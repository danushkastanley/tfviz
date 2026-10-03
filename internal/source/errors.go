package source

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/smithy-go"
)

// Error is a retrieval failure with a calm, actionable message. Messages
// name the object the user asked for but never credentials or content.
type Error struct {
	Message string
}

func (e *Error) Error() string { return e.Message }

// classify turns SDK errors into guidance (plan §12). An expired SSO
// session asks for `aws sso login`; tfviz never starts a login itself.
func classify(err error, loc S3Location, opts S3Options) error {
	object := fmt.Sprintf("s3://%s/%s", loc.Bucket, loc.Key)
	profile := opts.Profile
	if profile == "" {
		profile = "<profile>"
	}
	var (
		sso       *ssocreds.InvalidTokenError
		noProfile config.SharedConfigProfileNotExistError
		api       smithy.APIError
	)
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
		return &Error{Message: "Reading the state object timed out. Check your network connection and try again."}
	case errors.As(err, &sso):
		return &Error{Message: fmt.Sprintf("Your AWS SSO session has expired or is not signed in. Run `aws sso login --profile %s`, then try again.", profile)}
	case errors.As(err, &noProfile):
		return &Error{Message: fmt.Sprintf("The AWS profile %q is not configured on this machine.", opts.Profile)}
	case errors.As(err, &api):
		return classifyAPI(api.ErrorCode(), object, opts)
	case credentialsUnavailable(err):
		return &Error{Message: "No usable AWS credentials were found. Sign in (for example `aws sso login --profile " + profile + "`) or pass --aws-profile, then try again."}
	default:
		return &Error{Message: "The state object could not be retrieved from S3. Check the bucket, region and network, then try again."}
	}
}

func classifyAPI(code, object string, opts S3Options) error {
	switch code {
	case "AccessDenied", "Forbidden", "403":
		need := "s3:GetObject"
		if opts.VersionID != "" {
			need = "s3:GetObjectVersion"
		}
		msg := fmt.Sprintf("Access to %s was denied. The credentials need %s on the object (and kms:Decrypt on its key if it uses SSE-KMS).", object, need)
		msg += " Without s3:ListBucket, S3 also reports a missing object as access denied."
		if opts.ExpectedBucketOwner != "" {
			msg += " Check that --expected-bucket-owner matches the bucket's account."
		}
		return &Error{Message: msg}
	case "NoSuchKey", "NotFound", "404":
		return &Error{Message: fmt.Sprintf("%s does not exist.", object)}
	case "NoSuchVersion":
		return &Error{Message: fmt.Sprintf("Version %s of %s does not exist.", opts.VersionID, object)}
	case "NoSuchBucket":
		return &Error{Message: fmt.Sprintf("The bucket in %s does not exist.", object)}
	case "PermanentRedirect", "AuthorizationHeaderMalformed", "IllegalLocationConstraintException":
		return &Error{Message: "The state bucket is in a different region. Pass --aws-region with the bucket's region."}
	case "InvalidObjectState":
		return &Error{Message: fmt.Sprintf("%s is archived and must be restored before it can be read.", object)}
	case "ExpiredToken", "ExpiredTokenException", "InvalidToken", "InvalidAccessKeyId", "SignatureDoesNotMatch":
		return &Error{Message: "The AWS credentials were rejected or have expired. Refresh them (for example `aws sso login`) and try again."}
	case "KMS.AccessDeniedException", "KMS.DisabledException", "KMS.NotFoundException":
		return &Error{Message: fmt.Sprintf("%s is encrypted with a KMS key the credentials cannot use. They need kms:Decrypt on that key.", object)}
	default:
		return &Error{Message: fmt.Sprintf("S3 refused the request for %s (%s).", object, safeCode(code))}
	}
}

func credentialsUnavailable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "failed to retrieve credentials") || strings.Contains(msg, "failed to refresh cached credentials")
}

// safeCode keeps an AWS error code printable and short.
func safeCode(code string) string {
	var b strings.Builder
	for _, r := range code {
		if b.Len() >= 64 {
			break
		}
		if r == '.' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
