package qwery

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/qwery/parser"
	"github.com/redis/go-redis/v9"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Option struct {
	DB          *sql.DB
	QueryFiles  embed.FS
	DriverName  string
	Placeholder parser.Placeholder
	// Cache is an interface for caching.
	// Cache uses redis as the underlying caching library.
	Cache redis.Cmdable
}

// Init initializes a new sikat client.
func Init(log logger.Logger, opt Option) (*Client, error) {

	return initSikat(log, opt)

}

// initSikat initializes a new sikat client with the given options.
func initSikat(log logger.Logger, opt Option) (*Client, error) {

	db := sqlx.NewDb(opt.DB, opt.DriverName)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Initialize the runners map to store SQL queries
	var runners = make(map[string]string)

	// Walk through the embedded directory and its subdirectories
	err := fs.WalkDir(opt.QueryFiles, ".", func(path string, d fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Process only .sql files
		if strings.ToLower(filepath.Ext(path)) == ".sql" {
			// Read the SQL file from embed.FS
			content, err := fs.ReadFile(opt.QueryFiles, path)
			if err != nil {
				return fmt.Errorf("error reading file %s: %v", path, err)
			}

			// Get the directory name and file name without extension
			dir := filepath.Dir(path)
			fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

			// Use only the last directory name + file name as the key
			lastDir := filepath.Base(dir)
			key := lastDir + "." + fileName

			// Store the SQL query in the runners map
			runners[key] = string(content)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking through query location: %v", err)
	}

	return &Client{
		db:          &DB{DB: db},
		runners:     runners,
		placeholder: opt.Placeholder,
		log:         log,
		cache:       opt.Cache,
	}, nil

}

// InvalidateCache invalidates cache for the given key.
// It returns an error if there is a failure in invalidating cache.
// It uses the underlying cache object to invalidate cache.
func (c *Client) InvalidateCache(ctx context.Context, key string) error {

	keys := make([]string, 0)

	if strings.Contains(key, "*") {
		keys = c.cache.Keys(ctx, key).Val()
	} else {
		keys = append(keys, key)
	}

	for _, key := range keys {

		err := c.cache.Del(ctx, key).Err()
		if err != nil {
			err = errors.Wrapf(err, "failed to invalidate cache with key: %s", key)
			c.log.WithContext(ctx).WithStack(err).Error(err)
			return err
		}

	}
	return nil

}
