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

// Package tests contains REST API tests for following endpoints:
//
// apiPrefix
package tests

import (
	"github.com/verdverm/frisby"
)

// checkRestAPIEntryPoint check if the entry point (usually /api/v1/) responds correctly to HTTP GET command
func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API using HTTP GET method").Get(apiURL)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkNonExistentEntryPoint check whether non-existing endpoints are handled properly (HTTP code 404 etc.)
func checkNonExistentEntryPoint() {
	f := frisby.Create("Check the non-existent entry point to REST API").Get(apiURL + "foobar")
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeText)
	f.PrintReport()
}

// checkWrongEntryPoint check whether wrongly specified URLs are handled correctly
func checkWrongEntryPoint() {
	postfixes := [...]string{"..", "../", "...", "..?", "..?foobar"}
	for _, postfix := range postfixes {
		f := frisby.Create("Check the wrong entry point to REST API with postfix '" + postfix + "'").Get(apiURL + postfix)
		setAuthHeader(f)
		f.Send()
		f.ExpectStatus(404)
		f.ExpectHeader(contentTypeHeader, ContentTypeText)
		f.PrintReport()
	}
}

// check whether other HTTP methods are rejected correctly for the REST API entry point
func checkWrongMethodsForEntryPoint() {
	checkGetEndpointByOtherMethods(apiURL, false)
}
