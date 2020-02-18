package helpers

import (
	"testing"
	"time"
)

// TestFunctionPtr pointer to test function
type TestFunctionPtr = func(*testing.T)

// RunTestWithTimeout runs test with timeToRun timeout and fails if it wasn't in time
func RunTestWithTimeout(t *testing.T, test TestFunctionPtr, timeToRun time.Duration) {
	timeout := time.After(timeToRun)
	done := make(chan bool)

	go func() {
		test(t)
		done <- true
	}()

	select {
	case <-timeout:
		t.Fatal("Test ran out of time")
	case <-done:
	}
}
