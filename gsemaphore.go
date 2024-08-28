package gsemaphore

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type (
	Semaphore[T any] struct {
		f                               pipeline[T]
		itemsToProcess                  []T
		startingParallelPipelinesAmount int
		timeBetweenParallelismIncrease  time.Duration
		currentParallelPipelinesAmount  int
		maxParallelPipelinesAmount      int
		errorsChannel                   chan error

		semaphore chan struct{}

		workersPool sync.Pool
	}

	worker struct {
		id string
	}

	pipeline[T any] func(T, context.Context) error

	opts[T any] func(*Semaphore[T]) Semaphore[T]

	WorkerKey string
)

const (
	defaultParallelismAmount           = 1
	WorkerIDcontextKey       WorkerKey = "swid"
)

// NewSemaphore returns a new Semaphore properly configured with the given options.
func NewSemaphore[T any](options []opts[T]) *Semaphore[T] {
	sem := Semaphore[T]{
		workersPool: sync.Pool{
			New: func() any {
				return worker{
					id: uuid.NewString(),
				}
			},
		},
	}

	for _, opt := range options {
		sem = opt(&sem)
	}

	return &sem
}

// Run initialize the async processing of each item with the given pipeline function. It must receive a context which
// will be used to controll dead lines.
func (sem *Semaphore[T]) Run(ctx context.Context) {
	sem.semaphore = make(chan struct{}, sem.maxParallelPipelinesAmount)

	if sem.slowlyIncreaseParallelism() {
		for i := 0; i < sem.maxParallelPipelinesAmount-sem.startingParallelPipelinesAmount; i++ {
			sem.semaphore <- struct{}{}
		}

		ctxForNewSlots, cancel := context.WithCancel(ctx)
		defer cancel()

		go sem.slowlyOpenNewSlotsOnSemahore(ctxForNewSlots)
	}

	semaphoreWG := sync.WaitGroup{}

	for _, itemToProcess := range sem.itemsToProcess {
		semaphoreWG.Add(1)
		sem.semaphore <- struct{}{}
		worker := sem.workersPool.Get().(worker)

		go func(pipe pipeline[T], item T, errChan chan error) {
			ctxWithWorker := context.WithValue(ctx, WorkerIDcontextKey, worker.id)
			if err := sem.f(item, ctxWithWorker); err != nil {
				errChan <- err
			}

			<-sem.semaphore
			semaphoreWG.Done()
		}(sem.f, itemToProcess, sem.errorsChannel)
	}

	semaphoreWG.Wait()
	close(sem.errorsChannel)
}

func (sem *Semaphore[T]) slowlyIncreaseParallelism() bool {
	return sem.startingParallelPipelinesAmount > 0 && sem.timeBetweenParallelismIncrease > 0
}

func (sem *Semaphore[T]) slowlyOpenNewSlotsOnSemahore(ctx context.Context) {
	ticker := time.NewTicker(sem.timeBetweenParallelismIncrease)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			<-sem.semaphore
			sem.currentParallelPipelinesAmount++

			if sem.currentParallelPipelinesAmount >= sem.maxParallelPipelinesAmount {
				return
			}
		}
	}
}

// WithPipeline allows a pipeline to be passed to the semaphore that will be in charge of precessing each T element
// inside the list of itens to process.
func WithPipeline[T any](f pipeline[T]) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.f = f

		return s
	}
}

// WithItensToProcess allows the send of a list of T elements to be processed by the semaphore with given function.
func WithItensToProcess[T any](itemsToProcess []T) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.itemsToProcess = itemsToProcess

		return s
	}
}

// WithStartingParallelPipelinesAmount allows the change of the semaphore attribute startingParallelPipelinesAmount
// which dictates the amount of initial goroutines running simultaneously. The value must be greater than zero and
// equal or less than maxParallelPipelinesAmount.
func WithStartingParallelPipelinesAmount[T any](startingParallelPipelinesAmount int) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.startingParallelPipelinesAmount = startingParallelPipelinesAmount

		if s.startingParallelPipelinesAmount > s.maxParallelPipelinesAmount {
			s.startingParallelPipelinesAmount = s.maxParallelPipelinesAmount
		}

		if s.startingParallelPipelinesAmount <= 0 {
			startingParallelPipelinesAmount = defaultParallelismAmount
		}

		return s
	}
}

// WithMaxParallelPipelinesAmount allows the change of the semaphore attribute maxParallelPipelinesAmount which
// controls the max amount of goroutine runnining simultaneously. The value must be greater than 0.
func WithMaxParallelPipelinesAmount[T any](max int) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.maxParallelPipelinesAmount = max

		if s.maxParallelPipelinesAmount <= 0 {
			s.maxParallelPipelinesAmount = defaultParallelismAmount
		}

		return s
	}
}

// WithErrorChannel is a function that allows that a channel be passed to the semaphore allowing that the semaphore
// send errors that happened with the pipeline.
func WithErrorChannel[T any](errorsChannel chan error) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.errorsChannel = errorsChannel

		return s
	}
}

// WithTimeBetweenParallelismIncrease is a optional function that allows the implementation of a duration time
// between opening of each new slot for a goroutine to run.
func WithTimeBetweenParallelismIncrease[T any](d time.Duration) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.timeBetweenParallelismIncrease = d

		return s
	}
}
