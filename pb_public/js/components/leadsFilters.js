export function renderLeadsFilters() {
    return `
        <div class="flex flex-wrap items-center gap-3">
            <div class="flex-1 min-w-[200px]">
                <div class="relative">
                    <i data-feather="search" class="w-4 h-4 text-gray-400 absolute left-3 top-1/2 transform -translate-y-1/2"></i>
                    <input 
                        type="text" 
                        id="searchInput" 
                        placeholder="Search by name, phone, email..." 
                        class="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
                    >
                </div>
            </div>
            
            <select id="statusFilter" class="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm">
                <option value="">All Status</option>
                <option value="New">New</option>
                <option value="Called">Called</option>
                <option value="CNR">CNR</option>
                <option value="Voicemail">Voicemail</option>
                <option value="Denied">Denied</option>
                <option value="IP Approved">IP Approved</option>
                <option value="IP Decline">IP Decline</option>
                <option value="No Docs">No Docs</option>
                <option value="Already Carded">Already Carded</option>
                <option value="Not Eligible">Not Eligible</option>
                <option value="Follow Up">Follow Up</option>
            </select>

            <button id="dateFilterBtn" class="px-3 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors text-sm flex items-center space-x-2">
                <i data-feather="calendar" class="w-4 h-4"></i>
                <span id="dateFilterLabel">All Time</span>
            </button>

            <select id="agentFilter" class="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm">
                <option value="">All Agents</option>
            </select>

            <button id="clearFilters" class="px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors">
                Clear Filters
            </button>
        </div>
    `;
}
