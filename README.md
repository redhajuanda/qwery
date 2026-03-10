# qwery

[![Go Version](https://img.shields.io/badge/Go-1.24.2+-blue.svg)](https://golang.org)

**qwery** is a Go library that simplifies database operations through SQL templates and type-safe result mapping. It provides a clean abstraction over raw SQL while maintaining full control over your queries.


## Features

- **SQL-First Design**: Write and organize your SQL queries in separate files embedded via `embed.FS`
- **Template-Based Queries**: Use Go templates for dynamic SQL generation
- **Type-Safe Scanning**: Automatic mapping between SQL results and Go structs
- **Transaction Support**: Built-in transaction management with automatic rollback
- **Pagination**: Support for both offset and cursor-based pagination
- **Parameter Binding**: Safe parameter substitution with SQL injection protection
- **Flexible Scanning**: Support for maps, structs, and custom writers
- **Caching**: Optional query result caching with configurable TTL
- **Database Agnostic**: Works with any database supported by Go's `database/sql`

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Usage Examples](#usage-examples)
- [API Reference](#api-reference)
- [Configuration](#configuration)

## Installation

```bash
go get github.com/redhajuanda/qwery
```

## Quick Start

### 1. Initialize the Client

SQL files must be embedded using Go's `embed` package and passed via `QueryFiles`.

```go
package main

import (
    "database/sql"
    "embed"
    "log"

    "github.com/redhajuanda/komon/logger"
    "github.com/redhajuanda/qwery"
    _ "github.com/go-sql-driver/mysql" // MySQL/MariaDB driver
)

//go:embed queries
var queryFiles embed.FS

func main() {
    logr := logger.New("main")
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?parseTime=true")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Initialize qwery client
    client, err := qwery.Init(logr, qwery.Option{
        DB:          db,
        QueryFiles:  queryFiles,  // Embedded SQL files
        DriverName:  "mysql",     // Must match the driver used in sql.Open()
        Placeholder: qwery.Question, // Use `?` placeholder for MySQL/MariaDB
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2. Create SQL Query Files

Embed a directory of `.sql` files using Go's `embed` package:

```
queries/
├── user/
│   ├── GetUser.sql
│   ├── CreateUser.sql
│   └── ListUsers.sql
└── product/
    ├── GetProduct.sql
    └── SearchProducts.sql
```

Example SQL file (`queries/user/GetUser.sql`):
```sql
SELECT id, name, email, created_at
FROM users
WHERE id = {{ .id }}
```

### 3. Execute Queries

```go
// Define your struct
type User struct {
    ID        int       `qwery:"id"`
    Name      string    `qwery:"name"`
    Email     string    `qwery:"email"`
    CreatedAt time.Time `qwery:"created_at"`
}

// Get a single user
var user User
err := client.Run("user.GetUser").
    WithParam("id", 1).
    ScanStruct(&user).
    Query(context.Background())

// Get multiple users
var users []User
err = client.Run("user.ListUsers").
    ScanStructs(&users).
    Query(context.Background())
```

## Core Concepts

### Query Organization

Qwery maps `.sql` files using dot notation based on the last directory name and filename:

- File: `queries/user/GetUser.sql` → Query name: `user.GetUser`
- File: `queries/product/SearchProducts.sql` → Query name: `product.SearchProducts`

> **Note**: Only the last directory segment is used as the prefix, regardless of nesting depth.

### Parameter Binding

Qwery uses Go templates for parameter substitution. Parameters are passed safely as bound values, preventing SQL injection:

```sql
-- queries/user/SearchUsers.sql
SELECT id, name, email
FROM users
WHERE name LIKE {{ .name }}
AND is_active = {{ .is_active }}
```

### Scanning Results

Qwery provides multiple ways to scan query results:

- **Structs**: `ScanStruct(&user)` / `ScanStructs(&users)`
- **Maps**: `ScanMap(map[string]any{})` / `ScanMaps(&[]map[string]any{})`
- **Custom Writers**: `ScanWriter(io.Writer)`

## Usage Examples

### Basic CRUD Operations

```go
// Create a user
type CreateUserParams struct {
    Name  string `qwery:"name"`
    Email string `qwery:"email"`
}

params := CreateUserParams{
    Name:  "John Doe",
    Email: "john@example.com",
}

result, err := client.Run("user.CreateUser").
    WithParams(params).
    Exec(context.Background())

lastID, _ := result.LastInsertId()

// Update a user
updateParams := map[string]any{
    "id":    1,
    "name":  "Jane Doe",
    "email": "jane@example.com",
}

_, err = client.Run("user.UpdateUser").
    WithParams(updateParams).
    Exec(context.Background())

// Delete a user
_, err = client.Run("user.DeleteUser").
    WithParam("id", 1).
    Exec(context.Background())
```

### Raw Queries

For one-off or dynamic queries that don't have a corresponding `.sql` file, use `RunRaw`:

```go
var users []User
err := client.RunRaw("SELECT id, name FROM users WHERE status = {{ .status }}").
    WithParam("status", "active").
    ScanStructs(&users).
    Query(context.Background())
```

### Pagination

```go
import "github.com/redhajuanda/komon/pagination"

// Offset-based pagination
offsetPagination := &pagination.Pagination{
    Type:    "offset",
    Page:    1,
    PerPage: 10,
}

var users []User
err := client.Run("user.ListUsers").
    WithPagination(offsetPagination).
    ScanStructs(&users).
    Query(context.Background())

// Cursor-based pagination
cursorPagination := &pagination.Pagination{
    Type:    "cursor",
    Cursor:  "eyJpZCI6MTAwfQ==",
    PerPage: 10,
}

err = client.Run("user.ListUsers").
    WithPagination(cursorPagination).
    WithOrderBy("id"). // Required for cursor pagination
    ScanStructs(&users).
    Query(context.Background())
```

### Cursor Pagination Details

When using cursor-based pagination, there are several important considerations:

#### Required Order By
Cursor pagination **requires** an `OrderBy` clause. The column name must match a struct field tag in the result, as this column's value is used as the cursor pointer to track the last query position.

#### Unique Order Combination
The combination of columns in the `OrderBy` clause must be **guaranteed unique**. If not, the pagination will become inconsistent and may skip or duplicate records.

#### Handling Nullable Columns
If the order column is nullable, specify `nullable` after the column name. For example: `name nullable` or `-name nullable`. Only one column can be marked as nullable in the `OrderBy` clause because cursor pagination requires deterministic ordering. Since nullable columns are definitely not unique, you must combine them with another unique column.

#### OrderBy Prefix
- `+column` or `column` — ascending order
- `-column` — descending order

```go
// Order by name ascending, created_at descending
err = client.Run("user.ListUsers").
    WithPagination(cursorPagination).
    WithOrderBy("name", "-created_at").
    ScanStructs(&users).
    Query(ctx)
```

### How Pagination and Ordering Append to Queries

`WithPagination` and `WithOrderBy` do not modify your SQL file directly. Instead, they **append** `ORDER BY`, `LIMIT`, `OFFSET`, and (for cursor pagination) `WHERE` clauses to the **parsed query string** at build time. The insertion points are determined by standard SQL clause order (e.g., `ORDER BY` before `LIMIT`, `LIMIT` before `OFFSET`). If a clause already exists, the new one replaces or extends it as appropriate.

#### Offset Pagination

For `Type: "offset"`, the builder appends:

1. **ORDER BY** — Appended to the main query (or inserted before `LIMIT`/`OFFSET` if they exist). Required for deterministic page ordering.
2. **LIMIT** — Appended after `ORDER BY`, using `PerPage` as the value.
3. **OFFSET** — Appended after `LIMIT`, using `(Page - 1) * PerPage`.

**Example:** Your query `SELECT * FROM users WHERE status = ?` becomes:

```sql
SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
```

Parameters are appended in order: `[status, limit, offset]`.

#### Cursor Pagination

For `Type: "cursor"`, the builder appends:

1. **ORDER BY** — Appended to the main query. **Required** for cursor pagination; the order columns define the cursor.
2. **WHERE** — On pages after the first, a condition is appended (e.g. `(id, created_at) > (?, ?)`) to filter rows after the cursor. Existing `WHERE` conditions are preserved and combined with `AND`.
3. **LIMIT** — Appended after `ORDER BY`, using `PerPage + 1` (the extra row indicates whether a next page exists).

**Example (first page):** `SELECT * FROM users WHERE status = ?` becomes:

```sql
SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ?
```

**Example (next page):** With cursor `eyJpZCI6MTAwfQ==` (decoded to `id=100`):

```sql
SELECT * FROM users WHERE status = ? AND (created_at < ? OR (created_at = ? AND id < ?)) ORDER BY created_at DESC LIMIT ?
```

Parameters: `[status, cursor_value, cursor_value, cursor_value, limit+1]` (exact shape depends on order columns).

#### CTE Targeting (`WithCTETarget`)

When your query uses a Common Table Expression (CTE), e.g. `WITH cte AS (SELECT ...) SELECT * FROM cte`, pagination clauses can be applied **inside the CTE body** instead of (or in addition to) the outer query. Use `WithCTETarget("cte")` to target the CTE.

**Default behavior (no `CTEOptions`):**

- **ORDER BY** — Appended to **both** the CTE body and the main query. The main query mirrors it so that `SELECT * FROM cte` returns rows in the correct order.
- **LIMIT / OFFSET** — Appended **inside the CTE only**. The CTE is trimmed first; the outer query returns all rows from the CTE.
- **WHERE** (cursor) — Appended **inside the CTE only**.

**Example:** `WITH cte AS (SELECT * FROM users WHERE status = ?) SELECT * FROM cte` with offset pagination becomes:

```sql
WITH cte AS (SELECT * FROM users WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?) SELECT * FROM cte ORDER BY created_at DESC
```

**Why:** Applying `LIMIT`/`OFFSET` inside the CTE reduces the work done before the outer query. Mirroring `ORDER BY` on the main query ensures the final result set is ordered correctly.

**Per-clause control:** Use `CTEOptions` to override where each clause is applied:

```go
WithCTETarget("cte", qwery.CTEOptions{
    OrderBy:     qwery.CTETargetMain,   // ORDER BY only on outer query
    LimitOffset: qwery.CTETargetBoth,    // LIMIT/OFFSET in both CTE and main
    Where:       qwery.CTETargetMain,   // cursor WHERE only on outer query
})
```

- `CTETargetCTE` — clause only inside the CTE body
- `CTETargetMain` — clause only on the outer query
- `CTETargetBoth` — clause in both (useful for `LIMIT`/`OFFSET` when you need to restrict both the CTE and the final result)

### Transactions

#### Managed Transaction (Recommended)

`WithTransaction` automatically commits on success and rolls back on error or panic:

```go
type User struct {
    ID    int    `qwery:"id"`
    Name  string `qwery:"name"`
    Email string `qwery:"email"`
}

type Order struct {
    ID     int `qwery:"id"`
    UserID int `qwery:"user_id"`
    Amount int `qwery:"amount"`
}

result, err := client.WithTransaction(context.Background(), func(tx *qwery.Tx) (any, error) {
    // Create user
    userParams := map[string]any{
        "name":  "John Doe",
        "email": "john@example.com",
    }

    userResult, err := tx.Run("user.CreateUser").WithParams(userParams).Exec(ctx)
    if err != nil {
        return nil, err
    }

    userID, _ := userResult.LastInsertId()

    // Create order
    orderParams := map[string]any{
        "user_id": userID,
        "amount":  100,
    }

    _, err = tx.Run("order.CreateOrder").WithParams(orderParams).Exec(ctx)
    if err != nil {
        return nil, err
    }

    return userID, nil
})
```

#### Manual Transaction Control

For more control, use `BeginTransaction` and manage commit/rollback yourself:

```go
tx, err := client.BeginTransaction(ctx)
if err != nil {
    return err
}

_, err = tx.Run("user.CreateUser").WithParams(params).Exec(ctx)
if err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

### Caching

Use `WithCache` to cache query results. Requires a `Cache` implementation passed during initialization:

```go
import "time"

var users []User
err := client.Run("user.ListUsers").
    WithCache("user:list", 5*time.Minute).
    ScanStructs(&users).
    Query(context.Background())

// Invalidate a specific cache key
err = client.InvalidateCache(ctx, "user:list")

```

### Working with JSON and Custom Types

qwery provides custom types for common database patterns:

#### JSONMap

`JSONMap` is a `map[string]any` type that implements `sql.Scanner` and `driver.Valuer` for seamless JSON column handling. Use it for JSON/JSONB columns:

```go
type User struct {
    ID          int     `qwery:"id"`
    Name        string  `qwery:"name"`
    Preferences qwery.JSONMap `qwery:"preferences"` // JSON column
}

// Parse JSON map to a struct
var user User
err := client.Run("user.GetUser").WithParam("id", 1).ScanStruct(&user).Query(ctx)

var prefs UserPreferences
err = user.Preferences.Parse(&prefs)
```

#### Time

`qwery.Time` wraps `time.Time` with custom JSON formatting (`2006-01-02T15:04:05.000` for JSON, `2006-01-02 15:04:05` for text). Use it when you need consistent time serialization without timezone suffixes in API responses:

```go
type Event struct {
    ID        int         `qwery:"id"`
    CreatedAt qwery.Time   `qwery:"created_at"`
}
```

### Complex Queries with Conditions

```sql
-- queries/user/SearchUsers.sql
SELECT id, name, email, created_at
FROM users
WHERE 1=1
  {{ if .name }}AND name LIKE {{ .name }}{{ end }}
  {{ if .email }}AND email LIKE {{ .email }}{{ end }}
  {{ if .is_active }}AND is_active = {{ .is_active }}{{ end }}
  {{ if .created_after }}AND created_at >= {{ .created_after }}{{ end }}
ORDER BY created_at DESC
  {{ if .limit }}LIMIT {{ .limit }}{{ end }}
```

**Template helpers:** SQL templates support `IsTimeZero` and `IsTimeNotZero` for conditional time checks, and `DerefBool` for pointer booleans:

```sql
{{ if IsTimeNotZero .created_after }}AND created_at >= {{ .created_after }}{{ end }}
{{ if DerefBool .is_active }}AND is_active = true{{ end }}
```

```go
params := map[string]any{
    "name":          "%john%",
    "is_active":     true,
    "created_after": time.Now().AddDate(0, -1, 0),
    "limit":         50,
}

var users []User
err := client.Run("user.SearchUsers").
    WithParams(params).
    ScanStructs(&users).
    Query(context.Background())
```

## Configuration

### Client Options

```go
type Option struct {
    DB          *sql.DB           // Required: database connection
    QueryFiles  embed.FS          // Required: embedded directory of .sql files
    DriverName  string            // Required: database driver name (e.g. "mysql", "postgres")
    Placeholder parser.Placeholder // Required: placeholder format for the driver
    Cache       cache.Cache       // Optional: caching implementation (komon/cache)
}
```

### Supported Placeholders

| Constant         | Format          | Databases                          |
|------------------|-----------------|------------------------------------|
| `qwery.Question` | `?`             | MySQL, MariaDB, SQLite, Snowflake  |
| `qwery.Dollar`   | `$1, $2, ...`   | PostgreSQL                         |
| `qwery.Colon`    | `:1, :2, ...`   | Oracle                             |
| `qwery.AtP`      | `@p1, @p2, ...` | SQL Server                         |

### Struct Tags

Use the `qwery` tag to map database columns to struct fields:

```go
type User struct {
    ID        int       `qwery:"id"`
    Name      string    `qwery:"name"`
    Email     string    `qwery:"email"`
    CreatedAt time.Time `qwery:"created_at"`
    UpdatedAt time.Time `qwery:"updated_at"`
}
```

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `Run(queryName string) Runnerer` | Start a query by named SQL file |
| `RunRaw(query string) Runnerer` | Start a query using a raw SQL string |
| `WithTransaction(ctx, TxFunc) (any, error)` | Execute a callback inside a managed transaction |
| `BeginTransaction(ctx) (*Tx, error)` | Begin a transaction with manual control |
| `InvalidateCache(ctx, key string) error` | Invalidate cache by key |
| `DB() *sql.DB` | Return the underlying `*sql.DB` |

### Tx Methods

| Method | Description |
|--------|-------------|
| `Run(queryName string) Runnerer` | Start a named query within the transaction |
| `RunRaw(query string) Runnerer` | Start a raw query within the transaction |
| `Commit() error` | Commit the transaction |
| `Rollback() error` | Roll back the transaction |

### Runner Methods

| Method | Description |
|--------|-------------|
| `WithParam(key string, value any) Runnerer` | Add a single parameter |
| `WithParams(params any) Runnerer` | Add parameters from a map or struct |
| `WithPagination(p *pagination.Pagination) Runnerer` | Add pagination |
| `WithOrderBy(orderBy ...string) Runnerer` | Set ordering columns |
| `WithCTETarget(cteName string, opts ...CTEOptions) Runnerer` | Target a CTE for pagination clauses; see [How Pagination and Ordering Append to Queries](#how-pagination-and-ordering-append-to-queries) |
| `WithCache(key string, ttl time.Duration) Runnerer` | Enable result caching |
| `ScanStruct(dest any) Runnerer` | Scan result into a single struct |
| `ScanStructs(dest any) Runnerer` | Scan results into a slice of structs |
| `ScanMap(dest map[string]any) Runnerer` | Scan result into a map |
| `ScanMaps(dest *[]map[string]any) Runnerer` | Scan results into a slice of maps |
| `ScanWriter(dest io.Writer) Runnerer` | Scan results into a writer |
| `Exec(ctx context.Context) (*ResultExec, error)` | Execute a non-SELECT query |
| `Query(ctx context.Context) error` | Execute a SELECT query and scan results |
| `Build(ctx context.Context) (*BuildResult, error)` | Compile query and params without executing; useful for testing |

### ResultExec

`ResultExec` embeds `sql.Result`, providing access to:

```go
result, err := client.Run("user.CreateUser").WithParams(params).Exec(ctx)

lastID, err := result.LastInsertId()
rowsAffected, err := result.RowsAffected()
```

### BuildResult

`BuildResult` holds the output of `Runner.Build()` — the compiled query and parameters without executing against the database:

```go
result, err := client.Run("user.ListUsers").
    WithPagination(pagination).
    WithOrderBy("created_at").
    Build(ctx)

// result.Query — final SQL with ORDER BY / LIMIT / OFFSET applied
// result.Params — positional arguments for the query
// result.CountQuery, result.CountParams — populated when CountTotalData is true on pagination
```

### NewTestClient

For unit testing without a database, use `NewTestClient`. Only `Build()` may be called on runners from this client:

```go
runners := map[string]string{
    "user.GetUser": "SELECT id, name FROM users WHERE id = {{ .id }}",
}
client := qwery.NewTestClient(logger, runners, qwery.Question)
result, err := client.Run("user.GetUser").WithParam("id", 1).Build(ctx)
```
