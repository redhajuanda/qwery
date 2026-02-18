package qwery

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/qwery/parser"

	"github.com/jmoiron/sqlx"

	"github.com/pkg/errors"
)

type Runnerer interface {
	// WithParams initializes a new query with params.
	// Params can be a map or a struct, doesn't matter if you pass its pointer or its value.
	WithParams(any) Runnerer
	// WithParam initializes a new query with param.
	// Param is a key-value pair.
	// The key is the parameter name, and the value is the parameter value.
	// If the parameter already exists, it will be overwritten.
	WithParam(key string, value any) Runnerer
	// WithPagination adds pagination to the query.
	// pagination is a Pagination struct that contains the pagination options.
	// Pagination can be nil, in which case it will not be added to the query.
	// If you use cursor pagination, you also required to set the order by using WithOrderBy.
	WithPagination(pagination *pagination.Pagination) Runnerer
	// WithOrderBy initializes a new query with order by.
	// orderBy is a list of columns to order by.
	// It can be a single column or multiple columns.
	// The order can be ascending or descending using prefix: "+" for ascending and "-" for descending.
	// If no prefix is used, it will default to ascending order.
	// Example: WithOrderBy("name", "-created_at") will order by name ascending and created_at descending.
	WithOrderBy(orderBy ...string) Runnerer
	// WithCache initializes a new runner with cache.
	// key is the cache key.
	// ttl is the cache time to live.
	// If ttl is not specified, it will use the default ttl from runner config.
	// If ttl is specified, it will override the default ttl from runner config.
	WithCache(key string, ttl time.Duration) Runnerer
	// ScanMap initializes a runner with scanner map.
	// dest is the destination of the scanner.
	// It must be a map.
	ScanMap(dest map[string]any) Runnerer
	// ScanMaps initializes a runner with scanner maps.
	// dest is the destination of the scanner.
	// It must be a pointer to a slice of maps.
	// If you want to scan a single map, use ScanMap instead.
	ScanMaps(dest *[]map[string]any) Runnerer
	// ScanStruct initializes a runner with scanner struct.
	// dest is the destination of the scanner.
	// It must be a pointer to a struct.
	ScanStruct(dest any) Runnerer
	// ScanStructs initializes a runner with scanner structs.
	// dest is the destination of the scanner.
	// It must be a pointer to a slice of structs.
	ScanStructs(dest any) Runnerer
	// ScanWriter initializes a runner with scanner writer.
	// dest is the destination of the scanner.
	ScanWriter(dest io.Writer) Runnerer
	// Exec executes the query and returns the result.
	// It returns a ResultExec struct that contains the result of the execution.
	Exec(ctx context.Context) (*ResultExec, error)
	// Query executes the query and scans the result to the destination.
	// The destination must be set using ScanMap, ScanMaps, ScanStruct, ScanStructs, or ScanWriter.
	Query(ctx context.Context) error
}

type queryType int

const (
	queryTypeRaw queryType = iota
	queryTypeRunner
)

// // Runner is a struct that contains runner configs to be executed.
type Runner struct {
	runnerCode    string
	queryRaw      string
	queryType     queryType
	params        map[string]any
	client        *Client
	log           logger.Logger
	inTransaction bool
	tx            *Tx
	cacher        *Cacher
	scanner       *Scanner
	tabling       *Tabling
	errs          []error
}

type runnerParams struct {
	queryType     queryType
	runnerCode    string
	queryRaw      string
	client        *Client
	log           logger.Logger
	inTransaction bool
	tx            *Tx
}

// newRunner returns a new Runner.
func newRunner(runnerParams runnerParams) *Runner {

	return &Runner{
		queryType:     runnerParams.queryType,
		runnerCode:    runnerParams.runnerCode,
		queryRaw:      runnerParams.queryRaw,
		client:        runnerParams.client,
		params:        make(map[string]any),
		log:           runnerParams.log,
		inTransaction: runnerParams.inTransaction,
		tx:            runnerParams.tx,
		cacher:        &Cacher{},
	}

}

// WithParam initializes a new query with param.
// Param is a key-value pair.
// The key is the parameter name, and the value is the parameter value.
// If the parameter already exists, it will be overwritten.
func (r *Runner) WithParam(key string, value any) Runnerer {

	r.params[key] = value
	return r

}

// WithParams initializes a new query with params.
// Params can be a map or a struct, doesn't matter if you pass its pointer or its value.
func (r *Runner) WithParams(params any) Runnerer {

	// check if params is a map
	if p, ok := params.(map[string]any); ok {
		r.params = p
		return r
	}

	// check if params is a pointer to a map
	if p, ok := params.(*map[string]any); ok {
		r.params = *p
		return r
	}

	// check if params is a struct
	if isStruct(params) {

		mappedParams := StructToMap(params)
		maps.Copy(r.params, mappedParams)

		return r

	}

	r.errs = append(r.errs, errors.New("params must be a map or a struct"))
	return r

}

