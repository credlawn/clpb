package pb_hooks

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type SQLiteTable struct {
	Name string `db:"name"`
}

type SQLiteColumn struct {
	Name string `db:"name"`
}

// SetupEmployeeCodeSync listens for changes to a user's employee_code
// and cascades that change dynamically to all tables in the database.
func SetupEmployeeCodeSync(app *pocketbase.PocketBase) {
	app.OnRecordAfterUpdateSuccess("users").Bind(&hook.Handler[*core.RecordEvent]{
		Func: func(e *core.RecordEvent) error {
			newCode := e.Record.GetString("employee_code")
			oldCode := e.Record.Original().GetString("employee_code")

			// Trigger only if there's an actual change and neither is empty
			if newCode != oldCode && newCode != "" && oldCode != "" {
				// Run in background to avoid blocking the user update request
				go syncEmployeeCodeAcrossDatabase(app, oldCode, newCode)
			}
			return e.Next()
		},
	})
}

func syncEmployeeCodeAcrossDatabase(app *pocketbase.PocketBase, oldCode, newCode string) {
	app.Logger().Info("Starting Universal Employee Code Sync", "oldCode", oldCode, "newCode", newCode)

	// Fetch all tables from SQLite (ignoring system tables)
	var tables []SQLiteTable
	err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != '_collections' AND name != '_admins' AND name != '_superusers'").All(&tables)
	if err != nil {
		app.Logger().Error("EmployeeCodeSync: Failed to fetch tables from SQLite", "error", err)
		return
	}

	updatedTablesCount := 0
	totalRecordsAffected := int64(0)

	for _, table := range tables {
		tableName := table.Name

		// Skip users table to prevent infinite loops / messing with the source of truth
		if tableName == "users" {
			continue
		}

		// Check if this table has a column named 'employee_code'
		var columns []SQLiteColumn
		err := app.DB().NewQuery("PRAGMA table_info(`" + tableName + "`)").All(&columns)
		if err != nil {
			continue
		}

		hasEmployeeCode := false
		for _, col := range columns {
			if col.Name == "employee_code" {
				hasEmployeeCode = true
				break
			}
		}

		// If it has the column, execute the UPDATE replacement
		if hasEmployeeCode {
			query := app.DB().NewQuery("UPDATE `" + tableName + "` SET employee_code = {:newCode} WHERE employee_code = {:oldCode}")
			query.Bind(dbx.Params{
				"newCode": newCode,
				"oldCode": oldCode,
			})

			result, updateErr := query.Execute()
			if updateErr == nil {
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected > 0 {
					app.Logger().Info("EmployeeCodeSync: Replaced codes", "table", tableName, "rowsAffected", rowsAffected)
					updatedTablesCount++
					totalRecordsAffected += rowsAffected
				}
			} else {
				app.Logger().Error("EmployeeCodeSync: Failed to update table", "table", tableName, "error", updateErr)
			}
		}
	}

	app.Logger().Info("Finished Universal Employee Code Sync", 
		"tablesUpdated", updatedTablesCount, 
		"totalRecordsChanged", totalRecordsAffected,
	)
}
