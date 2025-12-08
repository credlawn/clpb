export function createLeadsSummaryCard(stats) {
    const total = stats?.total || 0;
    const newLeads = stats?.new || 0;
    const todayCnr = stats?.today_cnr || 0;
    const todayDenied = stats?.today_denied || 0;
    const used = stats?.used || 0;

    return `
        <div class="summary-card" onclick="openDetailPanel('leadsPanel')">
            <div class="card-header">
                <div class="card-icon bg-blue-100">
                    <i data-feather="trending-up" class="w-5 h-5 text-blue-600"></i>
                </div>
                <span class="card-title">Leads Pipeline</span>
            </div>
            <div class="card-value">${total}</div>
            <div class="card-label">Total Leads</div>
            <div class="card-footer">
                <div class="mini-stat">
                    <div class="mini-stat-value text-blue-600">${newLeads}</div>
                    <div class="mini-stat-label">New</div>
                </div>
                <div class="mini-stat">
                    <div class="mini-stat-value text-green-600">${used}</div>
                    <div class="mini-stat-label">Used</div>
                </div>
                <div class="mini-stat">
                    <div class="mini-stat-value text-yellow-600">${todayCnr}</div>
                    <div class="mini-stat-label">CNR</div>
                </div>
                <div class="mini-stat">
                    <div class="mini-stat-value text-red-600">${todayDenied}</div>
                    <div class="mini-stat-label">Denied</div>
                </div>
            </div>
        </div>
    `;
}

export function createLeadsDetailPanel() {
    return `
        <div id="leadsPanel" class="detail-panel">
            <div class="detail-header">
                <h2>Leads Analytics</h2>
                <button class="detail-close" onclick="closeDetailPanel()">
                    <i data-feather="x" class="w-5 h-5 text-gray-600"></i>
                </button>
            </div>
            <div class="detail-content" id="leadsDetailContent">
            </div>
        </div>
    `;
}
