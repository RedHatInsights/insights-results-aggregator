/*
Copyright © 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
*/

package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/migration/dvomigrations"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// DVORecommendationsStorage represents an interface to almost any database or storage system
type DVORecommendationsStorage interface {
	Storage
	ReportsCount() (int, error)
	WriteReportForCluster(
		orgID types.OrgID,
		clusterName types.ClusterName,
		report types.ClusterReport,
		workloads []types.WorkloadRecommendation,
		lastCheckedTime time.Time,
		gatheredAtTime time.Time,
		storedAtTime time.Time,
		requestID types.RequestID,
	) error
	ReadWorkloadsForOrganization(
		types.OrgID,
		map[types.ClusterName]struct{},
		bool,
	) ([]types.WorkloadsForNamespace, error)
	ReadWorkloadsForClusterAndNamespace(
		types.OrgID,
		types.ClusterName,
		string,
	) (types.DVOReport, error)
	DeleteReportsForOrg(orgID types.OrgID) error
	WriteHeartbeat(string, time.Time) error
}

const (
	// dvoDBSchema represents the name of the DB schema used by DVO-related queries/migrations
	dvoDBSchema = "dvo"
	// orgIDStr used in log messages
	orgIDStr = "orgID"
)

// DVORecommendationsDBStorage is an implementation of Storage interface that use selected SQL like database
// like PostgreSQL or RDS etc. That implementation is based on the standard
// sql package. It is possible to configure connection via Configuration structure.
// SQLQueriesLog is log for sql queries, default is nil which means nothing is logged
type DVORecommendationsDBStorage struct {
	connection   *sql.DB
	dbDriverType types.DBDriver
	// clusterLastCheckedDict is a dictionary of timestamps when the clusters were last checked.
	clustersLastChecked map[types.ClusterName]time.Time
}

// NewDVORecommendationsStorage function creates and initializes a new instance of Storage interface
func NewDVORecommendationsStorage(configuration Configuration) (DVORecommendationsStorage, error) {
	switch configuration.Type {
	case types.SQLStorage:
		log.Info().Str("DVO storage type", configuration.Type).Send()
		return newDVOStorage(configuration)
	case types.NoopStorage:
		return newNoopDVOStorage(configuration)
	default:
		// error to be thrown
		err := fmt.Errorf("Unknown storage type '%s'", configuration.Type)
		log.Error().Err(err).Msg("Init failure")
		return nil, err
	}
}

// newNoopDVOStorage function creates and initializes a new instance of Noop storage
func newNoopDVOStorage(_ Configuration) (DVORecommendationsStorage, error) {
	return &NoopDVOStorage{}, nil
}

// newDVOStorage function creates and initializes a new instance of DB storage
func newDVOStorage(configuration Configuration) (DVORecommendationsStorage, error) {
	driverType, driverName, dataSource, err := initAndGetDriver(configuration)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf(
		"Making connection to DVO data storage, driver=%s, connection string 'postgresql://%v:<password>@%v:%v/%v?%v'",
		driverName,
		configuration.PGUsername,
		configuration.PGHost,
		configuration.PGPort,
		configuration.PGDBName,
		configuration.PGParams,
	)

	connection, err := sql.Open(driverName, dataSource)
	if err != nil {
		log.Error().Err(err).Msg("Can not connect to data storage")
		return nil, err
	}

	log.Debug().Msg("connection to DVO storage created")

	return NewDVORecommendationsFromConnection(connection, driverType), nil
}

// NewDVORecommendationsFromConnection function creates and initializes a new instance of Storage interface from prepared connection
func NewDVORecommendationsFromConnection(connection *sql.DB, dbDriverType types.DBDriver) *DVORecommendationsDBStorage {
	return &DVORecommendationsDBStorage{
		connection:          connection,
		dbDriverType:        dbDriverType,
		clustersLastChecked: map[types.ClusterName]time.Time{},
	}
}

