package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ClassificationController struct {
	classificationService  classification.Service
	classifyRequestService classifyrequest.Service
	log                    *logrus.Logger
	baseController
}

func NewClassificationController(classificationService classification.Service, classifyRequestService classifyrequest.Service, log *logrus.Logger) *ClassificationController {
	return &ClassificationController{
		classificationService:  classificationService,
		classifyRequestService: classifyRequestService,
		log:                    log,
		baseController:         baseController{log},
	}
}

func (c ClassificationController) StartClassify(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var reqBody StartInboxClassifyRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
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
		c.respondError(w, err)
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
		c.respondError(w, err, withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	path := strings.Split(decodedPath, "/")
	if len(path) < 1 {
		c.respondError(w, errors.New("path cannot be empty"), withOverrideStatusCode(http.StatusBadRequest))
		return
	}

	cl, err := c.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
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
			HasInput:    cl.TakesInput,
			TypeKey:     cl.TypeKey,
			Name:        cl.Name,
			Description: cl.Description,
			InputLabel:  cl.InputLabel,
			InputFormat: convertInputFormat(cl.InputFormat),
		}
	}
	resBody.Classifiers = cls

	c.respond(w, http.StatusOK, resBody)
}

func (c ClassificationController) GetClassificationStatus(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	addr, err := mailboxsweep.UserAddressFromContext(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}

	resBody := ClassificationStatusResponse{
		Address: addr,
	}

	openRequests, err := c.classifyRequestService.FindOpenRequests(ctx)
	if errors.Is(err, classifyrequest.ErrNoOpenRequests) || len(openRequests) == 0 {
		c.respond(w, http.StatusOK, resBody)
		return
	} else if err != nil {
		c.respondError(w, err)
		return
	}

	var open classifyrequest.ClassifyRequest
	if len(openRequests) == 1 {
		open = openRequests[0]
	} else {
		for _, r := range openRequests {
			if r.RequestedAt > open.RequestedAt {
				open = r
			}
		}
	}

	var status ClassificationStatusStatus
	startedAt := time.Unix(open.RequestedAt, 0).Format(time.RFC3339)

	if open.Error != nil {
		status = Errored
	} else if open.CompletedAtUnix != nil {
		status = Complete
	} else {
		status = InProgress
	}
	resBody.Status = &ClassificationStatus{
		StartedAt: startedAt,
		Status:    status,
	}

	c.respond(w, http.StatusOK, open)
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

func convertInputFormat(format *classification.InputFormat) *InputFormat {
	if format == nil {
		return nil
	}

	switch *format {
	case classification.InputDate:
		return ptr(Date)
	case classification.InputString:
		return ptr(String)
	default:
		return ptr(String)
	}
}
