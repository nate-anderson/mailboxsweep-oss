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
}

func NewClassificationController(classificationService classification.Service, log *logrus.Logger) *ClassificationController {
	return &ClassificationController{
		classificationService: classificationService,
		log:                   log,
	}
}

func (c ClassificationController) PostInboxClassification(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody StartInboxClassifyRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		respondError(w, c.log, err, withStatusCode(http.StatusBadRequest))
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
		reqBody.Address,
		reqBody.Folders,
		serializedClassifiers,
	)
	if err != nil {
		respondError(w, c.log, fmt.Errorf("failed to enqueue inbox classification: %w", err))
		return
	}

	msg := "Classification job enqueued!"
	respond(w, http.StatusAccepted, StatusResponse{
		Status:  "OK",
		Message: &msg,
	})
	return
}

func (c ClassificationController) GetInboxClassificationSummary(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	address := req.Header.Get(testAddressHeader)
	root, err := c.classificationService.ClassificationSummary(ctx, address)
	if err != nil {
		respondError(w, c.log, err, withStatusCode(http.StatusBadRequest))
		return
	}

	resBody := ClassificationResponse{
		Classification: convertClassification(*root.Classification),
	}

	respond(w, http.StatusOK, resBody)
}

func (c ClassificationController) GetInboxClassification(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	address := req.Header.Get(testAddressHeader)

	pathStr := req.URL.Query().Get("path")
	decodedPath, err := url.QueryUnescape(pathStr)
	if err != nil {
		respondError(w, c.log, err, withStatusCode(http.StatusBadRequest))
		return
	}

	path := strings.Split(decodedPath, "/")
	if len(path) < 1 {
		respondError(w, c.log, errors.New("path cannot be empty"), withStatusCode(http.StatusBadRequest))
		return
	}

	cl, err := c.classificationService.ClassificationLeaf(ctx, address, path)
	if err != nil {
		if errors.Is(err, mailboxsweep.ErrNotFound) {
			respondError(w, c.log, err, withStatusCode(http.StatusNotFound))
			return
		}
		respondError(w, c.log, err)
		return
	}

	resBody := ClassificationResponse{
		Classification: convertClassification(*cl),
	}
	respond(w, http.StatusOK, resBody)
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
