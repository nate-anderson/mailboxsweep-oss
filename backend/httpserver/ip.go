package httpserver

import "net/http"

func requestIP(req *http.Request) string {
	if header, ok := req.Header["X-Forwarded-For"]; ok {
		return header[0]
	}

	return req.RemoteAddr
}
