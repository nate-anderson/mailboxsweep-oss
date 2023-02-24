package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mailboxsweep"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"golang.org/x/sync/errgroup"
)

const perSecondPrefixRateLimit = float32(3600)

type S3ClassificationRepository struct {
	client     *s3.Client
	bucketName string
	pageSize   int
}

func NewS3ClassificationRepository(client *s3.Client, bucket string, pageSize int) *S3ClassificationRepository {
	return &S3ClassificationRepository{
		client:     client,
		bucketName: bucket,
		pageSize:   pageSize,
	}
}

func (r S3ClassificationRepository) FetchSummary(ctx context.Context, inbox string, page int) (*ClassificationDocument, error) {
	pathStr := r.summaryPath(inbox, page)

	obj, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.bucketName,
		Key:    &pathStr,
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.(type) {
			case *types.NoSuchKey, *types.NoSuchBucket:
				return nil, mailboxsweep.NotFoundError(apiErr)
			default:
				return nil, err
			}
		}

		return nil, fmt.Errorf("failed to fetch summary: %s", err)
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
		if !errors.As(err, &one) {
			return nil, fmt.Errorf("failed to fetch leaf: %w", err)
		}
		return nil, mailboxsweep.NotFoundError(err)
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

func (r S3ClassificationRepository) summaryPath(inbox string, page int) string {
	return filepath.Join(r.summaryDirPath(inbox), strconv.Itoa(page))
}

func (r S3ClassificationRepository) summaryDirPath(inbox string) string {
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
		return fmt.Errorf("failed to store leaves: %w", err)
	}

	if err := r.storeSummaryDocument(ctx, inbox, classification, validityKeys); err != nil {
		return fmt.Errorf("failed to store summary: %w", err)
	}

	return nil
}

func (r S3ClassificationRepository) storeLeaves(ctx context.Context, inbox string, classification *Classification) error {
	leaves := classification.Leaves()
	maxGoroutines := 300
	guard := make(chan struct{}, maxGoroutines)

	group, groupCtx := errgroup.WithContext(ctx)
	for classPath, clPtr := range leaves {
		cl := clPtr
		path := classPath
		guard <- struct{}{}
		group.Go(func() error {
			leafDoc := bytes.Buffer{}
			err := json.NewEncoder(&leafDoc).Encode(cl)

			defer func() {
				<-guard
			}()

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
	summary := classification.withoutMessages()
	docs := sortAndPaginateChildren(summary, validityKeys, r.pageSize)

	for _, doc := range docs {
		summaryDocument := bytes.Buffer{}
		err := json.NewEncoder(&summaryDocument).Encode(doc)
		if err != nil {
			return fmt.Errorf("failed to encode summary document: %w", err)
		}

		_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: &r.bucketName,
			Key:    aws.String(r.summaryPath(inbox, doc.Page)),
			Body:   bytes.NewReader(summaryDocument.Bytes()),
		})

		if err != nil {
			return fmt.Errorf("failed to store summary document: %w", err)
		}
	}

	return nil
}

func (r S3ClassificationRepository) GetLeavesOfPrefix(ctx context.Context, inbox string, prefix []string) ([]Path, error) {
	filePathPrefix := r.leafPath(inbox, filepath.Join(prefix...))
	res, err := r.client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: &r.bucketName,
		Prefix: &filePathPrefix,
	})
	if err != nil {
		return nil, err
	}

	if len(res.Contents) == 0 {
		return nil, mailboxsweep.NotFoundError(fmt.Errorf("no leaves found for prefix %v", prefix))
	}

	paths := make([]Path, len(res.Contents))

	for i, c := range res.Contents {
		p := strings.Split(*c.Key, "/")[1:]
		path := Path(p)
		paths[i] = path
	}

	return paths, nil
}

var _ ClassificationRepository = &S3ClassificationRepository{}
