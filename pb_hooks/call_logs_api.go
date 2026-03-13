package pb_hooks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupCallLogsAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/call-logs/summary", handleCallLogsSummary)
		e.Router.GET("/api/call-logs/detail", handleCallLogsDetail)
		e.Router.GET("/api/call-logs/hourly", handleCallLogsHourly)
		return e.Next()
	})
}

// Summary endpoint for dashboard card
func handleCallLogsSummary(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	// Get date parameter (default: today)
	dateStr := c.Request.URL.Query().Get("date")
	var targetDate time.Time
	if dateStr == "" {
		// Get current IST time
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate = time.Now().In(istLocation)
	} else {
		var err error
		// Parse as IST date
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate, err = time.ParseInLocation("2006-01-02", dateStr, istLocation)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		}
	}

	// Convert IST date to UTC range
	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	startOfDayIST := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, istLocation)
	endOfDayIST := startOfDayIST.Add(24 * time.Hour)

	// Convert to UTC for database query
	startOfDay := startOfDayIST.UTC()
	endOfDay := endOfDayIST.UTC()

	dateFilter := fmt.Sprintf("attendance_date >= '%s' AND attendance_date < '%s'", startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))

	// Get present count from attendance
	type CountResult struct {
		Count int `db:"count" json:"count"`
	}
	var presentResult CountResult
	presentQuery := "SELECT COUNT(DISTINCT employee_code) as count FROM attendance WHERE " + dateFilter + " AND check_in_time IS NOT NULL"
	err := c.App.DB().NewQuery(presentQuery).One(&presentResult)
	presentCount := 0
	if err == nil {
		presentCount = presentResult.Count
	}

	// Get call logs stats with deduplication
	callDateFilter := fmt.Sprintf("call_timestamp >= '%s' AND call_timestamp < '%s'", startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))

	// Optimized: Single query with CTE for deduplication
	type StatsResult struct {
		CallCount     int `db:"call_count" json:"call_count"`
		TotalDuration int `db:"total_duration" json:"total_duration"`
	}
	var statsResult StatsResult

	// Deduplicated query: Groups by phone_number + timestamp_seconds + duration + employee_code
	statsQuery := `
		SELECT 
			COUNT(*) as call_count,
			COALESCE(SUM(call_duration), 0) as total_duration
		FROM (
			SELECT 
				phone_number,
				MAX(call_duration) as call_duration
			FROM call_logs
			WHERE ` + callDateFilter + ` AND call_duration > 0
			GROUP BY 
				phone_number,
				strftime('%Y-%m-%d %H:%M:%S', call_timestamp),
				call_duration,
				employee_code
		)
	`

	err = c.App.DB().NewQuery(statsQuery).One(&statsResult)
	totalCalls := 0
	totalDuration := 0
	if err == nil {
		totalCalls = statsResult.CallCount
		totalDuration = statsResult.TotalDuration
	} else {
		c.App.Logger().Error("❌ Stats query error", "error", err.Error())
	}

	// Avg per hour (10 AM - 7 PM)
	var hourlyResult CountResult
	hourlyQuery := "SELECT COUNT(*) as count FROM call_logs WHERE " + callDateFilter + " AND call_duration > 0 AND CAST(strftime('%H', call_timestamp) AS INTEGER) >= 10 AND CAST(strftime('%H', call_timestamp) AS INTEGER) < 19"
	err = c.App.DB().NewQuery(hourlyQuery).One(&hourlyResult)
	avgPerHour := 0
	if err == nil && hourlyResult.Count > 0 {
		avgPerHour = hourlyResult.Count / 9 // 9 working hours
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"present_count":  presentCount,
		"total_calls":    totalCalls,
		"total_duration": totalDuration,
		"avg_per_hour":   avgPerHour,
	})
}

// Detail endpoint for employee list
func handleCallLogsDetail(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	// Get date parameter
	dateStr := c.Request.URL.Query().Get("date")
	var targetDate time.Time
	if dateStr == "" {
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate = time.Now().In(istLocation)
	} else {
		var err error
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate, err = time.ParseInLocation("2006-01-02", dateStr, istLocation)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		}
	}

	// Convert IST date to UTC range
	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	startOfDayIST := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, istLocation)
	endOfDayIST := startOfDayIST.Add(24 * time.Hour)
	startOfDay := startOfDayIST.UTC()
	endOfDay := endOfDayIST.UTC()
	callDateFilter := fmt.Sprintf("call_timestamp >= '%s' AND call_timestamp < '%s'", startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))

	// Get active employees
	type User struct {
		EmployeeCode string `db:"employee_code" json:"employee_code"`
		EmployeeName string `db:"employee_name" json:"employee_name"`
		WFH          bool   `db:"wfh" json:"wfh"`
	}

	var users []User
	err := GetActiveEmployeesQuery(c.App).All(&users)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	results := []map[string]interface{}{}

	for _, user := range users {
		// Optimized: Single deduplicated query per employee instead of 3 separate queries
		type EmployeeStats struct {
			CallCount     int    `db:"call_count" json:"call_count"`
			TotalDuration int    `db:"total_duration" json:"total_duration"`
			LastCall      string `db:"last_call" json:"last_call"`
		}
		var stats EmployeeStats

		// Deduplicated query: Groups by phone_number + timestamp_seconds + duration
		statsQuery := `
			SELECT 
				COUNT(*) as call_count,
				COALESCE(SUM(call_duration), 0) as total_duration,
				COALESCE(MAX(call_timestamp), '') as last_call
			FROM (
				SELECT 
					phone_number,
					MAX(call_duration) as call_duration,
					MAX(call_timestamp) as call_timestamp
				FROM call_logs
				WHERE employee_code = {:code} AND ` + callDateFilter + ` AND call_duration > 0
				GROUP BY 
					phone_number,
					strftime('%Y-%m-%d %H:%M:%S', call_timestamp),
					call_duration
			)
		`

		c.App.DB().NewQuery(statsQuery).Bind(dbx.Params{"code": user.EmployeeCode}).One(&stats)

		// Only include employees with calls
		if stats.CallCount > 0 {
			results = append(results, map[string]interface{}{
				"employee_code":  user.EmployeeCode,
				"employee_name":  user.EmployeeName,
				"wfh":            user.WFH,
				"call_count":     stats.CallCount,
				"total_duration": stats.TotalDuration,
				"last_call_time": stats.LastCall,
			})
		}
	}
	return c.JSON(http.StatusOK, results)
}

