package pb_hooks

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type ShufflePreviewRequest struct {
	LeadStatuses []string `json:"lead_statuses"`
	MinAgeDays   int      `json:"min_age_days"`
}

type ShuffleRequest struct {
	LeadStatuses []string `json:"lead_statuses"`
	MinAgeDays   int      `json:"min_age_days"`
	Allocations  []struct {
		EmployeeCode string `json:"employee_code"`
		EmployeeName string `json:"employee_name"`
		Count        int    `json:"count"`
	} `json:"allocations"`
	AllocatedByCode string `json:"allocated_by_code"`
	AllocatedByName string `json:"allocated_by_name"`
}

type EligibleLead struct {
	DatabaseID     string
	LeadID         string
	MobileNo       string
	CustomerName   string
	CurrentEmpCode string
	RPScore        float64
	DaysSince      int
	AllocCount     int
	ConnectedCalls int
	ConnDuration   int
}

func SetupLeadShuffle(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/shuffle-preview", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can shuffle leads"})
			}

			var req ShufflePreviewRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
			}

			if req.MinAgeDays == 0 {
				req.MinAgeDays = 1
			}

			eligible := getEligibleLeads(e.App, req.LeadStatuses, req.MinAgeDays)

			return c.JSON(http.StatusOK, map[string]interface{}{
				"eligible_count": len(eligible),
				"leads":          eligible,
			})
		})

		e.Router.POST("/api/shuffle-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can shuffle leads"})
			}

			var req ShuffleRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
			}

			if req.MinAgeDays == 0 {
				req.MinAgeDays = 1
			}

			eligible := getEligibleLeads(e.App, req.LeadStatuses, req.MinAgeDays)

			shuffledCount := 0
			skippedCount := 0
			distribution := make(map[string]int)

			for _, alloc := range req.Allocations {
				if alloc.Count <= 0 {
					continue
				}

				user, err := e.App.FindFirstRecordByFilter("users", "employee_code = {:code}", dbx.Params{"code": alloc.EmployeeCode})
				if err != nil {
					continue
				}

				assigned := 0
				for i := 0; i < len(eligible) && assigned < alloc.Count; {
					lead := eligible[i]

					previousEmps := getPreviousEmployees(e.App, lead.DatabaseID)
					if previousEmps[alloc.EmployeeCode] {
						i++
						continue
					}

					if lead.CurrentEmpCode == alloc.EmployeeCode {
						i++
						continue
					}

					existingLead, _ := e.App.FindRecordById("leads", lead.LeadID)
					if existingLead == nil {
						i++
						skippedCount++
						continue
					}

					existingLead.Set("employee_code", alloc.EmployeeCode)
					existingLead.Set("employee_name", alloc.EmployeeName)
					existingLead.Set("assigned_to", user.Id)
					existingLead.Set("assigned_date", time.Now().UTC().Format(time.RFC3339))
					existingLead.Set("lead_status", "New")
					existingLead.Set("lead_status_date", time.Now().UTC().Format(time.RFC3339))

					if err := e.App.Save(existingLead); err != nil {
						i++
						skippedCount++
						continue
					}

					historyCollection, _ := e.App.FindCollectionByNameOrId("lead_allocation_history")
					if historyCollection != nil {
						e.App.DB().NewQuery("UPDATE lead_allocation_history SET is_active = FALSE, deallocated_date = {:date} WHERE lead_record_id = {:id} AND is_active = TRUE").
							Bind(dbx.Params{"id": lead.LeadID, "date": time.Now().Format(time.RFC3339)}).Execute()

						var maxSeq struct {
							Seq int `db:"seq"`
						}
						e.App.DB().NewQuery("SELECT COALESCE(MAX(allocation_sequence), 0) as seq FROM lead_allocation_history WHERE database_record_id = {:id}").
							Bind(dbx.Params{"id": lead.DatabaseID}).One(&maxSeq)

						historyRecord := core.NewRecord(historyCollection)
						historyRecord.Set("database_record_id", lead.DatabaseID)
						historyRecord.Set("lead_record_id", lead.LeadID)
						historyRecord.Set("mobile_no", lead.MobileNo)
						historyRecord.Set("customer_name", lead.CustomerName)
						historyRecord.Set("allocated_to_code", alloc.EmployeeCode)
						historyRecord.Set("allocated_to_name", alloc.EmployeeName)
						historyRecord.Set("allocated_by_code", req.AllocatedByCode)
						historyRecord.Set("allocated_by_name", req.AllocatedByName)
						historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
						historyRecord.Set("allocation_type", "shuffle")
						historyRecord.Set("is_active", true)
						historyRecord.Set("allocation_sequence", maxSeq.Seq+1)
						e.App.Save(historyRecord)
					}

					dbRecord, _ := e.App.FindFirstRecordByFilter("database", "mobile_no = {:mobile}", dbx.Params{"mobile": lead.MobileNo})
					if dbRecord != nil {
						shuffleCount := dbRecord.GetInt("shuffle_count")
						dbRecord.Set("shuffle_count", shuffleCount+1)
						dbRecord.Set("last_shuffle_date", time.Now().UTC().Format(time.RFC3339))

						var uniqueEmployees struct {
							Count int `db:"count"`
						}
						e.App.DB().NewQuery("SELECT COUNT(DISTINCT allocated_to_code) as count FROM lead_allocation_history WHERE database_record_id = {:id}").
							Bind(dbx.Params{"id": lead.DatabaseID}).One(&uniqueEmployees)

						currentAlloc := dbRecord.GetInt("allocation_count")
						dbRecord.Set("allocation_count", currentAlloc+1)
						dbRecord.Set("employee_count", uniqueEmployees.Count)

						leadStatus := existingLead.GetString("lead_status")
						totalCalls := dbRecord.GetInt("total_calls")
						if (leadStatus == "Denied" && uniqueEmployees.Count >= 3) ||
							(leadStatus == "CNR" && (uniqueEmployees.Count >= 3 || totalCalls >= 10)) {
							dbRecord.Set("data_status", "inactive")
						}

						e.App.Save(dbRecord)
					}

					eligible = append(eligible[:i], eligible[i+1:]...)
					assigned++
					shuffledCount++
					distribution[alloc.EmployeeCode]++
				}
			}

			return c.JSON(http.StatusOK, map[string]interface{}{
				"success":        true,
				"shuffled_count": shuffledCount,
				"skipped_count":  skippedCount,
				"distribution":   distribution,
			})
		})

		return e.Next()
	})
}

