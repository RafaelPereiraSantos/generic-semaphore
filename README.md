# gsemaphore

A small package that allow an easy async processing with goroutines controlled by the [semaphore pattern](https://en.wikipedia.org/wiki/Semaphore_(programming)).

## Usage
--------

```go
import semaphore "github.com/RafaelPereiraSantos/gsemaphore"

func main {
    // List of data that must be processed.
    myDataToProcess := []string {
        "a",
        "b",
        ...
    }

    // The funciton that will receive each "myDataToProcess" item.
    myPipelineFunc := func(str string) error {
        err := someService.ProcessString(str)

        if err != nil {
            return err
        }

        return nil
    }

    // the max amount of goroutines running, each goroutine will use "myPipelineFunc" to process the itens present into
    // the "myDataToProcess" list.
    maxGoRoutines := 99 

    // The channel that will receive incomming errors from "myPipelineFunc".
    errChan := make(chan error)

    semaphore.RunWithSemaphore(
		myPipelineFunc,
		myDataToProcess,
		maxGoRoutines,
		errChan,
	)

    // It is necessary to listen to the channel in order to prevent the application to finalize without processing all
    // data. When the list is entirely processed either successfully or error, the channel will be closed automatically.
    for err := range errorsChan {
		fmt.Printf("Some Error Ocurred: %v\n", err)
	}
}
```