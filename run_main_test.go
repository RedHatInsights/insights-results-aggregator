// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build testrunmain
// +build testrunmain

package main_test

import (
	"fmt"
	"os"
	"os/signal"
	"testing"

	main "github.com/RedHatInsights/insights-results-aggregator"
)

// TestRunMain function tries to start the service in test context with the
// ability to stop the service.
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
