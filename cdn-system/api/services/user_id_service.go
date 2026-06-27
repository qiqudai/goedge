package services

import (
	"cdn-api/models"
	"crypto/rand"
	"fmt"
	"math/big"

	"gorm.io/gorm"
)

const (
	ordinaryUserIDMin      int64 = 100000
	ordinaryUserIDRange    int64 = 100000
	ordinaryUserIDAttempts       = 64
)

// GenerateOrdinaryUserID returns a six-digit ordinary user ID starting with 1.
func GenerateOrdinaryUserID(tx *gorm.DB) (int64, error) {
	if tx == nil {
		return 0, fmt.Errorf("database handle is nil")
	}

	for i := 0; i < ordinaryUserIDAttempts; i++ {
		offset, err := rand.Int(rand.Reader, big.NewInt(ordinaryUserIDRange))
		if err != nil {
			return 0, fmt.Errorf("generate ordinary user id: %w", err)
		}
		id := ordinaryUserIDMin + offset.Int64()

		var count int64
		if err := tx.Model(&models.User{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return 0, fmt.Errorf("check ordinary user id: %w", err)
		}
		if count == 0 {
			return id, nil
		}
	}

	return 0, fmt.Errorf("failed to allocate ordinary user id after %d attempts", ordinaryUserIDAttempts)
}
