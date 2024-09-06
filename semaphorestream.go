package gsemaphore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type (
	pipelineWithOutput[T any, E any] func(context.Context, T) (E, error)

	SemaphoreStream[T any, E any] struct {
		timeout time.Duration

		amountOfWorkers int
		semaphore       chan struct{}

		source      chan T
		destination chan E

		workersPool sync.Pool
	}

	OptionStreamFunc[T any, E any] func(*SemaphoreStream[T, E]) *SemaphoreStream[T, E]
)

// NewSemaphore returns a new SemaphoreStream properly configured with the given options.
func NewSemaphoreStream[T any, E any](options []OptionStreamFunc[T, E]) *SemaphoreStream[T, E] {
	sem := &SemaphoreStream[T, E]{
		workersPool: sync.Pool{
			New: func() any {
				return &worker{
					id: uuid.NewString(),
				}
			},
		},
	}

	for _, opt := range options {
		sem = opt(sem)
	}

	return sem
}

func (sem *SemaphoreStream[T, E]) RunWithDestination(
	ctx context.Context,
	flow pipelineWithOutput[T, E],
	errorsChannel chan error,
) {
	if sem.amountOfWorkers <= 0 {
		sem.amountOfWorkers = defaultParallelismAmount
	}

	sem.semaphore = make(chan struct{}, sem.amountOfWorkers)

	semaphoreWG := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			semaphoreWG.Wait()

			return
		case sem.semaphore <- struct{}{}:
			itemToProcess := <-sem.source
			semaphoreWG.Add(1)

			go sem.runWorker(ctx, &semaphoreWG, flow, itemToProcess, errorsChannel)
		}
	}
}

func (sem *SemaphoreStream[T, E]) shouldApplyTimeout() bool {
	return sem.timeout > 0
}

// UpdateSettings allows that a already instantiated semaphore to be updated, it accepts any function with the signature
// of OptionFunc[T].
func (sem *SemaphoreStream[T, E]) UpdateSettings(options []OptionStreamFunc[T, E]) {
	for _, opts := range options {
		opts(sem)
	}
}

// runWorker process one item at a time with the given flow, it will send the result of the processing to the
// destination channel for another async processing and errors to the error channel.
func (sem *SemaphoreStream[T, E]) runWorker(
	ctx context.Context,
	semaphoreWG *sync.WaitGroup,
	flow pipelineWithOutput[T, E],
	item T,
	errChan chan error,
) {
	worker := sem.workersPool.Get().(*worker)

	defer semaphoreWG.Done()
	defer func() {
		<-sem.semaphore
	}()
	defer sem.workersPool.Put(worker)

	ctxWithWorker := context.WithValue(ctx, WorkerIDcontextKey, worker.id)

	if sem.shouldApplyTimeout() {
		var cancel context.CancelFunc
		ctxWithWorker, cancel = context.WithTimeout(ctxWithWorker, sem.timeout)
		defer cancel()
	}

	errGorChan := make(chan error)
	defer close(errGorChan)

	go func() {
		output, err := flow(ctxWithWorker, item)
		if err != nil {
			errGorChan <- err

			return
		}

		if sem.destination != nil {
			sem.destination <- output
		}
	}()

	workerErr := func() error {
		return fmt.Errorf("error with worker %s", worker.id)
	}

	select {
	case <-ctxWithWorker.Done():
		errChan <- fmt.Errorf("%w: %w", workerErr(), ErrSempahoreTimeout)

		return
	case err := <-errGorChan:
		errChan <- fmt.Errorf("%w: %w", workerErr(), err)
	}
}
