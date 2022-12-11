package session

import (
	"context"
	"encoding/json"
	"io"
	"mailboxsweep"
	"mailboxsweep/assert"
	"mailboxsweep/inboxuser"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testHandler struct{}

type testRes struct {
	UserUUID string `json:"userUUID"`
	OK       bool   `json:"ok"`
}

func (t testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	res := testRes{
		OK: true,
	}
	uuid, err := UserUUIDFromContext(ctx)
	if err != nil {
		res.OK = false
		w.WriteHeader(http.StatusUnauthorized)
	} else {
		res.UserUUID = uuid
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

	repo := NewDynamoSessionRepository(db, table)
	service := NewService(repo, 20, time.Minute)

	testAddr := "butterbeans@example.com"
	testUser, err := inboxuser.NewInboxUser(testAddr)
	assert.ThatError(t, err).IsNil()
	userRepo := inboxuser.NewDynamoInboxUserRepository(db, table)
	_, err = userRepo.Store(ctx, *testUser)
	assert.ThatError(t, err).IsNil()

	var sess *Session

	t.Run("start session", func(t *testing.T) {
		var err error
		sess, err = service.StartSession(ctx, *testUser)
		assert.ThatError(t, err).IsNil()
		assert.That(t, sess).NotEqual(nil)
	})

	t.Run("authorize context", func(t *testing.T) {
		authedCtx, err := service.Authorize(ctx, sess.Token)
		assert.ThatError(t, err).IsNil()

		t.Run("authed context yields UUID", func(t *testing.T) {
			uuid, err := UserUUIDFromContext(authedCtx)
			assert.ThatError(t, err).IsNil()
			assert.That(t, uuid).Equals(testUser.UUID)
		})

		t.Run("unauthed context yields Unauthenticated error", func(t *testing.T) {
			uuid, err := UserUUIDFromContext(context.Background())
			assert.ThatError(t, err).Is(mailboxsweep.ErrUnauthenticated)
			assert.That(t, uuid).Equals("")
		})
	})

	t.Run("auth middleware", func(t *testing.T) {
		mid := AuthMiddleware{service}
		handler := mid.Middleware(testHandler{})

		t.Run("authed ctx", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "localhost", nil)
			assert.ThatError(t, err).IsNil()
			c := sess.Cookie("localhost", "/")
			req.AddCookie(c)

			handler.ServeHTTP(w, req)

			res := testRes{}
			err = json.NewDecoder(w.Result().Body).Decode(&res)
			assert.ThatError(t, err).IsNil()

			assert.That(t, w.Result().StatusCode).Equals(http.StatusOK)
			assert.That(t, res.OK).Equals(true)
			assert.That(t, res.UserUUID).Equals(testUser.UUID)
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
