package dbops

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

// NormalizeExpression normalizes expression using ClickHouse formatQuerySingleLine function.
func (i *impl) NormalizeExpression(ctx context.Context, expression string) (string, error) {
	const prefix = "SELECT "
	sql := fmt.Sprintf("SELECT formatQuerySingleLine(concat('%s', %s)) AS formatted", prefix, chStringLiteral(expression))

	var formatted string
	found := false
	err := i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		v, err := data.GetString("formatted")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'formatted' field")
		}
		formatted = v
		found = true
		return nil
	})
	if err != nil {
		return expression, errors.WithMessage(err, "error running query")
	}
	if !found || !strings.HasPrefix(formatted, prefix) {
		return expression, nil
	}
	return strings.TrimPrefix(formatted, prefix), nil
}

// chStringLiteral renders s as a single-quoted ClickHouse string literal with backslash escaping.
func chStringLiteral(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}
