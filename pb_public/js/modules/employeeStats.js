import pb from '../utils/pb.js';
import { openCustomDateModal } from './customDateModal.js';

let currentEmployeeFilter = 'today';
let employeeCustomStartDate = '';
let employeeCustomEndDate = '';

export function setupEmployeeFilter() {
    const filterBtn = document.getElementById('employeeFilterBtn');
    const filterMenu = document.getElementById('employeeFilterMenu');
    const filterLabel = document.getElementById('employeeFilterLabel');

    if (!filterBtn || !filterMenu) return;

    filterBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        filterMenu.classList.toggle('hidden');
    });

    document.addEventListener('click', (e) => {
        if (!e.target.closest('#employeeFilterBtn') && !e.target.closest('#employeeFilterMenu')) {
            filterMenu.classList.add('hidden');
        }
    });

    filterMenu.querySelectorAll('button').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const filter = e.target.dataset.filter;

            if (filter === 'custom') {
                filterMenu.classList.add('hidden');
                openCustomDateModal((startDate, endDate) => {
                    employeeCustomStartDate = startDate;
                    employeeCustomEndDate = endDate;
                    currentEmployeeFilter = 'custom';

                    const formatDate = (dateStr) => {
                        const [year, month, day] = dateStr.split('-');
                        return `${day}-${month}-${year.slice(2)}`;
                    };

                    filterLabel.textContent = `(${formatDate(startDate)} to ${formatDate(endDate)})`;
                    fetchEmployeeStats();
                });
                return;
            }

            currentEmployeeFilter = filter;

            if (filter === 'all') {
                filterLabel.textContent = '';
            } else if (filter === 'today') {
                filterLabel.textContent = '(Today)';
            } else if (filter === 'yesterday') {
                filterLabel.textContent = '(Yesterday)';
            } else if (filter === 'month') {
                filterLabel.textContent = '(This Month)';
            }

            filterMenu.classList.add('hidden');

            await fetchEmployeeStats();
        });
    });
}

function getEmployeeDateFilter() {
    let filter = '';

    if (currentEmployeeFilter === 'today') {
        const now = new Date();
        const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const endOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
        filter = `lead_status_date >= "${startOfDay.toISOString()}" AND lead_status_date <= "${endOfDay.toISOString()}"`;
    } else if (currentEmployeeFilter === 'yesterday') {
        const now = new Date();
        const yesterday = new Date(now);
        yesterday.setDate(yesterday.getDate() - 1);
        const startOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate());
        const endOfDay = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate(), 23, 59, 59);
        filter = `lead_status_date >= "${startOfDay.toISOString()}" AND lead_status_date <= "${endOfDay.toISOString()}"`;
    } else if (currentEmployeeFilter === 'month') {
        const now = new Date();
        const startOfMonth = new Date(now.getFullYear(), now.getMonth(), 1);
        const endOfMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59);
        filter = `lead_status_date >= "${startOfMonth.toISOString()}" AND lead_status_date <= "${endOfMonth.toISOString()}"`;
    } else if (currentEmployeeFilter === 'custom' && employeeCustomStartDate && employeeCustomEndDate) {
        const start = new Date(employeeCustomStartDate + 'T00:00:00');
        const end = new Date(employeeCustomEndDate + 'T23:59:59');
        filter = `lead_status_date >= "${start.toISOString()}" AND lead_status_date <= "${end.toISOString()}"`;
    }

    return filter;
}

export async function fetchEmployeeStats(dateFilter) {
    if (dateFilter === undefined) {
        dateFilter = getEmployeeDateFilter();
    }
    try {
        const url = `/api/employee/stats${dateFilter ? `?filter=${encodeURIComponent(dateFilter)}` : ''}`;
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
            throw new Error('Failed to fetch employee stats');
        }

        const data = await response.json();
        renderEmployeeTable(data);
    } catch (error) {
        console.error('Error fetching employee stats:', error);
        const tbody = document.getElementById('employeeStatsBody');
        if (tbody) {
            tbody.innerHTML = '<tr><td colspan="3" class="text-center py-4 text-red-500">Error loading data</td></tr>';
        }
    }
}

function renderEmployeeTable(data) {
    const tbody = document.getElementById('employeeStatsBody');

    if (!tbody) return;

    if (data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-center py-4 text-gray-500">No employees found</td></tr>';
        return;
    }

    data.sort((a, b) => b.ipa - a.ipa);

    let totalIPA = 0;
    let totalIPD = 0;
    let grandTotal = 0;

    const rows = data.map(emp => {
        const total = emp.ipa + emp.ipd;
        const percentage = total > 0 ? ((emp.ipa / total) * 100).toFixed(0) : 0;

        totalIPA += emp.ipa;
        totalIPD += emp.ipd;
        grandTotal += total;

        return `
            <tr class="border-b border-gray-100 hover:bg-gray-50 transition-colors">
                <td class="py-2 px-3 text-gray-700 text-sm">${emp.employee_name}</td>
                <td class="py-2 px-2 text-center text-gray-400" style="font-size: 10px;">${percentage > 0 ? percentage + '%' : '-'}</td>
                <td class="py-2 px-2 text-center font-semibold text-green-600 text-sm">${emp.ipa > 0 ? emp.ipa : '-'}</td>
                <td class="py-2 px-3 text-center font-semibold text-red-600 text-sm">${emp.ipd > 0 ? emp.ipd : '-'}</td>
                <td class="py-2 px-3 text-center font-semibold text-blue-600 text-sm">${total > 0 ? total : '-'}</td>
            </tr>
        `;
    }).join('');

    const overallPercentage = grandTotal > 0 ? ((totalIPA / grandTotal) * 100).toFixed(0) : 0;

    const totalRow = `
        <tr class="bg-gray-100 font-bold border-t border-gray-200">
            <td class="py-2 px-3 text-gray-900 text-sm">Total</td>
            <td class="py-2 px-2 text-center text-gray-600" style="font-size: 10px;">${overallPercentage > 0 ? overallPercentage + '%' : '-'}</td>
            <td class="py-2 px-2 text-center text-green-700 text-sm font-bold">${totalIPA > 0 ? totalIPA : '-'}</td>
            <td class="py-2 px-3 text-center text-red-700 text-sm font-bold">${totalIPD > 0 ? totalIPD : '-'}</td>
            <td class="py-2 px-3 text-center text-blue-700 text-sm font-bold">${grandTotal > 0 ? grandTotal : '-'}</td>
        </tr>
    `;

    tbody.innerHTML = rows + totalRow;
}
