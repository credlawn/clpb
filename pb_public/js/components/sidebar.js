export function renderSidebar() {
    return `
        <aside id="sidebar"
            class="fixed w-64 h-full bg-white border-r border-gray-200 flex flex-col transition-transform duration-300 transform -translate-x-full z-50">
            <div class="p-3 border-b border-gray-200 flex items-center justify-between">
                <div class="flex items-center space-x-2">
                    <div class="w-8 h-8 bg-gradient-to-br from-blue-600 to-blue-700 rounded-lg flex items-center justify-center">
                        <i data-feather="layers" class="w-5 h-5 text-white"></i>
                    </div>
                    <span class="text-base font-bold text-gray-900 sidebar-text">CLPB</span>
                </div>
            </div>

            <nav class="flex-1 p-3 space-y-1 overflow-y-auto">
                <a href="/dashboard.html" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                    <i data-feather="home" class="w-4 h-4 flex-shrink-0"></i>
                    <span class="sidebar-text">Dashboard</span>
                </a>

                <a href="#" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                    <i data-feather="users" class="w-4 h-4 flex-shrink-0"></i>
                    <span class="sidebar-text">Employees</span>
                </a>

                <a href="/leads.html" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                <i data-feather="list" class="w-4 h-4 flex-shrink-0"></i>
                <span class="sidebar-text">Leads</span>
            </a>

            <a href="/allocate.html" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                <i data-feather="target" class="w-4 h-4 flex-shrink-0"></i>
                <span class="sidebar-text">Allocate Leads</span>
            </a>

            <a href="#" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                <i data-feather="upload" class="w-4 h-4 flex-shrink-0"></i>
                <span class="sidebar-text">Import Data</span>
            </a>

                <a href="#" class="flex items-center space-x-2 px-3 py-2 text-gray-700 hover:bg-gray-100 rounded-lg text-sm font-medium">
                    <i data-feather="file-text" class="w-4 h-4 flex-shrink-0"></i>
                    <span class="sidebar-text">Reports</span>
                </a>

                <button id="syncDataBtn" class="w-full flex items-center space-x-2 px-3 py-2 text-orange-600 hover:bg-orange-50 rounded-lg text-sm font-medium">
                    <i data-feather="refresh-cw" class="w-4 h-4 flex-shrink-0"></i>
                    <span class="sidebar-text">Sync Data</span>
                </button>
            </nav>

            <div class="p-3 border-t border-gray-200">
                <div class="flex items-center space-x-2 mb-2">
                    <div class="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center flex-shrink-0">
                        <i data-feather="user" class="w-4 h-4 text-blue-600"></i>
                    </div>
                    <div class="flex-1 min-w-0 sidebar-text">
                        <p class="text-xs font-medium text-gray-900 truncate" id="sidebarUserName">Manager</p>
                        <p class="text-xs text-gray-500">Manager</p>
                    </div>
                </div>
                <button id="logoutButton"
                    class="w-full flex items-center justify-center space-x-2 px-3 py-1.5 bg-red-50 text-red-600 rounded-lg hover:bg-red-100 transition-colors text-xs font-medium">
                    <i data-feather="log-out" class="w-3.5 h-3.5"></i>
                    <span class="sidebar-text">Logout</span>
                </button>
            </div>
        </aside>
    `;
}

