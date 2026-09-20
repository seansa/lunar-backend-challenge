package event

import "errors"

// ErrDuplicate means the (channel, message number) pair was already appended.
// Redeliveries are expected under an at-least-once guarantee, so appending the
// same message twice is reported, not treated as a failure.
var ErrDuplicate = errors.New("eventstore: event already appended")
