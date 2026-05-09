package pb_hooks

import (
    "github.com/pocketbase/pocketbase/core"
)

// SetupImportWorker registers all import-related hooks and APIs.
func SetupImportWorker(app core.App) {
	// Setup database-specific hooks for the 'database' collection
	SetupDatabaseHooks(app)

	// Register the import headers API endpoint
	SetupImportHeadersAPI(app)

	// Register the import job trigger hook
	app.OnRecordAfterUpdateSuccess(CollectionImportJobs).BindFunc(func(e *core.RecordEvent) error {
		status := e.Record.GetString("status")
		// Only trigger when status changes to 'needs_mapping' (user finished mapping)
		if status != "pending" {
			return e.Next()
		}

		jobID := e.Record.Id

		// Prevent duplicate goroutines for the same job
		if _, loaded := inProgressImports.LoadOrStore(jobID, true); loaded {
			return e.Next()
		}

		// Launch background import processor
		go func(jobID string) {
			ProcessImportJob(e.App, jobID)
		}(jobID)

		return e.Next()
	})
}
