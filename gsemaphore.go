package gsemaphore

import (
	"context"
	"sync"
)

type (
	Semaphore[T any] struct {
		f                               pipeline[T]
		itemsToProcess                  []T
		startingParallelPipelinesAmount int
		maxParallelPipelinesAmount      int
		errorsChannel                   chan error
	}

	pipeline[T any] func(T) error

	opts[T any] func(*Semaphore[T]) Semaphore[T]
)

func NewSemaphore[T any](options []opts[T]) *Semaphore[T] {
	sem := Semaphore[T]{}
	for _, opt := range options {
		sem = opt(&sem)
	}

	return &sem
}

func (sem *Semaphore[T]) Run(ctx context.Context) {
	semaphore := make(chan struct{}, sem.maxParallelPipelinesAmount)

	if sem.slowlyIncreaseParallelism() {
		for i := 0; i < sem.maxParallelPipelinesAmount-sem.startingParallelPipelinesAmount; i++ {
			semaphore <- struct{}{}
		}

		go sem.slowlyOpenNewSlotsOnSemahore()
	}

	semaphoreWG := sync.WaitGroup{}

	for _, item := range sem.itemsToProcess {
		semaphoreWG.Add(1)
		semaphore <- struct{}{}

		go func(pipe pipeline[T], i T, c chan error) {
			if err := sem.f(i); err != nil {
				c <- err
			}
			<-semaphore
			semaphoreWG.Done()
		}(sem.f, item, sem.errorsChannel)
	}

	semaphoreWG.Wait()
	close(sem.errorsChannel)
}

func (sem *Semaphore[T]) slowlyIncreaseParallelism() bool {
	return sem.startingParallelPipelinesAmount > 0
}

func (sem *Semaphore[T]) slowlyOpenNewSlotsOnSemahore(ctx context.Context) {

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
