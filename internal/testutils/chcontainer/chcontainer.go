// Package chcontainer provides the ClickHouse server tests run against.
package chcontainer

import (
	"cmp"
	"crypto/tls"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/tests"
)

const (
	nativePort = "9000/tcp"
	httpPort   = "8123/tcp"
)

// Server is a ClickHouse server reachable from tests.
type Server struct {
	Host       string
	NativePort uint16
	HTTPPort   uint16
	Username   string
	Password   string
	Secure     bool
}

// Start returns the TEST_CLICKHOUSE_HOST server, or a container of version (CLICKHOUSE_VERSION, then latest, by default) terminated with the test.
func Start(t testing.TB, version string) Server {
	t.Helper()

	return StartWithConfig(t, version, tests.ConfigSingle)
}

// StartWithConfig is Start with the container running config; the external server is skipped for any other config.
func StartWithConfig(t testing.TB, version, config string) Server {
	t.Helper()

	if host := os.Getenv("TEST_CLICKHOUSE_HOST"); host != "" {
		if version != "" {
			t.Skipf("requires ClickHouse %s, the external server can't be pinned", version)
		}
		if config != tests.ConfigSingle {
			t.Skip("requires a custom server config, the external server can't be configured")
		}
		return external(t, host)
	}

	ctx := t.Context()
	ctr, err := testcontainers.Run(ctx, "clickhouse/clickhouse-server:"+cmp.Or(version, os.Getenv("CLICKHOUSE_VERSION"), "latest"),
		testcontainers.WithEnv(map[string]string{"CLICKHOUSE_SKIP_USER_SETUP": "1"}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			Reader:            strings.NewReader(config),
			ContainerFilePath: "/etc/clickhouse-server/config.xml",
			FileMode:          0o644,
		}),
		testcontainers.WithExposedPorts(nativePort, httpPort),
		testcontainers.WithWaitStrategy(wait.ForAll(wait.ForHTTP("/ping").WithPort(httpPort), wait.ForListeningPort(nativePort)).WithStartupTimeoutDefault(2*time.Minute)),
	)
	testcontainers.CleanupContainer(t, ctr)
	if err != nil {
		t.Fatalf("starting ClickHouse container: %v", err)
	}

	server := Server{Username: "default", Password: "test"}
	if server.Host, err = ctr.Host(ctx); err != nil {
		t.Fatalf("resolving ClickHouse container host: %v", err)
	}
	for containerPort, hostPort := range map[string]*uint16{nativePort: &server.NativePort, httpPort: &server.HTTPPort} {
		port, err := ctr.MappedPort(ctx, containerPort)
		if err != nil {
			t.Fatalf("resolving ClickHouse container port %s: %v", containerPort, err)
		}
		*hostPort = port.Num()
	}
	return server
}

func external(t testing.TB, host string) Server {
	server := Server{
		Host:       host,
		NativePort: 9000,
		HTTPPort:   8123,
		Username:   cmp.Or(os.Getenv("TEST_CLICKHOUSE_USER"), "default"),
		Password:   os.Getenv("TEST_CLICKHOUSE_PASSWORD"),
		Secure:     os.Getenv("TEST_CLICKHOUSE_SECURE") == "true",
	}
	if server.Secure {
		server.NativePort, server.HTTPPort = 9440, 8443
	}
	for env, port := range map[string]*uint16{"TEST_CLICKHOUSE_NATIVE_PORT": &server.NativePort, "TEST_CLICKHOUSE_HTTP_PORT": &server.HTTPPort} {
		if value := os.Getenv(env); value != "" {
			parsed, err := strconv.ParseUint(value, 10, 16)
			if err != nil {
				t.Fatalf("invalid %s: %v", env, err)
			}
			*port = uint16(parsed)
		}
	}
	return server
}

// Client connects to the server with the native or http protocol.
func (s Server) Client(t testing.TB, protocol string) clickhouseclient.ClickhouseClient {
	t.Helper()

	var tlsConfig *tls.Config
	httpProtocol := "http"
	if s.Secure {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		httpProtocol = "https"
	}

	var (
		client clickhouseclient.ClickhouseClient
		err    error
	)
	switch protocol {
	case "native":
		client, err = clickhouseclient.NewNativeClient(clickhouseclient.NativeClientConfig{
			Host:             s.Host,
			Port:             s.NativePort,
			UserPasswordAuth: &clickhouseclient.UserPasswordAuth{Username: s.Username, Password: s.Password},
			TLSConfig:        tlsConfig,
		})
	case "http":
		client, err = clickhouseclient.NewHTTPClient(clickhouseclient.HTTPClientConfig{
			Protocol:  httpProtocol,
			Host:      s.Host,
			Port:      s.HTTPPort,
			BasicAuth: &clickhouseclient.BasicAuth{Username: s.Username, Password: s.Password},
			TLSConfig: tlsConfig,
		})
	default:
		t.Fatalf("unknown protocol %q", protocol)
	}
	if err != nil {
		t.Fatalf("connecting to ClickHouse with %s: %v", protocol, err)
	}
	return client
}
