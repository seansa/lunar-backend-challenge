package event

import "errors"

var (
	// ErrConflict means the message number is already used by a different payload.
	ErrConflict = errors.New("eventstore: message number already used by a different payload")
	// ErrDuplicate means the exact same event was already appended.
	ErrDuplicate = errors.New("eventstore: event already appended")
)
