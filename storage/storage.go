/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
*/

// Package storage contains an implementation of interface between Go code and
// (almost any) SQL database like PostgreSQL, SQLite, or MariaDB. An implementation
// named DBStorage is constructed via New function and it is mandatory to call Close
// for any opened connection to database. The storage might be initialized by Init
// method if database schema is empty.
//
// It is possible to configure connection to selected database by using Configuration
// structure. Currently that structure contains two configurable parameter:
//
// Driver - a SQL driver, like "sqlite3", "pq" etc.
// DataSource - specification of data source. The content of this parameter depends on the database used.
package storage

import (
	"database/sql"
	sql_driver "database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/lib/pq"           // PostgreSQL database driver
	"github.com/mattn/go-sqlite3" // SQLite database driver
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Storage represents an interface to almost any database or storage system
type Storage interface {
	Init() error
	Close() error
	ListOfOrgs() ([]types.OrgID, error)
	ListOfClustersForOrg(
		orgID types.OrgID, timeLimit time.Time) ([]types.ClusterName, error,
	)
	ListOfClustersForOrgSpecificRule(
		orgID types.OrgID, ruleID types.RuleSelector, activeClusters []string,
	) ([]ctypes.HittingClustersData, error)
	ReadReportForCluster(
		orgID types.OrgID, clusterName types.ClusterName) (
		[]types.RuleOnReport, types.Timestamp, types.Timestamp, types.Timestamp, error,
	)
	ReadReportInfoForCluster(
		types.OrgID, types.ClusterName) (
		types.Version, error,
	)
	ReadReportsForClusters(
		clusterNames []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error)
	ReadOrgIDsForClusters(
		clusterNames []types.ClusterName) ([]types.OrgID, error)
	ReadSingleRuleTemplateData(
		orgID types.OrgID, clusterName types.ClusterName, ruleID types.RuleID, errorKey types.ErrorKey,
	) (interface{}, error)
	ReadReportForClusterByClusterName(clusterName types.ClusterName) ([]types.RuleOnReport, types.Timestamp, error)
	GetLatestKafkaOffset() (types.KafkaOffset, error)
	WriteReportForCluster(
		orgID types.OrgID,
		clusterName types.ClusterName,
		report types.ClusterReport,
		rules []types.ReportItem,
		collectedAtTime time.Time,
		gatheredAtTime time.Time,
		storedAtTime time.Time,
		kafkaOffset types.KafkaOffset,
		requestID types.RequestID,
	) error
	WriteReportInfoForCluster(
		types.OrgID,
		types.ClusterName,
		[]types.InfoItem,
		time.Time,
	) error
	WriteRecommendationsForCluster(
		orgID types.OrgID,
		clusterName types.ClusterName,
		report types.ClusterReport,
		creationTime types.Timestamp,
	) error
	ReportsCount() (int, error)
	VoteOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		orgID types.OrgID,
		userID types.UserID,
		userVote types.UserVote,
		voteMessage string,
	) error
	AddOrUpdateFeedbackOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		orgID types.OrgID,
		userID types.UserID,
		message string,
	) error
	AddFeedbackOnRuleDisable(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		orgID types.OrgID,
		userID types.UserID,
		message string,
	) error
	GetUserFeedbackOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		userID types.UserID,
	) (*UserFeedbackOnRule, error)
	GetUserFeedbackOnRuleDisable(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		userID types.UserID,
	) (*UserFeedbackOnRule, error)
	DeleteReportsForOrg(orgID types.OrgID) error
	DeleteReportsForCluster(clusterName types.ClusterName) error
	ToggleRuleForCluster(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		orgID types.OrgID,
		ruleToggle RuleToggle,
	) error
	GetFromClusterRuleToggle(
		types.ClusterName,
		types.RuleID,
	) (*ClusterRuleToggle, error)
	GetTogglesForRules(
		clusterID types.ClusterName,
		rulesReport []types.RuleOnReport,
		orgID types.OrgID,
	) (map[types.RuleID]bool, error)
	DeleteFromRuleClusterToggle(
		clusterID types.ClusterName,
		ruleID types.RuleID,
	) error
	GetOrgIDByClusterID(cluster types.ClusterName) (types.OrgID, error)
	WriteConsumerError(msg *sarama.ConsumerMessage, consumerErr error) error
	GetUserFeedbackOnRules(
		clusterID types.ClusterName,
		rulesReport []types.RuleOnReport,
		userID types.UserID,
	) (map[types.RuleID]types.UserVote, error)
	GetUserDisableFeedbackOnRules(
		clusterID types.ClusterName,
		rulesReport []types.RuleOnReport,
		userID types.UserID,
	) (map[types.RuleID]UserFeedbackOnRule, error)
	DoesClusterExist(clusterID types.ClusterName) (bool, error)
	ListOfDisabledRules(orgID types.OrgID) ([]ctypes.DisabledRule, error)
	ListOfReasons(userID types.UserID) ([]DisabledRuleReason, error)
	ListOfDisabledRulesForClusters(
		clusterList []string,
		orgID types.OrgID,
	) ([]ctypes.DisabledRule, error)
	ListOfDisabledClusters(
		orgID types.OrgID,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
	) ([]ctypes.DisabledClusterInfo, error)
	RateOnRule(
		types.OrgID,
		types.RuleID,
		types.ErrorKey,
		types.UserVote,
	) error
	GetRuleRating(
		types.OrgID,
		types.RuleSelector,
	) (types.RuleRating, error)
	DisableRuleSystemWide(
		orgID types.OrgID, ruleID types.RuleID,
		errorKey types.ErrorKey, justification string,
	) error
	EnableRuleSystemWide(
		orgID types.OrgID,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
	) error
	UpdateDisabledRuleJustification(
		orgID types.OrgID,
		ruleID types.RuleID,
		errorKey types.ErrorKey,
		justification string,
	) error
	ReadDisabledRule(
		orgID types.OrgID, ruleID types.RuleID, errorKey types.ErrorKey,
	) (ctypes.SystemWideRuleDisable, bool, error)
	ListOfSystemWideDisabledRules(
		orgID types.OrgID,
	) ([]ctypes.SystemWideRuleDisable, error)
	ReadRecommendationsForClusters([]string, types.OrgID) (ctypes.RecommendationImpactedClusters, error)
	ReadClusterListRecommendations(clusterList []string, orgID types.OrgID) (
		ctypes.ClusterRecommendationMap, error,
	)
	MigrateToLatest() error
	GetConnection() *sql.DB
	PrintRuleDisableDebugInfo()
	GetDBDriverType() types.DBDriver
}

