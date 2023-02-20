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

func (c ClassificationController) GetClassificationSummary(w http.ResponseWriter, req *http.Request, params GetClassificationSummaryParams) {
	ctx := req.Context()
	page := 0
	if params.Page != nil {
		page = *params.Page
	}
	root, err := c.classificationService.ClassificationSummary(ctx, &page)
	if err != nil {
		c.respondError(w, err)
		return
	}

	resBody := ClassificationResponse{
		Classification: convertClassification(*root.Classification),
		PageCount:      root.TotalPages,
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
		Status:  &ClassificationStatus{},
	}

	openRequests, err := c.classifyRequestService.FindOpenRequests(ctx)
	if err == nil {
		// if there is an open request, respond with that
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

		resBody.Status.StartedAt = time.Unix(open.RequestedAt, 0).Format(time.RFC3339)

		if open.Error != nil {
			resBody.Status.Status = Errored
			resBody.Status.Details = open.Error
		} else if open.CompletedAtUnix != nil {
			resBody.Status.Status = Complete
		} else {
			resBody.Status.Status = InProgress
		}
		resBody.Status.StartedAt = time.Unix(open.RequestedAt, 0).Format(time.RFC3339)
		c.respond(w, http.StatusOK, resBody)
		return

	} else {
		if !errors.Is(err, classifyrequest.ErrNoOpenRequests) {
			c.respondError(w, err)
			return
		}
	}

	closedRequests, err := c.classifyRequestService.FindCompletedRequests(ctx)
	if err != nil {
		c.respondError(w, err)
		return
	}

	var closed classifyrequest.ClassifyRequest
	if len(closedRequests) == 0 {
		c.respondError(w, mailboxsweep.NotFoundError(errors.New("no completed classify requests found")))
	} else if len(closedRequests) == 1 {
		closed = closedRequests[0]
	} else {
		// if multiple completed requests, use the most recent
		for _, r := range closedRequests {
			if r.RequestedAt > closed.RequestedAt {
				closed = r
			}
		}
	}

	if closed.Error != nil {
		resBody.Status.Status = Errored
		resBody.Status.Details = closed.Error
	} else {
		resBody.Status.Status = Complete
	}
	resBody.Status.StartedAt = time.Unix(closed.RequestedAt, 0).Format(time.RFC3339)

	c.respond(w, http.StatusOK, resBody)
}

func convertClassification(cl classification.Classification) Classification {
	children := map[string]Classification{}
	for key, child := range cl.Children {
		children[key] = convertClassification(child)
	}

	var typeKey *string
	if cl.TypeKey != "" {
		typeKey = &cl.TypeKey
	}

	res := Classification{
		Label:     cl.Label,
		Dimension: cl.Dimension,
		TypeKey:   typeKey,
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
