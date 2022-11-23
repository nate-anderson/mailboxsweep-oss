package mailboxsweep

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

func (r S3ClassificationRepository) FetchSummary(ctx context.Context, inbox string) (*ClassificationDocument, error) {
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

	summary := ClassificationDocument{}
	err = json.NewDecoder(obj.Body).Decode(&summary)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r S3ClassificationRepository) FetchLeaf(ctx context.Context, inbox string, path []string) (*Classification, error) {
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

type pathLeafTuple struct {
	path string
	leaf Classification
}

func (r S3ClassificationRepository) fetchLeaves(ctx context.Context, inbox string) (map[string]Classification, error) {
	prefix := r.leavesPrefix(inbox)
	list, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &r.bucketName,
		Prefix: &prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list bucket contents with prefix '%s': %w", prefix, err)
	}

	group, groupCtx := errgroup.WithContext(ctx)

	resultChan := make(chan pathLeafTuple)
	results := make(map[string]Classification)

	go func() {
		for tuple := range resultChan {
			results[tuple.path] = tuple.leaf
		}
	}()

	// exclude summary, len - 1
	for _, obj := range list.Contents {
		if strings.HasSuffix(*obj.Key, "summary") {
			continue
		}
		o := obj
		group.Go(func() error {
			doc, err := r.client.GetObject(groupCtx, &s3.GetObjectInput{
				Bucket: &r.bucketName,
				Key:    o.Key,
			})
			if err != nil {
				return fmt.Errorf("failed to fetch leaf document '%s': %w", *o.Key, err)
			}

			cl := Classification{}
			err = json.NewDecoder(doc.Body).Decode(&cl)
			if err != nil {
				return fmt.Errorf("failed to decode leaf document '%s': %w", *o.Key, err)
			}

			resultChan <- pathLeafTuple{
				leaf: cl,
				path: strings.TrimPrefix(*o.Key, inbox+"/"),
			}

			return nil
		})
	}

	err = group.Wait()
	close(resultChan)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r S3ClassificationRepository) summaryPath(inbox string) string {
	return filepath.Join(inbox, "summary")
}

func (r S3ClassificationRepository) leafPath(inbox string, path string) string {
	return filepath.Join(inbox, path)
}

func (r S3ClassificationRepository) leavesPrefix(inbox string) string {
	return fmt.Sprintf("%s/", inbox)
}

/*
Store a summary document so that all tree nodes don't have to be fetched to reconstruct the
summary, then store all leaf nodes in full using their classification path
*/

func (r S3ClassificationRepository) Store(ctx context.Context, inbox string, classification *Classification, validityKeys map[string]uint32) error {
	if err := r.storeLeaves(ctx, inbox, classification); err != nil {
		return fmt.Errorf("failed to store leaves")
	}

	if err := r.storeSummaryDocument(ctx, inbox, classification, validityKeys); err != nil {
		return fmt.Errorf("failed to store summary: %w", err)
	}

	return nil
}

func (r S3ClassificationRepository) storeLeaves(ctx context.Context, inbox string, classification *Classification) error {
	leaves := classification.Leaves()
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

			p := r.leafPath(inbox, path)
			_, err = r.client.PutObject(groupCtx, &s3.PutObjectInput{
				Bucket: &r.bucketName,
				Key:    &p,
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

func (r S3ClassificationRepository) storeSummaryDocument(ctx context.Context, inbox string, classification *Classification, validityKeys map[string]uint32) error {
	// copy before stripping
	summary := classification.withoutMessages()
	doc := ClassificationDocument{
		Meta: ClassificationMeta{
			Created:      time.Now(),
			ValidityKeys: validityKeys,
		},
		Classification: &summary,
	}

	summaryDocument := bytes.Buffer{}
	err := json.NewEncoder(&summaryDocument).Encode(doc)
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
