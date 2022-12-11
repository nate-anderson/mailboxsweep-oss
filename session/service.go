package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"mailboxsweep"
	"net/http"
	"time"
)

type Service interface {
	// create + store session, authorize context, and return cookie value
	StartSession(ctx context.Context, userUUID string) (*Session, error)
	// lookup session and return authorized context
	Authorize(ctx context.Context, cookieValue string) (context.Context, error)
}

type DefaultService struct {
	repo            Repository
	sessionKeyLen   int
	sessionDuration time.Duration
}

func NewService(
	repo Repository,
	sessionKeyLen int,
	sessionDuration time.Duration,
) *DefaultService {
	return &DefaultService{
		repo:            repo,
		sessionKeyLen:   sessionKeyLen,
		sessionDuration: sessionDuration,
	}
}

func (s DefaultService) makeToken() (string, error) {
	buf := make([]byte, s.sessionKeyLen)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s DefaultService) StartSession(ctx context.Context, userAddress string) (*Session, error) {
	token, err := s.makeToken()
	if err != nil {
		return nil, err
	}

	expires := time.Now().Add(s.sessionDuration)

	sess := Session{
		Token:       token,
		UserAddress: userAddress,
		Expires:     expires.Unix(),
	}

	if err = s.repo.Store(ctx, sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

type authKeyT int

const authKey authKeyT = 1

func (s DefaultService) Authorize(ctx context.Context, cookieValue string) (context.Context, error) {
	sess, err := s.repo.Get(ctx, cookieValue)
	if err != nil {
		return ctx, err
	}

	authed := context.WithValue(ctx, authKey, sess.UserAddress)
	return authed, nil
}

type AuthMiddleware struct {
	Service Service
}

const sessionCookie = "X-MailboxSweep-Session"

func (a AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		cookie, err := req.Cookie(sessionCookie)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			log.Printf("missing session cookie: %s\n", err)
			return
		}

		authed, err := a.Service.Authorize(ctx, cookie.Value)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			log.Printf("authorize failed: %s\n", err)
			return
		}

		next.ServeHTTP(w, req.WithContext(authed))
	})
}

func UserAddressFromContext(ctx context.Context) (string, error) {
	userAddress := ctx.Value(authKey)
	if str, ok := userAddress.(string); ok {
		return str, nil
	}
	return "", mailboxsweep.ErrUnauthenticated
}