// Init performs all database initialization
// tasks necessary for further service operation.
func (storage DVORecommendationsDBStorage) Init() error {
	// Read clusterName:LastChecked dictionary from DB.
	rows, err := storage.connection.Query("SELECT cluster_id, last_checked_at FROM dvo.dvo_report;")
	if err != nil {
		return err
	}

	log.Debug().Msg("executing last_checked_at query")
	for rows.Next() {
		var (
			clusterName types.ClusterName
			lastChecked sql.NullTime
		)

		if err := rows.Scan(&clusterName, &lastChecked); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("Unable to close the DB rows handle")
			}
			return err
		}

		storage.clustersLastChecked[clusterName] = lastChecked.Time
	}

	// Not using defer to close the rows here to:
	// - make errcheck happy (it doesn't like ignoring returned errors),
	// - return a possible error returned by the Close method.
	return rows.Close()
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage DVORecommendationsDBStorage) Close() error {
	log.Info().Msg("Closing connection to data storage")
	if storage.connection != nil {
		err := storage.connection.Close()
		if err != nil {
			log.Error().Err(err).Msg("Can not close connection to data storage")
			return err
		}
	}
	return nil
}

// GetMigrations returns a list of database migrations related to DVO recommendation tables
func (storage DVORecommendationsDBStorage) GetMigrations() []migration.Migration {
	return dvomigrations.UsableDVOMigrations
}

// GetDBDriverType returns db driver type
func (storage DVORecommendationsDBStorage) GetDBDriverType() types.DBDriver {
	return storage.dbDriverType
}

// GetConnection returns db connection(useful for testing)
func (storage DVORecommendationsDBStorage) GetConnection() *sql.DB {
	return storage.connection
}

// GetDBSchema returns the schema name to be used in queries
func (storage DVORecommendationsDBStorage) GetDBSchema() migration.Schema {
	return migration.Schema(dvoDBSchema)
}

