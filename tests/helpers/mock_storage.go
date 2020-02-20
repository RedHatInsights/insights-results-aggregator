package helpers

import (
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// GetMockStorage creates mocked storage based on in-memory Sqlite instance
func GetMockStorage(init bool) (storage.Storage, error) {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
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

// MustGetMockStorage creates mocked storage based on in-memory Sqlite instance
// produces t.Fatal(err) on error
func MustGetMockStorage(t *testing.T, init bool) storage.Storage {
	mockStorage, err := GetMockStorage(init)
	if err != nil {
		t.Fatal(err)
	}

	return mockStorage
}
