import pb from '../utils/pb.js';

export async function fetchOtherStats() {

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
