package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	client, ok := NewClient().(*Client)
	require.True(t, ok)
	require.NotNil(t, client.client)
	assert.Equal(t, defaultTimeout, client.client.Timeout)
}

func TestNewClientWithHTTPClient(t *testing.T) {
	t.Parallel()

	custom := &http.Client{Timeout: time.Second}
	client, ok := NewClientWithHTTPClient(custom).(*Client)
	require.True(t, ok)
	assert.Same(t, custom, client.client)

	fallback, ok := NewClientWithHTTPClient(nil).(*Client)
	require.True(t, ok)
	require.NotNil(t, fallback.client)
	assert.Equal(t, defaultTimeout, fallback.client.Timeout)
}

func TestClient_DoHTTPRequest_Request(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *RequestParam
		assert  func(t *testing.T, r *http.Request, body []byte)
	}{
		{
			name: "default method is GET",
			request: &RequestParam{
				Method: "",
			},
			assert: func(t *testing.T, r *http.Request, body []byte) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Empty(t, body)
			},
		},
		{
			name: "json body sets content type",
			request: &RequestParam{
				Method: http.MethodPost,
				Body:   map[string]string{"name": "relievo"},
			},
			assert: func(t *testing.T, r *http.Request, body []byte) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, jsonContentType, r.Header.Get(contentTypeHeader))
				assert.JSONEq(t, `{"name":"relievo"}`, string(body))
			},
		},
		{
			name: "explicit content type is preserved",
			request: &RequestParam{
				Method: http.MethodPost,
				Header: map[string]string{
					contentTypeHeader: "application/problem+json",
				},
				Body: map[string]string{"error": "bad request"},
			},
			assert: func(t *testing.T, r *http.Request, body []byte) {
				assert.Equal(t, "application/problem+json", r.Header.Get(contentTypeHeader))
				assert.JSONEq(t, `{"error":"bad request"}`, string(body))
			},
		},
		{
			name: "reader body is sent as is",
			request: &RequestParam{
				Method: http.MethodPut,
				Header: map[string]string{
					"X-Trace-ID": "trace-1",
				},
				Body: strings.NewReader("raw body"),
			},
			assert: func(t *testing.T, r *http.Request, body []byte) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "trace-1", r.Header.Get("X-Trace-ID"))
				assert.Empty(t, r.Header.Get(contentTypeHeader))
				assert.Equal(t, "raw body", string(body))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				tt.assert(t, r, body)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			tt.request.RequestURI = server.URL
			err := NewClient().DoHTTPRequest(nil, tt.request)
			require.NoError(t, err)
		})
	}
}

func TestClient_DoHTTPRequest_Response(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		response any
		assert   func(t *testing.T, response any)
	}{
		{
			name:     "json response",
			body:     `{"ok":true,"name":"relievo"}`,
			response: &map[string]any{},
			assert: func(t *testing.T, response any) {
				got := *(response.(*map[string]any))
				assert.Equal(t, true, got["ok"])
				assert.Equal(t, "relievo", got["name"])
			},
		},
		{
			name:     "raw writer response",
			body:     "raw response",
			response: &bytes.Buffer{},
			assert: func(t *testing.T, response any) {
				assert.Equal(t, "raw response", response.(*bytes.Buffer).String())
			},
		},
		{
			name:     "empty response body",
			body:     "",
			response: &map[string]any{},
			assert: func(t *testing.T, response any) {
				assert.Empty(t, *(response.(*map[string]any)))
			},
		},
		{
			name:     "nil response ignores body",
			body:     `not json`,
			response: nil,
			assert:   func(t *testing.T, response any) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := NewClient().DoHTTPRequest(context.Background(), &RequestParam{
				RequestURI: server.URL,
				Response:   tt.response,
			})
			require.NoError(t, err)
			tt.assert(t, tt.response)
		})
	}
}

