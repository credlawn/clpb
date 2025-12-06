import { checkAuth, displayUserInfo, setupLogout } from './utils/auth.js';
import { setupSidebarToggle } from './utils/ui.js';
import { setupLeadsCard, fetchLeadsStats } from './modules/leads.js';
import { fetchOtherStats } from './modules/stats.js';
import { setupImportModal } from './modules/import.js';
import { renderSidebar } from './components/sidebar.js';
import { renderLeadsCard } from './components/leadsCard.js';
import { renderStatsCards } from './components/statsCards.js';
import { renderImportModal } from './components/importModal.js';

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
        fetchLeadsStats();
        fetchOtherStats();
    };

    if ('requestIdleCallback' in window) {
        requestIdleCallback(setupUI);
    } else {
        setTimeout(setupUI, 0);
    }
}
