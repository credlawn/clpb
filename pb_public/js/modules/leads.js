import pb from '../utils/pb.js';
import { openCustomDateModal } from './customDateModal.js';

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
                filterMenu.classList.add('hidden');
                openCustomDateModal((startDate, endDate) => {
                    customStartDate = startDate;
                    customEndDate = endDate;
                    currentDateFilter = 'custom';

                    const formatDate = (dateStr) => {
                        const [year, month, day] = dateStr.split('-');
                        return `${day}-${month}-${year.slice(2)}`;
                    };

                    filterLabel.textContent = `(${formatDate(startDate)} to ${formatDate(endDate)})`;
                    fetchLeadsStats();
                });
                return;
            } else {
                currentDateFilter = filter;
                if (filter === 'all') {
                    filterLabel.textContent = '';
                } else if (filter === 'today') {
                    filterLabel.textContent = '(Today)';
                } else if (filter === 'yesterday') {
                    filterLabel.textContent = '(Yesterday)';
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
    } else if (currentDateFilter === 'yesterday') {
        const yesterday = new Date(now);
        yesterday.setDate(yesterday.getDate() - 1);
        const startOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate());
        const endOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate(), 23, 59, 59);
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

export { getDateFilter };

export async function fetchLeadsStats() {
    const dateFilter = getDateFilter();

    document.getElementById('totalLeads').textContent = '...';

    try {
        const url = `/api/leads/stats${dateFilter ? `?filter=${encodeURIComponent(dateFilter)}` : ''}`;
        const response = await fetch(url, {
            headers: {
                'Authorization': pb.authStore.token
            }
        });

        if (response.status === 403) {
            pb.authStore.clear();

            const backdrop = document.createElement('div');
            backdrop.className = 'fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center animate-fade-in';

            const modal = document.createElement('div');
            modal.className = 'bg-white rounded-xl shadow-2xl p-8 max-w-md mx-4 animate-scale-in';
            modal.innerHTML = `
                <div class="text-center">
                    <div class="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-red-100 mb-4">
                        <svg class="h-8 w-8 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
                        </svg>
                    </div>
                    <h3 class="text-xl font-semibold text-gray-900 mb-2">Account Disabled</h3>
                    <p class="text-gray-600 mb-6">Your account has been disabled. Please contact the administrator for assistance.</p>
                    <div class="text-sm text-gray-500">Redirecting to login...</div>
                </div>
            `;

            backdrop.appendChild(modal);
            document.body.appendChild(backdrop);

            setTimeout(() => {
                backdrop.style.animation = 'fade-out 0.3s ease-out';
                setTimeout(() => {
                    window.location.href = '/';
                }, 300);
            }, 3000);

            return;
        }

        if (!response.ok) {
            throw new Error('Failed to fetch stats');
        }

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

        const totalExceptNew = (stats.called || 0) + (stats.cnr || 0) + (stats.denied || 0) +
            (stats.ip_approved || 0) + (stats.ip_decline || 0) + (stats.no_docs || 0) +
            (stats.already_carded || 0) + (stats.not_eligible || 0) + (stats.follow_up || 0);
        document.getElementById('leadsTotal').textContent = totalExceptNew > 0 ? totalExceptNew : '-';

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
        const cnrPct = ((counts.cnr / sumExceptNew) * 100).toFixed(0);
        document.getElementById('leadsCNRPct').textContent = `${cnrPct}%`;
    }

    if (sumExceptNewCNR > 0) {
        const deniedPct = ((counts.denied / sumExceptNewCNR) * 100).toFixed(0);
        document.getElementById('leadsDeniedPct').textContent = `${deniedPct}%`;

        const noDocsPct = ((counts.noDocs / sumExceptNewCNR) * 100).toFixed(0);
        document.getElementById('leadsNoDocsPct').textContent = `${noDocsPct}%`;

        const alreadyCardedPct = ((counts.alreadyCarded / sumExceptNewCNR) * 100).toFixed(0);
        document.getElementById('leadsAlreadyCardedPct').textContent = `${alreadyCardedPct}%`;

        const notEligiblePct = ((counts.notEligible / sumExceptNewCNR) * 100).toFixed(0);
        document.getElementById('leadsNotEligiblePct').textContent = `${notEligiblePct}%`;

        const followUpPct = ((counts.followUp / sumExceptNewCNR) * 100).toFixed(0);
        document.getElementById('leadsFollowUpPct').textContent = `${followUpPct}%`;
    }

    if (ipTotal > 0) {
        const ipApprovedPct = ((counts.ipApproved / ipTotal) * 100).toFixed(0);
        document.getElementById('leadsIPApprovedPct').textContent = `${ipApprovedPct}%`;

        const ipDeclinePct = ((counts.ipDecline / ipTotal) * 100).toFixed(0);
        document.getElementById('leadsIPDeclinePct').textContent = `${ipDeclinePct}%`;
    }
}
