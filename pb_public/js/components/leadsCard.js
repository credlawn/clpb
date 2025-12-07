export function renderLeadsCard() {
    return `
        <div class="bg-white rounded-lg shadow-md p-4 col-span-1 md:col-span-1 lg:col-span-1">
            <div class="flex items-center justify-between mb-3">
                <div class="flex items-center space-x-2">
                    <h3 class="text-base font-semibold text-gray-900" id="totalLeads">0</h3>
                    <span class="text-sm text-gray-600">Leads</span>
                    <span class="text-xs text-gray-600" id="leadsFilterLabel">(Today)</span>
                </div>
                <div class="relative">
                    <button id="leadsFilterBtn" class="p-1.5 hover:bg-gray-100 rounded-lg transition-colors flex items-center gap-1.5">
                        <i data-feather="filter" class="w-4 h-4 text-gray-600"></i>
                    </button>
                    <div id="leadsFilterMenu" class="hidden absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 z-10">
                        <div class="py-1">
                            <button data-filter="all" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">All Time</button>
                            <button data-filter="today" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">Today</button>
                            <button data-filter="yesterday" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">Yesterday</button>
                            <button data-filter="month" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">This Month</button>
                            <button data-filter="custom" class="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100">Custom Range</button>
                        </div>
                    </div>
                </div>
            </div>

            <div class="space-y-0">
                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">New</span>
                    <span class="text-sm font-semibold text-blue-600" id="leadsNew">0</span>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">Called</span>
                    <span class="text-sm font-semibold text-green-600" id="leadsCalled">0</span>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">CNR</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsCNRPct"></span>
                        <span class="text-sm font-semibold text-yellow-600" id="leadsCNR">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">Denied</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsDeniedPct"></span>
                        <span class="text-sm font-semibold text-red-600" id="leadsDenied">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm font-semibold text-green-700">IP Approved</span>
                    <div class="flex items-center gap-2">
                        <span class="text-green-600" style="font-size: 10px;" id="leadsIPApprovedPct"></span>
                        <span class="text-sm font-semibold text-green-700" id="leadsIPApproved">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm font-semibold text-red-700">IP Decline</span>
                    <div class="flex items-center gap-2">
                        <span class="text-red-600" style="font-size: 10px;" id="leadsIPDeclinePct"></span>
                        <span class="text-sm font-semibold text-red-700" id="leadsIPDecline">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">No Docs</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsNoDocsPct"></span>
                        <span class="text-sm font-semibold text-orange-600" id="leadsNoDocs">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">Already Carded</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsAlreadyCardedPct"></span>
                        <span class="text-sm font-semibold text-purple-600" id="leadsAlreadyCarded">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">Not Eligible</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsNotEligiblePct"></span>
                        <span class="text-sm font-semibold text-gray-600" id="leadsNotEligible">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 border-b border-gray-200">
                    <span class="text-sm text-gray-700">Follow Up</span>
                    <div class="flex items-center gap-2">
                        <span class="text-gray-400" style="font-size: 10px;" id="leadsFollowUpPct"></span>
                        <span class="text-sm font-semibold text-indigo-600" id="leadsFollowUp">0</span>
                    </div>
                </div>

                <div class="flex items-center justify-between py-2 px-3 bg-gray-100 font-bold border-t-2 border-gray-300">
                    <span class="text-sm text-gray-900">Total</span>
                    <span class="text-sm text-blue-700" id="leadsTotal">0</span>
                </div>
            </div>
        </div>
    `;
}
