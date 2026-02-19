package qwery

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/redhajuanda/komon/logger"

	"github.com/pkg/errors"
	"github.com/redhajuanda/komon/cache"
	"github.com/redhajuanda/qwery/mapper"
)

type CacheData struct {
	Dest any `sika:"dest" json:"dest"`
}

// Cacher is a struct that contains the cacher
type Cacher struct {

	// key is the key of the cacher
	key string

	// ttl is the time to live of the cache
	ttl time.Duration

	// cache
	cache cache.Cache

	// logger
	log logger.Logger

	runner *Runner

	// doCache is a flag to determine whether the cache should be used or not
	doCache bool
}

// newCacher returns a new cacher
func newCacher(key string, ttl time.Duration, cache cache.Cache, log logger.Logger, runner *Runner) *Cacher {

	return &Cacher{
		key:     key,
		ttl:     ttl,
		cache:   cache,
		log:     log,
		runner:  runner,
		doCache: true,
	}

}

// tryCache tries to get the object from the cache
func (c *Cacher) tryCache(ctx context.Context) bool {

	// if cacher is nil, return false
	if !c.doCache {
		return false
	}

	c.log.WithContext(ctx).SkipSource().Debug("getting object from cache")

	switch c.runner.scanner.scannerType {
	case scannerWriter:

		exists, err := c.tryCacheHandlerWriter(ctx)
		if err != nil {
			c.log.WithContext(ctx).SkipSource().WithStack(err).Error(err)
			return false
		}

		return exists

	default:

		exists, err := c.tryCacheHandlerDefault(ctx)
		if err != nil {
			c.log.WithContext(ctx).SkipSource().WithStack(err).Error(err)
			return false
		}

		return exists
	}

}

// tryCacheHandlerWriter attempts to retrieve data from the cache using the specified key and writes it to the destination writer.
// If the data is successfully retrieved from the cache, it is copied to the destination writer.
// If an error occurs during the retrieval or copying process, an error is returned.
// If the data is not found in the cache, a false value is returned.
// If any other error occurs, it is logged and returned.
func (c *Cacher) tryCacheHandlerWriter(ctx context.Context) (bool, error) {

	data, err := c.cache.Get(ctx, c.key)
	if err == nil {

		buf := bytes.NewBuffer(data)

		_, err = io.Copy(c.runner.scanner.dest.(io.Writer), buf)
		if err != nil {
			return false, errors.Wrap(err, "cannot copy writer")
		}

		return true, nil
	}

	if !errors.Is(err, cache.NotFound) { // if error occurs and error is not cache not found, log error
		err = errors.Wrapf(err, "failed to get object from cache with key %s", c.key)
		return false, err
	}

	return false, nil

}

// tryCacheHandlerDefault tries to retrieve data from cache and decode it into the appropriate structures.
// If the data is successfully retrieved and decoded, it returns true and no error.
// If there is an error during the retrieval or decoding process, it returns false and the corresponding error.
func (c *Cacher) tryCacheHandlerDefault(ctx context.Context) (bool, error) {

	var (
		data = CacheData{}
	)

	// get object from cache
	err := c.cache.GetObject(ctx, c.key, &data)
	if err == nil {

		err := mapper.Decode(data.Dest, &c.runner.scanner.dest)
		if err != nil {
			return false, err
		}

		return true, nil // if no error occurs, return response from cache
	}

	if !errors.Is(err, cache.NotFound) { // if error occurs and error is not cache not found, log error
		err = errors.Wrapf(err, "failed to get object from cache with key %s", c.key)
		return false, err
	}

	return false, nil
}

// setCache sets the object to the cache
func (c *Cacher) setCache(ctx context.Context) error {

	const (
		latencyType = "set_cache"
	)

	// if caching is nil, return nil
	if !c.doCache {
		return nil
	}

	c.log.WithContext(ctx).SkipSource().Debug("setting object to cache")

	return c.setCacheHandler(ctx)

}

// setCacheHandler sets the cache for the Cacher instance.
func (c *Cacher) setCacheHandler(ctx context.Context) error {

	ttl := c.ttl

	dest := new(interface{})

	switch c.runner.scanner.scannerType {

	case scannerWriter:

		dest = &c.runner.scanner.dest

	case scannerStruct:

		destMap := make(map[string]interface{})

		// decode dest
		err := mapper.Decode(c.runner.scanner.dest, &destMap)
		if err != nil {
			return err
		}

		destInterface := interface{}(destMap)
		dest = &destInterface

	case scannerStructs:

		destSlice := make([]map[string]interface{}, 0)

		// decode dest
		err := mapper.Decode(c.runner.scanner.dest, &destSlice)
		if err != nil {
			return err
		}

		destInterface := interface{}(destSlice)
		dest = &destInterface

	default:

		// decode dest
		err := mapper.Decode(c.runner.scanner.dest, &dest)
		if err != nil {
			return err
		}

	}

	data := CacheData{
		Dest: dest,
	}

	dataMap := make(map[string]interface{})

	// decode data
	err := mapper.Decode(data, &dataMap)
	if err != nil {
		return err
	}

	// set object to cache
	err = c.cache.Set(ctx, c.key, dataMap, ttl)
	if err != nil {
		return errors.Wrap(err, "failed to set object to cache")
	}

	return nil
}
