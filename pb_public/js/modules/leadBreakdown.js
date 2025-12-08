import pb from '../utils/pb.js';

let expandedStatus = null;

export async function fetchLeadBreakdown(status, dateFilter) {
    try {
        let url = `/api/leads/breakdown?status=${encodeURIComponent(status)}`;
        if (dateFilter) {
            url += `&filter=${encodeURIComponent(dateFilter)}`;
        }

        const response = await fetch(url, {
            headers: { 'Authorization': pb.authStore.token }
        });

        if (!response.ok) {
            throw new Error('Failed to fetch breakdown');
        }

        const data = await response.json();

        data.sort((a, b) => a.count - b.count);

        return data;
    } catch (error) {
        console.error('Error fetching breakdown:', error);
        return [];
    }
}

export function renderBreakdown(employees) {
    if (employees.length === 0) {
        return `
            <div class="px-4 py-3 text-sm text-gray-500 text-center bg-gray-50">
                No employees found
            </div>
        `;
    }

    const rows = employees.map(emp => `
        <div class="flex items-center justify-between py-2 px-4 border-b border-gray-100 bg-gray-50">
            <span class="text-sm ${emp.count === 0 ? 'text-gray-400' : 'text-gray-700'}">${emp.employee_name}</span>
            <span class="text-sm font-semibold ${emp.count === 0 ? 'text-gray-400' : 'text-blue-600'}">${emp.count}</span>
        </div>
    `).join('');

    return `
        <div class="breakdown-content border-l-2 border-blue-200">
            ${rows}
        </div>
    `;
}

export function setupLeadBreakdown(getDateFilter) {
    const statusRows = document.querySelectorAll('.status-row');

    statusRows.forEach(row => {
        row.addEventListener('click', async function () {
            const status = this.dataset.status;
            const breakdownContainer = document.getElementById(`breakdown-${status}`);
            const chevron = this.querySelector('.chevron-icon');

            if (expandedStatus === status) {
                breakdownContainer.classList.add('hidden');
                chevron.setAttribute('data-feather', 'chevron-right');
                feather.replace();
                expandedStatus = null;
                this.classList.remove('bg-blue-50');
            } else {
                if (expandedStatus) {
                    const prevContainer = document.getElementById(`breakdown-${expandedStatus}`);
                    const prevRow = document.querySelector(`.status-row[data-status="${expandedStatus}"]`);
                    const prevChevron = prevRow.querySelector('.chevron-icon');
                    prevContainer.classList.add('hidden');
                    prevChevron.setAttribute('data-feather', 'chevron-right');
                    prevRow.classList.remove('bg-blue-50');
                }

                breakdownContainer.innerHTML = '<div class="px-4 py-3 text-sm text-gray-500 text-center">Loading...</div>';
                breakdownContainer.classList.remove('hidden');
                chevron.setAttribute('data-feather', 'chevron-down');
                feather.replace();
                this.classList.add('bg-blue-50');

                const dateFilter = getDateFilter();
                const employees = await fetchLeadBreakdown(status, dateFilter);
                breakdownContainer.innerHTML = renderBreakdown(employees);

                expandedStatus = status;
            }
        });
    });
}
