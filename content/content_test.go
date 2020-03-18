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

package content_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/content"
)

const errYAMLBadToken = "yaml: line 14: found character that cannot start any token"

// TestContentParseOK checks whether reading from directory of correct content works as expected
func TestContentParseOK(t *testing.T) {
	con, err := content.ParseRuleContentDir("../tests/content/ok/")
	if err != nil {
		t.Fatal(err)
	}

	rule1Content, exists := con["rule1"]
	if !exists {
		t.Fatal("'rule1' content is missing")
	}

	_, exists = rule1Content.ErrorKeys["err_key"]
	if !exists {
		t.Fatal("'err_key' error content is missing")
	}
}

// TestContentParseInvalidDir checks how incorrect (non-existing) directory is handled
func TestContentParseInvalidDir(t *testing.T) {
	const invalidDirPath = "../tests/content/not-a-real-dir/"
	_, err := content.ParseRuleContentDir(invalidDirPath)
	if err == nil || err.Error() != fmt.Sprintf("open %s: no such file or directory", invalidDirPath) {
		t.Fatal(err)
	}
}

// TestContentParseNotDirectory1 checks how incorrect (non-existing) directory is handled
func TestContentParseNotDirectory1(t *testing.T) {
	// this is not a proper directory
	const notADirPath = "../tests/tests.toml"
	_, err := content.ParseRuleContentDir(notADirPath)
	if err == nil || err.Error() != "readdirent: not a directory" {
		t.Fatal(err)
	}
}

// TestContentParseNotDirectory2 checks how incorrect (non-existing) directory is handled
func TestContentParseInvalidDir2(t *testing.T) {
	// this is not a proper directory
	const notADirPath = "/dev/null"
	_, err := content.ParseRuleContentDir(notADirPath)
	if err == nil || err.Error() != "readdirent: not a directory" {
		t.Fatal(err)
	}
}

// TestContentParseMissingFile checks how missing file(s) in content directory are handled
func TestContentParseMissingFile(t *testing.T) {
	_, err := content.ParseRuleContentDir("../tests/content/missing/")
	if err == nil || !strings.HasSuffix(err.Error(), ": no such file or directory") {
		t.Fatal(err)
	}
}

// TestContentParseBadPluginYAML tests handling bad/incorrect plugin.yaml file
func TestContentParseBadPluginYAML(t *testing.T) {
	_, err := content.ParseRuleContentDir("../tests/content/bad_plugin/")
	if err == nil || err.Error() != errYAMLBadToken {
		t.Fatal(err)
	}
}

// TestContentParseBadMetadataYAML tests handling bad/incorrect metadata.yaml file
func TestContentParseBadMetadataYAML(t *testing.T) {
	_, err := content.ParseRuleContentDir("../tests/content/bad_metadata/")
	if err == nil || err.Error() != errYAMLBadToken {
		t.Fatal(err)
	}
}
