package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"mailboxsweep/classifyrequest"
	"net/http"
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

func (c ClassificationController) GetClassificationLeaf(w http.ResponseWriter, req *http.Request, params GetClassificationLeafParams) {
	ctx := req.Context()

	path := classification.ParsePath(params.Path)
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

	latestClassifyRequest, err := c.classifyRequestService.FindLatestRequest(ctx)
	if err != nil {
		c.respondAuthError(w, err)
		return
	}
	resBody.Status.StartedAt = time.Unix(latestClassifyRequest.RequestedAt, 0).Format(time.RFC3339)
	status, _ := latestClassifyRequest.Status()
	switch status {
	case classifyrequest.StatusClassifying:
		resBody.Status.Status = Classifying
	case classifyrequest.StatusClassified:
		resBody.Status.Status = Classified
	case classifyrequest.StatusExecuting:
		resBody.Status.Status = Executing
	case classifyrequest.StatusExecuted:
		resBody.Status.Status = Executed
	default:
		resBody.Status.Status = Errored
		resBody.Status.Details = ptr(fmt.Sprintf("Unknown classify status %d", status))
	}

	if latestClassifyRequest.Error != nil {
		resBody.Status.Details = latestClassifyRequest.Error
	}

	if latestClassifyRequest.Operation != nil {
		resBody.Status.Operation = ptr(Delete)
		if *latestClassifyRequest.Operation == string(changeset.MarkRead) {
			resBody.Status.Operation = ptr(MarkRead)
		}
	}

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
		Label:           cl.Label,
		Dimension:       cl.Dimension,
		TypeKey:         typeKey,
		Count:           int(cl.Count),
		UnreadCount:     int(cl.UnreadCount),
		SubjectPreviews: cl.SubjectPreviews,
		Children:        children,
	}
	if res.SubjectPreviews == nil {
		res.SubjectPreviews = []string{}
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