// WithPagination adds pagination to the query.
func (r *Runner) WithPagination(pagination *pagination.Pagination) Runnerer {

	pg, err := buildTabling(pagination)
	if err != nil {
		r.errs = append(r.errs, err)
	}

	if r.tabling == nil {
		r.tabling = &Tabling{
			Pagination: pg,
		}
	} else {
		r.tabling.Pagination = pg
	}

	return r

}

// WithOrderBy initializes a new query with order by.
// orderBy is a list of columns to order by.
// It can be a single column or multiple columns.
// The order can be ascending or descending using prefix: "+" for ascending and "-" for descending.
// If no prefix is used, it will default to ascending order.
// Example: WithOrderBy("name", "-created_at") will order by name ascending and created_at descending.
func (r *Runner) WithOrderBy(orderBy ...string) Runnerer {

	if r.tabling == nil {
		r.tabling = &Tabling{
			OrderBy: orderBy,
		}
	} else {
		r.tabling.OrderBy = orderBy
	}
	return r

}

// WithCache initializes a new runner with cache.
// key is the cache key.
// ttl is the cache time to live.
// If ttl is not specified, it will use the default ttl from runner config.
// If ttl is specified, it will override the default ttl from runner config.
func (r *Runner) WithCache(key string, ttl time.Duration) Runnerer {

	ttlDuration := ttl
	r.cacher = newCacher(key, ttlDuration, r.client.cache, r.log, r)
	return r

}

// ScanMap initializes a runner with scanner map.
// dest is the destination of the scanner.
// It must be a map.
func (r *Runner) ScanMap(dest map[string]any) Runnerer {

	r.scanner = newScanner(scannerMap, dest)
	return r

}

// ScanMaps initializes a runner with scanner maps.
// dest is the destination of the scanner.
// It must be a pointer to a slice of maps.
func (r *Runner) ScanMaps(dest *[]map[string]any) Runnerer {

	r.scanner = newScanner(scannerMaps, dest)
	return r

}

// ScanStruct initializes a runner with scanner struct.
// dest is the destination of the scanner.
// It must be a pointer to a struct.
func (r *Runner) ScanStruct(dest any) Runnerer {

	r.scanner = newScanner(scannerStruct, dest)
	return r

}

// ScanStructs initializes a runner with scanner structs.
// dest is the destination of the scanner.
// It must be a pointer to a slice of structs.
func (r *Runner) ScanStructs(dest any) Runnerer {

	r.scanner = newScanner(scannerStructs, dest)
	return r

}

// ScanWriter initializes a runner with scanner writer.
// dest is the destination of the scanner.
// It must be a writer.
func (r *Runner) ScanWriter(dest io.Writer) Runnerer {

	r.scanner = newScanner(scannerWriter, dest)
	return r

}

// Exec executes the query and returns the result.
func (r *Runner) Exec(ctx context.Context) (*ResultExec, error) {

	var (
		ps     = parser.New()
		result sql.Result
	)

	queryTemplate, err := r.getQueryTemplate()
	if err != nil {
		return nil, err
	}

	r.log.WithContext(ctx).WithParams(map[string]any{
		"runner_code": r.getRunnerCode(),
		"params":      r.params,
		"placeholder": r.client.placeholder,
	}).Debug("Parsing query")

	// parse query
	query, parameters, err := ps.Parse(ctx, queryTemplate, r.params, r.client.placeholder)
	if err != nil {
		return nil, err
	}

	if r.inTransaction {

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.getRunnerCode(), "query": query, "params": parameters}).Info("Executing query in transaction")

		// if in transaction, use the transaction context
		if r.tx == nil {
			return nil, errors.New("transaction not properly initialized")
		}
		tx := r.tx.GetTx()
		if tx == nil {
			return nil, errors.New("transaction not properly initialized")
		}

		// execute query
		result, err = tx.ExecContext(ctx, query, parameters...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute query in transaction")
		}

	} else {

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.getRunnerCode(), "query": query, "params": parameters}).Info("Executing query")

		// execute query
		result, err = r.client.db.ExecContext(ctx, query, parameters...)
		if err != nil {
			return nil, err
		}
	}

	return &ResultExec{
		result,
	}, nil

}

