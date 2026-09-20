package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seansa/lunar-backend-challenge/internal/domain"
)

type processor interface {
	Process(ctx context.Context, message domain.Message) (Result, error)
}

type Dispatcher interface {
	Dispatch(ctx context.Context, message domain.Message) (Result, error)
}

// PoolOptions configures the local worker pool.
type PoolOptions struct {
	Workers        int
	QueueSize      int
	ProcessTimeout time.Duration
}

// Pool is a bounded, in-process worker pool. It back-pressures the HTTP handlers
// by blocking Dispatch once the queue is full.
type Pool struct {
	processor processor
	options   PoolOptions

	jobs      chan *job
	quit      chan struct{}
	closeOnce sync.Once
	closed    atomic.Bool
	wg        sync.WaitGroup
	baseCtx   context.Context
}

type job struct {
	msg   domain.Message
	reply chan jobResult
}

type jobResult struct {
	result Result
	err    error
}

func NewPool(processor processor, options PoolOptions) *Pool {
	if options.Workers < 1 {
		options.Workers = 1
	}
	if options.QueueSize < 0 {
		options.QueueSize = 0
	}
	if options.ProcessTimeout <= 0 {
		options.ProcessTimeout = 10 * time.Second
	}

	return &Pool{
		processor: processor,
		options:   options,
		jobs:      make(chan *job, options.QueueSize),
		quit:      make(chan struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	p.baseCtx = ctx
	for i := 0; i < p.options.Workers; i++ {
		p.wg.Add(1)
		go p.work()
	}
	slog.Info("consumer pool started", "workers", p.options.Workers, "queue_size", p.options.QueueSize)
}

func (p *Pool) work() {
	defer p.wg.Done()
	for {
		select {
		case <-p.quit:
			return
		case j := <-p.jobs:
			p.process(j)
		}
	}
}

func (p *Pool) process(j *job) {
	var reply jobResult

	func() {
		defer func() {
			if r := recover(); r != nil {
				reply = jobResult{err: errPanic{value: r}}
			}
		}()

		ctx := p.baseCtx
		if ctx == nil {
			ctx = context.Background()
		}
		ctx, cancel := context.WithTimeout(ctx, p.options.ProcessTimeout)
		defer cancel()

		reply.result, reply.err = p.processor.Process(ctx, j.msg)
	}()

	if reply.err != nil {
		slog.Error("message processing failed",
			"channel", j.msg.Metadata.Channel,
			"message_number", j.msg.Metadata.MessageNumber,
			"message_type", j.msg.Metadata.MessageType,
			"error", reply.err)
	}

	j.reply <- reply
}

// Dispatch waits until the message is stored and projected. It reports an error
// as soon as the context is done, which the HTTP layer turns into a retryable
// status code; the message may still be processed afterwards, in which case the
// redelivery is deduplicated.
func (p *Pool) Dispatch(ctx context.Context, msg domain.Message) (Result, error) {
	if p.closed.Load() {
		return Result{}, ErrPoolClosed
	}

	j := &job{msg: msg, reply: make(chan jobResult, 1)}

	select {
	case p.jobs <- j:
	case <-p.quit:
		return Result{}, ErrPoolClosed
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}

	select {
	case r := <-j.reply:
		return r.result, r.err
	case <-p.quit:
		return Result{}, ErrPoolClosed
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

// Shutdown stops accepting work and waits for the workers to finish.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.closeOnce.Do(func() {
		p.closed.Store(true)
		close(p.quit)
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type errPanic struct {
	value any
}

func (e errPanic) Error() string {
	return fmt.Sprintf("consumer: worker panic recovered: %v", e.value)
}
