package program

// SilentError is used to indicate that Sugarkube should exit with a non-zero error code, but there's no need to print
// any error message (it's to be used where the error message has already been printed)
type SilentError struct {
}

func (e SilentError) Error() string {
	return "silent error"
}

// Simple errors are self-contained. There's no need to suggest viewing a stack trace
type SimpleError struct {
	Message string
}

func (e SimpleError) Error() string {
	return e.Message
}
