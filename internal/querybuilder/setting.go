package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

const (
	writabilityConst      = "CONST"
	writabilityWritable   = "WRITABLE"
	writabilityChangeable = "CHANGEABLE_IN_READONLY"
)

type settingData struct {
	Name        string
	Value       *string
	Min         *string
	Max         *string
	Writability *string
}

type setting interface {
	SQLDef() (string, error)
}

func (s *settingData) SQLDef() (string, error) {
	if s.Name == "" {
		return "", errors.New("Name can't be empty")
	}

	if s.Value == nil && s.Min == nil && s.Max == nil {
		return "", errors.New("Either Value, Min or Max should be set")
	}

	if s.Writability != nil && *s.Writability != writabilityConst && *s.Writability != writabilityWritable && *s.Writability != writabilityChangeable {
		return "", errors.New(fmt.Sprintf("Invalid value for Writability. Can be %q, %q or %q", writabilityConst, writabilityWritable, writabilityChangeable))
	}

	singleSetting := make([]string, 0)
	singleSetting = append(singleSetting, backtick(s.Name))
	if s.Value != nil {
		singleSetting = append(singleSetting, "=", quote(*s.Value))
	}
	if s.Min != nil {
		singleSetting = append(singleSetting, "MIN", quote(*s.Min))
	}
	if s.Max != nil {
		singleSetting = append(singleSetting, "MAX", quote(*s.Max))
	}
	if s.Writability != nil {
		singleSetting = append(singleSetting, *s.Writability)
	}

	return strings.Join(singleSetting, " "), nil
}
