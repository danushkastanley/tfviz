// Package source retrieves input documents from where they live: local
// files, standard input and exact S3 objects.
package source

import (
	"errors"
	"strings"
)

// S3Location is an exact S3 object.
type S3Location struct {
	Bucket string
	Key    string
}

// IsS3 reports whether an input refers to S3.
func IsS3(input string) bool {
	return strings.HasPrefix(input, "s3://")
}

// ParseS3 parses s3://bucket/key. Only exact objects are accepted: tfviz
// never lists buckets or resolves prefixes.
func ParseS3(uri string) (S3Location, error) {
	rest, ok := strings.CutPrefix(uri, "s3://")
	if !ok {
		return S3Location{}, errors.New("S3 inputs look like s3://bucket/path/to/terraform.tfstate")
	}
	bucket, key, ok := strings.Cut(rest, "/")
	switch {
	case !ok || key == "":
		return S3Location{}, errors.New("give the full S3 object path, for example s3://bucket/path/to/terraform.tfstate")
	case !validBucket(bucket):
		return S3Location{}, errors.New("the S3 bucket name is not valid")
	case len(key) > 1024:
		return S3Location{}, errors.New("the S3 object key is longer than 1,024 bytes")
	case strings.HasSuffix(key, "/"):
		return S3Location{}, errors.New("the S3 path ends with /; give the state object itself, not a prefix")
	}
	return S3Location{Bucket: bucket, Key: key}, nil
}

// validBucket applies the general-purpose bucket naming rules.
func validBucket(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for i, r := range name {
		alnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		edge := i == 0 || i == len(name)-1
		if !alnum && (edge || (r != '-' && r != '.')) {
			return false
		}
	}
	return !strings.Contains(name, "..")
}
