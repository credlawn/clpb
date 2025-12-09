package pb_hooks

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupLeadsAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/leads/stats", handleLeadsStats)
		e.Router.GET("/api/leads/breakdown", handleLeadsBreakdown)
		return e.Next()
	})
}

func handleLeadsStats(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	dateFilter := c.Request.URL.Query().Get("filter")

	type StatusCount struct {
		Status string `db:"lead_status" json:"status"`
		Count  int    `db:"count" json:"count"`
	}

	var results []StatusCount
	var err error

	if dateFilter != "" {
		query := "SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' AND " + dateFilter + " GROUP BY lead_status"
		err = c.App.DB().NewQuery(query).All(&results)
	} else {
		err = c.App.DB().NewQuery("SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' GROUP BY lead_status").All(&results)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "query": dateFilter})
	}

	type CountResult struct {
		Count int `db:"count" json:"count"`
	}

	var newCountResult CountResult
	err = c.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads WHERE lead_status = 'New'").One(&newCountResult)
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

	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
	todayFilter := fmt.Sprintf("lead_status_date >= '%s' AND lead_status_date <= '%s'", startOfDay.Format(time.RFC3339), endOfDay.Format(time.RFC3339))

	var todayResults []StatusCount
	todayQuery := "SELECT lead_status, COUNT(*) as count FROM leads WHERE lead_status != 'New' AND " + todayFilter + " GROUP BY lead_status"
	todayErr := c.App.DB().NewQuery(todayQuery).All(&todayResults)
	if todayErr != nil {
		log.Printf("Error fetching today's lead stats: %v", todayErr)
	}

	todayStatsMap := make(map[string]int)
	usedCount := 0
	for _, r := range todayResults {
		todayStatsMap[r.Status] = r.Count
		usedCount += r.Count
	}

	todayCnrCount := todayStatsMap["CNR"] + todayStatsMap["Voicemail"]
	todayDeniedCount := todayStatsMap["Denied"]

	return c.JSON(http.StatusOK, map[string]interface{}{
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
		"today_cnr":      todayCnrCount,
		"today_denied":   todayDeniedCount,
		"used":           usedCount,
	})
}

func handleLeadsBreakdown(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	status := c.Request.URL.Query().Get("status")
	dateFilter := c.Request.URL.Query().Get("filter")

	if status == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status parameter is required"})
	}

	type User struct {
		EmployeeCode string `db:"employee_code" json:"employee_code"`
		EmployeeName string `db:"employee_name" json:"employee_name"`
	}

	var users []User
	err := c.App.DB().NewQuery("SELECT employee_code, employee_name FROM users WHERE disabled = false AND LOWER(role) = 'employee'").All(&users)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	type CountResult struct {
		Count int `db:"count" json:"count"`
	}

	results := []map[string]interface{}{}

	for _, user := range users {
		var countResult CountResult

		query := "SELECT COUNT(*) as count FROM leads WHERE employee_code = {:code} AND lead_status = {:status}"
		if dateFilter != "" && status != "New" && status != "Called" {
			query += " AND " + dateFilter
		}

		c.App.DB().NewQuery(query).Bind(dbx.Params{
			"code":   user.EmployeeCode,
			"status": status,
		}).One(&countResult)

		results = append(results, map[string]interface{}{
			"employee_name": user.EmployeeName,
			"count":         countResult.Count,
		})
	}

	return c.JSON(http.StatusOK, results)
}
