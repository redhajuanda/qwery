package qwery

import (
	"context"
	"testing"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/komon/pagination"
	"github.com/stretchr/testify/assert"
)

func TestRunnerBuild(t *testing.T) {
	var (
		ctx     = context.Background()
		log     = logger.New("test")
		runners = map[string]string{
			"user.get": "SELECT * FROM users WHERE id = {{ .id }}",
		}
	)

	tests := []struct {
		name            string
		setup           func(client *Client) Runnerer
		wantQuery       string
		wantParams      []any
		wantCountQuery  string
		wantCountParams []any
		wantErr         bool
	}{
		{
			name: "raw query with single param",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE id = {{ .id }}`).
					WithParam("id", 1)
			},
			wantQuery:  "SELECT * FROM users WHERE id = ?",
			wantParams: []any{1},
		},
		{
			name: "raw query with optional param present",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE id = {{ .id }}{{- if .name }} AND name = {{ .name }}{{ end }}`).
					WithParam("id", 1).
					WithParam("name", "john")
			},
			wantQuery:  "SELECT * FROM users WHERE id = ? AND name = ?",
			wantParams: []any{1, "john"},
		},
		{
			name: "raw query with optional param absent",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE id = {{ .id }}{{- if .name }} AND name = {{ .name }}{{ end }}`).
					WithParam("id", 1)
			},
			wantQuery:  "SELECT * FROM users WHERE id = ?",
			wantParams: []any{1},
		},
		{
			name: "runner-code query",
			setup: func(c *Client) Runnerer {
				return c.Run("user.get").WithParam("id", 42)
			},
			wantQuery:  "SELECT * FROM users WHERE id = ?",
			wantParams: []any{42},
		},
		{
			name: "runner-code not found returns error",
			setup: func(c *Client) Runnerer {
				return c.Run("does.not.exist").WithParam("id", 1)
			},
			wantErr: true,
		},
		{
			name: "with offset pagination no count",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE status = {{ .status }}`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10})
			},
			wantQuery:      "SELECT * FROM users WHERE status = ? LIMIT ? OFFSET ?",
			wantParams:     []any{"active", 10, 0},
			wantCountQuery: "",
		},
		{
			name: "with offset pagination and CountTotalData",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE status = {{ .status }}`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10, CountTotalData: true})
			},
			wantQuery:       "SELECT * FROM users WHERE status = ? LIMIT ? OFFSET ?",
			wantParams:      []any{"active", 10, 0},
			wantCountQuery:  "SELECT COUNT(*) FROM users WHERE status = ?",
			wantCountParams: []any{"active"},
		},
		{
			name: "count query with CTE",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH filtered AS (SELECT id, name FROM users WHERE status = {{ .status }}) SELECT * FROM filtered`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10, CountTotalData: true})
			},
			wantQuery:       "WITH filtered AS (SELECT id, name FROM users WHERE status = ?) SELECT * FROM filtered LIMIT ? OFFSET ?",
			wantParams:      []any{"active", 10, 0},
			wantCountQuery:  "WITH filtered AS (SELECT id, name FROM users WHERE status = ?) SELECT COUNT(*) FROM filtered",
			wantCountParams: []any{"active"},
		},
		{
			name: "count query with JOIN",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT u.id, u.name FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE u.status = {{ .status }}`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10, CountTotalData: true})
			},
			wantQuery:       "SELECT u.id, u.name FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE u.status = ? LIMIT ? OFFSET ?",
			wantParams:      []any{"active", 10, 0},
			wantCountQuery:  "SELECT COUNT(*) FROM users u INNER JOIN orders o ON u.id = o.user_id WHERE u.status = ?",
			wantCountParams: []any{"active"},
		},
		{
			name: "count query with subquery in FROM",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM (SELECT id, name FROM users WHERE status = {{ .status }}) AS sub`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10, CountTotalData: true})
			},
			wantQuery:       "SELECT * FROM (SELECT id, name FROM users WHERE status = ?) AS sub LIMIT ? OFFSET ?",
			wantParams:      []any{"active", 10, 0},
			wantCountQuery:  "SELECT COUNT(*) FROM (SELECT id, name FROM users WHERE status = ?) AS sub",
			wantCountParams: []any{"active"},
		},
		{
			name: "count query with CTE and pagination in CTE",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT id, name FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10, CountTotalData: true}).
					WithCTETarget("cte")
			},
			wantQuery:       "WITH cte AS (SELECT id, name FROM users WHERE status = ? LIMIT ? OFFSET ?) SELECT * FROM cte",
			wantParams:      []any{"active", 10, 0},
			wantCountQuery:  "WITH cte AS (SELECT id, name FROM users WHERE status = ?) SELECT COUNT(*) FROM cte",
			wantCountParams: []any{"active"},
		},
		{
			name: "with order by",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`SELECT * FROM users WHERE status = {{ .status }}`).
					WithParam("status", "active").
					WithOrderBy("-created_at", "name")
			},
			wantQuery:  "SELECT * FROM users WHERE status = ? ORDER BY created_at DESC, name ASC",
			wantParams: []any{"active"},
		},
		{
			name: "with CTE target for offset pagination",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT * FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10}).
					WithCTETarget("cte")
			},
			wantQuery:  "WITH cte AS (SELECT * FROM users WHERE status = ? LIMIT ? OFFSET ?) SELECT * FROM cte",
			wantParams: []any{"active", 10, 0},
		},
		{
			name: "with CTE target for cursor pagination",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT * FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "cursor", PerPage: 10}).
					WithOrderBy("-created_at").
					WithCTETarget("cte")
			},
			wantQuery:  "WITH cte AS (SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ?) SELECT * FROM cte ORDER BY created_at DESC",
			wantParams: []any{"active", 11},
		},
		{
			name: "CTE target ORDER BY main only",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT * FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10}).
					WithOrderBy("-created_at").
					WithCTETarget("cte", CTEOptions{
						OrderBy: CTETargetMain,
					})
			},
			// LIMIT/OFFSET in CTE (default), ORDER BY only on outer query
			wantQuery:  "WITH cte AS (SELECT * FROM users WHERE status = ? LIMIT ? OFFSET ?) SELECT * FROM cte ORDER BY created_at DESC",
			wantParams: []any{"active", 10, 0},
		},
		{
			name: "CTE target LIMIT offset both",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT * FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "offset", Page: 1, PerPage: 10}).
					WithCTETarget("cte", CTEOptions{
						LimitOffset: CTETargetBoth,
					})
			},
			// LIMIT/OFFSET in both CTE body and outer query
			wantQuery:  "WITH cte AS (SELECT * FROM users WHERE status = ? LIMIT ? OFFSET ?) SELECT * FROM cte LIMIT ? OFFSET ?",
			wantParams: []any{"active", 10, 0, 10, 0},
		},
		{
			name: "CTE target cursor WHERE main only (first page)",
			setup: func(c *Client) Runnerer {
				return c.RunRaw(`WITH cte AS (SELECT * FROM users WHERE status = {{ .status }}) SELECT * FROM cte`).
					WithParam("status", "active").
					WithPagination(&pagination.Pagination{Type: "cursor", PerPage: 10}).
					WithOrderBy("-created_at").
					WithCTETarget("cte", CTEOptions{
						Where: CTETargetMain,
					})
			},
			// first page: no WHERE injection regardless of mode;
			// ORDER BY defaults to Both, LIMIT defaults to CTE — same shape as baseline cursor test
			wantQuery:  "WITH cte AS (SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ?) SELECT * FROM cte ORDER BY created_at DESC",
			wantParams: []any{"active", 11},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTestClient(log, runners, Question)
			result, err := tt.setup(client).Build(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.wantQuery != "" {
				assert.Equal(t, tt.wantQuery, result.Query)
			}
			if tt.wantParams != nil {
				assert.Equal(t, tt.wantParams, result.Params)
			}
			if tt.wantCountQuery != "" {
				assert.Equal(t, tt.wantCountQuery, result.CountQuery)
			} else {
				assert.Empty(t, result.CountQuery)
			}
			if tt.wantCountParams != nil {
				assert.Equal(t, tt.wantCountParams, result.CountParams)
			}
		})
	}
}
