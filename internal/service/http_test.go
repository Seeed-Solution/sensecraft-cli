package service

import (
	"net/http"
	"net/http/httptest"
	"time"

	"sensecraft-cloud-terminal/internal/client"
)

type handlerTransport struct {
	handler http.Handler
}

// RoundTrip runs the test HTTP handler in memory with the request's timeout context.
// RoundTrip 在内存中执行测试 HTTP 处理器，并使用请求的超时上下文。
func (transport handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		defer req.Body.Close()
	}
	response := httptest.NewRecorder()
	transport.handler.ServeHTTP(response, req)
	if err := req.Context().Err(); err != nil {
		return nil, err
	}
	return response.Result(), nil
}

func newHandlerAIClient(handler http.Handler) *client.AIApiClient {
	return client.NewAIApiClientAt("https://api.example", &http.Client{
		Transport: handlerTransport{handler: handler},
		Timeout:   100 * time.Millisecond,
	})
}
