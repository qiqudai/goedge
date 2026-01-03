package services

import (
	"cdn-api/db"
	"cdn-api/models"
	"fmt"
	"log"
	"time"
)

func StartUserPackageExpirationWorker() {
	go func() {
		for {
			checkUserPackageExpiration()
			time.Sleep(1 * time.Hour) // Check every hour
		}
	}()
}

func checkUserPackageExpiration() {
	var expiredPackages []models.UserPackage
	now := time.Now()
	// Find active packages that have expired
	// We check for packages where EndAt < Now AND (they are not already marked expired/disabled if there was a status field, 
	// but UserPackage doesn't have a direct 'status' enum persisted, logic usually infers it from EndAt).
	// However, SyncUserPackage logic needs to be triggered. 
	// The problem: if we trigger every hour for already expired packages, we spam tasks.
	// Solution: We need a way to know if we already synced the expiration.
	// Maybe check if the last Sync was for expiration? Or check if Version is up to date?
	// But `agent_config.go` status='expired' is derived.
	// 
	// Spec says: "Worker scans for expired packages -> Trigger SyncUserPackage(id, 'expire')"
	// To avoid repetition:
	// 1. We could add a `Status` field to `UserPackage` (UserPackageRow had it derived). 
	//    But schema migration just for this? Maybe.
	// 2. Or, we assume "expired" state is when `EndAt < Now`.
	//    We can check `Job` logs? No.
	//    The most robust way is to rely on `UserPackage` having a flag or strictly checking if we need to sync.
	//    
	//    Actually, if `SyncUserPackage` is idempotent/versioned, maybe it's fine?
	//    But creating Tasks every hour is bad.
	//    
	//    Let's check if there's a property `IsEnabled` or similar we can toggle 0?
	//    `UserPackage` has no `Enable`. It has `StartAt`/`EndAt`.
	//    
	//    Alternative: We scan for packages expired in the LAST HOUR (or interval).
	//    `EndAt` between `Now - Interval` and `Now`.
	//    This handles "just expired".
	//    But if worker crashes, we miss them.
	//    
	//    Better: Select where `EndAt < Now` AND `version` hasn't been updated for expiration? No.
	//    
	//    Let's use the "Scan last hour" approach combined with adequate overlap or state.
	//    Or, since I added `Version`, maybe I can add `Status` to `UserPackage` in DB?
	//    The `user_package_controller` derived `status`.
	//    Let's add `Status` column to `UserPackage`? 
	//    Wait, `models/user_package.go` doesn't have `Status`.
	//    
	//    Let's stick to "Just Expired" window for now, or finding packages that are expired but might not have been synced.
	//    Actually, the `agent_config` has `status`.
	//    
	//    Let's try: Find `UserPackage` where `EndAt < Now`.
	//    For each, we check if we recently created a "expire" task?
	//    
	//    Simpler: Just iterate all expired packages that are NOT YET synced as expired?
	//    How do we know?
	//    
	//    Let's go with:
	//    Iterate `EndAt < Now`.
	//    We can't easily know if we synced.
	//    
	//    Correction: UserPackage doesn't have `Enable`.
	//    If we want to stop service, we must set `Status='expired'` in Agent Config.
	//    
	//    Proposed Strategy:
	//    Add `IsExpired` (bool) to `UserPackage` model.
	//    When `EndAt < Now` AND `IsExpired == false`:
	//      Set `IsExpired = true`.
	//      Trigger Sync.
	//      Save.
	//    
	//    This requires DB migration for `is_expired`.
	//    I'll add `IsExpired` field to `models.UserPackage`.
	
	err := db.DB.Where("end_at < ? AND (is_expired = ? OR is_expired IS NULL)", now, false).Find(&expiredPackages).Error
	if err != nil {
		log.Println("[Error] Check expiration:", err)
		return
	}

	for _, p := range expiredPackages {
		// Mark as expired to avoid re-processing
		// We use UpdateColumn to verify it wasn't valid.
		if err := db.DB.Model(&p).Update("is_expired", true).Error; err != nil {
			continue
		}
		
		fmt.Printf("[INFO] Package %d expired. Syncing...\n", p.ID)
		notifyPackageExpired(p)
		if err := NewUserPackageService().SyncUserPackage(p.ID, "expire"); err != nil {
			log.Printf("[Error] Failed to sync expired package %d: %v\n", p.ID, err)
		}
	}
}

func notifyPackageExpired(pkg models.UserPackage) {
	userID := int64(pkg.UserID)
	if userID == 0 {
		return
	}
	title := "Package expired"
	content := fmt.Sprintf("Package %s expired at %s.", pkg.Name, pkg.EndAt.Format("2006-01-02 15:04:05"))
	_ = CreateUserMessage(userID, "package-expire", title, content, pkg.ID, 0)
}
