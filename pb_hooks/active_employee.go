package pb_hooks

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// GetActiveEmployeesFilter returns the standard WHERE clause for filtering active employees
// This ensures consistency across all APIs (leads, call logs, IPA stats, etc.)
//
// Filter criteria:
// - disabled = false (employee is not disabled)
// - no_atn = false (employee is not marked as no attendance)
// - role = 'employee' OR role = 'manager' (case-insensitive)
func GetActiveEmployeesFilter() dbx.Expression {
	return dbx.And(
		dbx.NewExp("disabled = false"),
		dbx.NewExp("no_atn = false"),
		dbx.Or(
			dbx.NewExp("LOWER(role) = 'employee'"),
			dbx.NewExp("LOWER(role) = 'manager'"),
		),
	)
}

// GetActiveEmployeesQuery returns a base query for fetching active employees
// This can be used as a starting point and extended with additional joins/filters
//
// Usage:
//
//	query := GetActiveEmployeesQuery(app)
//	query.LeftJoin("leads l", ...).GroupBy(...)
func GetActiveEmployeesQuery(app core.App) *dbx.SelectQuery {
	return app.DB().
		Select(
			"employee_code",
			"employee_name",
			"wfh",
			"role",
		).
		From("users").
		Where(GetActiveEmployeesFilter()).
		OrderBy("employee_name ASC")
}
