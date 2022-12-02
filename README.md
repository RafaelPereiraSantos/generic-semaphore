# gsemaphore

A small package that allow an easy async processing with goroutines controlled by the [semaphore pattern](https://en.wikipedia.org/wiki/Semaphore_(programming)).

## Usage
--------

```go
import semaphore "github.com/RafaelPereiraSantos/gsemaphore"

func main {
    myDataToProcess := []string {
        "a",
        "b",
        ...
    } // List of data that must be processed.

    myPipelineFunc := func(str string) error {
        err := someService.ProcessString(str)

        if err != nil {
            return err
        }

        return nil
    } // The funciton that will receive each "myDataToProcess" item.

    maxGoRoutines := 99 // the max amount goroutines running, each goroutine will use "myPipelineFunc" to process the itens present into the "myDataToProcess" list.

    errChan := make(chan error) // The channel that will receive incomming errors from "myPipelineFunc".

    semaphore.RunWithSemaphore(
		myPipelineFunc,
		myDataToProcess,
		maxGoRoutines,
		errChan,
	)

    for err := range errorsChan {
		fmt.Printf("Some Error Ocurred: %v\n", err)
	}
}
```