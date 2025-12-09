import pb from './pb.js';

export function checkAuth() {
    if (!pb.authStore.isValid) {
        window.location.href = '/';
        return false;
    }

    return true;
}

export function displayUserInfo() {
    const user = pb.authStore.model;

    if (!user) return;

    const userName = user.name || user.email?.split('@')[0] || 'Manager';

    const sidebarUserName = document.getElementById('sidebarUserName');
    if (sidebarUserName) {
        sidebarUserName.textContent = userName;
    }

    const currentDate = document.getElementById('currentDate');
    if (currentDate) {
        const now = new Date();
        const dateStr = now.toLocaleDateString('en-US', {
            weekday: 'short',
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
        currentDate.textContent = dateStr;
    }
}

export function setupLogout() {
    const logoutButton = document.getElementById('logoutButton');

    logoutButton.addEventListener('click', () => {
        pb.authStore.clear();
        window.location.href = '/';
    });
}
