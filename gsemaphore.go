package gsemaphore

import "sync"

type Pipeline[T interface{}] func(T) error

// RunWithSemaphore, receives a function F => error that iterate over itens of type T in parallel with the max amount of
// go routines defined by parameter, it does receive an error channel that will be feed by errors producer by the
// provided function.
func RunWithSemaphore[T interface{}](
	f Pipeline[T],
	itemsToProcess []T,
	maxAsyncProcess int,
	errorsChannel chan error) {
	go func(f Pipeline[T], c chan error) {

		semaphore := make(chan struct{}, maxAsyncProcess)
		semaphoreWG := sync.WaitGroup{}

		for _, item := range itemsToProcess {
			semaphoreWG.Add(1)

			go func(pipe Pipeline[T], i T, c chan error) {
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
