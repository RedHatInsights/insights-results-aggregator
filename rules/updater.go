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
	"context"
	"github.com/robfig/cron/v3"
	"log"
	"os/exec"
)

// Updater represents
type Updater struct {
	Config  Configuration
	Crontab *cron.Cron
}

// NewUpdater constructs an Updater from configuration, configures Cron job, but doesn't start it
func NewUpdater(config Configuration) *Updater {
	updater := Updater{Config: config}

	crontab := cron.New()
	crontab.AddFunc(config.CronJobConfig, updater.UpdateInsightsRulesContentCron)
	updater.Crontab = crontab

	return &updater
}

// StartUpdater starts the crontab
func (u *Updater) StartUpdater() {
	u.Crontab.Start()
}

// StopUpdater stops the crontab and returns context of running jobs
func (u *Updater) StopUpdater() context.Context {
	return u.Crontab.Stop()
}

// UpdateInsightsRulesContent runs the script update_insights_rules_content.sh.
// Ran either by cron or on demand
func (u *Updater) UpdateInsightsRulesContent() error {
	log.Println("Updating insights rules content")

	cmd := exec.Command(u.Config.ContentUpdateScript)
	err := cmd.Run()
	if err != nil {
		log.Printf("Error during the execution of the insights rules content update script: %v", err)
		return err
	}

	return nil
}

// UpdateInsightsRulesContentCron is just a wrapper for UpdateInsightsRulesContent
// beacuse the functions passed to cron.AddFunc mustn't return anything.
func (u *Updater) UpdateInsightsRulesContentCron() {
	err := u.UpdateInsightsRulesContent()
	if err != nil {
		log.Printf("Error during periodic Insights rules content update: %v", err)
	}
}
