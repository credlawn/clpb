package pb_hooks

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
)

// SetupVKYCCleanupCron sets up a daily cron job to delete VKYC records
// that are resolved (Success/Failed) or have expired (T-2 days).
func SetupVKYCCleanupCron(app *pocketbase.PocketBase) {
	// Cron expression: "40 19 * * *"
	// 19:40 UTC corresponds to 01:10 AM IST.
	app.Cron().MustAdd("vkyc_cleanup", "40 19 * * *", func() {
		app.Logger().Info("Cron: Starting VKYC Cleanup (1:10 AM IST)")

		records, err := app.FindRecordsByFilter("vkyc", "1=1", "", 0, 0)
		if err != nil {
			app.Logger().Error("Cron Cleanup: Failed to fetch vkyc records", "error", err)
			return
		}

		deletedCount := 0

		// Set up T-2 date (Day before yesterday, ignoring time)
		now := time.Now()
		t2Date := time.Date(now.Year(), now.Month(), now.Day()-2, 0, 0, 0, 0, time.UTC)

		for _, rec := range records {
			// 1. Terminal Status Deletion
			status := strings.ToLower(rec.GetString("bank_vkyc_status"))
			if status == "success" || status == "failed" {
				if err := app.Delete(rec); err == nil {
					deletedCount++
				} else {
					app.Logger().Error("Cron Cleanup: Failed to delete terminal VKYC record", "id", rec.Id, "error", err)
				}
				continue
			}

			// 2. Expiry Date Deletion (T-2 days)
			expTime := rec.GetDateTime("vkyc_expiry_date").Time()
			if !expTime.IsZero() {
				expDate := time.Date(expTime.Year(), expTime.Month(), expTime.Day(), 0, 0, 0, 0, time.UTC)
				
				// If expiry date is T-2 (or older), delete it
				if expDate.Unix() <= t2Date.Unix() {
					if err := app.Delete(rec); err == nil {
						deletedCount++
					} else {
						app.Logger().Error("Cron Cleanup: Failed to delete expired VKYC record", "id", rec.Id, "error", err)
					}
					continue
				}
			}
		}

		app.Logger().Info("Cron Cleanup: VKYC Cleanup Finished", 
			"total_checked", len(records),
			"records_deleted", deletedCount,
		)
	})
}
