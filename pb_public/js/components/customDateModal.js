export function createCustomDateModal() {
    return `
        <div id="customDateModal" class="hidden fixed inset-0 bg-black bg-opacity-60 backdrop-blur-sm z-[200] flex items-center justify-center">
            <div class="bg-white rounded-2xl shadow-2xl p-6 max-w-sm w-full mx-4" style="animation: scaleIn 0.2s ease-out;">
                <div class="flex items-center justify-between mb-5">
                    <div class="flex items-center gap-3">
                        <div class="w-10 h-10 bg-blue-100 rounded-xl flex items-center justify-center">
                            <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                            </svg>
                        </div>
                        <h3 class="text-lg font-bold text-gray-900">Custom Range</h3>
                    </div>
                    <button id="closeCustomDateModal" class="w-8 h-8 flex items-center justify-center hover:bg-gray-100 rounded-lg transition-colors">
                        <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                        </svg>
                    </button>
                </div>
                
                <div class="space-y-4">
                    <div>
                        <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">From</label>
                        <input type="date" id="customStartDate" class="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-gray-800 font-medium">
                    </div>
                    <div>
                        <label class="block text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">To</label>
                        <input type="date" id="customEndDate" class="w-full px-4 py-3 border-2 border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all text-gray-800 font-medium">
                    </div>
                    <div class="flex gap-3 pt-3">
                        <button id="cancelCustomDate" class="flex-1 px-5 py-3 text-gray-700 bg-gray-100 rounded-xl hover:bg-gray-200 font-semibold transition-colors">Cancel</button>
                        <button id="applyCustomDate" class="flex-1 px-5 py-3 text-white bg-blue-600 rounded-xl hover:bg-blue-700 font-semibold transition-colors shadow-lg shadow-blue-600/30">Apply</button>
                    </div>
                </div>
            </div>
        </div>
        
        <style>
            @keyframes scaleIn {
                from { transform: scale(0.95); opacity: 0; }
                to { transform: scale(1); opacity: 1; }
            }
        </style>
    `;
}
