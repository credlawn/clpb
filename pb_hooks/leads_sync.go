package pb_hooks

import (
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func SetupLeadsSync(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/sync-leads-to-database", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can run sync"})
			}

			type LeadRow struct {
				ID             string `db:"id"`
				CustomerName   string `db:"customer_name"`
				MobileNo       string `db:"mobile_no"`
				LeadStatus     string `db:"lead_status"`
				LeadStatusDate string `db:"lead_status_date"`
				Remarks        string `db:"remarks"`
				Segment        string `db:"segment"`
				Employer       string `db:"employer"`
				DeclineReason  string `db:"decline_reason"`
				Product        string `db:"product"`
				DateOfBirth    string `db:"date_of_birth"`
				EmployeeName   string `db:"employee_name"`
				EmployeeCode   string `db:"employee_code"`
				AssignedTo     string `db:"assigned_to"`
				AssignedDate   string `db:"assigned_date"`
			}

			var leads []LeadRow
			e.App.DB().NewQuery(`
				SELECT id, customer_name, mobile_no, lead_status, lead_status_date,
					   remarks, segment, employer, decline_reason, product,
					   date_of_birth, employee_name, employee_code, assigned_to, assigned_date
				FROM leads
			`).All(&leads)

			dbCollection, _ := e.App.FindCollectionByNameOrId("database")
			historyCollection, _ := e.App.FindCollectionByNameOrId("lead_allocation_history")

			createdCount := 0
			updatedCount := 0
			skippedCount := 0
			historyCreatedCount := 0

			for _, lead := range leads {
				if lead.MobileNo == "" {
					skippedCount++
					continue
				}

				dbRecord, _ := e.App.FindFirstRecordByFilter("database", "mobile_no = {:mobile}", dbx.Params{"mobile": lead.MobileNo})

				if dbRecord == nil {
					newRecord := core.NewRecord(dbCollection)
				newRecord.Set("customer_name", lead.CustomerName)
				newRecord.Set("mobile_no", lead.MobileNo)
				newRecord.Set("lead_status", lead.LeadStatus)
				newRecord.Set("lead_status_date", lead.LeadStatusDate)
				newRecord.Set("remarks", lead.Remarks)
				newRecord.Set("segment", lead.Segment)
				newRecord.Set("employer", lead.Employer)
				newRecord.Set("decline_reason", lead.DeclineReason)
				newRecord.Set("product", lead.Product)
				newRecord.Set("date_of_birth", lead.DateOfBirth)
				newRecord.Set("employee_name", lead.EmployeeName)
				newRecord.Set("employee_code", lead.EmployeeCode)
				newRecord.Set("data_status", "used")
				newRecord.Set("allocation_count", 1)
				newRecord.Set("employee_count", 1)

					if err := e.App.Save(newRecord); err == nil {
						createdCount++
						dbRecord = newRecord
					}
				} else {
					// Check for IP Approved / IP Decline status in database record
					dbLeadStatus := dbRecord.GetString("lead_status")
					if dbLeadStatus == "IP Approved" || dbLeadStatus == "IP Decline" {
						skippedCount++
						continue
					}

					shouldSave := false

					// Update lead status fields
					if dbLeadStatus != lead.LeadStatus || dbRecord.GetString("lead_status_date") != lead.LeadStatusDate {
						dbRecord.Set("lead_status", lead.LeadStatus)
						dbRecord.Set("lead_status_date", lead.LeadStatusDate)
						shouldSave = true
					}

					currentStatus := dbRecord.GetString("data_status")
					if strings.ToLower(currentStatus) == "new" {
						dbRecord.Set("data_status", "used")
						allocCount := dbRecord.GetInt("allocation_count")
						dbRecord.Set("allocation_count", allocCount+1)
						shouldSave = true
					}

					if shouldSave {
						if err := e.App.Save(dbRecord); err == nil {
							updatedCount++
						} else {
							skippedCount++ // Count as skipped if save failed? Or just don't count update
						}
					} else {
						skippedCount++
					}
				}

				if dbRecord != nil && historyCollection != nil {
					var existingHistory struct {
						Count int `db:"count"`
					}
					e.App.DB().NewQuery("SELECT COUNT(*) as count FROM lead_allocation_history WHERE mobile_no = {:mobile}").
						Bind(dbx.Params{"mobile": lead.MobileNo}).One(&existingHistory)

					if existingHistory.Count == 0 {
						historyRecord := core.NewRecord(historyCollection)
						historyRecord.Set("database_record_id", dbRecord.Id)
						historyRecord.Set("lead_record_id", lead.ID)
						historyRecord.Set("mobile_no", lead.MobileNo)
						historyRecord.Set("customer_name", lead.CustomerName)
						historyRecord.Set("allocated_to_code", lead.EmployeeCode)
						historyRecord.Set("allocated_to_name", lead.EmployeeName)
						historyRecord.Set("allocated_by_code", "SYNC")
						historyRecord.Set("allocated_by_name", "System Sync")
						if lead.AssignedDate != "" {
							historyRecord.Set("allocation_date", lead.AssignedDate)
						} else {
							historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
						}
						historyRecord.Set("allocation_type", "initial")
						historyRecord.Set("is_active", true)
						historyRecord.Set("allocation_sequence", 1)

						if err := e.App.Save(historyRecord); err == nil {
							historyCreatedCount++
						}
					}
				}
			}

			return c.JSON(http.StatusOK, map[string]interface{}{
				"success":               true,
				"total_leads":           len(leads),
				"database_created":      createdCount,
				"database_updated":      updatedCount,
				"skipped":               skippedCount,
				"history_records_created": historyCreatedCount,
			})
		})

		e.Router.POST("/api/sync-call-stats", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can run sync"})
			}

			type CallStats struct {
				PhoneNumber      string `db:"phone_number"`
				TotalCalls       int    `db:"total_calls"`
				ConnectedCalls   int    `db:"connected_calls"`
				ConnectedDuration int   `db:"connected_duration"`
			}

			var stats []CallStats
			e.App.DB().NewQuery(`
				SELECT 
					phone_number,
					COUNT(*) as total_calls,
					SUM(CASE WHEN call_duration > 0 THEN 1 ELSE 0 END) as connected_calls,
					SUM(COALESCE(call_duration, 0)) as connected_duration
				FROM call_logs
				GROUP BY phone_number
			`).All(&stats)

			updatedCount := 0

			for _, stat := range stats {
				dbRecord, _ := e.App.FindFirstRecordByFilter("database", "mobile_no = {:mobile}", dbx.Params{"mobile": stat.PhoneNumber})
				if dbRecord != nil {
					dbRecord.Set("total_calls", stat.TotalCalls)
					dbRecord.Set("connected_calls", stat.ConnectedCalls)
					dbRecord.Set("connected_duration", stat.ConnectedDuration)
					if err := e.App.Save(dbRecord); err == nil {
						updatedCount++
					}
				}
			}

			return c.JSON(http.StatusOK, map[string]interface{}{
				"success":        true,
				"total_phones":   len(stats),
				"database_updated": updatedCount,
			})
		})

		return e.Next()
	})
}
