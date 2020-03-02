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

// Package containing logic for parsing rule content.

package content

import (
	"io/ioutil"
	"path"
)

// RuleErrorContent wraps content of a single error key.
type RuleErrorContent struct {
	Generic  []byte
	Metadata []byte
}

// RuleContent wraps all the content available for a rule into a single structure.
type RuleContent struct {
	Summary    []byte
	Reason     []byte
	Resolution []byte
	MoreInfo   []byte
	Plugin     []byte
	Errors     map[string]RuleErrorContent
}

// parseErrorContents reads the contents of the specified directory
// and parses all subdirectories as error key contents.
// This implicitly checks that the directory exists,
// so it is not necessary to ever check that elsewhere.
func parseErrorContents(ruleDirPath string) (map[string]RuleErrorContent, error) {
	entries, err := ioutil.ReadDir(ruleDirPath)
	if err != nil {
		return nil, err
	}

	errorContents := map[string]RuleErrorContent{}

	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()

			generic, err := ioutil.ReadFile(path.Join(ruleDirPath, name, "generic.md"))
			if err != nil {
				return nil, err
			}

			metadata, err := ioutil.ReadFile(path.Join(ruleDirPath, name, "metadata.yaml"))
			if err != nil {
				return nil, err
			}

			errorContents[name] = RuleErrorContent{
				Generic:  generic,
				Metadata: metadata,
			}
		}
	}

	return errorContents, nil
}

// Parse attempts to parse all available rule content from the specified directory.
func Parse(ruleDirPath string) (RuleContent, error) {
	errorContents, err := parseErrorContents(ruleDirPath)
	if err != nil {
		return RuleContent{}, err
	}

	summary, err := ioutil.ReadFile(path.Join(ruleDirPath, "summary.md"))
	if err != nil {
		return RuleContent{}, err
	}

	reason, err := ioutil.ReadFile(path.Join(ruleDirPath, "reason.md"))
	if err != nil {
		return RuleContent{}, err
	}

	resolution, err := ioutil.ReadFile(path.Join(ruleDirPath, "resolution.md"))
	if err != nil {
		return RuleContent{}, err
	}

	moreInfo, err := ioutil.ReadFile(path.Join(ruleDirPath, "more_info.md"))
	if err != nil {
		return RuleContent{}, err
	}

	plugin, err := ioutil.ReadFile(path.Join(ruleDirPath, "plugin.yaml"))
	if err != nil {
		return RuleContent{}, err
	}

	return RuleContent{
		Summary:    summary,
		Reason:     reason,
		Resolution: resolution,
		MoreInfo:   moreInfo,
		Plugin:     plugin,
		Errors:     errorContents,
	}, nil
}
