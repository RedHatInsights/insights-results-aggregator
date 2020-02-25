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

package tests

import "github.com/verdverm/frisby"

const apiURL = "http://localhost:8080/api/v1/"

func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API").Get(apiURL)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader("Content-Type", "application/json; charset=utf-8")
	f.PrintReport()
}

func checkNonExistentEntryPoint() {
	f := frisby.Create("Check the non-existent entry point to REST API").Get(apiURL + "foobar")
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader("Content-Type", "text/plain; charset=utf-8")
	f.PrintReport()
}

func checkWrongEntryPoint() {
	f := frisby.Create("Check the wrong entry point to REST API").Get(apiURL + "../")
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader("Content-Type", "text/plain; charset=utf-8")
	f.PrintReport()
}

func checkWrongMethodsForEntryPoint() {
	f := frisby.Create("Check the entry point to REST API with wrong method: POST").Post(apiURL)
	f.Send()
	f.ExpectStatus(405)
	f.PrintReport()

	f = frisby.Create("Check the entry point to REST API with wrong method: PUT").Put(apiURL)
	f.Send()
	f.ExpectStatus(405)
	f.PrintReport()

	f = frisby.Create("Check the entry point to REST API with wrong method: DELETE").Delete(apiURL)
	f.Send()
	f.ExpectStatus(405)
	f.PrintReport()
}

func checkOpenAPISpecifications() {
	f := frisby.Create("Check the wrong entry point to REST API").Get(apiURL + "openapi.json")
	f.Send()
	f.ExpectStatus(200)
	f.PrintReport()
}

// ServerTests run all tests for basic REST API endpoints
func ServerTests() {
	checkRestAPIEntryPoint()
	checkNonExistentEntryPoint()
	checkWrongEntryPoint()
	checkWrongMethodsForEntryPoint()
	checkOpenAPISpecifications()
}
