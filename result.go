package qwery

import (
	"database/sql"
)

type ResultExec struct {
	sql.Result
}

// BuildResult holds the output of Runner.Build — the fully compiled query and
// parameters exactly as they would be sent to the database, without executing.
type BuildResult struct {
	// Query is the final SQL string (with ORDER BY / pagination applied if set).
	Query string
	// Params are the positional arguments that accompany Query.
	Params []any
	// CountQuery is only populated when CountTotalData is true on the pagination.
	// Built via kuysor.NewCount: replaces the main SELECT with COUNT(*), preserving
	// CTEs, JOINs, WHERE, GROUP BY, etc. Uses the base parsed query (before pagination).
	CountQuery string
	// CountParams are the positional arguments that accompany CountQuery.
	// They match the base parsed params (before pagination is applied).
	CountParams []any
}
