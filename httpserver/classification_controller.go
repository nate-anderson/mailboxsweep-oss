package httpserver

import (
	"encoding/json"
	"fmt"
	"mailboxsweep/classification"
	"net/http"

	"github.com/sirupsen/logrus"
)

type ClassificationController struct {
	classificationService classification.Service
	log                   *logrus.Logger
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
