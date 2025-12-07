import { checkAuth, displayUserInfo, setupLogout } from './utils/auth.js';
import { setupSidebarToggle } from './utils/ui.js';
import { setupLeadsCard, fetchLeadsStats } from './modules/leads.js';
import { fetchOtherStats } from './modules/stats.js';
import { setupImportModal } from './modules/import.js';
import { renderSidebar } from './components/sidebar.js';
import { renderLeadsCard } from './components/leadsCard.js';
import { renderStatsCards } from './components/statsCards.js';
import { renderImportModal } from './components/importModal.js';

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
    document.getElementById('statsGrid').innerHTML = renderLeadsCard() + renderStatsCards();
    document.getElementById('importModalContainer').innerHTML = renderImportModal();

    const setupUI = () => {
        displayUserInfo();
        setupSidebarToggle();
        setupLogout();
        setupLeadsCard();
        setupImportModal();
        feather.replace();

        registerAutoRefresh(fetchLeadsStats);

        fetchLeadsStats();
        fetchOtherStats();
        startAutoRefresh();
    };

    if ('requestIdleCallback' in window) {
        requestIdleCallback(setupUI);
    } else {
        setTimeout(setupUI, 0);
    }
}
