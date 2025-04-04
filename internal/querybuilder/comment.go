package querybuilder

import (
	"fmt"
)

type commentOption struct {
	comment string
}

func Comment(comment string) Option {
	return &commentOption{
		comment: comment,
	}
}

func (c *commentOption) String() string {
	return fmt.Sprintf("COMMENT %s", quote(c.comment))
}