func getEligibleLeads(app core.App, statuses []string, minAgeDays int) []EligibleLead {
	var eligible []EligibleLead

	if len(statuses) == 0 {
		statuses = []string{"CNR", "Denied"}
	}

	statusList := "'" + strings.Join(statuses, "','") + "'"

	cutoffTime := time.Now().UTC().AddDate(0, 0, -minAgeDays)

	query := `
		SELECT 
			l.id as lead_id,
			l.mobile_no,
			l.customer_name,
			l.employee_code as current_emp_code,
			l.lead_status_date,
			COALESCE(l.total_calls, 0) as total_calls,
			COALESCE(l.connected_calls, 0) as connected_calls,
			COALESCE(l.total_duration, 0) as conn_duration
		FROM leads l
		WHERE l.lead_status IN (` + statusList + `)
		  AND l.lead_status_date < {:cutoff}
	`

	type LeadRow struct {
		LeadID         string `db:"lead_id"`
		MobileNo       string `db:"mobile_no"`
		CustomerName   string `db:"customer_name"`
		CurrentEmpCode string `db:"current_emp_code"`
		LeadStatusDate string `db:"lead_status_date"`
		TotalCalls     int    `db:"total_calls"`
		ConnectedCalls int    `db:"connected_calls"`
		ConnDuration   int    `db:"conn_duration"`
	}

	var leads []LeadRow
	app.DB().NewQuery(query).Bind(dbx.Params{"cutoff": cutoffTime.Format("2006-01-02 15:04:05.000Z")}).All(&leads)

	for _, lead := range leads {
		daysSince := 0
		if lead.LeadStatusDate != "" {
			if t, err := time.Parse("2006-01-02 15:04:05.000Z", lead.LeadStatusDate); err == nil {
				daysSince = int(time.Since(t).Hours() / 24)
			}
		}

		rps := float64(daysSince*10) +
			float64(10-lead.ConnectedCalls*2) -
			float64(lead.ConnDuration)/60.0

		eligible = append(eligible, EligibleLead{
			DatabaseID:     lead.LeadID,
			LeadID:         lead.LeadID,
			MobileNo:       lead.MobileNo,
			CustomerName:   lead.CustomerName,
			CurrentEmpCode: lead.CurrentEmpCode,
			RPScore:        math.Round(rps*100) / 100,
			DaysSince:      daysSince,
			AllocCount:     0,
			ConnectedCalls: lead.ConnectedCalls,
			ConnDuration:   lead.ConnDuration,
		})
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].RPScore > eligible[j].RPScore
	})

	return eligible
}

func getPreviousEmployees(app core.App, dbRecordID string) map[string]bool {
	result := make(map[string]bool)

	type EmpRow struct {
		Code string `db:"allocated_to_code"`
	}

	var employees []EmpRow
	app.DB().NewQuery("SELECT DISTINCT allocated_to_code FROM lead_allocation_history WHERE database_record_id = {:id}").
		Bind(dbx.Params{"id": dbRecordID}).All(&employees)

	for _, emp := range employees {
		result[emp.Code] = true
	}

	return result
}
