export function renderStatsCards() {
    return `
        <div class="bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow p-4">
            <div class="flex items-start justify-between">
                <div>
                    <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Active Employees</p>
                    <h3 class="text-2xl font-bold text-gray-900 mt-1" id="activeEmployees">0</h3>
                </div>
                <div class="w-10 h-10 bg-gradient-to-br from-green-500 to-green-600 rounded-lg flex items-center justify-center">
                    <i data-feather="users" class="w-5 h-5 text-white"></i>
                </div>
            </div>
        </div>

        <div class="bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow p-4">
            <div class="flex items-start justify-between">
                <div>
                    <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Today Attendance</p>
                    <h3 class="text-2xl font-bold text-gray-900 mt-1" id="todayAttendance">0</h3>
                </div>
                <div class="w-10 h-10 bg-gradient-to-br from-purple-500 to-purple-600 rounded-lg flex items-center justify-center">
                    <i data-feather="check-circle" class="w-5 h-5 text-white"></i>
                </div>
            </div>
        </div>

        <div class="bg-white rounded-lg border border-gray-200 shadow-sm hover:shadow-md transition-shadow p-4">
            <div class="flex items-start justify-between">
                <div>
                    <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Today Calls</p>
                    <h3 class="text-2xl font-bold text-gray-900 mt-1" id="todayCalls">0</h3>
                </div>
                <div class="w-10 h-10 bg-gradient-to-br from-orange-500 to-orange-600 rounded-lg flex items-center justify-center">
                    <i data-feather="phone" class="w-5 h-5 text-white"></i>
                </div>
            </div>
        </div>
    `;
}
