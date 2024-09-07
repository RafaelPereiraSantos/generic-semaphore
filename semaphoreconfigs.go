package gsemaphore

import (
	"time"
)

// WithFlow allows a pipeline to be passed to the semaphore that will be in charge of precessing each T element
// inside the list of itens to process.
func WithFlow[T any](f flowFunc[T]) OptionFunc[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.flow = f

		return s
	}
}

// WithTimeout allows the specification of a timeout duration that will be used to control for how long a goroutine
// will wait for the pipeline to run before given up. The same context will be passed forward to the inner piepeline
// therefore, the pipeline will receive the Done signal and can try to gracefully shutdown.
// The timeout must be greater than 0.
func WithTimeout[T any](t time.Duration) OptionFunc[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		s.timeout = t

		if s.timeout < 0 {
			s.timeout = 0
		}

		return s
	}
}

// WithParallelismStrategyOf allows that the number of goroutines running at same time be defined following any strategy
// from maxing out from the very beginning with all goroutines running or a slow increase over time.
func WithParallelismStrategyOf[T any](str goroutinesRampUpStrategy[T]) OptionFunc[T] {
	return func(s *Semaphore[T]) *Semaphore[T] {
		if str == nil {
			return s
		}

		s.parallelismStrategy = str

		return s
	}
}

// WithTimeoutInStream allows the specification of a timeout duration that will be used to control for how long a goroutine
// will wait for the pipeline to run before given up. The same context will be passed forward to the inner piepeline
// therefore, the pipeline will receive the Done signal and can try to gracefully shutdown.
// The timeout must be greater than 0.
func WithTimeoutInStream[T, E any](t time.Duration) OptionStreamFunc[T, E] {
	return func(s *SemaphoreStream[T, E]) *SemaphoreStream[T, E] {
		s.timeout = t

		if s.timeout < 0 {
			s.timeout = 0
		}

		return s
	}
}

// WithAmountOfWorkersInStream allows to define the amount of workes in a stream.
func WithAmountOfWorkersInStream[T, E any](amountOfWorkers int) OptionStreamFunc[T, E] {
	return func(s *SemaphoreStream[T, E]) *SemaphoreStream[T, E] {
		if amountOfWorkers <= 0 {
			amountOfWorkers = defaultParallelismAmount
		}

		s.amountOfWorkers = amountOfWorkers

		return s
	}
}

// WithSourceStream it allows to define the source of the data that will be processed by this semaphore controler.
func WithSourceStream[T, E any](source chan T) OptionStreamFunc[T, E] {
	return func(s *SemaphoreStream[T, E]) *SemaphoreStream[T, E] {
		s.source = source

		return s
	}
}

// WithDestinationStream allows to define where the data should be sent after the processing inside this semaphore
// controler has being completed.
func WithDestinationStream[T, E any](destination chan E) OptionStreamFunc[T, E] {
	return func(s *SemaphoreStream[T, E]) *SemaphoreStream[T, E] {
		s.destination = destination

		return s
	}
}
