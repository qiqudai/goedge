package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock DB setup needed for real test, this is a template
func TestCreateDNSTask(t *testing.T) {
	// Setup checks
	if db.DB == nil {
		t.Skip("DB not initialized")
	}

	t.Run("Create New Task", func(t *testing.T) {
		key := "test_key_1"
		id, err := CreateDNSTask("TEST_TYPE", "{}", key)
		assert.NoError(t, err)
		assert.Greater(t, id, int64(0))

		// Clean up
		db.DB.Delete(&models.Task{}, id)
	})

	t.Run("Idempotency Check", func(t *testing.T) {
		key := "test_key_2"
		id1, err := CreateDNSTask("TEST_TYPE", "{}", key)
		assert.NoError(t, err)

		id2, err := CreateDNSTask("TEST_TYPE", "{}", key)
		assert.NoError(t, err)
		assert.Equal(t, id1, id2, "Should return same ID for existing active task")

		// Clean up
		db.DB.Delete(&models.Task{}, id1)
	})
}
