package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/osutils"
	"custompb/pb_hooks"
)

func main() {
	app := pocketbase.New()

	pb_hooks.InitFirebase()
	pb_hooks.SetupIPANotification(app)
	pb_hooks.SetupCaseLoginHook(app)
	pb_hooks.SetupCallCount(app)
	pb_hooks.SetupDisableUserCheck(app)
	pb_hooks.SetupLeadAllocation(app)
	pb_hooks.SetupEmployeeLeadsAPI(app)
	pb_hooks.SetupLeadsAPI(app)
	pb_hooks.SetupDashboardAPI(app)
	pb_hooks.SetupEmployeeStatsAPI(app)
	pb_hooks.SetupLeadReallocation(app)
	pb_hooks.SetupLeadShuffle(app)
	pb_hooks.SetupLeadsSync(app)
	pb_hooks.SetupN8NSync(app)

	app.OnRecordCreateExecute("database").BindFunc(func(e *core.RecordEvent) error {
		mobileNo := e.Record.GetString("mobile_no")
		
		if mobileNo == "" {
			return e.Next()
		}
		
		record, _ := e.App.FindFirstRecordByData("database", "mobile_no", mobileNo)
		
		if record != nil {
			return apis.NewBadRequestError("Mobile number already exists", nil)
		}
		
		return e.Next()
	})

	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			publicDir := defaultPublicDir()
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), true))
			}

			return e.Next()
		},
		Priority: 999,
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func defaultPublicDir() string {
	if osutils.IsProbablyGoRun() {
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
