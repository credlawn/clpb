package pb_hooks

import (
"github.com/pocketbase/pocketbase/apis"
"github.com/pocketbase/pocketbase/core"
)

func SetupDisableUserCheck(app core.App) {
	app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
if e.Record.GetBool("disabled") {
return apis.NewBadRequestError("Your account has been disabled. Please contact administrator.", nil)
}
return e.Next()
	})
	
	app.OnRecordAuthRefreshRequest().BindFunc(func(e *core.RecordAuthRefreshRequestEvent) error {
if e.Record != nil && e.Record.GetBool("disabled") {
			return apis.NewBadRequestError("Your account has been disabled. Please contact administrator.", nil)
		}
		return e.Next()
	})
}
