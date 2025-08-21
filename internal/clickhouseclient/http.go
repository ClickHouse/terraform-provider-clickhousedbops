package clickhouseclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pingcap/errors"
)

type httpClient struct {
	client  *http.Client
	baseUrl url.URL
}

type HTTPClientConfig struct {
	Protocol  string
	Host      string
	Port      uint16
	BasicAuth *BasicAuth
	TLSConfig *tls.Config
}

func NewHTTPClient(config HTTPClientConfig) (ClickhouseClient, error) {
	if config.Host == "" {
		return nil, errors.New("Host is required")
	}
	if config.Port == 0 {
		return nil, errors.New("Port is required")
	}
	if config.BasicAuth == nil {
		return nil, errors.New("Exactly one authentication method is required")
	}
	protocol := "http"
	if config.Protocol != "" {
		protocol = config.Protocol
	}

	urlStr := fmt.Sprintf("%s://%s", protocol, config.Host)

	if config.Port != 0 {
		urlStr = fmt.Sprintf("%s:%d", urlStr, config.Port)
	}

	baseUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot parse URL")
	}

	baseUrl.Path = "/"

	if config.BasicAuth != nil {
		if config.BasicAuth.Password == "" {
			baseUrl.User = url.User(config.BasicAuth.Username)
		} else {
			baseUrl.User = url.UserPassword(config.BasicAuth.Username, config.BasicAuth.Password)
		}
	}

	return &httpClient{
		baseUrl: *baseUrl,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: config.TLSConfig,
			},
		},
	}, nil
}

func (i *httpClient) Select(ctx context.Context, qry string, callback func(Row) error) error {
	body, err := i.runQuery(ctx, qry)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	parsed := jsonCompatStrings{}

	err = json.Unmarshal([]byte(body), &parsed)
	if err != nil {
		return errors.WithMessage(err, "error parsing response")
	}

	for _, row := range parsed.Rows() {
		err = callback(row)
		if err != nil {
			return errors.WithMessage(err, "error calling callback function")
		}
	}

	return nil
}

func (i *httpClient) Exec(ctx context.Context, qry string) error {
	_, err := i.runQuery(ctx, qry)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *httpClient) runQuery(ctx context.Context, qry string) (string, error) {
	ctx = tflog.SetField(ctx, "Query", qry)

	req, err := http.NewRequest(http.MethodPost, i.baseUrl.String(), strings.NewReader(qry))
	if err != nil {
		return "", errors.WithMessage(err, "error preparing HTTP request")
	}

	req.Header.Add("X-ClickHouse-Format", "JSONCompactStrings")

	resp, err := i.client.Do(req)
	if err != nil {
		return "", errors.WithMessage(err, "error executing query")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithMessage(err, "error reading response")
	}

	var buf bytes.Buffer
	err = json.Indent(&buf, body, "", "  ")
	if err == nil {
		// Best effort, if we fail parsing we leave the body as-is, otherwise we use the formatted version.
		body = buf.Bytes()
	}

	ctx = tflog.SetField(ctx, "QueryResult", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(string(body))
	}

	tflog.Debug(ctx, "Run Query")

	return string(body), nil
}
