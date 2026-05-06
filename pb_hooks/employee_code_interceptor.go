package pb_hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// SetupEmployeeCodeInterceptor creates a global middleware for all API create/update requests.
// It intercepts offline/stale data from the mobile app and forces the `employee_code`
// (and employee_name) to match the currently authenticated user's actual profile.
func SetupEmployeeCodeInterceptor(app *pocketbase.PocketBase) {

	// Intercept all API Create Requests
	app.OnRecordCreateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			return forceEmployeeCode(e)
		},
	})

	// Intercept all API Update Requests
	app.OnRecordUpdateRequest().Bind(&hook.Handler[*core.RecordRequestEvent]{
		Func: func(e *core.RecordRequestEvent) error {
			return forceEmployeeCode(e)
		},
	})
}

// forceEmployeeCode checks if the request is made by an authenticated user
// and if the target collection has an 'employee_code' field. If so, it overwrites it.
func forceEmployeeCode(e *core.RecordRequestEvent) error {
	// 1. If not an authenticated request (or not a user auth), skip.
	if e.Auth == nil || e.Auth.Collection().Name != "users" {
		return e.Next()
	}

	// 2. Skip if the collection being updated is the "users" collection itself
	if e.Collection.Name == "users" {
		return e.Next()
	}

	// 3. Check if the target collection has an 'employee_code' field
	if e.Collection.Fields.GetByName("employee_code") != nil {
		// Overwrite the incoming employee_code with the true one from Auth token
		actualCode := e.Auth.GetString("employee_code")
		if actualCode != "" {
			e.Record.Set("employee_code", actualCode)
		}

		// Optionally keep employee_name synced too if it exists in the collection
		if e.Collection.Fields.GetByName("employee_name") != nil {
			actualName := e.Auth.GetString("employee_name")
			if actualName != "" {
				e.Record.Set("employee_name", actualName)
			}
		}
	}

	return e.Next()
}
