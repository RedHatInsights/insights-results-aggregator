/*
Copyright © 2019, 2020 Red Hat, Inc.

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

package rules

import (
	"github.com/robfig/cron"
	"log"
	"os/exec"
)

// Updater represents
type Updater struct {
	Config Configuration
}

// New constructs a Updater from configuration, enables cron job
func New(config Configuration) *Updater {
	/*
	    updater := Updater{config}
		crontab := cron.New()
	    crontab.AddFunc(config.crontabConfig, updater.UpdateInsightsRulesContent)
	*/
}

// UpdateInsightsRulesContent runs the script update_insights_rules_content.sh.
// Ran either by cron or on demand
func (u *Updater) UpdateInsightsRulesContent() error {
	log.Println("Updating insights rules content")
	updateScript := u.Config.contentUpdateScript
	cmd := exec.Command(updateScript)
	err := cmd.Run()
	if err != nil {
		log.Printf("Error updating the insights rules content: %v", err)
		return err
	}
	return nil
}
