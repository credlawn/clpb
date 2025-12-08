export function createEmployeeStatsCard() {
    return `
        <div class="bg-white rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
                <span class="text-base font-semibold text-gray-800">Employee Performance</span>
                <div class="relative">
                    <button id="employeeFilterBtn" class="p-1.5 hover:bg-gray-100 rounded transition-colors flex items-center gap-1">
                        <i data-feather="filter" class="w-4 h-4 text-gray-500"></i>
                        <span id="employeeFilterLabel" class="text-sm text-gray-400">(Today)</span>
                    </button>
                    
                    <div id="employeeFilterMenu" class="hidden absolute right-0 mt-1 w-40 bg-white rounded-lg shadow-lg border border-gray-200 z-10">
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
            
            <div class="overflow-x-auto">
<table class="min-w-full">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="py-2 px-3 text-left text-sm font-semibold text-gray-600">Employee</th>
                            <th class="py-2 px-2 text-center text-sm font-medium text-gray-400">%</th>
                            <th class="py-2 px-2 text-center text-sm font-semibold text-green-600">IPA</th>
                            <th class="py-2 px-3 text-center text-sm font-semibold text-red-600">IPD</th>
                            <th class="py-2 px-3 text-center text-sm font-semibold text-blue-600">Total</th>
                        </tr>
                    </thead>
                    <tbody id="employeeStatsBody">
                        <tr>
                            <td colspan="5" class="text-center py-3 text-gray-500 text-xs">Loading...</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    `;
}

export function createActiveEmployeesCard() {
    return `
        <div class="bg-white rounded-lg shadow-md p-6">
            <div class="flex items-center justify-between">
                <div>
                    <p class="text-sm font-medium text-gray-600">Active Employees</p>
                    <p class="text-3xl font-bold text-gray-900 mt-2" id="activeEmployees">...</p>
                </div>
                <div class="p-3 bg-blue-100 rounded-full">
                    <svg class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
                    </svg>
                </div>
            </div>
        </div>
    `;
}