// ReportSuffix is used to strip away .report suffix from rule module names
const ReportSuffix = ".report"

// DBStorage is an implementation of Storage interface that use selected SQL like database
// like SQLite, PostgreSQL, MariaDB, RDS etc. That implementation is based on the standard
// sql package. It is possible to configure connection via Configuration structure.
// SQLQueriesLog is log for sql queries, default is nil which means nothing is logged
type DBStorage struct {
	connection   *sql.DB
	dbDriverType types.DBDriver
	// clusterLastCheckedDict is a dictionary of timestamps when the clusters were last checked.
	clustersLastChecked map[types.ClusterName]time.Time
}

// New function creates and initializes a new instance of Storage interface
func New(configuration Configuration) (Storage, error) {
	switch configuration.Type {
	case types.SQLStorage:
		return newSQLStorage(configuration)
	case types.RedisStorage:
		return newRedisStorage(configuration)
	case types.NoopStorage:
		return newNoopStorage(configuration)
	default:
		// error to be thrown
		err := fmt.Errorf("Unknown storage type '%s'", configuration.Type)
		log.Error().Err(err).Msg("Init failure")
		return nil, err
	}
}

// newNoopStorage function creates and initializes a new instance of Noop storage
func newNoopStorage(configuration Configuration) (Storage, error) {
	return &NoopStorage{}, nil
}

// newRedisStorage function creates and initializes a new instance of Redis storage
func newRedisStorage(configuration Configuration) (Storage, error) {
	redisCfg := configuration.RedisConfiguration
	log.Info().
		Str("Endpoint", redisCfg.RedisEndpoint).
		Int("Database index", redisCfg.RedisDatabase).
		Msg("Making connection to Redis storage")

	// pass for unit tests
	if redisCfg.RedisEndpoint == "" {
		return &RedisStorage{}, nil
	}

	client, err := redis.CreateRedisClient(
		redisCfg.RedisEndpoint,
		redisCfg.RedisDatabase,
		redisCfg.RedisPassword,
		redisCfg.RedisTimeoutSeconds,
	)
	// check for init error
	if err != nil {
		log.Error().Err(err).Msg("Error constructing Redis client")
		return nil, err
	}

	log.Info().Msg("Redis client has been initialized")

	redisStorage := &RedisStorage{
		Client: redis.Client{Connection: client},
	}

	err = redisStorage.Init()
	if err != nil {
		log.Error().Err(err).Msg("Error initializing Redis client")
		return nil, err
	}
	return redisStorage, nil
}

// newSQLStorage function creates and initializes a new instance of DB storage
func newSQLStorage(configuration Configuration) (Storage, error) {
	driverType, driverName, dataSource, err := initAndGetDriver(configuration)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf(
		"Making connection to data storage, driver=%s datasource=%s",
		driverName, dataSource,
	)

	connection, err := sql.Open(driverName, dataSource)
	if err != nil {
		log.Error().Err(err).Msg("Can not connect to data storage")
		return nil, err
	}

	return NewFromConnection(connection, driverType), nil
}

// NewFromConnection function creates and initializes a new instance of Storage interface from prepared connection
func NewFromConnection(connection *sql.DB, dbDriverType types.DBDriver) *DBStorage {
	return &DBStorage{
		connection:          connection,
		dbDriverType:        dbDriverType,
		clustersLastChecked: map[types.ClusterName]time.Time{},
	}
}

