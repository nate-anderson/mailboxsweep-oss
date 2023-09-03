package login

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

// TestServerLoginSender allows one configured user to log in during E2E testing by
// spinning up a temporary HTTP server with the login link in a JSON response that the E2E test
// process may call and retrieve the login link. The server then shuts down.
type TestServerLoginSender struct {
	port     int
	testUser string
	env      string
	baseURL  string
}

func NewTestServerLoginSender(port int, testUser string, baseURL string, env string) *TestServerLoginSender {
	if env != "e2e" {
		panic("cannot create test server login sender with environment other than 'e2e'")
	}
	return &TestServerLoginSender{
		port:     port,
		testUser: testUser,
		baseURL:  baseURL,
		env:      env,
	}
}

type JSONResBody struct {
	Link string `json:"link"`
}

func (s TestServerLoginSender) Send(ctx context.Context, req SendRequest) error {
	if s.env != "e2e" {
		panic("cannot create test server login sender with environment other than 'e2e'")
	}

	if req.Address != s.testUser {
		log.Println("ignoring login request for unknown user in TestServerLoginSender")
		return nil
	}

	query := url.Values{}
	query.Set("token", req.Token)
	query.Set("sig", req.Signature)
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return err
	}
	u.RawQuery = query.Encode()

	finish := make(chan struct{})

	link := u.String()

	go func() {
		handler := s.GetHTTPHandler(link, finish)

		addr := fmt.Sprintf(":%d", s.port)
		srv := http.Server{
			Addr:    addr,
			Handler: handler,
		}

		go srv.ListenAndServe()
		log.Printf("serving login link temporary server at %s for login link %s", addr, link)

		_ = <-finish
		if err = srv.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown temporary login sender server: %s", err)
		} else {
			log.Printf("temporary login link server shut down")
		}
	}()

	return nil
}

func (s TestServerLoginSender) GetHTTPHandler(link string, finish chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		err := json.NewEncoder(w).Encode(JSONResBody{Link: link})
		if err != nil {
			log.Printf("error encoding login link JSON web response in temporary login sender server: %s", err)
		}

		finish <- struct{}{}
	}
}