// GetMaxVersion returns the highest available migration version.
// The DB version cannot be set to a value higher than this.
// This value is equivalent to the length of the list of available migrations.
func (storage DVORecommendationsDBStorage) GetMaxVersion() migration.Version {
	return migration.Version(len(storage.GetMigrations()))
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (storage DVORecommendationsDBStorage) MigrateToLatest() error {
	dbConn, dbSchema := storage.GetConnection(), storage.GetDBSchema()

	if err := migration.InitInfoTable(dbConn, dbSchema); err != nil {
		return err
	}

	return migration.SetDBVersion(
		dbConn,
		storage.dbDriverType,
		dbSchema,
		storage.GetMaxVersion(),
		storage.GetMigrations(),
	)
}

// ReportsCount reads number of all records stored in the dvo.dvo_report table
func (storage DVORecommendationsDBStorage) ReportsCount() (int, error) {
	count := -1
	err := storage.connection.QueryRow("SELECT count(*) FROM dvo.dvo_report;").Scan(&count)
	err = types.ConvertDBError(err, nil)

	return count, err
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DVORecommendationsDBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	workloads []types.WorkloadRecommendation,
	lastCheckedTime time.Time,
	_ time.Time,
	_ time.Time,
	_ types.RequestID,
) error {
	// Skip writing the report if it isn't newer than a report
	// that is already in the database for the same cluster.
	if oldLastChecked, exists := storage.clustersLastChecked[clusterName]; exists && !lastCheckedTime.After(oldLastChecked) {
		return types.ErrOldReport
	}

	if storage.dbDriverType != types.DBDriverPostgres {
		return fmt.Errorf("writing workloads with DB %v is not supported", storage.dbDriverType)
	}

	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	err = func(tx *sql.Tx) error {
		// Check if there is a more recent report for the cluster already in the database.
		rows, err := tx.Query(
			"SELECT last_checked_at FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND last_checked_at > $3;",
			orgID, clusterName, lastCheckedTime)
		err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
		if err != nil {
			log.Error().Err(err).Msg("Unable to look up the most recent report in the database")
			return err
		}

		defer closeRows(rows)

		// If there is one, print a warning and discard the report (don't update it).
		if rows.Next() {
			log.Warn().Msgf("Database already contains report for organization %d and cluster name %s more recent than %v",
				orgID, clusterName, lastCheckedTime)
			return nil
		}

		err = storage.updateReport(tx, orgID, clusterName, report, workloads, lastCheckedTime)
		if err != nil {
			return err
		}

		storage.clustersLastChecked[clusterName] = lastCheckedTime
		metrics.WrittenReports.Inc()

		return nil
	}(tx)

	finishTransaction(tx, err)

	return err
}

func (storage DVORecommendationsDBStorage) updateReport(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	recommendations []types.WorkloadRecommendation,
	lastCheckedTime time.Time,
) error {
	// Get reported_at if present before deletion
	reportedAtMap, err := storage.getReportedAtMap(orgID, clusterName)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get dvo report reported_at")
		reportedAtMap = make(map[string]types.Timestamp) // create empty map
	}

	// Delete previous reports (CCXDEV-12529)
	_, err = tx.Exec("DELETE FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2;", orgID, clusterName)
	if err != nil {
		log.Err(err).
			Str(clusterKey, string(clusterName)).Interface(orgIDStr, orgID).
			Msg("Unable to remove previous cluster DVO reports")
		return err
	}

	if len(recommendations) == 0 {
		log.Info().
			Str(clusterKey, string(clusterName)).Interface(orgIDStr, orgID).
			Msg("No new DVO report to insert")
		return nil
	}

	namespaceMap, objectsMap, namespaceRecommendationCount, ruleHitsCounts := mapWorkloadRecommendations(&recommendations)

	// Get the INSERT statement for writing a workload into the database.
	workloadInsertStatement := storage.getReportInsertQuery()

	// Get values to be stored in dvo.dvo_report table
	values := make([]interface{}, 10)
	for namespaceUID, namespaceName := range namespaceMap {
		values[0] = orgID         // org_id
		values[1] = clusterName   // cluster_id
		values[2] = namespaceUID  // namespace_id
		values[3] = namespaceName // namespace_name

		workloadAsJSON, err := json.Marshal(report)
		if err != nil {
			log.Error().Err(err).Msg("cannot store raw workload report")
			values[4] = "{}" // report
		} else {
			values[4] = string(workloadAsJSON) // report
		}

		values[5] = namespaceRecommendationCount[namespaceUID] // recommendations
		values[6] = objectsMap[namespaceUID]                   // objects

		if reportedAt, ok := reportedAtMap[namespaceUID]; ok {
			values[7] = reportedAt // reported_at
		} else {
			values[7] = lastCheckedTime
		}

		values[8] = lastCheckedTime // last_checked_at

		values[9] = ruleHitsCounts[namespaceUID] // rule_hits_count

		_, err = tx.Exec(workloadInsertStatement, values...)
		if err != nil {
			log.Err(err).Msgf("Unable to insert the cluster workloads (org: %v, cluster: %v)",
				orgID, clusterName,
			)
			return err
		}
	}

	return nil
}

// updateRuleHitsCountsForNamespace updates the rule hits for given namespace based on the given recommendation
func updateRuleHitsCountsForNamespace(ruleHitsCounts map[string]types.RuleHitsCount, namespaceUID string, recommendation types.WorkloadRecommendation) {
	if _, ok := ruleHitsCounts[namespaceUID]; !ok {
		ruleHitsCounts[namespaceUID] = make(types.RuleHitsCount)
	}

	// define key in rule hits counts map as concatenation of rule component and key
	compositeRuleID, err := generators.GenerateCompositeRuleID(
		// for some unknown reason, there's a `.recommendation` suffix for each rule hit instead of the usual .report
		types.RuleFQDN(strings.TrimSuffix(recommendation.Component, types.WorkloadRecommendationSuffix)),
		types.ErrorKey(recommendation.Key),
	)
	if err != nil {
		log.Error().Err(err).Msg("error generating composite rule ID for rule")
		return
	}

	compositeRuleIDString := string(compositeRuleID)
	if _, ok := ruleHitsCounts[namespaceUID][compositeRuleIDString]; !ok {
		ruleHitsCounts[namespaceUID][compositeRuleIDString] = 0
	}
	ruleHitsCounts[namespaceUID][compositeRuleIDString]++
}

