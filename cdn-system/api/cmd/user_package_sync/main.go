// user_package_sync repairs or requeues a sold-package configuration after an
// interrupted update. It uses the same package task and all-primary-node
// config-sync paths as the API controller.
package main

import (
	"cdn-api/config"
	"cdn-api/db"
	"cdn-api/models"
	"cdn-api/services"
	"cdn-common/i18n"
	"encoding/json"
	"flag"
	"log"
	"os"
)

type syncSummary struct {
	UserPackageID int64 `json:"user_package_id"`
	Version       int   `json:"version"`
	ConfigVersion int64 `json:"config_version"`
}

func main() {
	userPackageID := flag.Int64("id", 0, "sold package ID to synchronize")
	flag.Parse()
	if *userPackageID <= 0 {
		log.Fatal("-id must be a positive sold package ID")
	}

	config.Load()
	if err := i18n.Load(""); err != nil {
		log.Fatal(err)
	}
	db.Init()
	if err := db.Ensure(); err != nil {
		log.Fatal(err)
	}
	if err := services.NewUserPackageService().SyncUserPackage(*userPackageID, "manual_recovery"); err != nil {
		log.Fatal(err)
	}
	configVersion := services.BumpUserPackageConfigVersion([]int64{*userPackageID})

	var userPackage models.UserPackage
	if err := db.DB.First(&userPackage, *userPackageID).Error; err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(syncSummary{
		UserPackageID: userPackage.ID,
		Version:       userPackage.Version,
		ConfigVersion: configVersion,
	}); err != nil {
		log.Fatal(err)
	}
}