// initAndGetDriver initializes driver(with logs if logSQLQueries is true),
// checks if it's supported and returns driver type, driver name, dataSource and error
func initAndGetDriver(configuration Configuration) (driverType types.DBDriver, driverName, dataSource string, err error) {
	var driver sql_driver.Driver
	driverName = configuration.Driver

	switch driverName {
	case "sqlite3":
		driverType = types.DBDriverSQLite3
		driver = &sqlite3.SQLiteDriver{}
		dataSource = configuration.SQLiteDataSource
	case "postgres":
		driverType = types.DBDriverPostgres
		driver = &pq.Driver{}
		dataSource = fmt.Sprintf(
			"postgresql://%v:%v@%v:%v/%v?%v",
			configuration.PGUsername,
			configuration.PGPassword,
			configuration.PGHost,
			configuration.PGPort,
			configuration.PGDBName,
			configuration.PGParams,
		)
	default:
		err = fmt.Errorf("driver %v is not supported", driverName)
		return
	}

	if configuration.LogSQLQueries {
		driverName = InitSQLDriverWithLogs(driver, driverName)
	}

	return
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (storage DBStorage) MigrateToLatest() error {
	if err := migration.InitInfoTable(storage.connection); err != nil {
		return err
	}

	return migration.SetDBVersion(storage.connection, storage.dbDriverType, migration.GetMaxVersion())
}

// Init performs all database initialization
// tasks necessary for further service operation.
func (storage DBStorage) Init() error {
	// Read clusterName:LastChecked dictionary from DB.
	rows, err := storage.connection.Query("SELECT cluster, last_checked_at FROM report;")
	if err != nil {
		return err
	}

	for rows.Next() {
		var (
			clusterName types.ClusterName
			lastChecked time.Time
		)

		if err := rows.Scan(&clusterName, &lastChecked); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("Unable to close the DB rows handle")
			}
			return err
		}

		storage.clustersLastChecked[clusterName] = lastChecked
	}

	// Not using defer to close the rows here to:
	// - make errcheck happy (it doesn't like ignoring returned errors),
	// - return a possible error returned by the Close method.
	return rows.Close()
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage DBStorage) Close() error {
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

// Report represents one (latest) cluster report.
//
//	Org: organization ID
//	Name: cluster GUID in the following format:
//	    c8590f31-e97e-4b85-b506-c45ce1911a12
type Report struct {
	Org        types.OrgID         `json:"org"`
	Name       types.ClusterName   `json:"cluster"`
	Report     types.ClusterReport `json:"report"`
	ReportedAt types.Timestamp     `json:"reported_at"`
}

func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

// ListOfOrgs reads list of all organizations that have at least one cluster report
func (storage DBStorage) ListOfOrgs() ([]types.OrgID, error) {
	orgs := make([]types.OrgID, 0)

	rows, err := storage.connection.Query("SELECT DISTINCT org_id FROM report ORDER BY org_id;")
	err = types.ConvertDBError(err, nil)
	if err != nil {
		return orgs, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var orgID types.OrgID

		err = rows.Scan(&orgID)
		if err == nil {
			orgs = append(orgs, orgID)
		} else {
			log.Error().Err(err).Msg("ListOfOrgID")
		}
	}
	return orgs, nil
}

// ListOfClustersForOrg reads list of all clusters fro given organization
func (storage DBStorage) ListOfClustersForOrg(orgID types.OrgID, timeLimit time.Time) ([]types.ClusterName, error) {
	clusters := make([]types.ClusterName, 0)

	q := `
		  SELECT cluster
		    FROM report
		   WHERE org_id = $1
		     AND reported_at >= $2
		ORDER BY cluster;
	`

	rows, err := storage.connection.Query(q, orgID, timeLimit)

	err = types.ConvertDBError(err, orgID)
	if err != nil {
		return clusters, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var clusterName string

		err = rows.Scan(&clusterName)
		if err == nil {
			clusters = append(clusters, types.ClusterName(clusterName))
		} else {
			log.Error().Err(err).Msg("ListOfClustersForOrg")
		}
	}
	return clusters, nil
}

// ListOfClustersForOrgSpecificRule returns list of all clusters for given organization that are affect by given rule
func (storage DBStorage) ListOfClustersForOrgSpecificRule(
	orgID types.OrgID,
	ruleID types.RuleSelector,
	activeClusters []string) (
	[]ctypes.HittingClustersData, error) {
	results := make([]ctypes.HittingClustersData, 0)

	var whereClause string
	if len(activeClusters) > 0 {
		// #nosec G201
		whereClause = fmt.Sprintf(`WHERE org_id = $1 AND rule_id = $2 AND cluster_id IN (%v)`,
			inClauseFromSlice(activeClusters))
	} else {
		whereClause = `WHERE org_id = $1 AND rule_id = $2`
	}
	// #nosec G202
	query := `SELECT cluster_id, created_at, impacted_since FROM recommendation ` + whereClause + ` ORDER BY cluster_id;`

	rows, err := storage.connection.Query(query, orgID, ruleID)

	err = types.ConvertDBError(err, orgID)
	if err != nil {
		return results, err
	}

	defer closeRows(rows)

	var (
		clusterName   types.ClusterName
		lastSeen      string
		impactedSince string
	)
	for rows.Next() {
		err = rows.Scan(&clusterName, &lastSeen, &impactedSince)
		if err != nil {
			log.Error().Err(err).Msg("ListOfClustersForOrgSpecificRule")
		}
		results = append(results, ctypes.HittingClustersData{
			Cluster:       clusterName,
			LastSeen:      lastSeen,
			ImpactedSince: impactedSince,
		})
	}

	// This is to ensure 404 when no recommendation is found for the given orgId + selector.
	// We can, alternatively, return something like this with a 204 (no content):
	// {"data":[],"meta":{"count":0,"component":"test.rule","error_key":"ek"},"status":"not_found"}
	if len(results) == 0 {
		return results, &types.ItemNotFoundError{ItemID: ruleID}
	}
	return results, nil
}

// GetOrgIDByClusterID reads OrgID for specified cluster
func (storage DBStorage) GetOrgIDByClusterID(cluster types.ClusterName) (types.OrgID, error) {
	row := storage.connection.QueryRow("SELECT org_id FROM report WHERE cluster = $1 ORDER BY org_id;", cluster)

	var orgID uint64
	err := row.Scan(&orgID)
	if err != nil {
		log.Error().Err(err).Msg("GetOrgIDByClusterID")
		return 0, err
	}
	return types.OrgID(orgID), nil
}

// parseTemplateData parses template data and returns a json raw message if it's a json or a string otherwise
func parseTemplateData(templateData []byte) interface{} {
	var templateDataJSON json.RawMessage

	err := json.Unmarshal(templateData, &templateDataJSON)
	if err != nil {
		log.Warn().Err(err).Msgf("unable to parse template data as json")
		return templateData
	}

	return templateDataJSON
}

func parseRuleRows(rows *sql.Rows) ([]types.RuleOnReport, error) {
	report := make([]types.RuleOnReport, 0)

	for rows.Next() {
		var (
			templateDataBytes []byte
			ruleFQDN          types.RuleID
			errorKey          types.ErrorKey
			createdAt         sql.NullTime
		)

		err := rows.Scan(&templateDataBytes, &ruleFQDN, &errorKey, &createdAt)
		if err != nil {
			log.Error().Err(err).Msg("ReportListForCluster")
			return report, err
		}

		templateData := parseTemplateData(templateDataBytes)
		var createdAtConverted time.Time
		if createdAt.Valid {
			createdAtConverted = createdAt.Time
		}
		rule := types.RuleOnReport{
			Module:       ruleFQDN,
			ErrorKey:     errorKey,
			TemplateData: templateData,
			CreatedAt:    types.Timestamp(createdAtConverted.UTC().Format(time.RFC3339)),
		}

		report = append(report, rule)
	}

	log.Info().Msgf("parseRuleRows final report %v", report)

	return report, nil
}

// constructInClausule is a helper function to construct `in` clause for SQL
// statement.
func constructInClausule(howMany int) (string, error) {
	// construct the `in` clause in SQL query statement
	if howMany < 1 {
		return "", fmt.Errorf("at least one value needed")
	}
	inClausule := "$1"
	for i := 2; i <= howMany; i++ {
		inClausule += fmt.Sprintf(",$%d", i)
	}
	return inClausule, nil
}

// argsWithClusterNames is a helper function to construct arguments for SQL
// statement.
func argsWithClusterNames(clusterNames []types.ClusterName) []interface{} {
	// prepare arguments
	args := make([]interface{}, len(clusterNames))

	for i, clusterName := range clusterNames {
		args[i] = clusterName
	}
	return args
}

// inClauseFromSlice is a helper function to construct `in` clause for SQL
// statement from a given slice of items. The received slice must be []string
// or any other type that can be asserted to []string, or else '1=1' will be
// returned, making the IN clause act like a wildcard.
func inClauseFromSlice(slice interface{}) string {
	if slice, ok := slice.([]string); ok {
		return "'" + strings.Join(slice, `','`) + `'`
	}
	return "1=1"
}

/*
func updateRecommendationsMetrics(cluster string, deleted float64, inserted float64) {
	metrics.SQLRecommendationsDeletes.WithLabelValues(cluster).Observe(deleted)
	metrics.SQLRecommendationsInserts.WithLabelValues(cluster).Observe(inserted)
}
*/

// ReadOrgIDsForClusters read organization IDs for given list of cluster names.
func (storage DBStorage) ReadOrgIDsForClusters(clusterNames []types.ClusterName) ([]types.OrgID, error) {
	// stub for return value
	ids := make([]types.OrgID, 0)

	if len(clusterNames) < 1 {
		return ids, nil
	}

	// prepare arguments
	args := argsWithClusterNames(clusterNames)

	// construct the `in` clause in SQL query statement
	inClausule, err := constructInClausule(len(clusterNames))
	if err != nil {
		log.Error().Err(err).Msg(inClauseError)
		return ids, err
	}

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	query := "SELECT DISTINCT org_id FROM report WHERE cluster in (" + inClausule + ");"

	// select results from the database
	rows, err := storage.connection.Query(query, args...)
	if err != nil {
		log.Error().Err(err).Msg("query to get org ids")
		return ids, err
	}

	// process results returned from database
	for rows.Next() {
		var orgID types.OrgID

		err := rows.Scan(&orgID)
		if err != nil {
			log.Error().Err(err).Msg("read one org id")
			return ids, err
		}

		ids = append(ids, orgID)
	}

	// everything seems ok -> return ids
	return ids, nil
}

// ReadReportsForClusters function reads reports for given list of cluster
// names.
func (storage DBStorage) ReadReportsForClusters(clusterNames []types.ClusterName) (map[types.ClusterName]types.ClusterReport, error) {
	// stub for return value
	reports := make(map[types.ClusterName]types.ClusterReport)

	if len(clusterNames) < 1 {
		return reports, nil
	}

	// prepare arguments
	args := argsWithClusterNames(clusterNames)

	// construct the `in` clause in SQL query statement
	inClausule, err := constructInClausule(len(clusterNames))
	if err != nil {
		log.Error().Err(err).Msg(inClauseError)
		return reports, err
	}

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	query := "SELECT cluster, report FROM report WHERE cluster in (" + inClausule + ");"

	// select results from the database
	rows, err := storage.connection.Query(query, args...)
	if err != nil {
		return reports, err
	}

	// process results returned from database
	for rows.Next() {
		// convert into requested type
		var (
			clusterName   types.ClusterName
			clusterReport types.ClusterReport
		)

		err := rows.Scan(&clusterName, &clusterReport)
		if err != nil {
			log.Error().Err(err).Msg("ReadReportsForClusters")
			return reports, err
		}

		reports[clusterName] = clusterReport
	}

	// everything seems ok -> return reports
	return reports, nil
}

// ReadReportForCluster reads result (health status) for selected cluster
func (storage DBStorage) ReadReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName,
) ([]types.RuleOnReport, types.Timestamp, types.Timestamp, types.Timestamp, error) {
	var lastChecked time.Time
	var reportedAt time.Time
	var gatheredAtInDB sql.NullTime // to avoid problems

	report := make([]types.RuleOnReport, 0)

	err := storage.connection.QueryRow(
		"SELECT last_checked_at, reported_at, gathered_at FROM report WHERE org_id = $1 AND cluster = $2;",
		orgID, clusterName,
	).Scan(&lastChecked, &reportedAt, &gatheredAtInDB)

	// convert timestamps to string
	var lastCheckedStr = types.Timestamp(lastChecked.UTC().Format(time.RFC3339))
	var reportedAtStr = types.Timestamp(reportedAt.UTC().Format(time.RFC3339))
	var gatheredAtStr types.Timestamp

	if gatheredAtInDB.Valid {
		gatheredAtStr = types.Timestamp(gatheredAtInDB.Time.UTC().Format(time.RFC3339))
	} else {
		gatheredAtStr = ""
	}

	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
	if err != nil {
		log.Error().Err(err).Str(clusterKey, string(clusterName)).Msg(
			"ReadReportForCluster query from report table error",
		)
		return report, lastCheckedStr, reportedAtStr, gatheredAtStr, err
	}

	rows, err := storage.connection.Query(
		"SELECT template_data, rule_fqdn, error_key, created_at FROM rule_hit WHERE org_id = $1 AND cluster_id = $2;", orgID, clusterName,
	)

	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
	if err != nil {
		log.Error().Err(err).Str(clusterKey, string(clusterName)).Msg(
			"ReadReportForCluster query from rule_hit table error",
		)
		return report, lastCheckedStr, reportedAtStr, gatheredAtStr, err
	}

	report, err = parseRuleRows(rows)

	return report, lastCheckedStr, reportedAtStr, gatheredAtStr, err
}

