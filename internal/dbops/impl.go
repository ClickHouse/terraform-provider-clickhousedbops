package dbops

import (
	"context"
	"sync"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type impl struct {
	clickhouseClient clickhouseclient.ClickhouseClient
	CapabilityFlags  *CapabilityFlags
	initOnce         sync.Once
	initErr          error
}

func NewClient(clickhouseClient clickhouseclient.ClickhouseClient) (Client, error) {
	return &impl{
		clickhouseClient: clickhouseClient,
		CapabilityFlags:  nil,
	}, nil
}

func (i *impl) initCapabilities(ctx context.Context) {
	i.initOnce.Do(func() {
		version, err := i.GetVersion(ctx)
		if err != nil {
			i.initErr = err
			return
		}
		i.CapabilityFlags = NewCapabilityFlags(version)
	})
}

// Returns initialized capability flags for the connected ClickHouse server.
func (i *impl) GetCapabilityFlags(ctx context.Context) (CapabilityFlags, error) {
	i.initCapabilities(ctx)

	if i.CapabilityFlags == nil {
		return CapabilityFlags{}, i.initErr
	}

	return *i.CapabilityFlags, i.initErr
}
