package gsemaphore

import "sync"

type pipeline[T interface{}] func(T) error

// RunWithSemaphore, receive a T list of elements to be process by a given f(T) -> error function in parallel with the
// specified amount of go routines, all errors returned by the given function will be published into the error channel.
func RunWithSemaphore[T interface{}](
	f pipeline[T],
	itemsToProcess []T,
	maxAsyncPipelinesRunning int,
	errorsChannel chan error) {
	go func(f pipeline[T], c chan error) {

		semaphore := make(chan struct{}, maxAsyncPipelinesRunning)
		semaphoreWG := sync.WaitGroup{}

		for _, item := range itemsToProcess {
			semaphoreWG.Add(1)

			go func(pipe pipeline[T], i T, c chan error) {
				semaphore <- struct{}{}
				if err := f(i); err != nil {
					c <- err
				}
				<-semaphore
				semaphoreWG.Done()
			}(f, item, c)
		}

		semaphoreWG.Wait()
		close(c)
	}(f, errorsChannel)
}
