package main_test

import (
	"os"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	main "github.com/RedHatInsights/insights-results-aggregator"
	"github.com/stretchr/testify/assert"
)

func TestStartHearbeatsConsumer(t *testing.T) {
	// It is necessary to perform migrations for this test
	// because the service won't run on top of an empty DB.
	*main.AutoMigratePtr = true

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		os.Clearenv()
		mustLoadConfiguration("./tests/tests")

		setEnvSettings(t, map[string]string{
			"INSIGHTS_RESULTS_AGGREGATOR__STORAGE_BACKEND__USE": "dvo_recommendations",
			"INSIGHTS_RESULTS_AGGREGATOR__METRICS__NAMESPACE":   "dvo_writer",
		})

		go func() {
			main.StartHeartbeatService()
		}()

		errCode := main.StopHeartbeatService()
		assert.Equal(t, main.ExitStatusOK, errCode)
	}, testsTimeout)

	*main.AutoMigratePtr = false
}
