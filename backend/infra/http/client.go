package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const (
	defaultTimeout       = 30 * time.Second
	defaultMethod        = http.MethodGet
	defaultErrorBodySize = 4 << 10
	contentTypeHeader    = "Content-Type"
	jsonContentType      = "application/json"
)

type Client struct {
	client *http.Client
}

func NewClient() IClient {
	return NewClientWithHTTPClient(&http.Client{Timeout: defaultTimeout})
}

func NewClientWithHTTPClient(client *http.Client) IClient {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{client: client}
}

func (c *Client) DoHTTPRequest(ctx context.Context, requestParam *RequestParam) error {
	if requestParam == nil {
		return fmt.Errorf("request param is nil")
	}
	ctx, cancel := requestContext(ctx, requestParam.Timeout)
	defer cancel()

	req, err := newRequest(ctx, requestParam)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return readResponse(resp, requestParam)
}

func requestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func newRequest(ctx context.Context, requestParam *RequestParam) (*http.Request, error) {
	body, isJSONBody, err := requestBody(requestParam.Body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, requestMethod(requestParam.Method), requestParam.RequestURI, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range requestParam.Header {
		req.Header.Set(key, value)
	}
	if requestParam.Body != nil && isJSONBody && req.Header.Get(contentTypeHeader) == "" {
		req.Header.Set(contentTypeHeader, jsonContentType)
	}
	return req, nil
}

func readResponse(resp *http.Response, requestParam *RequestParam) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, defaultErrorBodySize))
		_ = decodeResponse(bodyBytes, requestParam.ErrorResponse)
		bodyText := strings.TrimSpace(string(bodyBytes))
		if bodyText == "" {
			return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, bodyText)
	}

	if requestParam.Response == nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	return decodeResponse(bodyBytes, requestParam.Response)
}

func decodeResponse(bodyBytes []byte, target any) error {
	if target == nil {
		return nil
	}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return nil
	}
	if writer, ok := target.(io.Writer); ok {
		if _, err := writer.Write(bodyBytes); err != nil {
			return fmt.Errorf("failed to write response body: %w", err)
		}
		return nil
	}
	if err := jsoniter.ConfigFastest.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return nil
}

func requestBody(body any) (io.Reader, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	if reader, ok := body.(io.Reader); ok {
		return reader, false, nil
	}

	bodyBytes, err := jsoniter.ConfigFastest.Marshal(body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal request body: %w", err)
	}
	return bytes.NewReader(bodyBytes), true, nil
}

func requestMethod(method string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		return defaultMethod
	}
	return method
}
