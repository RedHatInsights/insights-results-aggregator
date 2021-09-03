// Copyright 2021 Red Hat, Inc
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

package amsclient_test

import (
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/amsclient"
	"github.com/rs/zerolog"
)

var defaultConfig amsclient.Configuration = amsclient.Configuration{
	Token: "",
	URL:   "https://api.stage.openshift.com",
}

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// TestClientCreation tests if creating a client works as expected
func TestClientCreationError(t *testing.T) {
	_, err := amsclient.NewAMSClient(defaultConfig)
	if err == nil {
		t.Fail()
	}
}
