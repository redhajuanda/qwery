package qwery

import (
	"database/sql"
	"io"
	"reflect"

	"github.com/redhajuanda/qwery/vars"

	"github.com/redhajuanda/kuysor"

	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/pkg/errors"
	"github.com/redhajuanda/komon/logger"

	"github.com/jmoiron/sqlx"
	// "database/sql"
)

type responser struct {
	rows            *sqlx.Rows
	mapScanFunc     func(r rower, dest map[string]any) error
	jsonMarshalFunc func(v any) ([]byte, error)
	kuysor          *kuysor.Result
	tabling         *Tabling
	log             logger.Logger
}

// ScanStruct scans the first row of the result set into the provided struct
func (r *responser) ScanStruct(dest any) error {

	r.log.Debug("Scanning into struct")

	if dest == nil {
		return errors.New("destination cannot be nil")
	}

	// Get the type of the provided value
	vType := reflect.TypeOf(dest)

	// Ensure that v is a pointer to a struct
	if vType.Kind() != reflect.Ptr || vType.Elem().Kind() != reflect.Struct {
		return errors.New("destination must be a pointer to a struct")
	}

	defer r.rows.Close()

	// Initialize the dbscan API with the provided struct tag key and column separator
	api, err := dbscan.NewAPI(
		dbscan.WithStructTagKey(vars.TagKey),
		dbscan.WithColumnSeparator("__"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create new API")
	}

	// Scan one row into the struct, return an error if no rows are found and if the row is more than one
	err = api.ScanOne(dest, r.rows)
	if err != nil {
		if errors.Is(err, dbscan.ErrNotFound) {
			return sql.ErrNoRows
		}
		return errors.Wrap(err, "failed to scan struct")
	}

	return nil

}

// ScanMap scans the first row of the result set into the provided map
// The destination must be a map with string keys
func (r *responser) ScanMap(dest map[string]any) error {

	r.log.Debug("Scanning into map[string]any")

	if dest == nil {
		return errors.New("destination cannot be nil")
	}

	defer r.rows.Close()

	if !r.rows.Next() {
		return sql.ErrNoRows
	}

	// Use the mapScanFunc to scan the row into the map
	if err := r.mapScanFunc(r.rows, dest); err != nil {
		return err
	}

	return nil

}

// ScanStructs scans all rows of the result set into the provided slice of structs
// The destination must be a pointer to a slice of structs
func (r *responser) ScanStructs(dest any) error {

	r.log.Debug("Scanning into slice of structs")

	// Ensure v is a pointer to a slice
	sliceValue := reflect.ValueOf(dest)
	if sliceValue.Kind() != reflect.Ptr || sliceValue.Elem().Kind() != reflect.Slice {
		return errors.Errorf("destination must be a pointer to a slice of structs")
	}

	defer r.rows.Close()

	// initialize the dbscan API with the provided struct tag key and column separator
	api, err := dbscan.NewAPI(
		dbscan.WithStructTagKey(vars.TagKey),
		dbscan.WithColumnSeparator("__"),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create new API")
	}

	// Scan all rows into the slice of structs
	err = api.ScanAll(dest, r.rows)
	if err != nil {
		return errors.Wrap(err, "failed to scan structs")
	}

	// handle data cursor pagination
	if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.Type == "cursor" {
		next, prev, err := r.kuysor.SanitizeStruct(dest)
		if err != nil {
			return err
		}
		r.tabling.Pagination.BuildResponseCursor(next, prev, r.tabling.TotalData)
	} else if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.Type == "offset" {
		r.tabling.Pagination.BuildResponseOffset(r.tabling.TotalData)
	}

	return nil

}

// ScanMaps scans all rows of the result set into the provided slice of maps
// The destination must be a pointer to a slice of maps
func (r *responser) ScanMaps(dest *[]map[string]any) error {

	r.log.Debug("Scanning into slice of maps")

	defer r.rows.Close()

	// loop through the rows and scan each row into a map
	for r.rows.Next() {

		// Initialize the map
		s := make(map[string]any)

		// Use the mapScanFunc to scan the row into the map
		err := r.mapScanFunc(r.rows, s)
		if err != nil {
			return err
		}

		// Append the scanned value to the slice
		*dest = append(*dest, s)

	}

	// handle data cursor pagination
	if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.Type == "cursor" {
		next, prev, err := r.kuysor.SanitizeMap(dest)
		if err != nil {
			return err
		}

		r.tabling.Pagination.BuildResponseCursor(next, prev, r.tabling.TotalData)
	} else if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.Type == "offset" {
		r.tabling.Pagination.BuildResponseOffset(r.tabling.TotalData)
	}

	return nil

}

// ScanWriter scans all rows of the result set into the provided writer
// The destination must be a writer
func (r *responser) ScanWriter(dest io.Writer) error {

	r.log.Debug("Scanning into writer")

	result := make([]map[string]any, 0)

	// Scan the rows into a slice of maps
	err := r.ScanMaps(&result)
	if err != nil {
		return err
	}

	// Convert the result to JSON
	jsonResult, err := r.jsonMarshalFunc(result)
	if err != nil {
		return errors.Wrap(err, "failed to marshal result to JSON")
	}

	// Write the JSON to the writer
	_, err = dest.Write(jsonResult)
	if err != nil {
		return errors.Wrap(err, "failed to write JSON to writer")
	}

	return nil

}

// Close closes the rows
func (r *responser) Close() error {

	if r.rows != nil {
		return r.rows.Close()
	}

	return nil

}
