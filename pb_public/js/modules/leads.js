let currentDateFilter = 'today';
let customStartDate = null;
let customEndDate = null;

export function setupLeadsCard() {
    setupLeadsFilter();
}

function setupLeadsFilter() {
    const filterBtn = document.getElementById('leadsFilterBtn');
    const filterMenu = document.getElementById('leadsFilterMenu');
    const filterLabel = document.getElementById('leadsFilterLabel');

    filterBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        filterMenu.classList.toggle('hidden');
    });

    document.addEventListener('click', (e) => {
        if (!e.target.closest('#leadsFilterBtn') && !e.target.closest('#leadsFilterMenu')) {
            filterMenu.classList.add('hidden');
        }
    });

    filterMenu.querySelectorAll('button').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const filter = e.target.dataset.filter;

            if (filter === 'custom') {
                const startDate = prompt('Start Date (YYYY-MM-DD):');
                const endDate = prompt('End Date (YYYY-MM-DD):');

                if (startDate && endDate) {
                    customStartDate = startDate;
                    customEndDate = endDate;
                    currentDateFilter = 'custom';
                    filterLabel.textContent = `(${startDate} to ${endDate})`;
                } else {
                    return;
                }
            } else {
                currentDateFilter = filter;
                if (filter === 'all') {
                    filterLabel.textContent = '';
                } else if (filter === 'today') {
                    filterLabel.textContent = '(Today)';
                } else if (filter === 'month') {
                    filterLabel.textContent = '(This Month)';
                }
            }

            filterMenu.classList.add('hidden');
            await fetchLeadsStats();
        });
    });
}

function getDateFilter() {
    const now = new Date();

    if (currentDateFilter === 'all') {
        return '';
    } else if (currentDateFilter === 'today') {
        const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const endOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
        return `lead_status_date >= "${startOfDay.toISOString()}" AND lead_status_date <= "${endOfDay.toISOString()}"`;
    } else if (currentDateFilter === 'month') {
        const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
        const endOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59);
        return `lead_status_date >= "${startOfMonth.toISOString()}" AND lead_status_date <= "${endOfMonth.toISOString()}"`;
    } else if (currentDateFilter === 'custom' && customStartDate && customEndDate) {
        const start = new Date(customStartDate + 'T00:00:00');
        const end = new Date(customEndDate + 'T23:59:59');
        return `lead_status_date >= "${start.toISOString()}" AND lead_status_date <= "${end.toISOString()}"`;
    }

    return '';
}

export async function fetchLeadsStats() {
    const dateFilter = getDateFilter();

    document.getElementById('totalLeads').textContent = '...';

    try {
        const url = `/api/leads/stats${dateFilter ? `?filter=${encodeURIComponent(dateFilter)}` : ''}`;
        const response = await fetch(url);
        const stats = await response.json();

        document.getElementById('totalLeads').textContent = stats.total || 0;
        document.getElementById('leadsNew').textContent = stats.new || 0;
        document.getElementById('leadsCalled').textContent = stats.called || 0;
        document.getElementById('leadsCNR').textContent = stats.cnr || 0;
        document.getElementById('leadsDenied').textContent = stats.denied || 0;
        document.getElementById('leadsIPApproved').textContent = stats.ip_approved || 0;
        document.getElementById('leadsIPDecline').textContent = stats.ip_decline || 0;
        document.getElementById('leadsNoDocs').textContent = stats.no_docs || 0;
        document.getElementById('leadsAlreadyCarded').textContent = stats.already_carded || 0;
        document.getElementById('leadsNotEligible').textContent = stats.not_eligible || 0;
        document.getElementById('leadsFollowUp').textContent = stats.follow_up || 0;

        calculatePercentages(stats);

    } catch (error) {
        console.error('Error fetching leads:', error);
        document.getElementById('totalLeads').textContent = '0';
    }
}

function calculatePercentages(stats) {
    const counts = {
        new: stats.new || 0,
        called: stats.called || 0,
        cnr: stats.cnr || 0,
        denied: stats.denied || 0,
        ipApproved: stats.ip_approved || 0,
        ipDecline: stats.ip_decline || 0,
        noDocs: stats.no_docs || 0,
        alreadyCarded: stats.already_carded || 0,
        notEligible: stats.not_eligible || 0,
        followUp: stats.follow_up || 0
    };

    const sumExceptNew = counts.called + counts.cnr + counts.denied + counts.ipApproved +
        counts.ipDecline + counts.noDocs + counts.alreadyCarded +
        counts.notEligible + counts.followUp;

    const sumExceptNewCNR = counts.called + counts.denied + counts.ipApproved +
        counts.ipDecline + counts.noDocs + counts.alreadyCarded +
        counts.notEligible + counts.followUp;

    const ipTotal = counts.ipApproved + counts.ipDecline;

    document.getElementById('leadsCNRPct').textContent = '';
    document.getElementById('leadsDeniedPct').textContent = '';
    document.getElementById('leadsIPApprovedPct').textContent = '';
    document.getElementById('leadsIPDeclinePct').textContent = '';
    document.getElementById('leadsNoDocsPct').textContent = '';
    document.getElementById('leadsAlreadyCardedPct').textContent = '';
    document.getElementById('leadsNotEligiblePct').textContent = '';
    document.getElementById('leadsFollowUpPct').textContent = '';

    if (sumExceptNew > 0) {
        const cnrPct = ((counts.cnr / sumExceptNew) * 100).toFixed(1);
        document.getElementById('leadsCNRPct').textContent = `${cnrPct}%`;
    }

    if (sumExceptNewCNR > 0) {
        const deniedPct = ((counts.denied / sumExceptNewCNR) * 100).toFixed(1);
        document.getElementById('leadsDeniedPct').textContent = `${deniedPct}%`;

        const noDocsPct = ((counts.noDocs / sumExceptNewCNR) * 100).toFixed(1);
        document.getElementById('leadsNoDocsPct').textContent = `${noDocsPct}%`;

        const alreadyCardedPct = ((counts.alreadyCarded / sumExceptNewCNR) * 100).toFixed(1);
        document.getElementById('leadsAlreadyCardedPct').textContent = `${alreadyCardedPct}%`;

        const notEligiblePct = ((counts.notEligible / sumExceptNewCNR) * 100).toFixed(1);
        document.getElementById('leadsNotEligiblePct').textContent = `${notEligiblePct}%`;

        const followUpPct = ((counts.followUp / sumExceptNewCNR) * 100).toFixed(1);
        document.getElementById('leadsFollowUpPct').textContent = `${followUpPct}%`;
    }

    if (ipTotal > 0) {
        const ipApprovedPct = ((counts.ipApproved / ipTotal) * 100).toFixed(1);
        document.getElementById('leadsIPApprovedPct').textContent = `${ipApprovedPct}%`;

        const ipDeclinePct = ((counts.ipDecline / ipTotal) * 100).toFixed(1);
        document.getElementById('leadsIPDeclinePct').textContent = `${ipDeclinePct}%`;
    }
}
