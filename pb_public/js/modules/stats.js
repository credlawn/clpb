import { pb } from '../utils/pb.js';

export async function fetchOtherStats() {
    try {
        const activeEmployees = await pb.collection('users').getList(1, 1, {
            filter: 'role = "employee"'
        });
        document.getElementById('activeEmployees').textContent = activeEmployees.totalItems || 0;
    } catch (error) {
        console.error('Error fetching employees:', error);
        document.getElementById('activeEmployees').textContent = '0';
    }

    try {
        const todayAttendance = await pb.collection('attendance').getList(1, 1);
        document.getElementById('todayAttendance').textContent = todayAttendance.totalItems || 0;
    } catch (error) {
        console.error('Error fetching attendance:', error);
        document.getElementById('todayAttendance').textContent = '0';
    }

    try {
        const todayCalls = await pb.collection('call_logs').getList(1, 1);
        document.getElementById('todayCalls').textContent = todayCalls.totalItems || 0;
    } catch (error) {
        console.error('Error fetching calls:', error);
        document.getElementById('todayCalls').textContent = '0';
    }
}
