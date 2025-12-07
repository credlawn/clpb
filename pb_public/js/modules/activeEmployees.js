import pb from '../utils/pb.js';

export async function fetchActiveEmployees() {
    try {
        const result = await pb.collection('users').getList(1, 1, {
            filter: 'disabled = false && role ~ "employee"',
            fields: 'id'
        });

        const count = result.totalItems || 0;
        const element = document.getElementById('activeEmployees');
        if (element) {
            element.textContent = count;
        }

        return count;
    } catch (error) {
        console.error('Error fetching active employees:', error);
        const element = document.getElementById('activeEmployees');
        if (element) {
            element.textContent = '0';
        }
        return 0;
    }
}
