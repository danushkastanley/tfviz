package source

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/danushkastanley/tfviz/internal/input"
)

// S3Options select the credentials and object. Credentials come only from
// the standard AWS SDK chain (environment, shared config and SSO sessions,
// web identity, container and instance roles); tfviz never handles keys.
type S3Options struct {
	Profile             string
	Region              string
	VersionID           string
	ExpectedBucketOwner string
	// Endpoint overrides the S3 endpoint. It exists for tests against a
	// local server and is not exposed on the command line.
	Endpoint string
}

// S3Object is a retrieved object and the metadata worth recording. The
// timestamp describes the object, not the live infrastructure.
type S3Object struct {
	Data         []byte
	LastModified *time.Time
	VersionID    string
}

// FetchS3 reads exactly one object with GetObject. It performs no listing,
// writing, locking or secret-value calls.
func FetchS3(ctx context.Context, loc S3Location, opts S3Options) (*S3Object, error) {
	var load []func(*config.LoadOptions) error
	if opts.Profile != "" {
		load = append(load, config.WithSharedConfigProfile(opts.Profile))
	}
	if opts.Region != "" {
		load = append(load, config.WithRegion(opts.Region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, load...)
	if err != nil {
		return nil, classify(err, loc, opts)
	}
	if cfg.Region == "" {
		return nil, &Error{Message: "No AWS region is configured. Pass --aws-region with the region of the state bucket."}
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
			o.UsePathStyle = true
		}
	})
	req := &s3.GetObjectInput{Bucket: aws.String(loc.Bucket), Key: aws.String(loc.Key)}
	if opts.VersionID != "" {
		req.VersionId = aws.String(opts.VersionID)
	}
	if opts.ExpectedBucketOwner != "" {
		req.ExpectedBucketOwner = aws.String(opts.ExpectedBucketOwner)
	}
	out, err := client.GetObject(ctx, req)
	if err != nil {
		return nil, classify(err, loc, opts)
	}
	defer out.Body.Close()
	if out.ContentLength != nil && *out.ContentLength > input.MaxInputBytes {
		return nil, &Error{Message: "The state object is larger than 512 MiB."}
	}
	data, err := input.ReadBounded(out.Body)
	if err != nil {
		var inputErr *input.Error
		if errors.As(err, &inputErr) {
			return nil, &Error{Message: inputErr.Message}
		}
		return nil, &Error{Message: "The state object could not be read completely. Try again."}
	}
	return &S3Object{Data: data, LastModified: out.LastModified, VersionID: aws.ToString(out.VersionId)}, nil
}
