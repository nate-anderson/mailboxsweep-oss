package session_test

import (
	"context"
	"encoding/json"
	"io"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"mailboxsweep/session"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testHandler struct{}

type testRes struct {
	UserAddress string `json:"userAddress"`
	OK          bool   `json:"ok"`
}

func (t testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	res := testRes{
		OK: true,
	}
	address, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		res.OK = false
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		res.UserAddress = address
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		panic(err)
	}
}

func TestSessionService(t *testing.T) {
	ctx := context.Background()
	db := mailboxsweep.SetupTestDynamoClient()
	table, teardown := mailboxsweep.SetupDynamoTestTable(ctx, t, db)
	defer teardown()

	repo := session.NewDynamoSessionRepository(db, table)
	service := session.NewService(repo, 20, time.Minute)

	testAddr := "butterbeans@example.com"
	testUser, err := inboxuser.NewInboxUser(testAddr)
	assert.ThatError(t, err).IsNil()
	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	_, err = userRepo.Store(ctx, *testUser)
	assert.ThatError(t, err).IsNil()

	var sess *session.Session

	t.Run("start session", func(t *testing.T) {
		var err error
		sess, err = service.StartSession(ctx, testUser.Address)
		assert.ThatError(t, err).IsNil()
		assert.That(t, sess).NotEqual(nil)
	})

	t.Run("authorize context", func(t *testing.T) {
		authedCtx, err := service.Authorize(ctx, sess.Token)
		assert.ThatError(t, err).IsNil()

		t.Run("authed context yields address", func(t *testing.T) {
			address, err := mailboxsweep.UserAddressFromContext(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, address).Equals(testUser.Address)
		})

		t.Run("unauthed context yields Unauthenticated error", func(t *testing.T) {
			address, err := mailboxsweep.UserAddressFromContext(context.Background())
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, address).Equals("")
		})
	})

	t.Run("auth middleware", func(t *testing.T) {
		mid := session.AuthMiddleware{service}
		handler := mid.Middleware(testHandler{})

		t.Run("authed ctx", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "localhost", nil)
			assert.ThatError(t, err).IsNil()
			c := sess.Cookie(false)
			req.AddCookie(c)

			handler.ServeHTTP(w, req)

			res := testRes{}
			err = json.NewDecoder(w.Result().Body).Decode(&res)
			assert.ThatError(t, err).IsNil()

			assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
			assert.That(t, res.OK).Equals(true)
			assert.That(t, res.UserAddress).Equals(testUser.Address)
		})

		t.Run("unauthed ctx", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "localhost", nil)
			assert.ThatError(t, err).IsNil()
			handler.ServeHTTP(w, req)
			var resBody string

			var buf []byte
			_, err = io.ReadFull(w.Result().Body, buf)
			assert.ThatError(t, err).IsNil()

			err = json.Unmarshal(buf, &resBody)
			assert.ThatError(t, err).IsNotNil()

			assert.That(t, w.Result().StatusCode).Equals(http.StatusUnauthorized)
		})
	})

}
