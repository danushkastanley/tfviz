package input

// Code classifies why an input was rejected.
type Code string

const (
	CodeRead          Code = "read"
	CodeTooLarge      Code = "too_large"
	CodeTooDeep       Code = "too_deep"
	CodeMalformed     Code = "malformed"
	CodeUnsupported   Code = "unsupported"
	CodeStreamingUI   Code = "streaming_ui"
	CodeEncrypted     Code = "encrypted"
	CodeWrongKind     Code = "wrong_kind"
	CodeTooManyThings Code = "too_many_resources"
)

// Error is a rejection with a calm, actionable message. Messages never
// include input content, which may be secret-bearing.
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string { return e.Message }

// Unsupported reports whether the error describes input tfviz does not
// support (as opposed to a failure while processing supported input).
func (e *Error) Unsupported() bool {
	return e.Code != CodeRead
}

func newError(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}
