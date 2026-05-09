package pb_hooks

import "time"

// Import job statuses
const (
	StatusNeedsMapping = "needs_mapping"
	StatusReading      = "reading"
	StatusValidating   = "validating"
	StatusProcessing   = "processing"
	StatusCompleted    = "completed"
	StatusFailed       = "failed"
)

// Import modes
const (
	ModeCreateUpdate = "create_update"
	ModeCreateOnly   = "create_only"
	ModeUpdateOnly   = "update_only"
)

// Field mapping special values
const (
	IgnoreValue  = "__IGNORE__"
	ManualValue  = "__MANUAL__"
	StaticPrefix = "__STATIC__:"
)

// Data status values
const (
	DataStatusNew      = "new"
	DataStatusUsed     = "used"
	DataStatusInactive = "inactive"
)

// Lead status values (used across collections)
const (
	LeadStatusNew       = "New"
	LeadStatusCNR       = "CNR"
	LeadStatusDenied    = "Denied"
	LeadStatusFollowUp  = "Follow Up"
	LeadStatusCalled    = "Called"
	LeadStatusVoicemail = "Voicemail"
)

// Default import job timeout
const DefaultImportTimeout = 30 * time.Minute

// Batch size for database operations
const DefaultBatchSize = 100

// Working hours for call analytics (10 AM - 7 PM IST)
const (
	WorkingHourStart  = 10
	WorkingHourEnd    = 19
	WorkingHoursCount = WorkingHourEnd - WorkingHourStart
)

// Default timezone
const DefaultTimezone = "Asia/Kolkata"

// System collection names (fixed)
const (
	CollectionImportJobs    = "import_jobs"
	CollectionImportMappings = "import_mappings"
)
