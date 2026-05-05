package pb_hooks

import (
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// SetupCaseLoginCascade sets up hooks to automatically propagate employee details
// to downstream collections whenever a case_login record is created or updated.
// It runs asynchronously so it doesn't block or rollback the case_login write.
func SetupCaseLoginCascade(app *pocketbase.PocketBase) {
	handler := func(e *core.RecordEvent) error {
		// Clone the record so we can safely use it inside the goroutine
		rec := e.Record.Clone()
		
		// Run in a goroutine so we don't block the API response
		go func(record *core.Record) {
			arnNo := strings.TrimSpace(record.GetString("arn_no"))
			if arnNo == "" {
				return
			}

			empName := record.GetString("employee_name")
			empCode := record.GetString("employee_code")
			mobNum := record.GetString("mobile_number") // note: field is mobile_number in case_login

			safeArn := strings.ReplaceAll(arnNo, "'", "''")
			filter := "arn_no = '" + safeArn + "'"

			collections := []string{"adobe_dump", "bank_approved_cards", "activation", "bkyc", "vkyc"}

			for _, collName := range collections {
				records, err := app.FindRecordsByFilter(collName, filter, "", 0, 0)
				if err != nil || len(records) == 0 {
					// No record found or error, skip silently as requested
					continue
				}

				for _, target := range records {
					modified := false
					if target.GetString("employee_name") != empName {
						target.Set("employee_name", empName)
						modified = true
					}
					if target.GetString("employee_code") != empCode {
						target.Set("employee_code", empCode)
						modified = true
					}
					if target.GetString("mobile_no") != mobNum {
						target.Set("mobile_no", mobNum)
						modified = true
					}

					if modified {
						if err := app.Save(target); err != nil {
							app.Logger().Error("Cascade Sync Failed", "collection", collName, "arn", arnNo, "error", err)
						}
					}
				}
			}
		}(rec)

		return e.Next()
	}

	app.OnRecordAfterCreateSuccess("case_login").BindFunc(handler)
	app.OnRecordAfterUpdateSuccess("case_login").BindFunc(handler)
}
