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
)

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

func (sem *Semaphore[T]) Run(ctx context.Context) {
	sem.semaphore = make(chan struct{}, sem.maxParallelPipelinesAmount)

	if sem.slowlyIncreaseParallelism() {
		for i := 0; i < sem.maxParallelPipelinesAmount-sem.startingParallelPipelinesAmount; i++ {
			sem.semaphore <- struct{}{}
		}

		go sem.slowlyOpenNewSlotsOnSemahore(ctx)
	}

	semaphoreWG := sync.WaitGroup{}

	for _, itemToProcess := range sem.itemsToProcess {
		semaphoreWG.Add(1)
		sem.semaphore <- struct{}{}

		go func(pipe pipeline[T], item T, errChan chan error) {
			if err := sem.f(item, ctx); err != nil {
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

func WithPipeline[T any](f pipeline[T]) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.f = f

		return s
	}
}

func WithItensToProcess[T any](itemsToProcess []T) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.itemsToProcess = itemsToProcess

		return s
	}
}

func WithStartingParallelPipelinesAmount[T any](startingParallelPipelinesAmount int) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.startingParallelPipelinesAmount = startingParallelPipelinesAmount

		return s
	}
}

func WithMaxParallelPipelinesAmount[T any](maxParallelPipelinesAmount int) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.maxParallelPipelinesAmount = maxParallelPipelinesAmount

		return s
	}
}

func WithErrorChannel[T any](errorsChannel chan error) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.errorsChannel = errorsChannel

		return s
	}
}

func WithTimeBetweenParallelismIncrease[T any](d time.Duration) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.timeBetweenParallelismIncrease = d

		return s
	}
}
