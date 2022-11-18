package mailsweeper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"golang.org/x/sync/errgroup"
)

type S3ClassificationRepository struct {
	client     *s3.Client
	bucketName string
}

func NewS3ClassificationRepository(client *s3.Client, bucket string) *S3ClassificationRepository {
	return &S3ClassificationRepository{
		client:     client,
		bucketName: bucket,
	}
}

/*
Store a summary document so that all tree nodes don't have to be fetched to reconstruct the
summary, then store all leaf nodes in full using their classification path
*/

func (r S3ClassificationRepository) Fetch(ctx context.Context, inbox string, path []string, includeMessages bool) (*Classification, error) {
	if includeMessages && len(path) > 0 {
		return r.fetchLeaf(ctx, inbox, path)
	}

	summary, err := r.fetchSummary(ctx, inbox)
	if err != nil {
		return nil, err
	}

	if !includeMessages {
		return summary, nil
	}

	// @TODO: need to fetch leaves here
	return summary, nil
}

func (r S3ClassificationRepository) fetchSummary(ctx context.Context, inbox string) (*Classification, error) {
	pathStr := r.summaryPath(inbox)

	obj, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucketName,
		Key:    &pathStr,
	})
	if err != nil {
		var one *types.NoSuchKey
		if errors.As(err, &one) {
			return nil, fmt.Errorf("failed to fetch summary: %w", one)
		}
	}

	summary := Classification{}
	err = json.NewDecoder(obj.Body).Decode(&summary)

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r S3ClassificationRepository) fetchLeaf(ctx context.Context, inbox string, path []string) (*Classification, error) {
	pathStr := r.leafPath(inbox, filepath.Join(path...))

	obj, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucketName,
		Key:    &pathStr,
	})
	if err != nil {
		var one *types.NoSuchKey
		if errors.As(err, &one) {
			return nil, fmt.Errorf("failed to fetch leaf: %w", one)
		}
	}

	cl := Classification{}
	err = json.NewDecoder(obj.Body).Decode(&cl)
	if err != nil {
		return nil, fmt.Errorf("failed to decode leaf document: %w", err)
	}
	return &cl, nil
}

func (r S3ClassificationRepository) summaryPath(inbox string) string {
	return filepath.Join(inbox, "summary")
}

func (r S3ClassificationRepository) leafPath(inbox string, path string) string {
	return filepath.Join(inbox, path)
}

func (r S3ClassificationRepository) Store(ctx context.Context, inbox string, classification *Classification) error {
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return r.storeSummaryDocument(groupCtx, inbox, classification)
	})

	group.Go(func() error {
		return r.storeLeaves(groupCtx, inbox, classification)
	})

	err := group.Wait()
	return err
}

func (r S3ClassificationRepository) storeLeaves(ctx context.Context, inbox string, classification *Classification) error {
	leaves := classification.Leaves("/")
	group, groupCtx := errgroup.WithContext(ctx)
	for classPath, clPtr := range leaves {
		cl := clPtr
		path := classPath
		group.Go(func() error {
			leafDoc := bytes.Buffer{}
			err := json.NewEncoder(&leafDoc).Encode(cl)
			if err != nil {
				return err
			}

			_, err = r.client.PutObject(groupCtx, &s3.PutObjectInput{
				Bucket: &r.bucketName,
				Key:    ptr(r.leafPath(inbox, path)),
				Body:   bytes.NewReader(leafDoc.Bytes()),
			})
			if err != nil {
				return fmt.Errorf("failed to store leaf: %w", err)
			}
			return nil
		})
	}
	err := group.Wait()
	return err
}

func (r S3ClassificationRepository) storeSummaryDocument(ctx context.Context, inbox string, classification *Classification) error {
	// copy before stripping
	summary := classification.Copy()
	stripMessages(summary)

	summaryDocument := bytes.Buffer{}
	err := json.NewEncoder(&summaryDocument).Encode(summary)
	if err != nil {
		return fmt.Errorf("failed to encode summary document: %w", err)
	}

	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &r.bucketName,
		Key:    ptr(r.summaryPath(inbox)),
		Body:   bytes.NewReader(summaryDocument.Bytes()),
	})

	if err != nil {
		return fmt.Errorf("failed to store summary document: %w", err)
	}
	return nil
}

var _ ClassificationRepository = &S3ClassificationRepository{}
