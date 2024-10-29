package response_recorder

import "net/http"

// ResponseRecorder is a helper to capture the response status and body.
type ResponseRecorder struct {
	StatusCode int
	Body       []byte
	header     http.Header
}

func New() *ResponseRecorder {
	return &ResponseRecorder{
		StatusCode: http.StatusOK,
		Body:       []byte{},
		header:     http.Header{},
	}
}

// WriteHeader captures the status code.
func (rec *ResponseRecorder) WriteHeader(statusCode int) {
	rec.StatusCode = statusCode
}

// Write captures the response body.
func (rec *ResponseRecorder) Write(body []byte) (int, error) {
	rec.Body = append(rec.Body, body...)
	return len(body), nil
}

func (rec *ResponseRecorder) Header() http.Header {
	return rec.header
}
