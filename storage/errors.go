package storage

import (
	"fmt"
)

// ItemNotFoundError shows that item with id ItemID wasn't found in the storage
type ItemNotFoundError struct {
	ItemID interface{}
}

func (e *ItemNotFoundError) Error() string {
	return fmt.Sprintf("Item with ID %+v was not found in the storage", e.ItemID)
}
