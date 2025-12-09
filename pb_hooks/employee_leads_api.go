package pb_hooks

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type EmployeeWithLeads struct {
	EmployeeCode   string `db:"employee_code" json:"employee_code"`
	EmployeeName   string `db:"employee_name" json:"employee_name"`
	NewLeadsCount  int    `db:"new_leads_count" json:"new_leads_count"`
	TotalLeads     int    `db:"total_leads" json:"total_leads"`
}

func SetupEmployeeLeadsAPI(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/employees/with-new-leads", func(c *core.RequestEvent) error {
			info, _ := c.RequestInfo()
			if info.Auth == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Unauthorized",
				})
			}

			role := info.Auth.GetString("role")
			if role != "manager" && role != "Manager" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Only managers can view this data",
				})
			}

			type User struct {
				EmployeeCode string `db:"employee_code"`
				EmployeeName string `db:"employee_name"`
			}

			var users []User
			err := e.App.DB().NewQuery("SELECT employee_code, employee_name FROM users WHERE disabled = false AND LOWER(role) = 'employee'").All(&users)

			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": err.Error(),
				})
			}

			results := []EmployeeWithLeads{}

			for _, user := range users {
				type CountResult struct {
					Count int `db:"count"`
				}

				var newLeadsResult, totalLeadsResult CountResult

				e.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads WHERE employee_code = {:code} AND lead_status = 'New'").
					Bind(dbx.Params{"code": user.EmployeeCode}).One(&newLeadsResult)

				e.App.DB().NewQuery("SELECT COUNT(*) as count FROM leads WHERE employee_code = {:code}").
					Bind(dbx.Params{"code": user.EmployeeCode}).One(&totalLeadsResult)

				results = append(results, EmployeeWithLeads{
					EmployeeCode:  user.EmployeeCode,
					EmployeeName:  user.EmployeeName,
					NewLeadsCount: newLeadsResult.Count,
					TotalLeads:    totalLeadsResult.Count,
				})
			}

			return c.JSON(http.StatusOK, results)
		})

		return e.Next()
	})
}
