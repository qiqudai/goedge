package services

import (
	"cdn-api/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGenerateOrdinaryUserIDRangeAndCollision(t *testing.T) {
	tx, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := tx.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	if err := tx.Create(&models.User{ID: ordinaryUserIDMin, Name: "existing", Type: 2, Enable: true}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	for i := 0; i < 128; i++ {
		id, err := GenerateOrdinaryUserID(tx)
		if err != nil {
			t.Fatalf("generate id: %v", err)
		}
		if id < ordinaryUserIDMin || id >= ordinaryUserIDMin+ordinaryUserIDRange {
			t.Fatalf("id out of range: %d", id)
		}
		if id == ordinaryUserIDMin {
			t.Fatalf("id collided with existing user: %d", id)
		}
		if err := tx.Create(&models.User{ID: id, Name: "user", Type: 2, Enable: true}).Error; err != nil {
			t.Fatalf("insert generated user %d: %v", id, err)
		}
	}
}

func TestGenerateOrdinaryUserIDRejectsNilDB(t *testing.T) {
	if _, err := GenerateOrdinaryUserID(nil); err == nil {
		t.Fatal("expected nil database error")
	}
}
