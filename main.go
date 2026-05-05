package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"custompb/pb_hooks"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/osutils"

	_ "custompb/migrations"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		// auto-run migrations on startup
		Automigrate:  true,
		TemplateLang: migratecmd.TemplateLangGo,
	})

	pb_hooks.InitFirebase()
	pb_hooks.SetupIPANotification(app)
	pb_hooks.SetupCaseLoginHook(app)
	pb_hooks.SetupCallCount(app)
	pb_hooks.SetupDisableUserCheck(app)
	pb_hooks.SetupUserNotificationSyncHook(app)
	pb_hooks.SetupLeadAllocation(app)
	pb_hooks.SetupEmployeeLeadsAPI(app)
	pb_hooks.SetupLeadsAPI(app)
	pb_hooks.SetupDashboardAPI(app)
	pb_hooks.SetupEmployeeStatsAPI(app)
	pb_hooks.SetupLeadReallocation(app)
	pb_hooks.SetupLeadShuffle(app)
	pb_hooks.SetupDatabaseFilters(app)
	pb_hooks.SetupDatabaseCount(app)
	pb_hooks.SetupMobileLeadAllocation(app)
	pb_hooks.SetupMobileReallocationAvailability(app) // NEW: Check reallocation availability
	pb_hooks.SetupLeadsSync(app)
	pb_hooks.SetupN8NSync(app)
	pb_hooks.SetupCallLogsAPI(app)
	pb_hooks.SetupLeadsPivotAPI(app)
	pb_hooks.SetupAttendanceSyncHook(app)
	pb_hooks.SetupOnDutyCron(app)
	pb_hooks.SetupBirthdayReminderCron(app)
	pb_hooks.SetupDatabaseSyncCron(app)         // NEW: Daily sync cron at 1 AM
	pb_hooks.SetupAutoLeadReallocationCron(app) // NEW: Auto lead reallocation every 5 minutes
	pb_hooks.SetupImportWorker(app)             // NEW: Background Excel Import Worker
	pb_hooks.SetupImportCleanup(app)            // NEW: Daily Excel File Cleanup Cron (1 AM IST)
	pb_hooks.SetupActivationCleanupCron(app)    // NEW: Activation cleanup Cron (1:05 AM IST)
	pb_hooks.SetupVKYCCleanupCron(app)          // NEW: VKYC cleanup Cron (1:10 AM IST)
	pb_hooks.SetupCaseLoginCascade(app)         // NEW: Auto-cascade employee details on case_login update

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
