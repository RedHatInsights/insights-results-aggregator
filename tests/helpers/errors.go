package helpers

import (
	"testing"
)

// FailOnError wraps result of function with one argument
func FailOnError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
