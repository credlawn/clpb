import { checkAuth, displayUserInfo, setupLogout } from './utils/auth.js';
import { setupSidebarToggle } from './utils/ui.js';
import { setupImportModal } from './modules/import.js';
import { renderSidebar } from './components/sidebar.js';
import { renderImportModal } from './components/importModal.js';
import { createCustomDateModal } from './components/customDateModal.js';
import { setupCustomDateModal } from './modules/customDateModal.js';
import { setupDetailPanel } from './modules/detailPanel.js';
import { createLeadsSummaryCard, createLeadsDetailPanel } from './components/leadsSummaryCard.js';
import { createEmployeeSummaryCard, createEmployeeDetailPanel } from './components/employeeSummaryCard.js';
import { renderLeadsCard } from './components/leadsCard.js';
import { createEmployeeStatsCard } from './components/employeeComponents.js';
import { setupLeadsCard, fetchLeadsStats } from './modules/leads.js';
import { fetchEmployeeStats, setupEmployeeFilter } from './modules/employeeStats.js';
import pb from './utils/pb.js';

let refreshInterval = null;
const refreshFunctions = [];
let leadsData = {};
let employeeData = [];

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

async function loadLeadsData() {
    try {
        const response = await fetch('/api/dashboard/summary', {
            headers: { 'Authorization': pb.authStore.token }
        });
        if (response.ok) {
            leadsData = await response.json();
            updateLeadsSummaryCard();
        }
    } catch (error) {
        console.error('Error loading leads:', error);
    }
}

async function loadEmployeeData() {
    try {
        const now = new Date();
        const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate());
        const endOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
        const todayFilter = `lead_status_date >= "${startOfDay.toISOString()}" AND lead_status_date <= "${endOfDay.toISOString()}"`;

        const response = await fetch(`/api/employee/stats?filter=${encodeURIComponent(todayFilter)}`, {
            headers: { 'Authorization': pb.authStore.token }
        });
        if (response.ok) {
            employeeData = await response.json();
            updateEmployeeSummaryCard();
        }
    } catch (error) {
        console.error('Error loading employees:', error);
    }
}

function updateLeadsSummaryCard() {
    const container = document.querySelector('[onclick*="leadsPanel"]');
    if (container) {
        container.outerHTML = createLeadsSummaryCard(leadsData);
        feather.replace();
    }
}

function updateEmployeeSummaryCard() {
    const container = document.querySelector('[onclick*="employeePanel"]');
    if (container) {
        container.outerHTML = createEmployeeSummaryCard(employeeData);
        feather.replace();
    }
}

function renderDetailPanelContent() {
    const leadsContent = document.getElementById('leadsDetailContent');
    const employeeContent = document.getElementById('employeeDetailContent');

    if (leadsContent) {
        leadsContent.innerHTML = renderLeadsCard();
    }

    if (employeeContent) {
        employeeContent.innerHTML = createEmployeeStatsCard();
    }

    feather.replace();
    setupLeadsCard();
    setupEmployeeFilter();
}

if (checkAuth()) {
    document.getElementById('sidebarContainer').innerHTML = renderSidebar();
    document.getElementById('importModalContainer').innerHTML = renderImportModal();

    async function setupUI() {
        const statsGrid = document.getElementById('statsGrid');
        const detailPanels = document.getElementById('detailPanels');

        statsGrid.innerHTML = createLeadsSummaryCard(leadsData) + createEmployeeSummaryCard(employeeData);
        detailPanels.innerHTML = createLeadsDetailPanel() + createEmployeeDetailPanel();

        document.body.insertAdjacentHTML('beforeend', createCustomDateModal());

        displayUserInfo();
        setupSidebarToggle();
        setupLogout();
        setupImportModal();
        setupCustomDateModal();
        setupDetailPanel();

        feather.replace();

        renderDetailPanelContent();

        registerAutoRefresh(loadLeadsData);
        registerAutoRefresh(loadEmployeeData);
        registerAutoRefresh(fetchLeadsStats);
        registerAutoRefresh(fetchEmployeeStats);

        await Promise.all([loadLeadsData(), loadEmployeeData()]);

        fetchLeadsStats();
        fetchEmployeeStats();

        startAutoRefresh();
    }

    if ('requestIdleCallback' in window) {
        requestIdleCallback(setupUI);
    } else {
        setTimeout(setupUI, 0);
    }
}
