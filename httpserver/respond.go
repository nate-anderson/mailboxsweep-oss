package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

func respond(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		respondError(w, logrus.New(), err)
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

func withStatusCode(statusCode int) respondErrorOptFunc {
	return func(o *respondErrorOptions) {
		o.code = statusCode
	}
}

func respondError(w http.ResponseWriter, log *logrus.Logger, err error, opts ...respondErrorOptFunc) {
	options := respondErrorOptions{
		code: http.StatusInternalServerError,
	}

	for _, opt := range opts {
		opt(&options)
	}

	w.WriteHeader(options.code)
	res := errorResponse{
		Error: err.Error(),
	}
	if writeErr := json.NewEncoder(w).Encode(res); writeErr != nil {
		log.Errorf("error writting error to response: %s", err)
		return
	}
}
