let activePanel = null;

export function openDetailPanel(panelId) {
    const overlay = document.getElementById('detailOverlay');
    const panel = document.getElementById(panelId);

    if (overlay && panel) {
        overlay.classList.add('active');
        panel.classList.add('active');
        document.body.style.overflow = 'hidden';
        activePanel = panelId;
    }
}

export function closeDetailPanel() {
    const overlay = document.getElementById('detailOverlay');

    if (overlay && activePanel) {
        const panel = document.getElementById(activePanel);
        overlay.classList.remove('active');
        if (panel) panel.classList.remove('active');
        document.body.style.overflow = '';
        activePanel = null;
    }
}

export function setupDetailPanel() {
    const overlay = document.getElementById('detailOverlay');

    if (overlay) {
        overlay.addEventListener('click', closeDetailPanel);
    }

    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape' && activePanel) {
            closeDetailPanel();
        }
    });
}

window.openDetailPanel = openDetailPanel;
window.closeDetailPanel = closeDetailPanel;
