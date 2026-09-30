package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProviderSchema_HasTimeoutAttribute(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	attr, ok := resp.Schema.Attributes["dial_timeout"]
	if !ok {
		t.Fatal("expected 'dial_timeout' attribute in provider schema, not found")
	}

	if attr.IsRequired() {
		t.Error("'dial_timeout' attribute should be optional, not required")
	}
}

func TestProviderSchema_HasReadAfterWriteTimeoutAttribute(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	if _, ok := resp.Schema.Attributes["read_after_write_timeout"]; !ok {
		t.Fatal("expected 'read_after_write_timeout' attribute in provider schema, not found")
	}
}

func TestProviderSchema_RequiredAttributes(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	required := []string{"protocol", "host", "port", "auth_config"}
	for _, name := range required {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected required attribute %q not found", name)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be required", name)
		}
	}
}

func TestProviderSchema_TLSConfigHasServerNameAttribute(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	tlsAttr, ok := resp.Schema.Attributes["tls_config"]
	if !ok {
		t.Fatal("expected 'tls_config' attribute in provider schema, not found")
	}

	nested, ok := tlsAttr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected 'tls_config' to be a SingleNestedAttribute, got %T", tlsAttr)
	}

	if _, ok := nested.Attributes["server_name"]; !ok {
		t.Fatal("expected 'server_name' attribute in tls_config schema, not found")
	}
}

func TestBuildTLSConfig_ServerNameUnset(t *testing.T) {
	cfg := &TLSConfig{
		InsecureSkipVerify: types.BoolNull(),
		CACert:             types.StringNull(),
		ServerName:         types.StringNull(),
	}

	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("buildTLSConfig returned an error: %v", err)
	}

	if tlsConfig.ServerName != "" {
		t.Errorf("expected empty ServerName when server_name is unset, got %q", tlsConfig.ServerName)
	}
}

func TestBuildTLSConfig_ServerNameSet(t *testing.T) {
	cfg := &TLSConfig{
		InsecureSkipVerify: types.BoolNull(),
		CACert:             types.StringNull(),
		ServerName:         types.StringValue("real-hostname.clickhouse.cloud"),
	}

	tlsConfig, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("buildTLSConfig returned an error: %v", err)
	}

	if tlsConfig.ServerName != "real-hostname.clickhouse.cloud" {
		t.Errorf("expected ServerName %q, got %q", "real-hostname.clickhouse.cloud", tlsConfig.ServerName)
	}
}

func TestBuildTLSConfig_NilTLSConfig(t *testing.T) {
	tlsConfig, err := buildTLSConfig(nil)
	if err != nil {
		t.Fatalf("buildTLSConfig returned an error: %v", err)
	}

	if tlsConfig.ServerName != "" {
		t.Errorf("expected empty ServerName when tls_config is nil, got %q", tlsConfig.ServerName)
	}
}

func TestProviderSchema_PasswordIsSensitive(t *testing.T) {
	p := &Provider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema returned errors: %v", resp.Diagnostics.Errors())
	}

	authAttr, ok := resp.Schema.Attributes["auth_config"]
	if !ok {
		t.Fatal("expected 'auth_config' attribute in provider schema, not found")
	}

	nested, ok := authAttr.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected 'auth_config' to be a SingleNestedAttribute, got %T", authAttr)
	}

	password, ok := nested.Attributes["password"]
	if !ok {
		t.Fatal("expected 'password' attribute in auth_config schema, not found")
	}

	if !password.IsSensitive() {
		t.Error("'password' attribute should be sensitive")
	}
}
