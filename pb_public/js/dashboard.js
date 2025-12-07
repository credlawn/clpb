import { checkAuth, displayUserInfo, setupLogout } from './utils/auth.js';
import { setupSidebarToggle } from './utils/ui.js';
import { setupLeadsCard, fetchLeadsStats } from './modules/leads.js';
import { setupImportModal } from './modules/import.js';
import { renderSidebar } from './components/sidebar.js';
import { renderLeadsCard } from './components/leadsCard.js';
import { renderImportModal } from './components/importModal.js';
import { createEmployeeStatsCard } from './components/employeeComponents.js';
import { fetchEmployeeStats, setupEmployeeFilter } from './modules/employeeStats.js';
import { createCustomDateModal } from './components/customDateModal.js';
import { setupCustomDateModal } from './modules/customDateModal.js';

let refreshInterval = null;
const refreshFunctions = [];

export function registerAutoRefresh(fn) {
    if (typeof fn === 'function' && !refreshFunctions.includes(fn)) {
        refreshFunctions.push(fn);
    }
}

export function unregisterAutoRefresh(fn) {
    const index = refreshFunctions.indexOf(fn);
    if (index > -1) {
        refreshFunctions.splice(index, 1);
    }
}

function refreshAllStats() {
    refreshFunctions.forEach(fn => {
        try {
            fn();
        } catch (error) {
            console.error('Error in auto-refresh function:', error);
        }
    });
}

function startAutoRefresh() {
    if (!document.hidden && !refreshInterval) {
        refreshInterval = setInterval(() => {
            refreshAllStats();
        }, 30000);
    }
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        stopAutoRefresh();
    } else {
        refreshAllStats();
        startAutoRefresh();
    }
});

if (checkAuth()) {
    document.getElementById('sidebarContainer').innerHTML = renderSidebar();
    document.getElementById('importModalContainer').innerHTML = renderImportModal();
    document.body.insertAdjacentHTML('beforeend', createCustomDateModal());

    async function setupUI() {
        const statsGrid = document.getElementById('statsGrid');

        statsGrid.innerHTML = renderLeadsCard() + createEmployeeStatsCard();

        displayUserInfo();
        setupSidebarToggle();
        setupLogout();
        setupImportModal();

        feather.replace();

        setupLeadsCard();
        setupEmployeeFilter();
        setupCustomDateModal();

        registerAutoRefresh(fetchLeadsStats);
        registerAutoRefresh(fetchEmployeeStats);

        fetchLeadsStats();
        fetchEmployeeStats();
        startAutoRefresh();
    };

    if ('requestIdleCallback' in window) {
        requestIdleCallback(setupUI);
    } else {
        setTimeout(setupUI, 0);
    }
}
