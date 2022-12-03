package httpserver

type Controller struct {
	CredentialController
	ClassificationController
}

func NewController() *Controller {
	return &Controller{}
}
