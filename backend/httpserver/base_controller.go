package httpserver

import (
	"encoding/json"
	"errors"
	"mailboxsweep"
	"net/http"

	"github.com/sirupsen/logrus"
)

const contentType = "application/json"

type baseController struct {
	logger *logrus.Logger
}

func newBaseController(logger *logrus.Logger) *baseController {
	return &baseController{
		logger: logger,
	}
}

func (r baseController) respond(w http.ResponseWriter, status int, body any) {
	w.Header().Add("Content-Type", contentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		r.respondError(w, err)
		return
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type respondErrorOptions struct {
	code int
}

type respondErrorOptFunc func(o *respondErrorOptions)

func withOverrideStatusCode(statusCode int) respondErrorOptFunc {
	return func(o *respondErrorOptions) {
		o.code = statusCode
	}
}

func (r baseController) respondError(w http.ResponseWriter, err error, opts ...respondErrorOptFunc) {
	w.Header().Add("Content-Type", contentType)

	options := respondErrorOptions{
		code: http.StatusInternalServerError,
	}

	var statusError mailboxsweep.StatusError
	if errors.As(err, &statusError) {
		options.code = statusError.StatusCode()
	}

	for _, opt := range opts {
		opt(&options)
	}

	r.logger.Errorf("response error: %s", err)

	w.WriteHeader(options.code)
	res := errorResponse{
		Error: err.Error(),
	}
	if writeErr := json.NewEncoder(w).Encode(res); writeErr != nil {
		r.logger.Errorf("error writting error to response: %s", err)
		return
	}
}

func (r baseController) respondAuthError(w http.ResponseWriter, internalError error) {
	w.Header().Add("Content-Type", contentType)
	w.WriteHeader(http.StatusUnauthorized)
	res := errorResponse{
		Error: "operation not permitted",
	}

	r.logger.Errorf("response auth error: %s", internalError)

	if writeErr := json.NewEncoder(w).Encode(res); writeErr != nil {
		r.logger.Errorf("error writting error to response: %s", writeErr)
		return
	}
}

func ptr[t any](v t) *t {
	return &v
}
