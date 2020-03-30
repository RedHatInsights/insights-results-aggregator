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

import (
	"fmt"
	"strings"

	"github.com/verdverm/frisby"
)

// checkPrometheusMetrics checks the Prometheus metrics API endpoint
func checkPrometheusMetrics() {
	f := frisby.Create("Check the Prometheus metrics API endpoint").Get(apiURL + "metrics")
	f.Send()
	f.ExpectStatus(200)
	// the content type header set by metrics handler is a bit complicated
	// but it must start with "text/plain" in any case
	f.Expect(func(F *frisby.Frisby) (bool, string) {
		header := F.Resp.Header.Get(contentTypeHeader)
		if strings.HasPrefix(header, "text/plain") {
			return true, "ok"
		}
		return false, fmt.Sprintf("Expected Header %q to be %q, but got %q", contentTypeHeader, "text/plain", header)
	})
	f.PrintReport()
}
