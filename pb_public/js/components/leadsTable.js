export function renderLeadsTable(leads) {
    if (!leads || leads.length === 0) {
        return `
            <div class="flex flex-col items-center justify-center py-16 px-4">
                <i data-feather="inbox" class="w-16 h-16 text-gray-300 mb-4"></i>
                <h3 class="text-lg font-medium text-gray-900 mb-1">No leads found</h3>
                <p class="text-sm text-gray-500">Try adjusting your filters or import new leads</p>
            </div>
        `;
    }

    const rows = leads.map(lead => `
        <tr class="hover:bg-gray-50 transition-colors border-b border-gray-100">
            <td class="px-4 py-3 whitespace-nowrap">
                <input type="checkbox" class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500" data-lead-id="${lead.id}">
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
                <div class="font-medium text-gray-900">${lead.customer_name || '-'}</div>
            </td>
            <td class="px-4 py-3 text-sm text-gray-700 whitespace-nowrap">${lead.mobile_no || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-700 whitespace-nowrap">${lead.city || '-'}</td>
            <td class="px-4 py-3 whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(lead.lead_status)}">
                    ${lead.lead_status || 'New'}
                </span>
            </td>
            <td class="px-4 py-3 text-sm text-gray-700 whitespace-nowrap">${lead.agent_name || lead.employee_name || 'Unassigned'}</td>
            <td class="px-4 py-3 text-sm text-gray-500 whitespace-nowrap">${formatDate(lead.created)}</td>
            <td class="px-4 py-3 text-sm text-gray-500 whitespace-nowrap">${formatDate(lead.updated)}</td>
            <td class="px-4 py-3 whitespace-nowrap">
                <button class="p-1 hover:bg-gray-100 rounded transition-colors" data-lead-id="${lead.id}">
                    <i data-feather="more-vertical" class="w-4 h-4 text-gray-600"></i>
                </button>
            </td>
        </tr>
    `).join('');

    return `
        <div class="overflow-x-auto border border-gray-200 rounded-lg shadow-sm">
            <table class="min-w-full divide-y divide-gray-200">
                <thead class="bg-gray-50 sticky top-0">
                    <tr>
                        <th class="px-4 py-3 text-left whitespace-nowrap">
                            <input type="checkbox" id="selectAll" class="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500">
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 whitespace-nowrap" data-sort="customer_name">
                            Customer Name <i data-feather="chevron-down" class="w-3 h-3 inline"></i>
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider whitespace-nowrap">Mobile No</th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider whitespace-nowrap">City</th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 whitespace-nowrap" data-sort="lead_status">
                            Status <i data-feather="chevron-down" class="w-3 h-3 inline"></i>
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 whitespace-nowrap" data-sort="employee_name">
                            Agent <i data-feather="chevron-down" class="w-3 h-3 inline"></i>
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 whitespace-nowrap" data-sort="created">
                            Created <i data-feather="chevron-down" class="w-3 h-3 inline"></i>
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 whitespace-nowrap" data-sort="updated">
                            Updated <i data-feather="chevron-down" class="w-3 h-3 inline"></i>
                        </th>
                        <th class="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider whitespace-nowrap">Actions</th>
                    </tr>
                </thead>
                <tbody class="bg-white divide-y divide-gray-200">
                    ${rows}
                </tbody>
            </table>
        </div>
    `;
}

function getStatusColor(status) {
    const colors = {
        'New': 'bg-blue-100 text-blue-700',
        'Called': 'bg-green-100 text-green-700',
        'CNR': 'bg-yellow-100 text-yellow-700',
        'Voicemail': 'bg-yellow-100 text-yellow-700',
        'Denied': 'bg-red-100 text-red-700',
        'IP Approved': 'bg-green-100 text-green-700',
        'IP Decline': 'bg-red-100 text-red-700',
        'No Docs': 'bg-orange-100 text-orange-700',
        'Already Carded': 'bg-purple-100 text-purple-700',
        'Not Eligible': 'bg-gray-100 text-gray-700',
        'Follow Up': 'bg-indigo-100 text-indigo-700'
    };
    return colors[status] || 'bg-gray-100 text-gray-700';
}

function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const year = date.getFullYear();
    return `${day}-${month}-${year}`;
}
