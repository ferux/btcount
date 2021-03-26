package btcount

// Error is a domain error type.
type Error string

// Error implements error interface.
func (e Error) Error() string { return string(e) }

const (
	// ErrNegativeValue is returned when no negative values allowed.
	ErrNegativeValue    Error = "negative value not allowed"
	ErrServerNotInited  Error = "server not inited"
	ErrParamNotFound    Error = "param not found"
	ErrInvalidParameter Error = "invalid parameter"
	ErrNotFound         Error = "not found"
	ErrUnexpectedType   Error = "unexpected type"
)
