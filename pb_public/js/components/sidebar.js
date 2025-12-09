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
