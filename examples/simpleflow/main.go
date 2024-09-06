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

	pipeline := func(ctx context.Context, u string) error {
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

	sem := gsemaphore.NewSemaphore([]gsemaphore.OptionFunc[string]{
		gsemaphore.WithFlow(pipeline),
		gsemaphore.WithParallelismStrategyOf(
			gsemaphore.BuildLinearParallelismIncreaseStrategy[string](1, 10, time.Second),
		),
		gsemaphore.WithTimeout[string](time.Second),
	})

	go sem.Run(context.Background(), userNames, errorChannel)

	for err := range errorChannel {
		fmt.Println(err)
	}
}
