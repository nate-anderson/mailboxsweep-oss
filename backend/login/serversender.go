package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mailboxsweep/pkg/slice"
	"net/http"
	"net/url"

	"sync"
)

// TestServerLoginSender allows one configured user to log in during E2E testing by
// spinning up a temporary HTTP server with the login link in a JSON response that the E2E test
// process may call and retrieve the login link. The server then shuts down.
type TestServerLoginSender struct {
	port      int
	testUsers []string
	env       string
	baseURL   string

	// mux covers two state variables because this is only for testing and i'm being lazy
	mux             *sync.Mutex
	isServerRunning bool
	links           map[string]string
}

func NewTestServerLoginSender(port int, testUsers []string, baseURL string, env string) *TestServerLoginSender {
	if env != "e2e" {
		panic("cannot create test server login sender with environment other than 'e2e'")
	}
	return &TestServerLoginSender{
		port:      port,
		testUsers: testUsers,
		baseURL:   baseURL,
		env:       env,
		mux:       &sync.Mutex{},
		links:     make(map[string]string),
	}
}

type JSONResBody struct {
	Link string `json:"link"`
}

func (s *TestServerLoginSender) setIsServing(isServing bool) {
	s.mux.Lock()
	s.isServerRunning = isServing
	s.mux.Unlock()
}

func (s *TestServerLoginSender) isServing() bool {
	isServing := false
	s.mux.Lock()
	isServing = s.isServerRunning
	s.mux.Unlock()
	return isServing
}

func (s *TestServerLoginSender) registerLoginLink(address, link string) {
	s.mux.Lock()
	s.links[address] = link
	s.mux.Unlock()
}

func (s *TestServerLoginSender) clearLoginLink(address string) {
	s.mux.Lock()
	delete(s.links, address)
	s.mux.Unlock()
}

func (s *TestServerLoginSender) getLoginLink(address string) (link string, err error) {
	s.mux.Lock()
	var ok bool
	link, ok = s.links[address]
	s.mux.Unlock()
	if !ok {
		return "", fmt.Errorf("unprepared login address %s", address)
	}
	return link, err
}

func (s *TestServerLoginSender) hasRemainingLinks() bool {
	has := false
	s.mux.Lock()
	has = len(s.links) > 0
	s.mux.Unlock()
	return has
}

func (s TestServerLoginSender) Send(ctx context.Context, req SendRequest) error {
	if s.env != "e2e" {
		panic("cannot create test server login sender with environment other than 'e2e'")
	}

	if !slice.Contains(s.testUsers, req.Address) {
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

	s.registerLoginLink(req.Address, link)

	go func() {
		handler := s.GetHTTPHandler(finish)

		addr := fmt.Sprintf(":%d", s.port)
		srv := http.Server{
			Addr:    addr,
			Handler: handler,
		}

		if !s.isServing() {
			go func() {
				err := srv.ListenAndServe()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("test closed unexpectedly: %s", err)
				}
			}()
			log.Printf("serving login link temporary server at %s for login link %s", addr, link)
			s.setIsServing(true)
		}

		<-finish
		if err = srv.Shutdown(context.Background()); err != nil {
			log.Printf("failed to shutdown temporary login sender server: %s", err)
		} else {
			log.Printf("temporary login link server shut down")
			s.setIsServing(false)
		}
	}()

	return nil
}

func (s TestServerLoginSender) GetHTTPHandler(finish chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")

		address := r.URL.Query().Get("address")
		link, err := s.getLoginLink(address)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		err = json.NewEncoder(w).Encode(JSONResBody{Link: link})
		if err != nil {
			log.Printf("error encoding login link JSON web response in temporary login sender server: %s", err)
		}
		s.clearLoginLink(address)

		if !s.hasRemainingLinks() {
			finish <- struct{}{}
		}
	}
}
