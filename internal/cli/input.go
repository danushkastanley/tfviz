package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/danushkastanley/tfviz/internal/input"
	"github.com/danushkastanley/tfviz/internal/report/model"
	"github.com/danushkastanley/tfviz/internal/source"
)

// s3Timeout bounds retrieval, including credential resolution.
const s3Timeout = 2 * time.Minute

// s3Flags select an exact S3 object and the credentials to read it with.
type s3Flags struct {
	profile string
	region  string
	version string
	owner   string
}

func (f s3Flags) set() bool {
	return f.profile != "" || f.region != "" || f.version != "" || f.owner != ""
}

// retrieved is an input document and what is known about where it came from.
type retrieved struct {
	data    []byte
	kind    model.SourceKind
	time    *time.Time
	version string
}

func readInput(opts reportOptions, env Env) (*retrieved, error) {
	switch {
	case source.IsS3(opts.input):
		return readS3(opts, env)
	case opts.s3.set():
		return nil, &usageError{"--aws-profile, --aws-region, --s3-version and --expected-bucket-owner apply only to s3:// inputs"}
	case opts.input == "-":
		data, err := input.ReadBounded(env.Stdin)
		return &retrieved{data: data, kind: model.SourceStdin}, err
	}
	data, err := readFile(opts.input)
	return &retrieved{data: data, kind: model.SourceFile}, err
}

func readFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the input file: %w", errors.Unwrap(err))
	}
	if !info.Mode().IsRegular() {
		return nil, &usageError{"the input must be a regular file or - for standard input"}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the input file: %w", errors.Unwrap(err))
	}
	defer f.Close()
	return input.ReadBounded(io.Reader(f))
}

// readS3 retrieves one state object (plan §12). Nothing is listed, written
// or locked, and credentials never leave the AWS SDK.
func readS3(opts reportOptions, env Env) (*retrieved, error) {
	if opts.offline {
		return nil, &usageError{"--offline: an s3:// input needs network access to AWS; download the state and pass a local file instead"}
	}
	loc, err := source.ParseS3(opts.input)
	if err != nil {
		return nil, &usageError{err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), s3Timeout)
	defer cancel()
	obj, err := source.FetchS3(ctx, loc, source.S3Options{
		Profile:             opts.s3.profile,
		Region:              opts.s3.region,
		VersionID:           opts.s3.version,
		ExpectedBucketOwner: opts.s3.owner,
		Endpoint:            env.S3Endpoint,
	})
	if err != nil {
		return nil, err
	}
	return &retrieved{data: obj.Data, kind: model.SourceS3, time: obj.LastModified, version: obj.VersionID}, nil
}
