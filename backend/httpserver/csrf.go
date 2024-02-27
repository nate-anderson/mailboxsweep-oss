package httpserver

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// CSRFMiddleware is a simple implementation of CSRF tokens using cookie storage
type CSRFMiddleware struct {
	signingKey    []byte
	life          time.Duration
	tokenLen      int
	secureCookies bool
	// SkipCheckPathFunc should be defined if some routes need to be skipped, and return true for routes to skip
	SkipCheckPathFunc func(path string) bool
	logger            *logrus.Logger
}

func NewCSRFMiddleware(hmacKey string, tokenLife time.Duration, tokenLen int, secure bool, logger *logrus.Logger) *CSRFMiddleware {
	return &CSRFMiddleware{
		signingKey:    []byte(hmacKey),
		life:          tokenLife,
		tokenLen:      tokenLen,
		secureCookies: secure,
		logger:        logger,
	}
}

func (m CSRFMiddleware) generateToken() (string, error) {
	buf := make([]byte, m.tokenLen)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(buf), nil
}

func (m CSRFMiddleware) respondCSRFError(err error, w http.ResponseWriter) {
	resErr := errorResponse{
		Error: "Forbidden",
		Key:   "CSRF",
	}
	w.WriteHeader(http.StatusForbidden)
	if writeErr := json.NewEncoder(w).Encode(resErr); writeErr != nil {
		m.logger.Printf("error writing CSRF error: %s", writeErr)
	}
	m.logger.WithField("error_type", "CSRF").Error(err)
}

// sign cookies
func (m CSRFMiddleware) signMAC(data []byte) (string, error) {
	h := hmac.New(sha256.New, m.signingKey)
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(h.Sum(nil)), nil
}

func (m CSRFMiddleware) verifyMAC(value []byte, signatureStr string) error {
	h := hmac.New(sha256.New, m.signingKey)
	_, err := h.Write(value)
	if err != nil {
		return err
	}

	signature, err := base64.RawStdEncoding.DecodeString(signatureStr)
	if err != nil {
		return err
	}

	calculated := h.Sum(nil)
	if !hmac.Equal(calculated, signature) {
		return errors.New("invalid cookie")
	}
	return nil
}

// cookiename|token|timestamp
func (m CSRFMiddleware) Cookie(token string) (*http.Cookie, error) {
	issued := time.Now()
	value := fmt.Sprintf("%s|%s|%d", csrfCookie, token, issued.Unix())
	signature, err := m.signMAC([]byte(value))
	if err != nil {
		return nil, err
	}

	total := value + ":" + signature
	encoded := base64.RawStdEncoding.EncodeToString([]byte(total))
	return &http.Cookie{
		Name:     csrfCookie,
		Value:    encoded,
		Path:     "/",
		Secure:   m.secureCookies,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  issued,
		MaxAge:   int(m.life.Seconds()),
	}, nil
}

func (m CSRFMiddleware) ParseCookieValue(val string) (name, token string, issued time.Time, err error) {
	decoded, err := base64.RawStdEncoding.DecodeString(val)
	if err != nil {
		return "", "", time.Time{}, err
	}

	components := strings.Split(string(decoded), ":")
	if len(components) != 2 {
		return "", "", time.Time{}, errors.New("invalid cookie (decoded)")
	}

	signature := components[1]
	concatValue := components[0]

	if err := m.verifyMAC([]byte(concatValue), signature); err != nil {
		return "", "", time.Time{}, err
	}

	components = strings.Split(concatValue, "|")
	if len(components) != 3 {
		return "", "", time.Time{}, errors.New("invalid cookie concat value")
	}

	issuedUnix, err := strconv.ParseInt(components[2], 10, 64)
	if err != nil {
		return "", "", time.Time{}, errors.New("invalid cookie expiration")
	}

	issued = time.Unix(issuedUnix, 0)

	token = components[1]
	name = components[0]

	return name, token, issued, nil
}

func (m CSRFMiddleware) GetStoredToken(req *http.Request) (string, error) {
	cookie, err := req.Cookie(csrfCookie)
	if err != nil {
		return "", err
	}

	val := cookie.Value
	name, token, issued, err := m.ParseCookieValue(val)
	if err != nil {
		return "", err
	}

	if time.Since(issued) > m.life {
		return "", errors.New("invalid cookie: expired")
	}

	if name != csrfCookie {
		return "", errors.New("invalid cookie")
	}

	return token, nil
}

func (m CSRFMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		newToken, err := m.generateToken()
		if err != nil {
			m.respondCSRFError(err, w)
			return
		}

		header := r.Header.Get(csrfHeader)
		// ignore error - if a token is invalid, just create a new one
		expectedHeader, cookieErr := m.GetStoredToken(r)

		// set new token
		cookie, err := m.Cookie(newToken)
		if err != nil {
			m.respondCSRFError(err, w)
			return
		}
		http.SetCookie(w, cookie)
		w.Header().Set(csrfHeader, newToken)

		if m.SkipCheckPathFunc != nil {
			if m.SkipCheckPathFunc(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		switch r.Method {
		// idempotent methods can be ignored
		case http.MethodGet, http.MethodOptions, http.MethodHead:
			next.ServeHTTP(w, r)
			return
		default:
			if cookieErr != nil {
				m.respondCSRFError(err, w)
				return
			}

			if header != expectedHeader {
				m.respondCSRFError(errors.New("Forbidden"), w)
				return
			}

			next.ServeHTTP(w, r)
		}

	})
}

const (
	csrfCookie = "mailboxsweep_csrf"
	csrfHeader = "X-CSRF-Token"
)
