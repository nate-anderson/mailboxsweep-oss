package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classification"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

type ClassificationController struct {
	classificationService classification.Service
	log                   *logrus.Logger
	baseController
}

func NewClassificationController(classificationService classification.Service, log *logrus.Logger) *ClassificationController {
	return &ClassificationController{
		classificationService: classificationService,
		log:                   log,
		baseController:        baseController{log},
	}
}

func (c ClassificationController) StartClassify(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody StartInboxClassifyRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	serializedClassifiers := make([]classification.SerializableClassifier, len(reqBody.Classifiers))
	for i, cl := range reqBody.Classifiers {
		initValue := ""
		if cl.InitValue != nil {
			initValue = *cl.InitValue
		}
		serializedClassifiers[i] = classification.SerializableClassifier{
			TypeKey:   cl.TypeKey,
			InitValue: initValue,
		}
	}

	err := c.classificationService.EnqueueClassify(
		ctx,
		reqBody.Folders,
		serializedClassifiers,
	)
	if err != nil {
		c.respondError(w, fmt.Errorf("failed to enqueue inbox classification: %w", err))
		return
	}

	msg := "Classification job enqueued!"
	c.respond(w, http.StatusAccepted, StatusResponse{
		Status:  "OK",
		Message: &msg,
	})
	return
}

func (c ClassificationController) GetClassificationSummary(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	root, err := c.classificationService.ClassificationSummary(ctx)
	if err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	resBody := ClassificationResponse{
		Classification: convertClassification(*root.Classification),
	}

	c.respond(w, http.StatusOK, resBody)
}

func (c ClassificationController) GetClassificationLeaf(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	pathStr := req.URL.Query().Get("path")
	decodedPath, err := url.QueryUnescape(pathStr)
	if err != nil {
		c.respondError(w, err, withStatusCode(http.StatusBadRequest))
		return
	}

	path := strings.Split(decodedPath, "/")
	if len(path) < 1 {
		c.respondError(w, errors.New("path cannot be empty"), withStatusCode(http.StatusBadRequest))
		return
	}

	cl, err := c.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		if errors.Is(err, mailboxsweep.ErrNotFound) {
			c.respondError(w, err, withStatusCode(http.StatusNotFound))
			return
		}
		c.respondError(w, err)
		return
	}

	resBody := ClassificationResponse{
		Classification: convertClassification(*cl),
	}
	c.respond(w, http.StatusOK, resBody)
}

func (c ClassificationController) GetClassifiers(w http.ResponseWriter, req *http.Request) {
	classifiers := c.classificationService.ListClassifiers()

	resBody := ListClassifiersResponse{}
	cls := make([]ClassifierInfo, len(classifiers))
	for i, cl := range classifiers {
		cls[i] = ClassifierInfo{
			HasInput:         cl.TakesInput,
			TypeKey:          cl.TypeKey,
			Name:             cl.Name,
			Description:      cl.Description,
			InputDescription: stringOrNil(cl.InputDescription),
		}
	}
	resBody.Classifiers = cls

	c.respond(w, http.StatusOK, resBody)
}

func convertClassification(cl classification.Classification) Classification {
	children := map[string]Classification{}
	for key, child := range cl.Children {
		children[key] = convertClassification(child)
	}
	res := Classification{
		Label:     cl.Label,
		Dimension: cl.Dimension,
		Count:     int(cl.Count),
		Children:  children,
	}
	return res
}

func stringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
