export function renderLeadsCard() {
    return `
        <div class="bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow">
            <div class="p-4 border-b border-gray-100">
                <div class="flex items-center justify-between">
                    <div class="flex items-center space-x-2">
                        <h3 class="text-xl font-bold text-gray-900" id="totalLeads">0</h3>
                        <span class="text-sm text-gray-500">Leads</span>
                        <span class="text-xs text-gray-400" id="leadsFilterLabel">(Today)</span>
                    </div>
                    <div class="relative">
                        <button id="leadsFilterBtn" class="p-1.5 hover:bg-gray-100 rounded-lg transition-colors">
                            <i data-feather="filter" class="w-4 h-4 text-gray-600"></i>
                        </button>
                        <div id="leadsFilterMenu" class="hidden absolute left-1/2 transform -translate-x-1/2 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 z-10">
                            <div class="py-1">
                                <button class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" data-filter="all">All Time</button>
                                <button class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" data-filter="today">Today</button>
                                <button class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" data-filter="month">This Month</button>
                                <button class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100" data-filter="custom">Custom Date</button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div id="leadsBreakdown" class="bg-gray-50">
                <div class="p-4 space-y-2">
                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-blue-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-blue-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">New</span>
                        </div>
                        <span class="text-sm font-bold text-gray-900" id="leadsNew">0</span>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-green-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-green-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">Called</span>
                        </div>
                        <span class="text-sm font-bold text-gray-900" id="leadsCalled">0</span>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-yellow-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-yellow-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">CNR</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsCNRPct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsCNR">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-red-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-red-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">Denied</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsDeniedPct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsDenied">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-green-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-green-600 rounded-full"></div>
                            <span class="text-sm font-medium text-green-600">IP Approved</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-green-600" id="leadsIPApprovedPct"></span>
                            <span class="text-sm font-bold text-green-600" id="leadsIPApproved">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-red-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-red-600 rounded-full"></div>
                            <span class="text-sm font-medium text-red-600">IP Decline</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-red-600" id="leadsIPDeclinePct"></span>
                            <span class="text-sm font-bold text-red-600" id="leadsIPDecline">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-orange-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-red-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">No Docs</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsNoDocsPct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsNoDocs">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-purple-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-purple-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">Already Carded</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsAlreadyCardedPct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsAlreadyCarded">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-gray-100 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-gray-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">Not Eligible</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsNotEligiblePct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsNotEligible">0</span>
                        </div>
                    </div>

                    <div class="flex items-center justify-between p-2 bg-white rounded-lg hover:bg-indigo-50 transition-colors">
                        <div class="flex items-center space-x-2">
                            <div class="w-2 h-2 bg-indigo-500 rounded-full"></div>
                            <span class="text-sm font-medium text-gray-700">Follow Up</span>
                        </div>
                        <div class="flex items-center space-x-3">
                            <span class="text-xs text-gray-500" id="leadsFollowUpPct"></span>
                            <span class="text-sm font-bold text-gray-900" id="leadsFollowUp">0</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
}