// ReadSingleRuleTemplateData reads template data for a single rule
func (storage DBStorage) ReadSingleRuleTemplateData(
	orgID types.OrgID, clusterName types.ClusterName, ruleID types.RuleID, errorKey types.ErrorKey,
) (interface{}, error) {
	var templateDataBytes []byte

	err := storage.connection.QueryRow(`
		SELECT template_data FROM rule_hit
		WHERE org_id = $1 AND cluster_id = $2 AND rule_fqdn = $3 AND error_key = $4;
	`,
		orgID,
		clusterName,
		ruleID,
		errorKey,
	).Scan(&templateDataBytes)
	err = types.ConvertDBError(err, []interface{}{orgID, clusterName, ruleID, errorKey})

	return parseTemplateData(templateDataBytes), err
}

// ReadReportForClusterByClusterName reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForClusterByClusterName(
	clusterName types.ClusterName,
) ([]types.RuleOnReport, types.Timestamp, error) {
	report := make([]types.RuleOnReport, 0)
	var lastChecked time.Time

	err := storage.connection.QueryRow(
		"SELECT last_checked_at FROM report WHERE cluster = $1;", clusterName,
	).Scan(&lastChecked)

	switch {
	case err == sql.ErrNoRows:
		return report, "", &types.ItemNotFoundError{
			ItemID: fmt.Sprintf("%v", clusterName),
		}
	case err != nil:
		return report, "", err
	}

	rows, err := storage.connection.Query(
		"SELECT template_data, rule_fqdn, error_key, created_at FROM rule_hit WHERE cluster_id = $1;", clusterName,
	)

	if err != nil {
		return report, types.Timestamp(lastChecked.UTC().Format(time.RFC3339)), err
	}

	report, err = parseRuleRows(rows)

	return report, types.Timestamp(lastChecked.UTC().Format(time.RFC3339)), err
}

