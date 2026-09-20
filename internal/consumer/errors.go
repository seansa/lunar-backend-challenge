package consumer

import "errors"

var ErrPoolClosed = errors.New("consumer: worker pool is closed")
