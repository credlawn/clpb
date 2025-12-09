import { checkAuth } from './utils/auth.js';
import { renderSidebar, setupSyncButton } from './components/sidebar.js';
import { renderLeadsTable } from './components/leadsTable.js';
import { renderLeadsFilters } from './components/leadsFilters.js';
import { fetchLeads, fetchAgents } from './modules/leadsData.js';

let currentFilters = {
    page: 1,
    limit: 50,
    search: '',
    status: '',
    agent: '',
    dateFrom: '',
    dateTo: '',
    sortBy: 'created',
    sortOrder: 'desc'
};

let leadsData = null;

async function init() {
    if (!await checkAuth()) return;

    document.getElementById('sidebarContainer').innerHTML = renderSidebar();
    setupSyncButton();
    document.getElementById('filtersContainer').innerHTML = renderLeadsFilters();

    setupEventListeners();
    await loadAgents();
    await loadLeads();

    feather.replace();
}

async function loadLeads() {
    try {
        const tableContainer = document.getElementById('tableContainer');
        tableContainer.innerHTML = '<div class="flex items-center justify-center py-16"><div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div></div>';

        leadsData = await fetchLeads(currentFilters);

        tableContainer.innerHTML = renderLeadsTable(leadsData.items);
        updateLeadsCount();
        renderPagination();

        feather.replace();
        setupTableEventListeners();
    } catch (error) {
        console.error('Error loading leads:', error);
        document.getElementById('tableContainer').innerHTML = `
            <div class="flex flex-col items-center justify-center py-16 px-4">
                <i data-feather="alert-circle" class="w-16 h-16 text-red-300 mb-4"></i>
                <h3 class="text-lg font-medium text-gray-900 mb-1">Error loading leads</h3>
                <p class="text-sm text-gray-500">${error.message}</p>
            </div>
        `;
        feather.replace();
    }
}

async function loadAgents() {
    try {
        const agents = await fetchAgents();
        const agentFilter = document.getElementById('agentFilter');

        agents.forEach(agent => {
            const option = document.createElement('option');
            option.value = agent.id;
            option.textContent = agent.name;
            agentFilter.appendChild(option);
        });
    } catch (error) {
        console.error('Error loading agents:', error);
    }
}

function setupEventListeners() {
    const sidebarToggle = document.getElementById('sidebarToggle');
    const sidebar = document.getElementById('sidebar');
    const sidebarOverlay = document.getElementById('sidebarOverlay');

    sidebarToggle.addEventListener('click', () => {
        sidebar.classList.toggle('-translate-x-full');
        sidebarOverlay.classList.toggle('hidden');
    });

    sidebarOverlay.addEventListener('click', () => {
        sidebar.classList.add('-translate-x-full');
        sidebarOverlay.classList.add('hidden');
    });

    let searchTimeout;
    document.getElementById('searchInput').addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            currentFilters.search = e.target.value;
            currentFilters.page = 1;
            loadLeads();
        }, 500);
    });

    document.getElementById('statusFilter').addEventListener('change', (e) => {
        currentFilters.status = e.target.value;
        currentFilters.page = 1;
        loadLeads();
    });

    document.getElementById('agentFilter').addEventListener('change', (e) => {
        currentFilters.agent = e.target.value;
        currentFilters.page = 1;
        loadLeads();
    });

    document.getElementById('clearFilters').addEventListener('click', () => {
        currentFilters = {
            page: 1,
            limit: 50,
            search: '',
            status: '',
            agent: '',
            dateFrom: '',
            dateTo: '',
            sortBy: 'created',
            sortOrder: 'desc'
        };

        document.getElementById('searchInput').value = '';
        document.getElementById('statusFilter').value = '';
        document.getElementById('agentFilter').value = '';
        document.getElementById('dateFilterLabel').textContent = 'All Time';

        loadLeads();
    });

    document.getElementById('exportBtn').addEventListener('click', async () => {
        alert('Export functionality coming soon!');
    });

    document.getElementById('importBtn').addEventListener('click', () => {
        window.location.href = '/dashboard.html';
    });
}

function setupTableEventListeners() {
    document.querySelectorAll('th[data-sort]').forEach(th => {
        th.addEventListener('click', () => {
            const sortBy = th.dataset.sort;

            if (currentFilters.sortBy === sortBy) {
                currentFilters.sortOrder = currentFilters.sortOrder === 'asc' ? 'desc' : 'asc';
            } else {
                currentFilters.sortBy = sortBy;
                currentFilters.sortOrder = 'asc';
            }

            loadLeads();
        });
    });

    const selectAll = document.getElementById('selectAll');
    if (selectAll) {
        selectAll.addEventListener('change', (e) => {
            document.querySelectorAll('input[data-lead-id]').forEach(checkbox => {
                checkbox.checked = e.target.checked;
            });
        });
    }
}

function updateLeadsCount() {
    const count = document.getElementById('leadsCount');
    if (leadsData) {
        count.textContent = `${leadsData.totalItems} total leads`;
    }
}

function renderPagination() {
    const container = document.getElementById('paginationContainer');

    if (!leadsData || leadsData.totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    const { currentPage, totalPages, totalItems, perPage } = leadsData;
    const startItem = (currentPage - 1) * perPage + 1;
    const endItem = Math.min(currentPage * perPage, totalItems);

    let pages = [];

    if (totalPages <= 7) {
        for (let i = 1; i <= totalPages; i++) {
            pages.push(i);
        }
    } else {
        if (currentPage <= 4) {
            pages = [1, 2, 3, 4, 5, '...', totalPages];
        } else if (currentPage >= totalPages - 3) {
            pages = [1, '...', totalPages - 4, totalPages - 3, totalPages - 2, totalPages - 1, totalPages];
        } else {
            pages = [1, '...', currentPage - 1, currentPage, currentPage + 1, '...', totalPages];
        }
    }

    const pageButtons = pages.map(page => {
        if (page === '...') {
            return '<span class="px-3 py-1 text-gray-500">...</span>';
        }

        const isActive = page === currentPage;
        return `
            <button 
                class="px-3 py-1 rounded ${isActive ? 'bg-blue-600 text-white' : 'text-gray-700 hover:bg-gray-100'} transition-colors"
                ${isActive ? 'disabled' : ''}
                onclick="window.goToPage(${page})"
            >
                ${page}
            </button>
        `;
    }).join('');

    container.innerHTML = `
        <div class="flex items-center justify-between">
            <div class="text-sm text-gray-700">
                Showing <span class="font-medium">${startItem}</span> to <span class="font-medium">${endItem}</span> of <span class="font-medium">${totalItems}</span> results
            </div>
            <div class="flex items-center space-x-1">
                <button 
                    class="px-3 py-1 rounded text-gray-700 hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    ${currentPage === 1 ? 'disabled' : ''}
                    onclick="window.goToPage(${currentPage - 1})"
                >
                    Previous
                </button>
                ${pageButtons}
                <button 
                    class="px-3 py-1 rounded text-gray-700 hover:bg-gray-100 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    ${currentPage === totalPages ? 'disabled' : ''}
                    onclick="window.goToPage(${currentPage + 1})"
                >
                    Next
                </button>
            </div>
        </div>
    `;
}

window.goToPage = function (page) {
    currentFilters.page = page;
    loadLeads();
    window.scrollTo({ top: 0, behavior: 'smooth' });
};

document.addEventListener('DOMContentLoaded', init);