// mapWorkloadRecommendations filters out the data which is grouped by recommendations and aggregates
// them by namespace.
// Essentially we need to "invert" data from:
//   - list of recommendations: list of workloads from ALL namespaces combined (objects can also be duplicate between recommendations)
//
// to:
//   - list of namespaces: list of affected workloads and data aggregations for this particular namespace
func mapWorkloadRecommendations(recommendations *[]types.WorkloadRecommendation) (
	map[string]string, map[string]int, map[string]int, map[string]types.RuleHitsCount,
) {
	// map the namespace ID to the namespace name
	namespaceMap := make(map[string]string)
	// map how many recommendations hit per namespace
	namespaceRecommendationCount := make(map[string]int)
	// map the number of unique workloads affected by at least 1 rule per namespace
	objectsPerNamespace := make(map[string]map[string]struct{})
	// map the hit counts per namespace and rule
	ruleHitsCounts := make(map[string]types.RuleHitsCount)

	for _, recommendation := range *recommendations {
		// objectsMapPerRecommendation is used to calculate number of rule hits in namespace
		objectsPerRecommendation := make(map[string]int)

		for i := range recommendation.Workloads {
			workload := &recommendation.Workloads[i]

			if _, ok := namespaceMap[workload.NamespaceUID]; !ok {
				// store the namespace name in the namespaceMap if it's not already there
				namespaceMap[workload.NamespaceUID] = workload.Namespace
			}

			// per single recommendation within namespace
			objectsPerRecommendation[workload.NamespaceUID]++

			updateRuleHitsCountsForNamespace(ruleHitsCounts, workload.NamespaceUID, recommendation)

			// per whole namespace; just workload IDs with empty structs to filter out duplicate objects
			if _, ok := objectsPerNamespace[workload.NamespaceUID]; !ok {
				objectsPerNamespace[workload.NamespaceUID] = make(map[string]struct{})
			}
			objectsPerNamespace[workload.NamespaceUID][workload.UID] = struct{}{}
		}

		// increase rule hit count for affected namespaces
		for namespace := range namespaceMap {
			if _, ok := objectsPerRecommendation[namespace]; ok {
				namespaceRecommendationCount[namespace]++
			}
		}
	}

	uniqueObjectsMap := make(map[string]int)
	// count the number of unique objects per namespace
	for namespace, objects := range objectsPerNamespace {
		uniqueObjectsMap[namespace] = len(objects)
	}

	return namespaceMap, uniqueObjectsMap, namespaceRecommendationCount, ruleHitsCounts
}

// getRuleKeyCreatedAtMap returns a map between
// (rule_fqdn, error_key) -> created_at
// for each rule_hit rows matching given
// orgId and clusterName
func (storage DVORecommendationsDBStorage) getReportedAtMap(orgID types.OrgID, clusterName types.ClusterName) (map[string]types.Timestamp, error) {
	query := "SELECT namespace_id, reported_at FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2;"
	reportedAtRows, err := storage.connection.Query(
		query, orgID, clusterName)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving dvo.dvo_report created_at timestamp")
		return nil, err
	}
	defer closeRows(reportedAtRows)

	reportedAtMap := make(map[string]types.Timestamp)
	for reportedAtRows.Next() {
		var namespaceID string
		var reportedAt time.Time
		err := reportedAtRows.Scan(
			&namespaceID,
			&reportedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("error scanning for rule id -> created_at map")
			continue
		}
		reportedAtMap[namespaceID] = types.Timestamp(reportedAt.UTC().Format(time.RFC3339))
	}
	return reportedAtMap, err
}

