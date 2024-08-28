package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/RafaelPereiraSantos/gsemaphore"
)

func main() {
	errorChannel := make(chan error)

	pipeline := func(u string, ctx context.Context) error {
		workerID := ctx.Value(gsemaphore.WorkerIDcontextKey)

		fmt.Printf("worker: [%s] starting to process user: [%s]\n", workerID, u)
		veryLongProcesingJob := rand.Intn(5)
		time.Sleep(time.Duration(veryLongProcesingJob) * time.Second)

		fmt.Printf("worker: [%s] finished processing user: [%s] in %ds\n", workerID, u, veryLongProcesingJob)

		return nil
	}

	userNames := []string{
		"John",
		"Clara",
		"Fabio",
		"Yasmin",
		"Henrique",
		"Claudia",
	}

	sem := gsemaphore.NewSemaphore([]gsemaphore.Option[string]{
		gsemaphore.WithPipeline(pipeline),
		gsemaphore.WithItensToProcess(userNames),
		gsemaphore.WithStartingParallelPipelinesAmount[string](1),
		gsemaphore.WithTimeBetweenParallelismIncrease[string](1 * time.Second),
		gsemaphore.WithMaxParallelPipelinesAmount[string](10),
		gsemaphore.WithErrorChannel[string](errorChannel),
	})

	go sem.Run(context.Background())

	for err := range errorChannel {
		fmt.Println(err)
	}
}
