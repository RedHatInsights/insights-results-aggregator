// +build testrunmain

package main_test

import (
	"fmt"
	"os"
	"os/signal"
	"testing"

	main "github.com/RedHatInsights/insights-results-aggregator"
)

func TestRunMain(t *testing.T) {
	interruptSignal := make(chan os.Signal, 1)
	signal.Notify(interruptSignal, os.Interrupt)
	go func() {
		<-interruptSignal
		errCode := main.StopService()
		if errCode != 0 {
			panic(fmt.Sprintf("service has exited with a code %v", errCode))
		}
	}()

	fmt.Println("starting...")
	os.Args = []string{"./insights-results-aggregator", "start-service"}
	main.Main()
	fmt.Println("exiting...")
}
