package inboxchangeset

import (
	"context"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
	"strconv"
)

type DefaultService struct {
	changeSetService      changeset.Service
	classificationService classification.Service
}

func (s DefaultService) AddLeafForDelete(ctx context.Context, folder string, path []string) (int, error) {
	leaf, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	uids := make([]string, len(leaf.Messages))
	for i, msg := range leaf.Messages {
		uids[i] = strconv.FormatUint(uint64(msg.UID), 10)
	}

	return s.changeSetService.MarkForDelete(ctx, folder, uids)
}

func (s DefaultService) AddLeafForMarkRead(ctx context.Context, folder string, path []string) (int, error) {
	leaf, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	uids := make([]string, len(leaf.Messages))
	for i, msg := range leaf.Messages {
		uids[i] = strconv.FormatUint(uint64(msg.UID), 10)
	}

	return s.changeSetService.MarkForMarkRead(ctx, folder, uids)
}

func (s DefaultService) RemoveLeafForMarkRead()
