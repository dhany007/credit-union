package fault

// Kind classifies application errors without coupling services to HTTP.
type Kind uint8

// Application error kinds are independent of HTTP status codes.
const (
	Invalid Kind = iota + 1
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Rejected
	Unavailable
)

// Error is a safe application failure with an optional underlying cause.
type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string { return e.Message }

// New constructs the configured component with explicit dependencies.
func New(kind Kind, message string) error { return &Error{Kind: kind, Message: message} }

// Wrap preserves an infrastructure cause while keeping its public message safe.
func Wrap(kind Kind, message string, cause error) error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}
func (e *Error) Unwrap() error { return e.Cause }