func TestClient_DoHTTPRequest_StatusError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		statusCode    int
		body          string
		errorResponse any
		wantError     string
		assert        func(t *testing.T, errorResponse any)
	}{
		{
			name:       "includes response body",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"bad request"}`,
			errorResponse: &struct {
				Message string `json:"message"`
			}{},
			wantError: "HTTP request failed with status 400",
			assert: func(t *testing.T, errorResponse any) {
				got := errorResponse.(*struct {
					Message string `json:"message"`
				})
				assert.Equal(t, "bad request", got.Message)
			},
		},
		{
			name:       "empty response body",
			statusCode: http.StatusInternalServerError,
			body:       "",
			wantError:  "HTTP request failed with status 500",
			assert:     func(t *testing.T, errorResponse any) {},
		},
		{
			name:          "invalid error response json is ignored",
			statusCode:    http.StatusBadGateway,
			body:          "upstream failed",
			errorResponse: &map[string]any{},
			wantError:     "HTTP request failed with status 502: upstream failed",
			assert: func(t *testing.T, errorResponse any) {
				assert.Empty(t, *(errorResponse.(*map[string]any)))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			err := NewClient().DoHTTPRequest(context.Background(), &RequestParam{
				RequestURI:    server.URL,
				ErrorResponse: tt.errorResponse,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
			tt.assert(t, tt.errorResponse)
		})
	}
}

func TestClient_DoHTTPRequest_InputError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *RequestParam
		want    string
	}{
		{
			name:    "nil request",
			request: nil,
			want:    "request param is nil",
		},
		{
			name: "invalid url",
			request: &RequestParam{
				RequestURI: "://invalid-url",
			},
			want: "missing protocol scheme",
		},
		{
			name: "body marshal error",
			request: &RequestParam{
				Method:     http.MethodPost,
				RequestURI: "http://example.com",
				Body:       make(chan int),
			},
			want: "unsupported type",
		},
		{
			name: "response unmarshal error",
			request: &RequestParam{
				Response: &map[string]any{},
			},
			want: "failed to unmarshal response body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.request != nil && tt.request.RequestURI == "" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("not json"))
				}))
				defer server.Close()
				tt.request.RequestURI = server.URL
			}

			err := NewClient().DoHTTPRequest(context.Background(), tt.request)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestClient_DoHTTPRequest_Context(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		context func() context.Context
		request *RequestParam
		want    string
	}{
		{
			name: "context cancelled",
			context: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			request: &RequestParam{},
			want:    "context canceled",
		},
		{
			name:    "request timeout",
			context: context.Background,
			request: &RequestParam{Timeout: 10 * time.Millisecond},
			want:    "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			tt.request.RequestURI = server.URL
			err := NewClient().DoHTTPRequest(tt.context(), tt.request)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestClient_DoHTTPRequest_TransportError(t *testing.T) {
	t.Parallel()

	client := &Client{client: &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("network unavailable")
		}),
	}}

	err := client.DoHTTPRequest(context.Background(), &RequestParam{
		RequestURI: "http://example.com",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
	assert.Contains(t, err.Error(), "network unavailable")
}

func TestClient_DoHTTPRequest_ResponseReadError(t *testing.T) {
	t.Parallel()

	client := &Client{client: &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errReadCloser{},
				Header:     make(http.Header),
			}, nil
		}),
	}}

	err := client.DoHTTPRequest(context.Background(), &RequestParam{
		RequestURI: "http://example.com",
		Response:   &map[string]any{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response body")
}

func TestClient_DoHTTPRequest_WriterError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("raw response"))
	}))
	defer server.Close()

	err := NewClient().DoHTTPRequest(context.Background(), &RequestParam{
		RequestURI: server.URL,
		Response:   errWriter{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write response body")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

type errReadCloser struct{}

func (errReadCloser) Read(p []byte) (int, error) {
	return 0, errors.New("read failed")
}

func (errReadCloser) Close() error {
	return nil
}

type errWriter struct{}

func (errWriter) Write(p []byte) (int, error) {
	return 0, errors.New("write failed")
}