// Hourly endpoint for employee call history
func handleCallLogsHourly(c *core.RequestEvent) error {
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Your account has been disabled. Please contact administrator."})
	}

	// Get parameters
	employeeCode := c.Request.URL.Query().Get("employee_code")
	dateStr := c.Request.URL.Query().Get("date")

	if employeeCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "employee_code parameter is required"})
	}

	var targetDate time.Time
	if dateStr == "" {
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate = time.Now().In(istLocation)
	} else {
		var err error
		istLocation, _ := time.LoadLocation("Asia/Kolkata")
		targetDate, err = time.ParseInLocation("2006-01-02", dateStr, istLocation)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		}
	}

	istLocation, _ := time.LoadLocation("Asia/Kolkata")
	results := []map[string]interface{}{}

	// First check: Total calls for this employee on this date (IST)
	startOfDayIST := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, istLocation)
	endOfDayIST := startOfDayIST.Add(24 * time.Hour)
	startOfDayUTC := startOfDayIST.UTC()
	endOfDayUTC := endOfDayIST.UTC()

	dayFilter := fmt.Sprintf("call_timestamp >= '%s' AND call_timestamp < '%s'", startOfDayUTC.Format("2006-01-02 15:04:05"), endOfDayUTC.Format("2006-01-02 15:04:05"))

	type CountResult struct {
		Count int `db:"count" json:"count"`
	}
	var totalDayCount CountResult
	testQuery := "SELECT COUNT(*) as count FROM call_logs WHERE employee_code = {:code} AND " + dayFilter + " AND call_duration > 0"
	c.App.DB().NewQuery(testQuery).Bind(dbx.Params{"code": employeeCode}).One(&totalDayCount)

	// Loop through hours 11 AM to 7 PM (IST display hours)
	for displayHour := 11; displayHour <= 19; displayHour++ {
		// Data hour is one less (11 AM shows 10-11 AM data)
		dataHour := displayHour - 1

		// Create IST hour range using startOfDayIST as base
		hourStartIST := startOfDayIST.Add(time.Duration(dataHour) * time.Hour)
		hourEndIST := hourStartIST.Add(1 * time.Hour)

		// Convert to UTC for database query
		hourStart := hourStartIST.UTC()
		hourEnd := hourEndIST.UTC()
		hourFilter := fmt.Sprintf("call_timestamp >= '%s' AND call_timestamp < '%s'", hourStart.Format("2006-01-02 15:04:05"), hourEnd.Format("2006-01-02 15:04:05"))

		// Optimized: Single deduplicated query for count and duration
		type HourStats struct {
			CallCount int `db:"call_count" json:"call_count"`
			Duration  int `db:"duration" json:"duration"`
		}
		var hourStats HourStats

		// Deduplicated query for this hour
		hourQuery := `
			SELECT 
				COUNT(*) as call_count,
				COALESCE(SUM(call_duration), 0) as duration
			FROM (
				SELECT 
					phone_number,
					MAX(call_duration) as call_duration
				FROM call_logs
				WHERE employee_code = {:code} AND ` + hourFilter + ` AND call_duration > 0
				GROUP BY 
					phone_number,
					strftime('%Y-%m-%d %H:%M:%S', call_timestamp),
					call_duration
			)
		`

		err := c.App.DB().NewQuery(hourQuery).Bind(dbx.Params{"code": employeeCode}).One(&hourStats)
		if err != nil {
			c.App.Logger().Error("Error in hourly query", "hour", displayHour, "error", err.Error())
		}

		// Calculate idle time (3600 seconds - call duration)
		idleTime := 3600 - hourStats.Duration

		results = append(results, map[string]interface{}{
			"hour":           displayHour,
			"call_count":     hourStats.CallCount,
			"total_duration": hourStats.Duration,
			"idle_time":      idleTime,
		})
	}

	return c.JSON(http.StatusOK, results)
}
