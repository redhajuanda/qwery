package qwery

import (
	"context"

	"github.com/redhajuanda/komon/pagination"
	"github.com/redhajuanda/qwery/parser"
	"github.com/redhajuanda/qwery/vars"

	"github.com/VauntDev/tqla"
	"github.com/pkg/errors"
	"github.com/redhajuanda/kuysor"
)

var (
	// Question is a PlaceholderFormat instance that replaces placeholders with
	// question-prefixed positional placeholders (e.g. ?, ?, ?).
	Question = tqla.Question
	// Dollar is a PlaceholderFormat instance that replaces placeholders with
	// dollar-prefixed positional placeholders (e.g. $1, $2, $3).
	Dollar = tqla.Dollar
	// Colon is a PlaceholderFormat instance that replaces placeholders with
	// colon-prefixed positional placeholders (e.g. :1, :2, :3).
	Colon = tqla.Colon
	// AtP is a PlaceholderFormat instance that replaces placeholders with
	// "@p"-prefixed positional placeholders (e.g. @p1, @p2, @p3).
	AtP = tqla.AtP
)

// CTETargetMode controls where a specific SQL clause is injected when
// WithCTETarget is active. Mirrors kuysor.CTETargetMode.
type CTETargetMode int

const (
	// CTETargetDefault uses the natural default for each clause:
	// ORDER BY defaults to Both, LIMIT/OFFSET and WHERE default to CTE.
	CTETargetDefault CTETargetMode = iota
	// CTETargetCTE injects the clause inside the named CTE body only.
	CTETargetCTE
	// CTETargetMain injects the clause on the outer/main query only.
	CTETargetMain
	// CTETargetBoth injects the clause in both the CTE body and the main query.
	CTETargetBoth
)

// CTEOptions provides per-clause control over where each SQL modification is
// applied when WithCTETarget is active.
// Zero value (CTETargetDefault) uses natural defaults:
//   - OrderBy     → CTETargetBoth  (CTE body + mirrored on main)
//   - LimitOffset → CTETargetCTE   (CTE body only)
//   - Where       → CTETargetCTE   (CTE body only)
type CTEOptions struct {
	// OrderBy controls where ORDER BY is injected.
	OrderBy CTETargetMode
	// LimitOffset controls where LIMIT and OFFSET are injected.
	LimitOffset CTETargetMode
	// Where controls where the cursor WHERE clause is injected.
	Where CTETargetMode
}

type Tabling struct {
	OrderBy    []string
	Pagination *pagination.Pagination
	TotalData  *int
	CTETarget  string
	CTEOptions *CTEOptions
}

var placeholderMapping = map[parser.Placeholder]kuysor.PlaceHolderType{
	Question: kuysor.Question,
	Dollar:   kuysor.Dollar,
	Colon:    kuysor.Colon,
	AtP:      kuysor.At,
}

func buildTabling(pagination *pagination.Pagination) (*pagination.Pagination, error) {

	if pagination == nil {
		return nil, errors.New("pagination cannot be nil")
	}

	if pagination.PerPage <= 0 {
		pagination.PerPage = vars.DefaultPaginationPerPage
	}

	if pagination.Page <= 0 {
		pagination.Page = vars.DefaultPaginationPage
	}

	return pagination, nil

}

// mapCTEMode converts a qwery CTETargetMode to the equivalent kuysor.CTETargetMode.
func mapCTEMode(m CTETargetMode) kuysor.CTETargetMode {
	switch m {
	case CTETargetCTE:
		return kuysor.CTETargetModeCTE
	case CTETargetMain:
		return kuysor.CTETargetModeMain
	case CTETargetBoth:
		return kuysor.CTETargetModeBoth
	default:
		return kuysor.CTETargetModeDefault
	}
}

func processTabling(_ context.Context, client *Client, tabling *Tabling, query string, parameters ...any) (*kuysor.Result, error) {

	if tabling == nil {
		return &kuysor.Result{Query: query, Args: parameters}, nil
	}

	kys := kuysor.NewInstance(kuysor.Options{
		StructTag:       vars.TagKey,
		PlaceHolderType: placeholderMapping[client.placeholder],
		NullSortMethod:  kuysor.BoolSort,
		DefaultLimit:    vars.DefaultPaginationPage,
	})

	// if pagination is set, build kuysor query
	if tabling.Pagination != nil && tabling.Pagination.Type == "cursor" {

		ks := kys.NewQuery(query, kuysor.Cursor).
			WithArgs(parameters...)

		if len(tabling.OrderBy) > 0 {
			ks = ks.WithOrderBy(tabling.OrderBy...)
		} else {
			return nil, errors.New("orderBy is required when using cursor pagination")
		}

		ks = ks.WithLimit(tabling.Pagination.PerPage).
			WithCursor(tabling.Pagination.Cursor)

		if tabling.CTETarget != "" {
			if tabling.CTEOptions != nil {
				ks = ks.WithCTETarget(tabling.CTETarget, kuysor.CTEOptions{
					OrderBy:     mapCTEMode(tabling.CTEOptions.OrderBy),
					LimitOffset: mapCTEMode(tabling.CTEOptions.LimitOffset),
					Where:       mapCTEMode(tabling.CTEOptions.Where),
				})
			} else {
				ks = ks.WithCTETarget(tabling.CTETarget)
			}
		}

		res, err := ks.Build()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build kuysor query")
		}

		return res, nil

	} else if tabling.Pagination != nil && tabling.Pagination.Type == "offset" {
		ks := kys.NewQuery(query, kuysor.Offset).
			WithArgs(parameters...).
			WithOffset(tabling.Pagination.GetOffset())

		if len(tabling.OrderBy) > 0 {
			ks = ks.WithOrderBy(tabling.OrderBy...)
		}

		if tabling.CTETarget != "" {
			if tabling.CTEOptions != nil {
				ks = ks.WithCTETarget(tabling.CTETarget, kuysor.CTEOptions{
					OrderBy:     mapCTEMode(tabling.CTEOptions.OrderBy),
					LimitOffset: mapCTEMode(tabling.CTEOptions.LimitOffset),
					Where:       mapCTEMode(tabling.CTEOptions.Where),
				})
			} else {
				ks = ks.WithCTETarget(tabling.CTETarget)
			}
		}

		ks = ks.WithLimit(tabling.Pagination.PerPage)

		res, err := ks.Build()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build kuysor query")
		}

		return res, nil

	} else if len(tabling.OrderBy) > 0 {
		ks := kys.NewQuery(query, "").
			WithOrderBy(tabling.OrderBy...).
			WithArgs(parameters...)

		res, err := ks.Build()
		if err != nil {
			return nil, errors.Wrap(err, "failed to build kuysor query")
		}

		return res, nil
	}

	return nil, errors.New("pagination type not found")

}
