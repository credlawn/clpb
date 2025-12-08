export function renderLeadsCard() {
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

            <div class="space-y-0">
                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">New</span>
                    <span class="text-sm font-semibold text-blue-600" id="leadsNew">0</span>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">Called</span>
                    <span class="text-sm font-semibold text-green-600" id="leadsCalled">0</span>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">CNR</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsCNRPct"></span>
                        <span class="text-sm font-semibold text-yellow-600" id="leadsCNR">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">Denied</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsDeniedPct"></span>
                        <span class="text-sm font-semibold text-red-600" id="leadsDenied">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100 bg-green-50">
                    <span class="text-sm font-medium text-green-700">IP Approved</span>
                    <div class="flex items-center gap-2">
                        <span class="text-green-500" style="font-size: 10px;" id="leadsIPApprovedPct"></span>
                        <span class="text-sm font-bold text-green-700" id="leadsIPApproved">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100 bg-red-50">
                    <span class="text-sm font-medium text-red-700">IP Decline</span>
                    <div class="flex items-center gap-2">
                        <span class="text-red-500" style="font-size: 10px;" id="leadsIPDeclinePct"></span>
                        <span class="text-sm font-bold text-red-700" id="leadsIPDecline">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">No Docs</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsNoDocsPct"></span>
                        <span class="text-sm font-semibold text-orange-600" id="leadsNoDocs">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">Already Carded</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsAlreadyCardedPct"></span>
                        <span class="text-sm font-semibold text-purple-600" id="leadsAlreadyCarded">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">Not Eligible</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsNotEligiblePct"></span>
                        <span class="text-sm font-semibold text-gray-600" id="leadsNotEligible">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 border-b border-gray-100">
                    <span class="text-sm text-gray-600">Follow Up</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsFollowUpPct"></span>
                        <span class="text-sm font-semibold text-indigo-600" id="leadsFollowUp">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2.5 px-3 bg-gray-100 font-bold border-t border-gray-200">
                    <span class="text-sm text-gray-900">Total</span>
                    <span class="text-sm font-bold text-blue-700" id="leadsTotal">0</span>
                </div>
            </div>
        </div>
    `;
}
