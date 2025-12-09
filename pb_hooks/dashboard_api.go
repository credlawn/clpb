package pb_hooks

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupDashboardAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/dashboard/summary", handleDashboardSummary)
		return e.Next()
	})
}

func handleDashboardSummary(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account disabled"})
	}

	type CountResult struct {
		Count int `db:"count"`
	}

	var totalResult, newResult CountResult
	c.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads").One(&totalResult)
	c.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads WHERE lead_status = 'New'").One(&newResult)

	type StatusCount struct {
		Status string `db:"lead_status"`
		Count  int    `db:"count"`
	}

	var todayResults []StatusCount
	todayQuery := "SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' AND date(lead_status_date) = date('now') GROUP BY lead_status"
	c.App.DB().NewQuery(todayQuery).All(&todayResults)

	todayStats := make(map[string]int)
	usedCount := 0
	for _, r := range todayResults {
		todayStats[r.Status] = r.Count
		usedCount += r.Count
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":        totalResult.Count,
		"new":          newResult.Count,
		"used":         usedCount,
		"today_cnr":    todayStats["CNR"] + todayStats["Voicemail"],
		"today_denied": todayStats["Denied"],
	})
}
