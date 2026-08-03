package clickhouseclient

import (
	"context"
	"strings"
	"testing"
)

func Test_httpClient_execErrorDoesNotLeakParams(t *testing.T) {
	client, err := NewHTTPClient(HTTPClientConfig{
		Host:      "127.0.0.1",
		Port:      1,
		BasicAuth: &BasicAuth{Username: "default"},
	})
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	err = client.Exec(context.Background(), "CREATE USER `u` IDENTIFIED WITH sha256_hash BY {secret_0:String};", map[string]string{"secret_0": "supersecret"})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if strings.Contains(err.Error(), "supersecret") {
		t.Errorf("error message leaks query parameter value: %v", err)
	}
}
