package pb_hooks

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
)

// SetupBKYCCleanupCron sets up a daily cron job to delete BKYC records
// that are completed (Done/remove_data) or have aged beyond 60 days.
func SetupBKYCCleanupCron(app *pocketbase.PocketBase) {
	// Cron expression: "45 19 * * *"
	// 19:45 UTC corresponds to 01:15 AM IST.
	app.Cron().MustAdd("bkyc_cleanup", "45 19 * * *", func() {
		app.Logger().Info("Cron: Starting BKYC Cleanup (1:15 AM IST)")

		records, err := app.FindRecordsByFilter("bkyc", "1=1", "", 0, 0)
		if err != nil {
			app.Logger().Error("Cron Cleanup: Failed to fetch bkyc records", "error", err)
			return
		}

		deletedCount := 0

		for _, rec := range records {
			// 1. Terminal Status Deletion
			status := strings.ToLower(strings.TrimSpace(rec.GetString("bank_status")))
			shouldRemove := rec.GetBool("remove_data")

			if status == "done" || shouldRemove {
				if err := app.Delete(rec); err == nil {
					deletedCount++
				} else {
					app.Logger().Error("Cron Cleanup: Failed to delete terminal BKYC record", "id", rec.Id, "error", err)
				}
				continue
			}

			// 2. Aging Deletion (Older than T-60 days)
			arnTime := rec.GetDateTime("arn_date").Time()
			if !arnTime.IsZero() {
				daysSince := time.Since(arnTime).Hours() / 24
				
				// If older than 60 days, delete it
				if daysSince > 60 {
					if err := app.Delete(rec); err == nil {
						deletedCount++
					} else {
						app.Logger().Error("Cron Cleanup: Failed to delete expired BKYC record", "id", rec.Id, "error", err)
					}
					continue
				}
			}
		}

		app.Logger().Info("Cron Cleanup: BKYC Cleanup Finished", 
			"total_checked", len(records),
			"records_deleted", deletedCount,
		)
	})
}
