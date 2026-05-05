package pb_hooks

import (
	"time"

	"github.com/pocketbase/pocketbase"
)

// SetupImportCleanup sets up a daily cron job to delete processed Excel files.
// This prevents the server storage from filling up with old import files.
func SetupImportCleanup(app *pocketbase.PocketBase) {
	// Cron expression: "30 19 * * *"
	// 19:30 UTC corresponds to 01:00 AM IST (UTC + 5:30).
	app.Cron().MustAdd("import_file_cleanup", "30 19 * * *", func() {
		app.Logger().Info("Cron: Starting Refined Import Cleanup (1 AM IST)")

		// 1. Fetch ALL records from import_jobs to apply different rules
		records, err := app.FindRecordsByFilter("import_jobs", "1=1", "-created", 0, 0)
		if err != nil {
			app.Logger().Error("Cron Cleanup: Failed to fetch jobs", "error", err)
			return
		}

		fsys, err := app.NewFilesystem()
		if err != nil {
			app.Logger().Error("Cron Cleanup: Filesystem error", "error", err)
			return
		}
		defer fsys.Close()

		now := time.Now()
		thirtyDaysAgo := now.AddDate(0, 0, -30)
		
		filesDeleted := 0
		recordsDeleted := 0

		for _, rec := range records {
			status := rec.GetString("status")
			importDateStr := rec.GetString("import_date")
			fileName := rec.GetString("file")
			createdTime := rec.GetDateTime("created").Time()

			shouldDeleteRecord := false
			shouldDeleteFileOnly := false

			// Rule A: Delete records older than 30 days
			if createdTime.Before(thirtyDaysAgo) {
				shouldDeleteRecord = true
			} else if status == "needs_mapping" || importDateStr == "" {
				// Rule B: Delete draft or incomplete jobs
				shouldDeleteRecord = true
			} else if (status == "completed" || status == "failed") && fileName != "" {
				// Rule C: For finished jobs, delete file but keep record history
				shouldDeleteFileOnly = true
			}

			if shouldDeleteRecord {
				// Delete physical file first if it exists
				if fileName != "" {
					_ = fsys.Delete(rec.BaseFilesPath() + "/" + fileName)
				}
				// Delete database record
				if err := app.Delete(rec); err == nil {
					recordsDeleted++
				}
			} else if shouldDeleteFileOnly {
				// Delete physical file
				if err := fsys.Delete(rec.BaseFilesPath() + "/" + fileName); err == nil {
					filesDeleted++
					// Clear file field in DB
					rec.Set("file", "")
					_ = app.Save(rec)
				}
			}
		}

		app.Logger().Info("Cron Cleanup: Finished", 
			"records_deleted", recordsDeleted, 
			"files_only_deleted", filesDeleted,
		)
	})
}
