package helpers

import (
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// GetMockStorage creates mocked storage based on in-memory Sqlite instance
func GetMockStorage(init bool) (storage.Storage, error) {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	})
	if err != nil {
		return nil, err
	}

	// initialize the database by all required tables
	if init {
		err = mockStorage.Init()
		if err != nil {
			return nil, err
		}
	}

	return mockStorage, nil
}
