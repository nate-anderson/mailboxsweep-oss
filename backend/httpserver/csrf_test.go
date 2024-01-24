package httpserver

import (
	"mailboxsweep/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestCSRFMiddleware(t *testing.T) {
	m := NewCSRFMiddleware("aaa_bbb_ccc_123", time.Minute, 15, true, logrus.New())

	t.Run("instantiate", func(t *testing.T) {
		assert.That(t, m).NotEqual(nil)
	})

	t.Run("generateToken", func(t *testing.T) {
		t.Run("token creation succeeds", func(t *testing.T) {
			tok, err := m.generateToken()
			assert.ThatError(t, err).IsNil()
			assert.That(t, tok).NotEqual("")
		})

		t.Run("tokens are distinct", func(t *testing.T) {
			n := 200
			tokens := []string{`H/eEAV3YkdgBpFZ0rm/P`}
			for i := 0; i < n; i++ {
				token, err := m.generateToken()
				assert.ThatError(t, err).IsNil()
				assert.ThatSlice(t, tokens).NotContains(token)
				tokens = append(tokens, token)
			}
		})
	})

	t.Run("signatures", func(t *testing.T) {
		var signature string
		str := "hello, world! ;)"
		data := []byte(str)

		t.Run("sign", func(t *testing.T) {
			var err error
			signature, err = m.signMAC(data)
			assert.ThatError(t, err).IsNil()
			assert.That(t, signature).NotEqual("")
		})

		t.Run("verify", func(t *testing.T) {
			t.Run("valid signature passes", func(t *testing.T) {
				err := m.verifyMAC(data, signature)
				assert.ThatError(t, err).IsNil()
			})

			t.Run("invalid signature fails", func(t *testing.T) {
				invalidSignature, err := m.signMAC([]byte("waterford"))
				assert.ThatError(t, err).IsNil()
				err = m.verifyMAC(data, invalidSignature)
				assert.ThatError(t, err).IsNotNil()
			})
		})
	})

	t.Run("cookies", func(t *testing.T) {
		var (
			cookie *http.Cookie
			token  string
		)
		t.Run("generate cookie", func(t *testing.T) {
			var err error
			token, err = m.generateToken()
			assert.ThatError(t, err).IsNil()

			cookie, err = m.Cookie(token)
			assert.ThatError(t, err).IsNil()
			assert.That(t, cookie).NotEqual(nil)

			assert.That(t, cookie.HttpOnly).Equals(true)
			assert.That(t, cookie.Secure).Equals(true)
			assert.That(t, cookie.HttpOnly).Equals(true)
			assert.That(t, cookie.Value).NotEqual("")
			assert.That(t, cookie.Name).Equals(csrfCookie)
		})

		t.Run("parse cookie", func(t *testing.T) {
			t.Run("valid", func(t *testing.T) {
				name, value, issued, err := m.ParseCookieValue(cookie.Value)
				assert.ThatError(t, err).IsNil()
				assert.That(t, name).Equals(csrfCookie)
				assert.That(t, value).Equals(token)
				assert.That(t, issued.IsZero()).Equals(false)
			})

			t.Run("invalid", func(t *testing.T) {
				name, value, issued, err := m.ParseCookieValue("aoiufj")
				assert.ThatError(t, err).IsNotNil()
				assert.That(t, name).Equals("")
				assert.That(t, value).Equals("")
				assert.That(t, issued.IsZero()).Equals(true)
			})

			t.Run("incorrect signing key", func(t *testing.T) {
				m2 := NewCSRFMiddleware("waterbug", time.Minute, 15, true, logrus.New())

				token, err := m2.generateToken()
				assert.ThatError(t, err).IsNil()

				cookie, err = m2.Cookie(token)
				assert.ThatError(t, err).IsNil()

				name, value, issued, err := m.ParseCookieValue(cookie.Value)
				assert.ThatError(t, err).IsNotNil()
				assert.That(t, name).Equals("")
				assert.That(t, value).Equals("")
				assert.That(t, issued.IsZero()).Equals(true)
			})
		})

		t.Run("GetStoredToken", func(t *testing.T) {
			t.Run("no cookie", func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, "/bingo", strings.NewReader("Hello world"))
				assert.ThatError(t, err).IsNil()

				token, err := m.GetStoredToken(req)
				assert.ThatError(t, err).IsNotNil()
				assert.That(t, token).Equals("")
			})

			t.Run("cookie present", func(t *testing.T) {
				token, err := m.generateToken()
				assert.ThatError(t, err).IsNil()

				cookie, err := m.Cookie(token)
				assert.ThatError(t, err).IsNil()

				req := httptest.NewRequest(http.MethodPost, "/bingo", strings.NewReader("Hello world"))
				req.AddCookie(cookie)

				retrieved, err := m.GetStoredToken(req)
				assert.ThatError(t, err).IsNil()
				assert.That(t, retrieved).Equals(token)
			})

			t.Run("bad cookie present", func(t *testing.T) {
				m2 := NewCSRFMiddleware("waterbug", time.Minute, 15, true, logrus.New())

				token, err := m2.generateToken()
				assert.ThatError(t, err).IsNil()

				cookie, err = m2.Cookie(token)
				assert.ThatError(t, err).IsNil()

				req := httptest.NewRequest(http.MethodPost, "/bingo", strings.NewReader("Hello world"))
				req.AddCookie(cookie)

				retrieved, err := m.GetStoredToken(req)
				assert.ThatError(t, err).IsNotNil()
				assert.That(t, retrieved).Equals("")
			})
		})

		t.Run("middleware", func(t *testing.T) {
			matchMethods := []string{
				http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				http.MethodPatch,
			}

			skipMethods := []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodOptions,
			}

			t.Run("applicable, non-idempotent methods", func(t *testing.T) {
				for _, method := range matchMethods {
					var called int
					testHandler := func(w http.ResponseWriter, req *http.Request) {
						called++
					}
					wrapped := m.Middleware(http.HandlerFunc(testHandler))
					t.Run("no header, handler responds with token", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						wrapped.ServeHTTP(w, req)
						header := w.Header().Get(csrfHeader)
						assert.That(t, header).NotEqual("")
						assert.That(t, called).Equals(0)

						var (
							cookie http.Cookie
							found  bool
						)
						for _, c := range w.Result().Cookies() {
							cookie = *c
							found = true
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(header)
						assert.That(t, issued.IsZero()).Equals(false)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusForbidden)
					})

					t.Run("mismatched header and cookie", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						m2 := NewCSRFMiddleware("waterbug", time.Minute, 15, true, logrus.New())
						token, err := m2.generateToken()
						assert.ThatError(t, err).IsNil()

						setCookie, err := m2.Cookie(token)
						assert.ThatError(t, err).IsNil()
						req.AddCookie(setCookie)
						token2, err := m.generateToken()
						assert.ThatError(t, err).IsNil()
						req.Header.Set(csrfHeader, token2)

						wrapped.ServeHTTP(w, req)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusForbidden)
						assert.That(t, called).Equals(0)

						// response has cookie
						var (
							found  bool
							cookie http.Cookie
						)

						for _, c := range w.Result().Cookies() {
							if c.Name == csrfCookie {
								cookie = *c
								found = true
							}
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(token)
						assert.That(t, issued.IsZero()).Equals(false)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusForbidden)

						// response has header
						headerVal := w.Result().Header.Get(csrfHeader)
						assert.That(t, headerVal).Equals(token)
					})

					t.Run("matching header and cookie", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						token, err := m.generateToken()
						assert.ThatError(t, err).IsNil()
						req.Header.Set(csrfHeader, token)

						setCookie, err := m.Cookie(token)
						assert.ThatError(t, err).IsNil()
						req.AddCookie(setCookie)

						wrapped.ServeHTTP(w, req)

						assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
						assert.That(t, called).Equals(1)

						// response has cookie
						var (
							found  bool
							cookie http.Cookie
						)

						for _, c := range w.Result().Cookies() {
							if c.Name == csrfCookie {
								cookie = *c
								found = true
							}
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(token)
						assert.That(t, issued.IsZero()).Equals(false)

					})
				}
			})

			t.Run("idempotent methods", func(t *testing.T) {
				for _, method := range skipMethods {
					var called int
					testHandler := func(w http.ResponseWriter, req *http.Request) {
						called++
					}
					wrapped := m.Middleware(http.HandlerFunc(testHandler))
					t.Run("no header, handler responds with token", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						wrapped.ServeHTTP(w, req)
						header := w.Header().Get(csrfHeader)
						assert.That(t, header).NotEqual("")
						assert.That(t, called).Equals(1)

						var (
							cookie http.Cookie
							found  bool
						)
						for _, c := range w.Result().Cookies() {
							cookie = *c
							found = true
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(header)
						assert.That(t, issued.IsZero()).Equals(false)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
					})

					t.Run("mismatched header and cookie", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						m2 := NewCSRFMiddleware("waterbug", time.Minute, 15, true, logrus.New())
						token, err := m2.generateToken()
						assert.ThatError(t, err).IsNil()

						setCookie, err := m2.Cookie(token)
						assert.ThatError(t, err).IsNil()
						req.AddCookie(setCookie)
						token2, err := m.generateToken()
						assert.ThatError(t, err).IsNil()
						req.Header.Set(csrfHeader, token2)

						wrapped.ServeHTTP(w, req)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
						assert.That(t, called).Equals(2)

						// response has cookie
						var (
							found  bool
							cookie http.Cookie
						)

						for _, c := range w.Result().Cookies() {
							if c.Name == csrfCookie {
								cookie = *c
								found = true
							}
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(token)
						assert.That(t, issued.IsZero()).Equals(false)
						assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)

						// response has header
						headerVal := w.Result().Header.Get(csrfHeader)
						assert.That(t, headerVal).Equals(token)
					})

					t.Run("matching header and cookie", func(t *testing.T) {
						w := httptest.NewRecorder()
						req := httptest.NewRequest(method, "/bingo", strings.NewReader("Hello world"))

						token, err := m.generateToken()
						assert.ThatError(t, err).IsNil()

						setCookie, err := m.Cookie(token)
						assert.ThatError(t, err).IsNil()
						req.AddCookie(setCookie)

						wrapped.ServeHTTP(w, req)

						assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
						assert.That(t, called).Equals(3)

						// response has cookie
						var (
							found  bool
							cookie http.Cookie
						)

						for _, c := range w.Result().Cookies() {
							if c.Name == csrfCookie {
								cookie = *c
								found = true
							}
						}
						assert.That(t, found).Equals(true)
						assert.That(t, cookie.Name).Equals(csrfCookie)
						assert.That(t, cookie.Value).NotEqual("")

						name, token, issued, err := m.ParseCookieValue(cookie.Value)
						assert.ThatError(t, err).IsNil()
						assert.That(t, name).Equals(csrfCookie)
						assert.That(t, token).Equals(token)
						assert.That(t, issued.IsZero()).Equals(false)

					})
				}
			})

		})
	})
}