// GetLatestKafkaOffset returns latest kafka offset from report table
func (storage DBStorage) GetLatestKafkaOffset() (types.KafkaOffset, error) {
	var offset types.KafkaOffset
	err := storage.connection.QueryRow("SELECT COALESCE(MAX(kafka_offset), 0) FROM report;").Scan(&offset)
	return offset, err
}

// GetRuleHitInsertStatement method prepares DB statement to be used to write
// rule FQDN + rule error key into rule_hit table for given cluster_id
func (storage DBStorage) GetRuleHitInsertStatement(rules []types.ReportItem) string {
	const ruleInsertStatement = "INSERT INTO rule_hit(org_id, cluster_id, rule_fqdn, error_key, template_data, created_at) VALUES %s"

	var placeholders []string

	// fill-in placeholders for INSERT statement
	for index := range rules {
		placeholders = append(
			placeholders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d)",
				index*6+1,
				index*6+2,
				index*6+3,
				index*6+4,
				index*6+5,
				index*6+6,
			))
	}

	// construct INSERT statement for multiple values
	return fmt.Sprintf(ruleInsertStatement, strings.Join(placeholders, ","))
}

// valuesForRuleHitsInsert function prepares values to insert rules into
// rule_hit table.
func valuesForRuleHitsInsert(
	orgID types.OrgID,
	clusterName types.ClusterName,
	rules []types.ReportItem,
	ruleKeyCreatedAt map[string]types.Timestamp,
) []interface{} {
	// fill-in values for INSERT statement
	var values []interface{}

	for _, rule := range rules {
		ruleKey := string(rule.Module) + string(rule.ErrorKey)
		var impactedSince types.Timestamp
		if val, ok := ruleKeyCreatedAt[ruleKey]; ok {
			impactedSince = val
		} else {
			impactedSince = types.Timestamp(time.Now().UTC().Format(time.RFC3339))
		}
		values = append(values,
			orgID,
			clusterName,
			rule.Module,
			rule.ErrorKey,
			string(rule.TemplateData),
			impactedSince,
		)
	}
	return values
}

