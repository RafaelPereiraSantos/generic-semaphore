package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/RafaelPereiraSantos/gsemaphore"
)

func main() {
	firstFlow := func(ctx context.Context, u string) (string, error) {
		workerID := ctx.Value(gsemaphore.WorkerIDcontextKey)

		fmt.Printf("first worker: [%s] starting to process user: [%s]\n", workerID, u)
		veryLongProcesingJob := rand.Intn(5)
		time.Sleep(time.Duration(veryLongProcesingJob) * time.Second)

		fmt.Printf("first worker: [%s] finished processing user: [%s] in %ds\n", workerID, u, veryLongProcesingJob)

		return fmt.Sprintf("first processing %s", u), nil
	}

	secondFlow := func(ctx context.Context, u string) (string, error) {
		workerID := ctx.Value(gsemaphore.WorkerIDcontextKey)

		fmt.Printf("second worker: [%s] starting to process user: [%s]\n", workerID, u)
		veryLongProcesingJob := rand.Intn(2)
		time.Sleep(time.Duration(veryLongProcesingJob) * time.Second)

		fmt.Printf("second worker: [%s] finished processing user: [%s] in %ds\n", workerID, u, veryLongProcesingJob)

		return fmt.Sprintf("second processing %s", u), nil
	}

	userNames := []string{
		"John",
		"Clara",
		"Fabio",
		"Yasmin",
		"Henrique",
		"Claudia",
	}

	ctx, cancel := context.WithCancel(context.Background())
	originalSource := make(chan string)

	go func() {
		defer cancel()

		for _, name := range userNames {
			originalSource <- name

			time.Sleep(time.Second)
		}
	}()

	firstDestinationChan := make(chan string)
	errorChannel := make(chan error)

	firstSem := gsemaphore.NewSemaphoreStream([]gsemaphore.OptionStreamFunc[string, string]{
		gsemaphore.WithSourceStream[string, string](originalSource),
		gsemaphore.WithAmountOfWorkersInStream[string, string](10),
		gsemaphore.WithTimeoutInStream[string, string](time.Second),
		gsemaphore.WithDestinationStream[string, string](firstDestinationChan),
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
}