// Query executes the query and scans the result to the destination.
func (r *Runner) Query(ctx context.Context) error {

	var (
		ps              = parser.New()
		rows            *sqlx.Rows
		totalData       int
		queryFinal      string
		parametersFinal []any
	)

	// try to get object from cache, if exists, return response from cache
	exists := r.cacher.tryCache(ctx)
	if exists {
		return nil
	}

	queryTemplate, err := r.getQueryTemplate()
	if err != nil {
		return err
	}

	r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.getRunnerCode(), "params": r.params, "placeholder": r.client.placeholder}).Debug("Parsing query")

	// parse query
	queryParsed, parametersParsed, err := ps.Parse(ctx, queryTemplate, r.params, r.client.placeholder)
	if err != nil {
		return err
	}

	rs, err := processTabling(ctx, r.client, r.tabling, queryParsed, parametersParsed...)
	if err != nil {
		return errors.Wrap(err, "failed to build pagination cursor")
	}

	queryFinal = rs.Query
	parametersFinal = rs.Args

	// replace new line, tab, and carriage return with space
	replacer := strings.NewReplacer("\n", " ", "\t", " ", "\r", " ") // also handle carriage returns
	queryParsed = replacer.Replace(queryParsed)
	queryFinal = replacer.Replace(queryFinal)

	if r.inTransaction {

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode, "query": queryFinal, "params": parametersFinal}).Info("Querying query in transaction")

		// if in transaction, use the transaction context
		if r.tx == nil {
			return errors.New("transaction not properly initialized")
		}
		tx := r.tx.GetTx()
		if tx == nil {
			return errors.New("transaction not properly initialized")
		}

		// execute query
		rows, err = tx.QueryxContext(ctx, queryFinal, parametersFinal...)
		if err != nil {
			return errors.Wrap(err, "failed to execute query in transaction")
		}

		if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.CountTotalData {

			countQuery := fmt.Sprintf(`SELECT COUNT(1) AS total_data FROM ( %s ) AS sub`, queryParsed)

			r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode, "query": countQuery, "params": parametersParsed}).Info("Querying count query")

			countRow := tx.QueryRowxContext(ctx, countQuery, parametersParsed...)
			err = countRow.Scan(&totalData)
			if err != nil {
				return errors.Wrap(err, "failed to execute count query for offset pagination")
			}
			r.tabling.TotalData = &totalData
		}

	} else {

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode, "query": queryFinal, "params": parametersFinal}).Info("Querying query")

		// execute query
		rows, err = r.client.db.QueryxContext(ctx, queryFinal, parametersFinal...)
		if err != nil {
			return err
		}

		if r.tabling != nil && r.tabling.Pagination != nil && r.tabling.Pagination.CountTotalData {

			countQuery := fmt.Sprintf(`SELECT COUNT(1) AS total_data FROM ( %s ) AS sub`, queryParsed)

			r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode, "query": countQuery, "params": parametersParsed}).Info("Querying count query for offset pagination")

			countRow := r.client.db.QueryRowxContext(ctx, countQuery, parametersParsed...)
			err = countRow.Scan(&totalData)
			if err != nil {
				return errors.Wrap(err, "failed to execute count query for offset pagination")
			}

			r.tabling.TotalData = &totalData
		}
	}

	responser := &responser{
		rows:        rows,
		mapScanFunc: MapScan,
		jsonMarshalFunc: func(v interface{}) ([]byte, error) {
			return json.Marshal(v)
		},
		kuysor:  rs,
		tabling: r.tabling,
		log:     r.log,
	}

	err = r.scan(ctx, responser)
	if err != nil {
		return err
	}

	// httpData := result.Metadata.HTTP
	err = r.cacher.setCache(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to set cache")
	}

	return nil

}

// scan scans the result to the destination.
func (r *Runner) scan(ctx context.Context, sc Scannerer) error {

	if r.scanner == nil {
		r.scanner = newScanner(noScanner, nil)
	}

	// scan result
	switch r.scanner.scannerType {
	case scannerMap:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("scanning result into scanner map")

		err := sc.ScanMap(r.scanner.dest.(map[string]any))
		if err != nil {
			return err
		}

	case scannerMaps:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("scanning result into scanner maps")

		err := sc.ScanMaps(r.scanner.dest.(*[]map[string]any))
		if err != nil {
			return err
		}

	case scannerStruct:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("scanning result into scanner struct")

		err := sc.ScanStruct(r.scanner.dest)
		if err != nil {
			return err
		}

	case scannerStructs:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("scanning result into scanner structs")

		err := sc.ScanStructs(r.scanner.dest)
		if err != nil {
			return err
		}

	case scannerWriter:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("scanning result into scanner writer")

		err := sc.ScanWriter(r.scanner.dest.(io.Writer))
		if err != nil {
			return err
		}

	default:

		r.log.WithContext(ctx).WithParams(map[string]any{"runner_code": r.runnerCode}).Debug("no scanner type found, closing scanner")

		err := sc.Close()
		if err != nil {
			return err
		}

	}

	return nil
}

// getQueryTemplate returns the query template based on the query type.
func (r *Runner) getQueryTemplate() (string, error) {

	switch r.queryType {
	case queryTypeRunner:
		// check if runner code is valid
		if _, ok := r.client.runners[r.runnerCode]; !ok {
			return "", errors.New(fmt.Sprintf("runner [%s] does not exist", r.runnerCode))
		}
		return r.client.runners[r.runnerCode], nil // return runner query
	case queryTypeRaw:
		// check if query is empty
		if r.queryRaw == "" {
			return "", errors.New("query cannot be empty")
		}
		return r.queryRaw, nil // return raw query
	}

	return "", errors.New("invalid query type") // return error if query type is invalid

}

func (r *Runner) getRunnerCode() string {

	switch r.queryType {
	case queryTypeRunner:
		return r.runnerCode
	case queryTypeRaw:
		return "RAW"
	}

	return ""
}