func (storage DBStorage) updateReport(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	rules []types.ReportItem,
	lastCheckedTime time.Time,
	gatheredAt time.Time,
	reportedAtTime time.Time,
	kafkaOffset types.KafkaOffset,
) error {
	// Get the UPSERT query for writing a report into the database.
	reportUpsertQuery := storage.getReportUpsertQuery()

	// Get created_at if present before deletion
	query := "SELECT rule_fqdn, error_key, created_at FROM rule_hit WHERE org_id = $1 AND cluster_id = $2;"
	RuleKeyCreatedAt, err := storage.getRuleKeyCreatedAtMap(
		query, orgID, clusterName,
	)
	if err != nil {
		log.Error().Err(err).Msgf("Unable to get recommendation impacted_since")
		RuleKeyCreatedAt = make(map[string]types.Timestamp) // create empty map
	}

	deleteQuery := "DELETE FROM rule_hit WHERE org_id = $1 AND cluster_id = $2;"
	_, err = tx.Exec(deleteQuery, orgID, clusterName)
	if err != nil {
		log.Err(err).Msgf("Unable to remove previous cluster reports (org: %v, cluster: %v)", orgID, clusterName)
		return err
	}

	// Perform the report insert.
	// All older rule hits has been deleted for given cluster so it is
	// possible to just insert new hits w/o the need to update on conflict
	if len(rules) > 0 {
		// Get the INSERT statement for writing a rule into the database.
		ruleInsertStatement := storage.GetRuleHitInsertStatement(rules)

		// Get values to be stored in rule_hits table
		values := valuesForRuleHitsInsert(orgID, clusterName, rules, RuleKeyCreatedAt)

		_, err = tx.Exec(ruleInsertStatement, values...)
		if err != nil {
			log.Err(err).Msgf("Unable to insert the cluster report rules (org: %v, cluster: %v)",
				orgID, clusterName,
			)
			return err
		}
	}

	if gatheredAt.IsZero() {
		_, err = tx.Exec(reportUpsertQuery, orgID, clusterName, report, reportedAtTime, lastCheckedTime, kafkaOffset, sql.NullTime{Valid: false})
	} else {
		_, err = tx.Exec(reportUpsertQuery, orgID, clusterName, report, reportedAtTime, lastCheckedTime, kafkaOffset, gatheredAt)
	}

	if err != nil {
		log.Err(err).Msgf("Unable to upsert the cluster report (org: %v, cluster: %v)", orgID, clusterName)
		return err
	}

	return nil
}

func prepareInsertRecommendationsStatement(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ReportRules,
	createdAt types.Timestamp,
	impactedSinceMap map[string]types.Timestamp,
) (selectors []string, statement string, statementArgs []interface{}) {
	statement = `INSERT INTO recommendation (org_id, cluster_id, rule_fqdn, error_key, rule_id, created_at, impacted_since) VALUES %s`

	var valuesIdx []string
	statementIdx := 0
	selectors = make([]string, len(report.HitRules))

	for idx, rule := range report.HitRules {
		ruleFqdn := strings.TrimSuffix(string(rule.Module), ReportSuffix)
		ruleID := ruleFqdn + "|" + string(rule.ErrorKey)
		impactedSince, ok := impactedSinceMap[ruleFqdn+string(rule.ErrorKey)]
		if !ok {
			impactedSince = createdAt
		}
		selectors[idx] = ruleID
		statementArgs = append(statementArgs, orgID, clusterName, ruleFqdn, rule.ErrorKey, ruleID, createdAt, impactedSince)
		statementIdx = len(statementArgs)
		valuesIdx = append(valuesIdx, "($"+fmt.Sprint(statementIdx-6)+
			", $"+fmt.Sprint(statementIdx-5)+
			", $"+fmt.Sprint(statementIdx-4)+
			", $"+fmt.Sprint(statementIdx-3)+
			", $"+fmt.Sprint(statementIdx-2)+
			", $"+fmt.Sprint(statementIdx-1)+
			", $"+fmt.Sprint(statementIdx)+")")
	}

	statement = fmt.Sprintf(statement, strings.Join(valuesIdx, ","))
	return
}

func (storage DBStorage) insertRecommendations(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ReportRules,
	createdAt types.Timestamp,
	impactedSince map[string]types.Timestamp,
) (inserted int, err error) {
	if len(report.HitRules) == 0 {
		log.Info().
			Int(organizationKey, int(orgID)).
			Str(clusterKey, string(clusterName)).
			Int(issuesCountKey, 0).
			Msg("No new recommendation to insert")
		return 0, nil
	}

	selectors, statement, args := prepareInsertRecommendationsStatement(orgID, clusterName, report, createdAt, impactedSince)

	if _, err = tx.Exec(statement, args...); err != nil {
		log.Error().
			Int(organizationKey, int(orgID)).
			Str(clusterKey, string(clusterName)).
			Int(issuesCountKey, inserted).
			Interface(createdAtKey, createdAt).
			Strs(selectorsKey, selectors).
			Err(err).
			Msg("Unable to insert the recommendations")
		return 0, err
	}
	log.Info().
		Int(organizationKey, int(orgID)).
		Str(clusterKey, string(clusterName)).
		Int(issuesCountKey, inserted).
		Interface(createdAtKey, createdAt).
		Strs(selectorsKey, selectors).
		Msg("Recommendations inserted successfully")

	inserted = len(selectors)
	return

}

