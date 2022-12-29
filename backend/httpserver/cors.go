package httpserver

import (
	"net/http"
)

const corsAllowOriginHeader = "Access-Control-Allow-Origin"
const corsAllowHeadersHeader = "Access-Control-Allow-Headers"
const corsAllowedHeaders = "Content-Type,Set-Cookie"
const corsAllowMethodsHeader = "Access-Control-Allow-Methods"
const corsAllowMethods = "POST, GET, PATCH, OPTIONS, DELETE, PUT"
const corsAllowCredentialsHeader = "Access-Control-Allow-Credentials"

// CORS middleware
type corsHandler struct {
	allowOrigin string
}

func NewCORSHandler(allowOrigin string) *corsHandler {
	return &corsHandler{
		allowOrigin: allowOrigin,
	}
}

func (m corsHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add(corsAllowMethodsHeader, corsAllowMethods)
		w.Header().Add(corsAllowOriginHeader, m.allowOrigin)
		w.Header().Add(corsAllowHeadersHeader, corsAllowedHeaders)
		w.Header().Add(corsAllowCredentialsHeader, "true")

		// handle preflight
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, req)
	})
}
