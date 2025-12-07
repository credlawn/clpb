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