// getRuleKeyCreatedAtMap returns a map between
// (rule_fqdn, error_key) -> created_at
// for each rule_hit rows matching given
// orgId and clusterName
func (storage DBStorage) getRuleKeyCreatedAtMap(
	query string,
	orgID types.OrgID,
	clusterName types.ClusterName,
) (
	map[string]types.Timestamp,
	error) {

	impactedSinceRows, err := storage.connection.Query(
		query, orgID, clusterName)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving recommendation timestamp")
		return nil, err
	}
	defer closeRows(impactedSinceRows)

	RuleKeyCreatedAt := make(map[string]types.Timestamp)
	for impactedSinceRows.Next() {
		var ruleFqdn string
		var errorKey string
		var oldTime time.Time
		err := impactedSinceRows.Scan(
			&ruleFqdn,
			&errorKey,
			&oldTime,
		)
		if err != nil {
			log.Error().Err(err).Msg("error scanning for rule id -> created_at map")
			continue
		}
		newTime := types.Timestamp(oldTime.UTC().Format(time.RFC3339))
		RuleKeyCreatedAt[ruleFqdn+errorKey] = newTime
	}
	return RuleKeyCreatedAt, err
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	rules []types.ReportItem,
	lastCheckedTime time.Time,
	gatheredAt time.Time,
	storedAtTime time.Time,
	kafkaOffset types.KafkaOffset,
	requestID types.RequestID,
) error {
	// Skip writing the report if it isn't newer than a report
	// that is already in the database for the same cluster.
	if oldLastChecked, exists := storage.clustersLastChecked[clusterName]; exists && !lastCheckedTime.After(oldLastChecked) {
		return types.ErrOldReport
	}

	if storage.dbDriverType != types.DBDriverSQLite3 && storage.dbDriverType != types.DBDriverPostgres {
		return fmt.Errorf("writing report with DB %v is not supported", storage.dbDriverType)
	}

	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	err = func(tx *sql.Tx) error {

		// Check if there is a more recent report for the cluster already in the database.
		rows, err := tx.Query(
			"SELECT last_checked_at FROM report WHERE org_id = $1 AND cluster = $2 AND last_checked_at > $3;",
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

		err = storage.updateReport(tx, orgID, clusterName, report, rules, lastCheckedTime, gatheredAt, storedAtTime, kafkaOffset)
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

// WriteRecommendationsForCluster writes hitting rules in received report for selected cluster
func (storage DBStorage) WriteRecommendationsForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	stringReport types.ClusterReport,
	creationTime types.Timestamp,
) (err error) {
	var report types.ReportRules
	err = json.Unmarshal([]byte(stringReport), &report)
	if err != nil {
		return err
	}
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	impactedSinceMap := make(map[string]ctypes.Timestamp)
	err = func(tx *sql.Tx) error {
		var deleted int64 = 0
		// Delete current recommendations for the cluster if some report has been previously stored for this cluster
		if _, ok := storage.clustersLastChecked[clusterName]; ok {

			// Get impacted_since if present
			query := "SELECT rule_fqdn, error_key, impacted_since FROM recommendation WHERE org_id = $1 AND cluster_id = $2 LIMIT 1;"
			impactedSinceMap, err = storage.getRuleKeyCreatedAtMap(
				query, orgID, clusterName)
			if err != nil {
				log.Error().Err(err).Msgf("Unable to get recommendation impacted_since")
			}

			// it is needed to use `org_id = $1` condition there
			// because it allows DB to use proper btree indexing
			// and not slow sequential scan
			result, err := tx.Exec(
				"DELETE FROM recommendation WHERE org_id = $1 AND cluster_id = $2;", orgID, clusterName)
			err = types.ConvertDBError(err, []interface{}{clusterName})
			if err != nil {
				log.Error().Err(err).Msgf("Unable to delete the existing recommendations for %s", clusterName)
				return err
			}

			// As the documentation says:
			// RowsAffected returns the number of rows affected by an
			// update, insert, or delete. Not every database or database
			// driver may support this.
			// So we might run in a scenario where we don't have metrics
			// if the driver doesn't help.
			deleted, err = result.RowsAffected()
			if err != nil {
				log.Error().Err(err).Msg("Unable to retrieve number of deleted rows with current driver")
				return err
			}
		}

		inserted, err := storage.insertRecommendations(tx, orgID, clusterName, report, creationTime, impactedSinceMap)
		if err != nil {
			return err
		}

		log.Info().
			Int64("Deleted", deleted).
			Int("Inserted", inserted).
			Int(organizationKey, int(orgID)).
			Str(clusterKey, string(clusterName)).
			Msg("Updated recommendation table")
		// updateRecommendationsMetrics(string(clusterName), float64(deleted), float64(inserted))

		return nil
	}(tx)

	finishTransaction(tx, err)

	return err
}

// finishTransaction finishes the transaction depending on err. err == nil -> commit, err != nil -> rollback
func finishTransaction(tx *sql.Tx, err error) {
	if err != nil {
		rollbackError := tx.Rollback()
		if rollbackError != nil {
			log.Err(rollbackError).Msgf("error when trying to rollback a transaction")
		}
	} else {
		commitError := tx.Commit()
		if commitError != nil {
			log.Err(commitError).Msgf("error when trying to commit a transaction")
		}
	}
}

// ReadRecommendationsForClusters reads all recommendations from recommendation table for given organization
func (storage DBStorage) ReadRecommendationsForClusters(
	clusterList []string,
	orgID types.OrgID,
) (ctypes.RecommendationImpactedClusters, error) {

	impactedClusters := make(ctypes.RecommendationImpactedClusters, 0)

	if len(clusterList) < 1 {
		return impactedClusters, nil
	}

	// #nosec G201
	whereClause := fmt.Sprintf(`WHERE org_id = $1 AND cluster_id IN (%v)`, inClauseFromSlice(clusterList))

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	query := `
	SELECT
		rule_id, cluster_id
	FROM
		recommendation
	` + whereClause

	rows, err := storage.connection.Query(query, orgID)
	if err != nil {
		log.Error().Err(err).Msg("query to get recommendations")
		return impactedClusters, err
	}

	for rows.Next() {
		var (
			ruleID    types.RuleID
			clusterID types.ClusterName
		)

		err := rows.Scan(
			&ruleID,
			&clusterID,
		)
		if err != nil {
			log.Error().Err(err).Msg("read one recommendation")
			return impactedClusters, err
		}

		impactedClusters[ruleID] = append(impactedClusters[ruleID], clusterID)
	}

	return impactedClusters, nil
}

// ReadClusterListRecommendations retrieves cluster IDs and a list of hitting rules for each one
func (storage DBStorage) ReadClusterListRecommendations(
	clusterList []string,
	orgID types.OrgID,
) (ctypes.ClusterRecommendationMap, error) {

	clusterMap := make(ctypes.ClusterRecommendationMap, 0)

	if len(clusterList) < 1 {
		return clusterMap, nil
	}

	// we have to select from report table primarily because we need to show last_checked_at even if there
	// are no rule hits (which means there are no rows in recommendation table for that cluster)

	// disable "G202 (CWE-89): SQL string concatenation"
	// #nosec G202
	query := `
	SELECT
		rep.cluster, rep.last_checked_at, COALESCE(rec.rule_id, '')
	FROM
		report rep
	LEFT JOIN
		recommendation rec
	ON
		rep.org_id = rec.org_id AND
		rep.cluster = rec.cluster_id
	WHERE
		rep.org_id = $1 AND rep.cluster IN (%v)
	`
	// #nosec G201
	query = fmt.Sprintf(query, inClauseFromSlice(clusterList))

	rows, err := storage.connection.Query(query, orgID)
	if err != nil {
		log.Error().Err(err).Msg("query to get recommendations")
		return clusterMap, err
	}

	for rows.Next() {
		var (
			clusterID    ctypes.ClusterName
			ruleID       ctypes.RuleID
			timestampStr string
			timestamp    time.Time
		)

		if storage.dbDriverType != types.DBDriverSQLite3 {
			// postgres is able to auto scan created_at into time.Time
			err := rows.Scan(
				&clusterID,
				&timestamp,
				&ruleID,
			)
			if err != nil {
				log.Error().Err(err).Msg("problem reading one recommendation")
				return clusterMap, err
			}
		} else {
			// sqlite cannot auto scan into time.Time, needs manual parse
			err := rows.Scan(
				&clusterID,
				&timestampStr,
				&ruleID,
			)
			if err != nil {
				log.Error().Err(err).Msg("problem reading one recommendation")
				return clusterMap, err
			}

			timestamp, err = time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				log.Error().Err(err).Msgf("unparsable timestamp %v", timestamp)
				return clusterMap, err
			}
		}

		if cluster, exists := clusterMap[clusterID]; exists {
			cluster.Recommendations = append(cluster.Recommendations, ruleID)
			clusterMap[clusterID] = cluster
		} else {
			// create entry in map for new cluster ID
			clusterMap[clusterID] = ctypes.ClusterRecommendationList{
				// created at is the same for all rows for each cluster
				CreatedAt:       timestamp,
				Recommendations: []ctypes.RuleID{ruleID},
			}
		}
	}

	log.Info().Msgf("Filling metadata for clustermap %v", clusterMap)
	storage.fillInMetadata(orgID, clusterMap)
	return clusterMap, nil
}

// ReportsCount reads number of all records stored in database
func (storage DBStorage) ReportsCount() (int, error) {
	count := -1
	err := storage.connection.QueryRow("SELECT count(*) FROM report;").Scan(&count)
	err = types.ConvertDBError(err, nil)

	return count, err
}

// DeleteReportsForOrg deletes all reports related to the specified organization from the storage.
func (storage DBStorage) DeleteReportsForOrg(orgID types.OrgID) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE org_id = $1;", orgID)
	return err
}

