package dbops

import (
	"context"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type impl struct {
	clickhouseClient clickhouseclient.ClickhouseClient
	CapabilityFlags  *CapabilityFlags
}

func NewClient(clickhouseClient clickhouseclient.ClickhouseClient) (Client, error) {
	return &impl{
		clickhouseClient: clickhouseClient,
		CapabilityFlags:  nil,
	}, nil
}

// Retrieves the ClickHouse version and sets the capability flags accordingly.
// Used to determine which features are supported by the connected ClickHouse server.
func (i *impl) SetCapabilityFlags(ctx context.Context) error {
	version, err := i.GetVersion(ctx)
	if err != nil {
		return err
	}
	i.CapabilityFlags = NewCapabilityFlags(version)
	return nil
}

// Returns initialized capability flags for the connected ClickHouse server.
func (i *impl) GetCapabilityFlags() CapabilityFlags {
	if i.CapabilityFlags == nil {
		return CapabilityFlags{}
	}
	return *i.CapabilityFlags
}
