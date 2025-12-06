package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/ghupdate"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/osutils"
)

func main() {
	app := pocketbase.New()

	// ---------------------------------------------------------------
	// Optional plugin flags:
	// ---------------------------------------------------------------

	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(
		&hooksDir,
		"hooksDir",
		"",
		"the directory with the JS app hooks",
	)

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(
		&hooksWatch,
		"hooksWatch",
		true,
		"auto restart the app on pb_hooks file change; it has no effect on Windows",
	)

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(
		&hooksPool,
		"hooksPool",
		15,
		"the total prewarm goja.Runtime instances for the JS app hooks execution",
	)

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path, e.g. when pretty urls are used with SPA",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	// ---------------------------------------------------------------
	// Plugins and hooks:
	// ---------------------------------------------------------------

	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: migrationsDir,
		HooksDir:      hooksDir,
		HooksWatch:    hooksWatch,
		HooksPoolSize: hooksPool,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	// GitHub selfupdate
	ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{})

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
			e.Router.GET("/api/leads/stats", func(c *core.RequestEvent) error {
				dateFilter := c.Request.URL.Query().Get("filter")
				
				type StatusCount struct {
					Status string `db:"lead_status" json:"status"`
					Count  int    `db:"count" json:"count"`
				}
				
				var results []StatusCount
				var err error
				
				if dateFilter != "" {
					query := "SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' AND " + dateFilter + " GROUP BY lead_status"
					err = e.App.DB().NewQuery(query).All(&results)
				} else {
					err = e.App.DB().NewQuery("SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' GROUP BY lead_status").All(&results)
				}
				
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"error": err.Error(),
						"query": dateFilter,
					})
				}
				
				type CountResult struct {
					Count int `db:"count" json:"count"`
				}
				
				var newCountResult CountResult
				err = e.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads WHERE lead_status = 'New'").One(&newCountResult)
				newCount := 0
				if err == nil {
					newCount = newCountResult.Count
				}
				
				statsMap := make(map[string]int)
				totalCount := newCount
				
				for _, r := range results {
					statsMap[r.Status] = r.Count
					totalCount += r.Count
				}
				
				cnrCount := statsMap["CNR"] + statsMap["Voicemail"]
				
				response := map[string]interface{}{
					"total":          totalCount,
					"new":            newCount,
					"called":         statsMap["Called"],
					"cnr":            cnrCount,
					"denied":         statsMap["Denied"],
					"ip_approved":    statsMap["IP Approved"],
					"ip_decline":     statsMap["IP Decline"],
					"no_docs":        statsMap["No Docs"],
					"already_carded": statsMap["Already Carded"],
					"not_eligible":   statsMap["Not Eligible"],
					"follow_up":      statsMap["Follow Up"],
				}
				
				return c.JSON(http.StatusOK, response)
			})
			
			return e.Next()
		},
		Priority: 100,
	})

	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), indexFallback))
			}

			return e.Next()
		},
		Priority: 999, // execute as latest as possible to allow users to provide their own route
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if osutils.IsProbablyGoRun() {
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
