package responsewriter

import "net/http"

// ResponseWriter implements http.ResponseWriter for API Gateway responses
type ResponseWriter struct {
    Headers    map[string]string
    Body       []byte
    StatusCode int
}

func NewResponseWriter() *ResponseWriter {
    return &ResponseWriter{
        Headers:    make(map[string]string),
        StatusCode: http.StatusOK,
    }
}

// Standard http.ResponseWriter interface implementations
func (w *ResponseWriter) Header() http.Header {
    h := http.Header{}
    for k, v := range w.Headers {
        h.Set(k, v)
    }
    return h
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
    w.Body = append(w.Body, b...)
    return len(b), nil
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
    w.StatusCode = statusCode
}
