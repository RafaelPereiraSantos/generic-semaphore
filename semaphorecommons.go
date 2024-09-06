package gsemaphore

import (
	"errors"
)

type (
	worker struct {
		id string
	}

	WorkerKey string

	strategyFollowUp func()
)

const (
	WorkerIDcontextKey       WorkerKey = "swid"
	defaultParallelismAmount int       = 3
)

var (
	ErrSempahoreTimeout = errors.New("semaphore pipeline timeout")
)
