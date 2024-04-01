package dvomigrations

import "github.com/RedHatInsights/insights-results-aggregator/migration"

// UsableDVOMigrations contains all usable DVO-related migrations
var UsableDVOMigrations = []migration.Migration{
	mig0001CreateDVOReport,
	mig0002CreateDVOReportIndexes,
	mig0003CCXDEV12602DeleteBuggyRecords,
}