// DeleteReportsForCluster deletes all reports related to the specified cluster from the storage.
func (storage DBStorage) DeleteReportsForCluster(clusterName types.ClusterName) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE cluster = $1;", clusterName)
	return err
}

// GetConnection returns db connection(useful for testing)
func (storage DBStorage) GetConnection() *sql.DB {
	return storage.connection
}

// WriteConsumerError writes a report about a consumer error into the storage.
func (storage DBStorage) WriteConsumerError(msg *sarama.ConsumerMessage, consumerErr error) error {
	_, err := storage.connection.Exec(`
		INSERT INTO consumer_error (topic, partition, topic_offset, key, produced_at, consumed_at, message, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Timestamp, time.Now().UTC(), msg.Value, consumerErr.Error())

	return err
}

// GetDBDriverType returns db driver type
func (storage DBStorage) GetDBDriverType() types.DBDriver {
	return storage.dbDriverType
}

// DoesClusterExist checks if cluster with this id exists
func (storage DBStorage) DoesClusterExist(clusterID types.ClusterName) (bool, error) {
	err := storage.connection.QueryRow(
		"SELECT cluster FROM report WHERE cluster = $1", clusterID,
	).Scan(&clusterID)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// ListOfDisabledClusters function returns list of all clusters disabled for a rule from a
// specified account.
func (storage DBStorage) ListOfDisabledClusters(
	orgID types.OrgID,
	ruleID types.RuleID,
	errorKey types.ErrorKey,
) (
	disabledClusters []ctypes.DisabledClusterInfo,
	err error,
) {
	// select disabled rules from toggle table and the latest feedback from disable_feedback table
	// LEFT join and COALESCE are used for the feedback, because feedback is filled by different
	// request than toggle, so it might be empty/null
	query := `
	SELECT
        toggle.cluster_id,
		toggle.disabled_at,
		COALESCE(feedback.message, '')
	FROM
		cluster_rule_toggle toggle
	LEFT JOIN
		cluster_user_rule_disable_feedback feedback
	ON feedback.updated_at = (
		SELECT updated_at
		FROM cluster_user_rule_disable_feedback
		WHERE cluster_id = toggle.cluster_id
		AND org_id = $1
		AND rule_id = $2
		AND error_key = $3
		ORDER BY updated_at DESC
		LIMIT 1
	)
	WHERE
		toggle.org_id = $1
		AND toggle.rule_id = $2
		AND toggle.error_key = $3
		AND toggle.disabled = $4
	ORDER BY
		toggle.disabled_at DESC
	`

	// run the query against database
	rows, err := storage.connection.Query(query, orgID, ruleID, errorKey, RuleToggleDisable)

	// return empty list in case of any error
	if err != nil {
		return disabledClusters, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var disabledCluster ctypes.DisabledClusterInfo

		err = rows.Scan(
			&disabledCluster.ClusterID,
			&disabledCluster.DisabledAt,
			&disabledCluster.Justification,
		)

		if err != nil {
			log.Error().Err(err).Msg("ReadListOfDisabledRules")
			// return partially filled slice + error
			return disabledClusters, err
		}

		// append disabled cluster read from database to a slice
		disabledClusters = append(disabledClusters, disabledCluster)
	}

	return disabledClusters, nil
}
