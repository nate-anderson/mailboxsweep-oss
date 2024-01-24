package httpserver

import (
	"mailboxsweep/pkg/slice"
	"net/http"
)

const corsAllowOriginHeader = "Access-Control-Allow-Origin"
const corsAllowHeadersHeader = "Access-Control-Allow-Headers"
const corsAllowedHeaders = "Content-Type,Set-Cookie,X-Csrf-Token"
const corsAllowMethodsHeader = "Access-Control-Allow-Methods"
const corsAllowMethods = "POST, GET, PATCH, OPTIONS, DELETE, PUT"
const corsAllowCredentialsHeader = "Access-Control-Allow-Credentials"

// CORS middleware
type corsHandler struct {
	allowOrigins []string
}

func NewCORSHandler(allowOrigins []string) *corsHandler {
	return &corsHandler{
		allowOrigins: allowOrigins,
	}
}

func (m corsHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		origin := req.Header.Get("origin")
		w.Header().Add(corsAllowMethodsHeader, corsAllowMethods)
		if slice.Contains(m.allowOrigins, origin) {
			w.Header().Add(corsAllowOriginHeader, origin)
		}
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
