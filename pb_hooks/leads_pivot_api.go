package pb_hooks

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Constants for better maintainability
const (
	timezoneIST        = "Asia/Kolkata"
	roleEmployee       = "employee"
	roleManager        = "manager"
	statusNew          = "New"
	statusCNR          = "CNR"
	statusVoicemail    = "Voicemail"
	statusDenied       = "Denied"
	statusCalled       = "Called"
	maxEmployeeCodeLen = 50
)

func SetupLeadsPivotAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/leads/pivot", handleLeadsAnalytics)
		return e.Next()
	})
}

func handleLeadsAnalytics(c *core.RequestEvent) error {
	// Auth check
	info, _ := c.RequestInfo()
	if info.Auth == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	if info.Auth.GetBool("disabled") {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account disabled"})
	}

	// Get and validate parameters
	dateStr := c.Request.URL.Query().Get("date")
	filterType := c.Request.URL.Query().Get("filter_type")
	employeeCode := strings.TrimSpace(c.Request.URL.Query().Get("employee_code"))

	// Validate employee code length to prevent abuse
	if len(employeeCode) > maxEmployeeCodeLen {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid employee code"})
	}

	// Load timezone once
	istLocation, err := time.LoadLocation(timezoneIST)
	if err != nil {
		c.App.Logger().Error("Failed to load timezone", "timezone", timezoneIST, "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Server configuration error"})
	}

	// Parse date and convert IST to UTC
	var targetDate time.Time
	if dateStr == "" {
		targetDate = time.Now().In(istLocation)
	} else {
		targetDate, err = time.ParseInLocation("2006-01-02", dateStr, istLocation)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
		}
	}

	// Build date filter
	var startOfDayIST, endOfDayIST time.Time

	switch filterType {
	case "yesterday":
		yesterday := targetDate.AddDate(0, 0, -1)
		startOfDayIST = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, istLocation)
		endOfDayIST = startOfDayIST.Add(24 * time.Hour)
	default: // today
		startOfDayIST = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, istLocation)
		endOfDayIST = startOfDayIST.Add(24 * time.Hour)
	}

	// Convert to UTC for database
	startOfDay := startOfDayIST.UTC()
	endOfDay := endOfDayIST.UTC()

	// Build query with parameterized values using dbx
	query := c.App.DB().
		Select(
			"u.employee_code",
			"u.employee_name",
			"u.wfh",
			"u.role",
			"u.disabled",
			"u.designation",

			"(SELECT COUNT(id) FROM leads WHERE employee_code = u.employee_code AND lead_status = {:statusNew}) as new_count",
			"COUNT(CASE WHEN l.lead_status IS NOT NULL AND l.lead_status != {:statusNew2} THEN 1 END) as total_activity",
			"COUNT(CASE WHEN l.lead_status = 'IP Approved' THEN 1 END) as ip_approved_count",
			"COUNT(CASE WHEN l.lead_status = 'IP Decline' THEN 1 END) as ip_decline_count",
			"COUNT(CASE WHEN l.lead_status = 'Follow Up' THEN 1 END) as follow_up_count",
			"COUNT(CASE WHEN l.lead_status = 'No Docs' THEN 1 END) as no_docs_count",
			"COUNT(CASE WHEN l.lead_status = 'Already Carded' THEN 1 END) as already_carded_count",
			"COUNT(CASE WHEN l.lead_status = 'Not Eligible' THEN 1 END) as not_eligible_count",
			"COUNT(CASE WHEN l.lead_status IN ({:statusCNR}, {:statusVoicemail}) THEN 1 END) as cnr_count",
			"COUNT(CASE WHEN l.lead_status = {:statusDenied} THEN 1 END) as denied_count",
			"COUNT(CASE WHEN l.lead_status = {:statusCalled} THEN 1 END) as called_count",
		).
		From("users u").
		LeftJoin("leads l", dbx.And(
			dbx.NewExp("u.employee_code = l.employee_code"),
			dbx.NewExp("l.lead_status_date >= {:startDate}", dbx.Params{"startDate": startOfDay.Format("2006-01-02 15:04:05")}),
			dbx.NewExp("l.lead_status_date < {:endDate}", dbx.Params{"endDate": endOfDay.Format("2006-01-02 15:04:05")}),
		)).
		Where(GetActiveEmployeesFilter()).
		GroupBy("u.employee_code", "u.employee_name", "u.wfh").
		OrderBy("total_activity DESC").
		Bind(dbx.Params{
			"statusNew":       statusNew,
			"statusNew2":      statusNew,
			"statusCNR":       statusCNR,
			"statusVoicemail": statusVoicemail,
			"statusDenied":    statusDenied,
			"statusCalled":    statusCalled,
		})

	// Add employee filter if provided (using parameterized query)
	if employeeCode != "" {
		query.AndWhere(dbx.NewExp("l.employee_code = {:employeeCode}", dbx.Params{"employeeCode": employeeCode}))
	}

	rows, err := query.Rows()
	if err != nil {
		c.App.Logger().Error("Database query failed", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to fetch data",
		})
	}
	defer rows.Close()

	employees := []map[string]interface{}{}
	var summaryNew, summaryActivity, summaryIpApproved, summaryIpDecline int
	var summaryFollowUp, summaryNoDocs, summaryAlreadyCarded, summaryNotEligible int
	var summaryCnr, summaryDenied, summaryCalled int

	for rows.Next() {
		var employeeCode, employeeName, role, designation string
		var wfh, disabled bool

		var newCount, totalActivity, ipApproved, ipDecline, followUp int
		var noDocs, alreadyCarded, notEligible, cnr, denied, called int

		err := rows.Scan(
			&employeeCode, &employeeName, &wfh, &role, &disabled, &designation,

			&newCount, &totalActivity,
			&ipApproved, &ipDecline, &followUp,
			&noDocs, &alreadyCarded, &notEligible,
			&cnr, &denied, &called,
		)

		if err != nil {
			c.App.Logger().Warn("Row scan failed", "employee", employeeCode, "error", err)
			continue
		}

		// Calculate employee metrics dynamically
		// Unproductive = CNR + Voicemail + Denied + Called (fixed list)
		unproductive := cnr + denied + called
		// Productive = Everything else (Total Activity - Unproductive)
		// This is future-proof: if new statuses are added, they automatically count as productive
		productive := totalActivity - unproductive
		worked := productive

		productivity := 0.0
		if totalActivity > 0 {
			productivity = float64(productive) / float64(totalActivity) * 100
		}

		employees = append(employees, map[string]interface{}{
			"employee_code":  employeeCode,
			"employee_name":  employeeName,
			"wfh":            wfh,
			"role":           role,
			"disabled":       disabled,
			"designation":    designation,

			"new":            newCount,
			"total":          totalActivity,
			"productive":     productive,
			"unproductive":   unproductive,
			"worked":         worked,
			"productivity":   fmt.Sprintf("%.1f", productivity),
			"ip_approved":    ipApproved,
			"ip_decline":     ipDecline,
			"follow_up":      followUp,
			"no_docs":        noDocs,
			"already_carded": alreadyCarded,
			"not_eligible":   notEligible,
			"cnr":            cnr,
			"denied":         denied,
			"called":         called,
		})

		// Aggregate for summary
		summaryNew += newCount
		summaryActivity += totalActivity
		summaryIpApproved += ipApproved
		summaryIpDecline += ipDecline
		summaryFollowUp += followUp
		summaryNoDocs += noDocs
		summaryAlreadyCarded += alreadyCarded
		summaryNotEligible += notEligible
		summaryCnr += cnr
		summaryDenied += denied
		summaryCalled += called
	}

	// Calculate summary metrics dynamically
	summaryUnproductive := summaryCnr + summaryDenied + summaryCalled
	summaryProductive := summaryActivity - summaryUnproductive
	summaryWorked := summaryProductive
	summaryProductivity := 0.0
	if summaryActivity > 0 {
		summaryProductivity = float64(summaryProductive) / float64(summaryActivity) * 100
	}

	response := map[string]interface{}{
		"summary": map[string]interface{}{
			"new_leads":      summaryNew,
			"total_activity": summaryActivity,
			"productive":     summaryProductive,
			"unproductive":   summaryUnproductive,
			"worked":         summaryWorked,
			"productivity":   fmt.Sprintf("%.1f", summaryProductivity),
			"breakdown": map[string]interface{}{
				"productive": map[string]interface{}{
					"ip_approved":    summaryIpApproved,
					"ip_decline":     summaryIpDecline,
					"follow_up":      summaryFollowUp,
					"no_docs":        summaryNoDocs,
					"already_carded": summaryAlreadyCarded,
					"not_eligible":   summaryNotEligible,
				},
				"unproductive": map[string]interface{}{
					"cnr":    summaryCnr,
					"denied": summaryDenied,
					"called": summaryCalled,
				},
			},
		},
		"employees": employees,
	}

	return c.JSON(http.StatusOK, response)
}
