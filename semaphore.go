package gsemaphore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type (
	flowFunc[T any] func(context.Context, T) error

	goroutinesRampUpStrategy[T any] func(context.Context, *Semaphore[T]) (strategyFollowUp, context.CancelFunc)

	Semaphore[T any] struct {
		flow flowFunc[T]

		startingParallelPipelinesAmount int
		timeout                         time.Duration

		semaphore chan struct{}

		workersPool sync.Pool

		parallelismStrategy goroutinesRampUpStrategy[T]
	}

	OptionFunc[T any] func(*Semaphore[T]) *Semaphore[T]
)

// NewSemaphore returns a new Semaphore properly configured with the given options.
func NewSemaphore[T any](options []OptionFunc[T]) *Semaphore[T] {
	sem := &Semaphore[T]{
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

// Run initialize the async processing of each item with the given pipeline function. It must receive a context which
// will be used to controll dead lines.
// If the startingParallelPipelinesAmount was greater than zero, the async processing will start with the amount of
// goroutines equal to the startingParallelPipelinesAmount attribute and slowly increase its capacity up to
// maxParallelPipelinesAmount, incrementing 1 extra goroutine each timeBetweenParallelismIncrease.
// If timeout was specified the pipeline has that amount of time to run, otherwise it will receive a termination signal
// the goroutine will pass to the next item of the list and the errorsChannel will receive a ErrSempahoreTimeout error.
func (sem *Semaphore[T]) Run(ctx context.Context, source []T, errorsChannel chan error) {
	if sem.parallelismStrategy == nil {
		sem.parallelismStrategy = BuildFullCapacityFromStartStrategy[T](defaultParallelismAmount)
	}

	followUp, followUpCancel := sem.parallelismStrategy(ctx, sem)

	if followUp != nil {
		go followUp()
	}

	if followUpCancel != nil {
		defer followUpCancel()
	}

	semaphoreWG := sync.WaitGroup{}

	for _, itemToProcess := range source {
		semaphoreWG.Add(1)
		sem.semaphore <- struct{}{}

		go sem.runWorker(ctx, &semaphoreWG, itemToProcess, errorsChannel)
	}

	semaphoreWG.Wait()
	close(errorsChannel)
}

// runWorker receives one item at a time and process it forwarding any error returned by the flow to the specified
// channel.
func (sem *Semaphore[T]) runWorker(ctx context.Context, semaphoreWG *sync.WaitGroup, item T, errChan chan error) {
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
		if err := sem.flow(ctxWithWorker, item); err != nil {
			errGorChan <- err
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

func (sem *Semaphore[T]) shouldApplyTimeout() bool {
	return sem.timeout > 0
}

// UpdateSettings allows that a already instantiated semaphore to be updated, it accepts any function with the signature
// of OptionFunc[T].
func (sem *Semaphore[T]) UpdateSettings(options []OptionFunc[T]) {
	for _, opts := range options {
		opts(sem)
	}
}

// BuildLinearParallelismIncreaseStrategy creates a strategy function that follows the linear progression of goroutines
// increase.
func BuildLinearParallelismIncreaseStrategy[T any](
	startingParallelPipelinesAmount int,
	maxParallelPipelinesAmount int,
	timeBetweenParallelismIncrease time.Duration,
) goroutinesRampUpStrategy[T] {
	return func(
		ctx context.Context,
		sem *Semaphore[T],
	) (strategyFollowUp, context.CancelFunc) {
		sem.semaphore = make(chan struct{}, maxParallelPipelinesAmount)

		for i := 0; i < maxParallelPipelinesAmount-sem.startingParallelPipelinesAmount-1; i++ {
			sem.semaphore <- struct{}{}
		}

		ctxForNewSlots, cancel := context.WithCancel(ctx)

		linearSpotsIncreaser := func() {
			ticker := time.NewTicker(timeBetweenParallelismIncrease)
			defer ticker.Stop()

			currentParallelPipelinesAmount := sem.startingParallelPipelinesAmount

			for {
				select {
				case <-ctxForNewSlots.Done():
					return
				case <-ticker.C:
					<-sem.semaphore
					currentParallelPipelinesAmount++

					if currentParallelPipelinesAmount >= maxParallelPipelinesAmount {
						return
					}
				}
			}
		}

		return linearSpotsIncreaser, cancel
	}
}

// BuildFullCapacityFromStartStrategy creates a strategy function that enables all goroutines to run from the very
// beginning.
func BuildFullCapacityFromStartStrategy[T any](maxParallelPipelinesAmount int) goroutinesRampUpStrategy[T] {
	return func(_ context.Context, sem *Semaphore[T]) (strategyFollowUp, context.CancelFunc) {
		sem.semaphore = make(chan struct{}, maxParallelPipelinesAmount)

		return nil, nil
	}
}
