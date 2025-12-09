package pb_hooks

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type ReallocationRequest struct {
	DatabaseRecordIDs []string `json:"database_record_ids"`
	Allocations       []struct {
		EmployeeCode string `json:"employee_code"`
		EmployeeName string `json:"employee_name"`
		Count        int    `json:"count"`
	} `json:"allocations"`
	AllocatedByCode string `json:"allocated_by_code"`
	AllocatedByName string `json:"allocated_by_name"`
}

type ReallocationResponse struct {
	Success          bool           `json:"success"`
	TotalSelected    int            `json:"total_selected"`
	ReallocatedCount int            `json:"reallocated_count"`
	SkippedCount     int            `json:"skipped_count"`
	Distribution     map[string]int `json:"distribution"`
}

func SetupLeadReallocation(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/reallocate-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			}

			role := info.Auth.GetString("role")
			if strings.ToLower(role) != "manager" {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only managers can reallocate leads"})
			}

			var req ReallocationRequest
			if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			}

			totalSelected := len(req.DatabaseRecordIDs)
			reallocatedCount := 0
			skippedCount := 0
			distribution := make(map[string]int)

			availableRecords := make([]string, len(req.DatabaseRecordIDs))
			copy(availableRecords, req.DatabaseRecordIDs)

			for _, alloc := range req.Allocations {
				if alloc.Count > len(availableRecords) {
					alloc.Count = len(availableRecords)
				}

				selectedRecords := availableRecords[:alloc.Count]
				availableRecords = availableRecords[alloc.Count:]

				user, err := e.App.FindFirstRecordByFilter("users", "employee_code = {:code}", dbx.Params{"code": alloc.EmployeeCode})
				if err != nil {
					skippedCount += len(selectedRecords)
					continue
				}

				for _, dbRecordID := range selectedRecords {
					dbRecord, err := e.App.FindRecordById("database", dbRecordID)
					if err != nil {
						skippedCount++
						continue
					}

					mobileNo := dbRecord.GetString("mobile_no")

					existingLead, _ := e.App.FindFirstRecordByFilter("leads", "mobile_no = {:mobile}", dbx.Params{"mobile": mobileNo})
					if existingLead == nil {
						skippedCount++
						continue
					}

					if existingLead.GetString("employee_code") == alloc.EmployeeCode {
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
						skippedCount++
						continue
					}

					historyCollection, _ := e.App.FindCollectionByNameOrId("lead_allocation_history")
					if historyCollection != nil {
						e.App.DB().NewQuery("UPDATE lead_allocation_history SET is_active = FALSE, deallocated_date = {:date} WHERE lead_record_id = {:id} AND is_active = TRUE").
							Bind(dbx.Params{"id": existingLead.Id, "date": time.Now().Format(time.RFC3339)}).Execute()

						var maxSeq struct {
							Seq int `db:"seq"`
						}
						e.App.DB().NewQuery("SELECT COALESCE(MAX(allocation_sequence), 0) as seq FROM lead_allocation_history WHERE database_record_id = {:id}").
							Bind(dbx.Params{"id": dbRecordID}).One(&maxSeq)

						historyRecord := core.NewRecord(historyCollection)
						historyRecord.Set("database_record_id", dbRecordID)
						historyRecord.Set("lead_record_id", existingLead.Id)
						historyRecord.Set("mobile_no", mobileNo)
						historyRecord.Set("customer_name", dbRecord.GetString("customer_name"))
						historyRecord.Set("allocated_to_code", alloc.EmployeeCode)
						historyRecord.Set("allocated_to_name", alloc.EmployeeName)
						historyRecord.Set("allocated_by_code", req.AllocatedByCode)
						historyRecord.Set("allocated_by_name", req.AllocatedByName)
						historyRecord.Set("allocation_date", time.Now().Format(time.RFC3339))
						historyRecord.Set("allocation_type", "reallocation")
						historyRecord.Set("is_active", true)
						historyRecord.Set("allocation_sequence", maxSeq.Seq+1)
						e.App.Save(historyRecord)
					}

					var uniqueEmployees struct {
						Count int `db:"count"`
					}
					e.App.DB().NewQuery("SELECT COUNT(DISTINCT allocated_to_code) as count FROM lead_allocation_history WHERE database_record_id = {:id}").
						Bind(dbx.Params{"id": dbRecordID}).One(&uniqueEmployees)

					currentCount := dbRecord.GetInt("allocation_count")
					e.App.DB().NewQuery("UPDATE database SET allocation_count = {:count}, employee_count = {:emp_count} WHERE id = {:id}").
						Bind(dbx.Params{"count": currentCount + 1, "emp_count": uniqueEmployees.Count, "id": dbRecordID}).Execute()

					reallocatedCount++
					distribution[alloc.EmployeeCode]++
				}
			}

			return c.JSON(http.StatusOK, ReallocationResponse{
				Success:          true,
				TotalSelected:    totalSelected,
				ReallocatedCount: reallocatedCount,
				SkippedCount:     skippedCount,
				Distribution:     distribution,
			})
		})

		return e.Next()
	})
}
