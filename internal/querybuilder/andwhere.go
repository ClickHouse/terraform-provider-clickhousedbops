package querybuilder

import (
	"fmt"
	"strings"
)

func AndWhere(clauses ...Where) Where {
	return &andWhere{
		clauses: clauses,
	}
}

type andWhere struct {
	clauses []Where
}

func (s *andWhere) Clause() string {
	tokens := make([]string, 0)

	for _, c := range s.clauses {
		tokens = append(tokens, strings.TrimPrefix(c.Clause(), "WHERE "))
	}

	return fmt.Sprintf("WHERE (%s)", strings.Join(tokens, " AND "))
}
