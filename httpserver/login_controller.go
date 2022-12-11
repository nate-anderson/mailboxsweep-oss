package httpserver

import (
	"encoding/json"
	"mailboxsweep/login"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LoginController struct {
	log     *logrus.Logger
	service *login.Service
	baseController
}

func NewLoginController(service *login.Service, log *logrus.Logger) *LoginController {
	return &LoginController{
		log,
		service,
		baseController{log},
	}
}

func (c LoginController) PostLogin(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	reqBody := LoginRequest{}
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	if err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	err = c.service.CreateLoginLink(ctx, reqBody.Address, requestIP(req))
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	msg := "OK"
	resBody := StatusResponse{
		Message: &msg,
		Status:  msg,
	}
	c.respond(w, http.StatusOK, resBody)
}

func (c LoginController) GetLoginToken(w http.ResponseWriter, req *http.Request, token string) {
	ctx := req.Context()
	token = strings.TrimSpace(
		strings.ToLower(
			token,
		),
	)

	// slow down login spamming
	time.Sleep(time.Second)

	user, err := c.service.ValidateLoginLink(ctx, token)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	resBody := LoginResponse{
		Address: user.Address,
	}

	c.respond(w, http.StatusOK, resBody)
}
