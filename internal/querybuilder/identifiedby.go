package querybuilder

import (
	"fmt"
)

type Identification string

const (
	IdentificationSHA256Hash Identification = "sha256_hash"
)

func Identified(with Identification, by string) Option {
	return &identified{
		with: with,
		by:   by,
	}
}

type identified struct {
	with Identification
	by   string
}

func (i *identified) String() string {
	return fmt.Sprintf("IDENTIFIED WITH %s BY %s", i.with, quote(i.by))
}