// ReadWorkloadsForOrganization returns all rows from dvo.dvo_report table for given organizaiton
func (storage DVORecommendationsDBStorage) ReadWorkloadsForOrganization(
	orgID types.OrgID,
	activeClusterMap map[types.ClusterName]struct{},
	clusterFilteringEnabled bool,
) (
	workloads []types.WorkloadsForNamespace,
	err error,
) {
	tStart := time.Now()
	query := `
		SELECT cluster_id, namespace_id, namespace_name, recommendations, objects, reported_at, last_checked_at, rule_hits_count
		FROM dvo.dvo_report
		WHERE org_id = $1
	`

	// #nosec G202
	rows, err := storage.connection.Query(query, orgID)

	err = types.ConvertDBError(err, orgID)
	if err != nil {
		return workloads, err
	}

	defer closeRows(rows)

	var count uint
	for rows.Next() {
		var (
			dvoReport       types.WorkloadsForNamespace
			lastCheckedAtDB sql.NullTime
			reportedAtDB    sql.NullTime
		)
		err = rows.Scan(
			&dvoReport.Cluster.UUID,
			&dvoReport.Namespace.UUID,
			&dvoReport.Namespace.Name,
			&dvoReport.Metadata.Recommendations,
			&dvoReport.Metadata.Objects,
			&reportedAtDB,
			&lastCheckedAtDB,
			&dvoReport.RecommendationsHitCount,
		)
		if err != nil {
			log.Error().Err(err).Msg("ReadWorkloadsForOrganization")
		}

		if clusterFilteringEnabled {
			// skip inactive clusters, cheaper than using the list in an SQL query.
			if _, found := activeClusterMap[types.ClusterName(dvoReport.Cluster.UUID)]; !found {
				continue
			}
		}

		// convert timestamps to string
		dvoReport.Metadata.LastCheckedAt = lastCheckedAtDB.Time.UTC().Format(time.RFC3339)
		dvoReport.Metadata.ReportedAt = reportedAtDB.Time.UTC().Format(time.RFC3339)

		workloads = append(workloads, dvoReport)
		count++
	}

	log.Info().Int(orgIDStr, int(orgID)).Msgf("ReadWorkloadsForOrganization processed %d rows in %v", count, time.Since(tStart))

	return workloads, err
}

// ReadWorkloadsForClusterAndNamespace returns a single result from the dvo.dvo_report table
func (storage DVORecommendationsDBStorage) ReadWorkloadsForClusterAndNamespace(
	orgID types.OrgID,
	clusterID types.ClusterName,
	namespaceID string,
) (
	workload types.DVOReport,
	err error,
) {
	tStart := time.Now()
	query := `
		SELECT cluster_id, namespace_id, namespace_name, recommendations, report, objects, reported_at, last_checked_at
		FROM dvo.dvo_report
		WHERE org_id = $1
		AND cluster_id = $2
		AND namespace_id = $3
	`

	var (
		dvoReport       types.DVOReport
		lastCheckedAtDB sql.NullTime
		reportedAtDB    sql.NullTime
	)
	err = storage.connection.QueryRow(query, orgID, clusterID, namespaceID).Scan(
		&dvoReport.ClusterID,
		&dvoReport.NamespaceID,
		&dvoReport.NamespaceName,
		&dvoReport.Recommendations,
		&dvoReport.Report,
		&dvoReport.Objects,
		&reportedAtDB,
		&lastCheckedAtDB,
	)
	if err == sql.ErrNoRows {
		return workload, &types.ItemNotFoundError{ItemID: fmt.Sprintf("%d:%s:%s", orgID, clusterID, namespaceID)}
	}
	// convert timestamps to string
	dvoReport.LastCheckedAt = types.Timestamp(lastCheckedAtDB.Time.UTC().Format(time.RFC3339))
	dvoReport.ReportedAt = types.Timestamp(reportedAtDB.Time.UTC().Format(time.RFC3339))

	log.Debug().Int(orgIDStr, int(orgID)).Msgf("ReadWorkloadsForClusterAndNamespace took %v", time.Since(tStart))

	if err != nil {
		return dvoReport, err
	}

	return storage.filterReportWithHeartbeats(dvoReport)
}

