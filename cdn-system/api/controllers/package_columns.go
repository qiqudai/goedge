package controllers

import (
	"cdn-api/db"
	"cdn-api/models"
)

func ensurePackageL2OriginColumn() error {
	if db.DB == nil {
		return nil
	}
	if db.DB.Migrator().HasColumn(&models.Package{}, "l2_origin") {
		return nil
	}
	return db.DB.Migrator().AddColumn(&models.Package{}, "L2Origin")
}

func ensureUserPackageL2OriginColumn() error {
	if db.DB == nil {
		return nil
	}
	if db.DB.Migrator().HasColumn(&models.UserPackage{}, "l2_origin") {
		return nil
	}
	return db.DB.Migrator().AddColumn(&models.UserPackage{}, "L2Origin")
}
