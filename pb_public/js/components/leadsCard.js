export function renderLeadsCard() {
    const statuses = [
        { id: 'New', label: 'New', color: 'blue' },
        { id: 'Called', label: 'Called', color: 'green' },
        { id: 'CNR', label: 'CNR', color: 'yellow', hasPercent: true },
        { id: 'Denied', label: 'Denied', color: 'red', hasPercent: true },
        { id: 'IP Approved', label: 'IP Approved', color: 'green', hasPercent: true, bgClass: 'bg-green-50' },
        { id: 'IP Decline', label: 'IP Decline', color: 'red', hasPercent: true, bgClass: 'bg-red-50' },
        { id: 'No Docs', label: 'No Docs', color: 'orange', hasPercent: true },
        { id: 'Already Carded', label: 'Already Carded', color: 'purple', hasPercent: true },
        { id: 'Not Eligible', label: 'Not Eligible', color: 'gray', hasPercent: true },
        { id: 'Follow Up', label: 'Follow Up', color: 'indigo', hasPercent: true }
    ];

    const statusRows = statuses.map(status => {
        const elementId = `leads${status.id.replace(/ /g, '')}`;
        const pctId = `${elementId}Pct`;
        const colorClass = status.color === 'green' && status.id === 'IP Approved' ? 'green-700' :
            status.color === 'red' && status.id === 'IP Decline' ? 'red-700' :
                `${status.color}-600`;
        const fontWeight = status.id.includes('IP') ? 'font-bold' : 'font-semibold';
        const textClass = status.id.includes('IP') ? 'font-medium' : '';

        return `
                <div class="status-row flex items-center justify-between py-2.5 px-3 border-b border-gray-100 cursor-pointer hover:bg-gray-50 transition-colors ${status.bgClass || ''}" data-status="${status.id}">
                    <span class="text-sm ${textClass} ${status.id.includes('IP') ? `text-${status.color}-700` : 'text-gray-600'}">${status.label}</span>
                    <div class="flex items-center gap-2">
                        ${status.hasPercent ? `<span class="text-gray-400" style="font-size: 10px;" id="${pctId}"></span>` : ''}
                        <span class="text-sm ${fontWeight} text-${colorClass}" id="${elementId}">0</span>
                        <i data-feather="chevron-right" class="w-4 h-4 text-gray-400 chevron-icon"></i>
                    </div>
                </div>
                <div class="breakdown-container hidden" id="breakdown-${status.id}"></div>`;
    }).join('');

    return `
        <div class="bg-white rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
                <div class="flex items-center gap-2">
                    <span class="text-base font-semibold text-gray-800" id="totalLeads">0</span>
                    <span class="text-sm text-gray-500">Leads</span>
                    <span class="text-sm text-gray-400" id="leadsFilterLabel">(Today)</span>
                </div>
                <div class="relative">
                    <button id="leadsFilterBtn" class="p-1.5 hover:bg-gray-100 rounded transition-colors flex items-center gap-1">
                        <i data-feather="filter" class="w-4 h-4 text-gray-500"></i>
                    </button>
                    <div id="leadsFilterMenu" class="hidden absolute right-0 mt-1 w-40 bg-white rounded-lg shadow-lg border border-gray-200 z-10">
                        <div class="py-0.5">
                            <button data-filter="all" class="w-full text-left px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100">All Time</button>
                            <button data-filter="today" class="w-full text-left px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100">Today</button>
                            <button data-filter="yesterday" class="w-full text-left px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100">Yesterday</button>
                            <button data-filter="month" class="w-full text-left px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100">This Month</button>
                            <button data-filter="custom" class="w-full text-left px-3 py-1.5 text-xs text-gray-700 hover:bg-gray-100">Custom Range</button>
                        </div>
                    </div>
                </div>
            </div>

            <div class="space-y-0" id="leadsStatusList">
                ${statusRows}
                
                <div class="flex items-center justify-between py-2.5 px-3 bg-gray-100 font-bold border-t border-gray-200">
                    <span class="text-sm text-gray-900">Total</span>
                    <span class="text-sm font-bold text-blue-700" id="leadsTotal">0</span>
                </div>
            </div>
        </div>
    `;
}