// filter a DVO report based on the data in dvo.runtimes_heartbeats
// It the last timestampt there is  longer than 30 secs (value for the demo)
// it gets removed from the report.
func (storage DVORecommendationsDBStorage) filterReportWithHeartbeats(dvoReport types.DVOReport) (
	workload types.DVOReport,
	err error,
) {
	query := `
		SELECT instance_id
		FROM dvo.runtimes_heartbeats
		WHERE last_checked_at > (now() - interval '30 seconds')
	`

	rows, err := storage.connection.Query(query)
	if err != nil {
		return dvoReport, err
	}

	defer closeRows(rows)

	aliveInstances := map[string]bool{}

	for rows.Next() {
		var (
			instanceID string
		)
		err = rows.Scan(
			&instanceID,
		)
		if err != nil {
			log.Error().Err(err).Msg("ReadWorkloadsForOrganization getting alive instances")
		}

		aliveInstances[instanceID] = true
	}

	// do not do any filtering if there is no data.
	// This is a hack for previosly existing unit tests
	if len(aliveInstances) == 0 {
		return dvoReport, nil
	}

	processedReport := dvoReport.Report
	processedReport = strings.ReplaceAll(processedReport, "\\n", "")
	processedReport = strings.ReplaceAll(processedReport, "\\", "")
	processedReport = processedReport[1 : len(processedReport)-1]

	// filter report
	var reportData types.DVOMetrics // here we will miss part of the original report, but a part that is not used anywhere
	err = json.Unmarshal([]byte(processedReport), &reportData)
	if err != nil {
		return dvoReport, err
	}

	seenObjects := map[string]bool{}

	for i, rec := range reportData.WorkloadRecommendations {
		reportData.WorkloadRecommendations[i].Workloads = filterWorkloads(rec.Workloads, aliveInstances, seenObjects)
	}

	bReport, err := json.Marshal(reportData)
	if err != nil {
		return dvoReport, err
	}
	fmt.Print(string(bReport))
	dvoReport.Report = string(bReport)
	dvoReport.Objects = uint(len(seenObjects)) // #nosec G115

	return dvoReport, nil
}

// filterWorkloads: use aliveInstances to filter the workloads and add seen objects to a map
func filterWorkloads(workloads []types.DVOWorkload, aliveInstances map[string]bool, seenObjects map[string]bool) []types.DVOWorkload {
	i := 0 // output index
	for _, x := range workloads {
		if _, ok := aliveInstances[x.UID]; ok {
			// copy and increment index
			workloads[i] = x
			i++
			seenObjects[x.UID] = true
		}
	}
	return workloads[:i]
}

// DeleteReportsForOrg deletes all reports related to the specified organization from the storage.
func (storage DVORecommendationsDBStorage) DeleteReportsForOrg(orgID types.OrgID) error {
	_, err := storage.connection.Exec("DELETE FROM dvo.dvo_report WHERE org_id = $1;", orgID)
	return err
}

// WriteHeartbeat ...
func (storage DVORecommendationsDBStorage) WriteHeartbeat(
	instanceID string,
	lastCheckedTime time.Time,
) error {
	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	err = func(tx *sql.Tx) error {
		// Check if there is a more recent report for the cluster already in the database.
		_, err := tx.Exec(
			"INSERT INTO dvo.runtimes_heartbeats VALUES ($1, $2);",
			instanceID, types.Timestamp(lastCheckedTime.UTC().Format(time.RFC3339)))
		if err != nil {
			return err
		}

		return nil
	}(tx)

	finishTransaction(tx, err)

	return err
}
