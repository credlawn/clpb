package pb_hooks

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// SetupDatabaseHooks registers collection-specific rules for the database collection
func SetupDatabaseHooks(app core.App) {
	
	// 1. Hook for NEW records (Data Cleaning)
	app.OnRecordCreate("database").BindFunc(func(e *core.RecordEvent) error {
		cleanDatabaseRecord(e.Record)
		return e.Next()
	})

	// 2. Hook for UPDATING records (Filtering, Resetting, Marking)
	app.OnRecordUpdate("database").BindFunc(func(e *core.RecordEvent) error {
		// Only apply business rules if the update is triggered by an Import
		// We detect this by checking if import_job_id is being set or is present
		if e.Record.GetString("import_job_id") == "" {
			return e.Next()
		}

		// A. FILTERING: Check the EXISTING status before it's overwritten
		rawStatus := e.Record.Original().GetString("lead_status")
		status := strings.ToUpper(strings.TrimSpace(rawStatus))

		if strings.Contains(status, "IP APPROVED") || 
		   strings.Contains(status, "IP DECLINE") || 
		   strings.Contains(status, "ALREADY CARDED") {
			// Returning an error will skip this update in the worker
			return fmt.Errorf("SKIP: Restricted Lead Status (%s)", rawStatus)
		}

		// B. DATA CLEANING: Clean the incoming data
		cleanDatabaseRecord(e.Record)

		// C. RESETTING: Reset specific fields as per requirements
		e.Record.Set("lead_status", "")
		e.Record.Set("lead_status_date", "")
		e.Record.Set("data_status", "")
		e.Record.Set("employee_name", "")
		e.Record.Set("employee_code", "")
		e.Record.Set("no_reallocation", false)

		// D. MARKING: Append /E to custom_code if not already present
		customCode := strings.TrimSpace(e.Record.GetString("custom_code"))
		if customCode != "" && customCode != "/E" {
			if !strings.HasSuffix(customCode, "/E") {
				e.Record.Set("custom_code", customCode+"/E")
			}
		} else {
			// If it was empty or just "/E", we keep it as "/E" only if it's truly an update without a code
			if customCode == "" {
				e.Record.Set("custom_code", "/E")
			}
		}

		return e.Next()
	})
}

// cleanDatabaseRecord performs basic field normalization
func cleanDatabaseRecord(record *core.Record) {
	// 1. Date Transformation
	if val := record.Get("old_decision_date"); val != nil {
		record.Set("old_decision_date", TransformDate(val))
	}

	// 2. String Normalization
	if val := record.GetString("customer_name"); val != "" {
		record.Set("customer_name", strings.ToUpper(strings.TrimSpace(val)))
	}
	if val := record.GetString("city"); val != "" {
		record.Set("city", strings.ToUpper(strings.TrimSpace(val)))
	}
	if val := record.GetString("product"); val != "" {
		record.Set("product", strings.ToUpper(strings.TrimSpace(val)))
	}
	if val := record.GetString("custom_code"); val != "" {
		record.Set("custom_code", strings.ToUpper(strings.TrimSpace(val)))
	}

	// 3. Global Trim for all fields
	for _, field := range record.Collection().Fields {
		name := field.GetName()
		if val := record.GetString(name); val != "" {
			record.Set(name, strings.TrimSpace(val))
		}
	}
}
