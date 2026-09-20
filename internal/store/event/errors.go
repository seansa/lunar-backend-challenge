package event

import "errors"

var (
	// ErrConflict means the message number is already used by a different payload.
	ErrConflict = errors.New("eventstore: message number already used by a different payload")
)
