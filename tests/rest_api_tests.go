/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Implementation of REST API tests that checks all REST API endpoints of
// Insights aggregator service.
//
// These test should be started by using one of following commands in order to be configured properly:
//
//   ./test.sh rest_api
//   make rest_api_tests
package main

import (
	"os"

	tests "github.com/RedHatInsights/insights-results-aggregator/tests/rest"
	"github.com/verdverm/frisby"
)

func main() {
	tests.ServerTests()
	frisby.Global.PrintReport()
	os.Exit(frisby.Global.NumErrored)
}