export function setupSyncButton() {
    const syncBtn = document.getElementById('syncDataBtn');
    if (!syncBtn) return;

    // Add modal HTML to body if not exists
    if (!document.getElementById('syncModal')) {
        const modalHTML = `
            <div id="syncModal" class="fixed inset-0 bg-black bg-opacity-50 z-50 hidden flex items-center justify-center">
                <div class="bg-white rounded-lg shadow-lg w-full max-w-md mx-4 overflow-hidden">
                    <div class="p-4 border-b border-gray-100 flex justify-between items-center bg-gray-50">
                        <h3 class="font-semibold text-gray-800">Sync Data</h3>
                        <button id="closeSyncModal" class="text-gray-400 hover:text-gray-600">
                            <i data-feather="x" class="w-4 h-4"></i>
                        </button>
                    </div>
                    
                    <div id="syncInitial" class="p-6">
                        <div class="mb-6 flex justify-center">
                            <div class="w-16 h-16 bg-orange-100 rounded-full flex items-center justify-center">
                                <i data-feather="refresh-cw" class="w-8 h-8 text-orange-600"></i>
                            </div>
                        </div>
                        <h4 class="text-center text-lg font-medium text-gray-900 mb-2">Sync All Data?</h4>
                        <p class="text-center text-sm text-gray-500 mb-6">This action will sync existing leads to the master database, create history records, and update call statistics.</p>
                        
                        <div class="space-y-2 text-xs text-gray-500 bg-gray-50 p-3 rounded border border-gray-100 mb-6">
                            <div class="flex items-center gap-2"><i data-feather="check-circle" class="w-3 h-3 text-green-500"></i> Sync leads to database</div>
                            <div class="flex items-center gap-2"><i data-feather="check-circle" class="w-3 h-3 text-green-500"></i> Create allocation history</div>
                            <div class="flex items-center gap-2"><i data-feather="check-circle" class="w-3 h-3 text-green-500"></i> Sync call stats</div>
                        </div>

                        <div class="flex gap-3">
                            <button id="cancelSync" class="flex-1 px-4 py-2 border border-gray-300 rounded text-sm font-medium text-gray-700 hover:bg-gray-50">Cancel</button>
                            <button id="confirmSync" class="flex-1 px-4 py-2 bg-orange-600 text-white rounded text-sm font-medium hover:bg-orange-700">Start Sync</button>
                        </div>
                    </div>

                    <div id="syncProgress" class="p-6 hidden">
                        <div class="flex flex-col items-center justify-center py-8">
                            <div class="w-12 h-12 border-4 border-orange-200 border-t-orange-600 rounded-full animate-spin mb-4"></div>
                            <p class="text-gray-900 font-medium">Syncing Data...</p>
                            <p class="text-sm text-gray-500 mt-1">Please wait, do not close this window.</p>
                        </div>
                    </div>

                    <div id="syncResults" class="p-6 hidden">
                        <div class="mb-4 text-center">
                            <div class="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-3">
                                <i data-feather="check" class="w-6 h-6 text-green-600"></i>
                            </div>
                            <h4 class="text-lg font-medium text-gray-900">Sync Complete!</h4>
                        </div>
                        
                        <div class="bg-gray-50 rounded border border-gray-200 p-4 mb-6">
                            <h5 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Leads Database</h5>
                            <div class="grid grid-cols-2 gap-y-2 text-sm mb-4">
                                <div class="text-gray-600">Created:</div>
                                <div class="font-medium text-right" id="syncCreated">0</div>
                                <div class="text-gray-600">Updated:</div>
                                <div class="font-medium text-right" id="syncUpdated">0</div>
                                <div class="text-gray-600">History Records:</div>
                                <div class="font-medium text-right" id="syncHistory">0</div>
                            </div>
                            
                            <div class="border-t border-gray-200 my-2"></div>
                            
                            <h5 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2 mt-2">Call Stats</h5>
                            <div class="grid grid-cols-2 gap-y-2 text-sm">
                                <div class="text-gray-600">Records Updated:</div>
                                <div class="font-medium text-right" id="syncCalls">0</div>
                            </div>
                        </div>

                        <button id="closeSyncResult" class="w-full px-4 py-2 bg-gray-900 text-white rounded text-sm font-medium hover:bg-gray-800">Close</button>
                    </div>
                </div>
            </div>
        `;
        document.body.insertAdjacentHTML('beforeend', modalHTML);
    }

    const modal = document.getElementById('syncModal');
    const initialView = document.getElementById('syncInitial');
    const progressView = document.getElementById('syncProgress');
    const resultsView = document.getElementById('syncResults');

    function openModal() {
        modal.classList.remove('hidden');
        initialView.classList.remove('hidden');
        progressView.classList.add('hidden');
        resultsView.classList.add('hidden');
        feather.replace();
    }

    function closeModal() {
        modal.classList.add('hidden');
    }

    syncBtn.addEventListener('click', openModal);
    document.getElementById('closeSyncModal').addEventListener('click', closeModal);
    document.getElementById('cancelSync').addEventListener('click', closeModal);
    document.getElementById('closeSyncResult').addEventListener('click', closeModal);

    document.getElementById('confirmSync').addEventListener('click', async () => {
        initialView.classList.add('hidden');
        progressView.classList.remove('hidden');

        try {
            const token = localStorage.getItem('pocketbase_auth') ? JSON.parse(localStorage.getItem('pocketbase_auth')).token : '';

            const res1 = await fetch('/api/sync-leads-to-database', {
                method: 'POST',
                headers: { 'Authorization': token }
            });
            const data1 = await res1.json();

            const res2 = await fetch('/api/sync-call-stats', {
                method: 'POST',
                headers: { 'Authorization': token }
            });
            const data2 = await res2.json();

            document.getElementById('syncCreated').textContent = data1.database_created || 0;
            document.getElementById('syncUpdated').textContent = data1.database_updated || 0;
            document.getElementById('syncHistory').textContent = data1.history_records_created || 0;
            document.getElementById('syncCalls').textContent = data2.database_updated || 0;

            progressView.classList.add('hidden');
            resultsView.classList.remove('hidden');
            feather.replace();

        } catch (error) {
            alert('Sync failed: ' + error.message);
            closeModal();
        }
    });

    feather.replace();
}
