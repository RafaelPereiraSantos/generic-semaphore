# gsemaphore

A small package that allow an easy async processing with goroutines controlled by the [semaphore pattern](https://en.wikipedia.org/wiki/Semaphore_(programming)).

## Usage

The usage is pretty straightforward, import the package and follow as the example bellow.

```go
import semaphore "github.com/RafaelPereiraSantos/gsemaphore"

func main() {
    /**
  	  Creates a error channel that will be used by the semaphore to
	  return any errors with the pipeline.
    **/
	errorChannel := make(chan error)

    /**
  	  Creates the function that will process each item from a list of itens to
	  be processed individually per goroutine.
    **/
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
    /**
  	  Creates the async pipeline passing all the necessary configurations.
	  Note that if no configuration is given it will the follow its default values
	  (refers to the internal code to see them).
    **/
	sem := gsemaphore.NewSemaphore([]gsemaphore.OptionFunc[string]{
		gsemaphore.WithFlow(pipeline),
      /**
    	  Defines which parallelism strategy the pipeline will follow,
		  in this case the pipeline will start from 1
    	goroutine and slowly increase up to 10 with a pace of a increase
		  of 1 per second.
      **/
		gsemaphore.WithParallelismStrategyOf(
			gsemaphore.BuildLinearParallelismIncreaseStrategy[string](1, 10, time.Second),
		),
		gsemaphore.WithErrorChannel[string](errorChannel),
	})

    /**
  	  Runs the semaphore passing the list of itens to be processed.
	  The context that is passed could be used to end
  	  all pipelines by calling Done(), note that you must implement
	  a mechanism inside the function to handle the done
  	  signal and finish the job. 
    **/
	go sem.Run(context.Background(), userNames)

  	/**
      Keeps listening to all errors returned by the channel until it is closed.
    **/
	for err := range errorChannel {
		fmt.Println(err)
	}
}

```