package clickhouseclient

import (
	"context"
	"errors"
)

var errConfigUnknown = errors.New("provider configuration depends on values that are not known until apply")

type unknownConfigClient struct{}

func NewUnknownConfigClient() ClickhouseClient {
	return unknownConfigClient{}
}

func (unknownConfigClient) Select(context.Context, string, func(Row) error) error {
	return errConfigUnknown
}

func (unknownConfigClient) Exec(context.Context, string, ...map[string]string) error {
	return errConfigUnknown
}
