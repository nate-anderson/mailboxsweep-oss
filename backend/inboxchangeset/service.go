package inboxchangeset

import (
	"context"
	"mailboxsweep/changeset"
	"mailboxsweep/classification"
)

type Service interface {
	AddLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error)
	AddLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error)
	RemoveLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error)
	RemoveLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error)
}

type DefaultService struct {
	changeSetService      changeset.Service
	classificationService classification.Service
}

func NewService(changeSetService changeset.Service, classificationService classification.Service) *DefaultService {
	return &DefaultService{
		changeSetService:      changeSetService,
		classificationService: classificationService,
	}
}

func (s DefaultService) AddLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForDelete(ctx, folder, []classification.Path{path})
}

func (s DefaultService) AddLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.MarkForMarkRead(ctx, folder, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForMarkRead(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForMarkRead(ctx, folder, []classification.Path{path})
}

func (s DefaultService) RemoveLeafForDelete(ctx context.Context, folder string, path classification.Path) (int, error) {
	_, err := s.classificationService.ClassificationLeaf(ctx, path)
	if err != nil {
		return 0, err
	}

	return s.changeSetService.UnmarkForDelete(ctx, folder, []classification.Path{path})
}
