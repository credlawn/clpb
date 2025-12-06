import { pb } from './pb.js';

export function checkAuth() {
    if (!pb.authStore.isValid) {
        window.location.href = '/';
        return false;
    }

    const user = pb.authStore.model;
    const userRole = (user.role || '').toLowerCase();

    if (userRole !== 'manager') {
        pb.authStore.clear();
        alert('Access denied. Manager access only.');
        window.location.href = '/';
        return false;
    }

    return true;
}

export function displayUserInfo() {
    const user = pb.authStore.model;

    if (!user) return;

    const userName = user.name || user.email?.split('@')[0] || 'Manager';

    document.getElementById('sidebarUserName').textContent = userName;

    const now = new Date();
    const dateStr = now.toLocaleDateString('en-US', {
        weekday: 'short',
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
    document.getElementById('currentDate').textContent = dateStr;
}

export function setupLogout() {
    const logoutButton = document.getElementById('logoutButton');

    logoutButton.addEventListener('click', () => {
        pb.authStore.clear();
        window.location.href = '/';
    });
}
