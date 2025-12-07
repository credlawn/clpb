export function createEmployeeSummaryCard(stats) {
    const employees = stats || [];
    const totalIPA = employees.reduce((sum, e) => sum + (e.ipa || 0), 0);
    const totalIPD = employees.reduce((sum, e) => sum + (e.ipd || 0), 0);
    const grandTotal = totalIPA + totalIPD;
    const activeCount = employees.length;

    return `
        <div class="summary-card" onclick="openDetailPanel('employeePanel')">
            <div class="card-header">
                <div class="card-icon bg-purple-100">
                    <i data-feather="users" class="w-5 h-5 text-purple-600"></i>
                </div>
                <span class="card-title">Employee Performance</span>
            </div>
            <div class="card-value">${activeCount}</div>
            <div class="card-label">Active Employees</div>
            <div class="card-footer">
                <div class="mini-stat">
                    <div class="mini-stat-value text-green-600">${totalIPA}</div>
                    <div class="mini-stat-label">IPA</div>
                </div>
                <div class="mini-stat">
                    <div class="mini-stat-value text-red-600">${totalIPD}</div>
                    <div class="mini-stat-label">IPD</div>
                </div>
                <div class="mini-stat">
                    <div class="mini-stat-value text-blue-600">${grandTotal}</div>
                    <div class="mini-stat-label">Total</div>
                </div>
            </div>
        </div>
    `;
}

export function createEmployeeDetailPanel() {
    return `
        <div id="employeePanel" class="detail-panel">
            <div class="detail-header">
                <h2>Employee Performance</h2>
                <button class="detail-close" onclick="closeDetailPanel()">
                    <i data-feather="x" class="w-5 h-5 text-gray-600"></i>
                </button>
            </div>
            <div class="detail-content" id="employeeDetailContent">
            </div>
        </div>
    `;
}
