package gsemaphore

import "sync"

type (
	Semaphore[T any] struct {
		f                    pipeline[T]
		itemsToProcess       []T
		maxParallelPipelines int
		errorsChannel        chan error
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

func (sem *Semaphore[T]) Run() {
	semaphore := make(chan struct{}, sem.maxParallelPipelines)
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

func WithMaxParallelPipelines[T any](maxParallelPipelines int) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.maxParallelPipelines = maxParallelPipelines

		return s
	}
}

func WithErrorChannel[T any](errorsChannel chan error) func(*Semaphore[T]) *Semaphore[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.errorsChannel = errorsChannel

		return s
	}
}
