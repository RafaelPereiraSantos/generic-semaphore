package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/RafaelPereiraSantos/gsemaphore"
)

// In the following example we have a list of names that must be processed sequencially by two distinct pipelines:
// firstFlow and secondFlow, the former represents a very long process to complete that could vary from 0s to 10s to
// finish whereas the latter could take between 0s and 2s to complete.
//
// Because of that, the firstFlow receives 10 workers and the secondFlow receives only 2.
//
// When a name is processed by the firstFlow it is then forwarded to the
// secondFlow through a commum channel.
func main() {
	userNames := []string{
		"John",
		"Clara",
		"Fabio",
		"Yasmin",
		"Henrique",
		"Claudia",
	}

	mu := sync.Mutex{}

	processedNames := []string{}

	firstFlow := func(ctx context.Context, u string) (string, error) {
		workerID := ctx.Value(gsemaphore.WorkerIDcontextKey)

		fmt.Printf("first worker: [%s] starting to process user: [%s]\n", workerID, u)
		veryLongProcesingJob := rand.Intn(20)
		time.Sleep(time.Duration(veryLongProcesingJob) * time.Second)

		fmt.Printf("first worker: [%s] finished processing user: [%s] in %ds\n", workerID, u, veryLongProcesingJob)

		return fmt.Sprintf("[1st] %s", u), nil
	}

	secondFlow := func(ctx context.Context, u string) (string, error) {
		workerID := ctx.Value(gsemaphore.WorkerIDcontextKey)

		fmt.Printf("second worker: [%s] starting to process user: [%s]\n", workerID, u)
		shortProcessingjob := rand.Intn(5)
		time.Sleep(time.Duration(shortProcessingjob) * time.Second)

		fmt.Printf("second worker: [%s] finished processing user: [%s] in %ds\n", workerID, u, shortProcessingjob)

		newName := fmt.Sprintf("[2nd] %s", u)

		mu.Lock()
		processedNames = append(processedNames, newName)
		mu.Unlock()

		return newName, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	originalSource := make(chan string)

	errorChannel := make(chan error)

	go func() {
		defer close(errorChannel)
		defer cancel()

		for _, name := range userNames {
			// This is the source of the names, it is slowly feeding the pipeline
			originalSource <- name

			time.Sleep(time.Second)
		}

		time.Sleep(10 * time.Second)
	}()

	firstDestinationChan := make(chan string)

	firstSem := gsemaphore.NewSemaphoreStream([]gsemaphore.OptionStreamFunc[string, string]{
		gsemaphore.WithSourceStream[string, string](originalSource),
		gsemaphore.WithAmountOfWorkersInStream[string, string](1000),
		gsemaphore.WithTimeoutInStream[string, string](10 * time.Second),
		gsemaphore.WithDestinationStream[string](firstDestinationChan),
	})

	secondSem := gsemaphore.NewSemaphoreStream([]gsemaphore.OptionStreamFunc[string, string]{
		gsemaphore.WithSourceStream[string, string](firstDestinationChan),
		gsemaphore.WithAmountOfWorkersInStream[string, string](2),
		gsemaphore.WithTimeoutInStream[string, string](time.Second),
	})

	go firstSem.RunWithDestination(
		ctx,
		firstFlow,
		errorChannel,
	)

	go secondSem.RunWithDestination(
		ctx,
		secondFlow,
		errorChannel,
	)

	for err := range errorChannel {
		fmt.Println(err)
	}

	fmt.Println("Only the following users where processed by the two pipelines:")
	for _, u := range processedNames {
		fmt.Println(u)
	}
}
