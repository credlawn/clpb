import pb from '../utils/pb.js';

export async function fetchLeads(filters = {}) {
    try {
        const {
            page = 1,
            limit = 50,
            search = '',
            status = '',
            agent = '',
            dateFrom = '',
            dateTo = '',
            sortBy = 'created',
            sortOrder = 'desc'
        } = filters;

        let filter = [];

        if (search) {
            filter.push(`(customer_name ~ "${search}" || mobile_no ~ "${search}")`);
        }

        if (status) {
            filter.push(`lead_status = "${status}"`);
        }

        if (agent) {
            filter.push(`assigned_to = "${agent}"`);
        }

        if (dateFrom && dateTo) {
            filter.push(`created >= "${dateFrom} 00:00:00" && created <= "${dateTo} 23:59:59"`);
        }

        const filterString = filter.length > 0 ? filter.join(' && ') : '';
        const sort = `${sortOrder === 'desc' ? '-' : ''}${sortBy}`;

        const result = await pb.collection('leads').getList(page, limit, {
            filter: filterString,
            sort: sort,
            expand: 'assigned_to'
        });

        return {
            items: result.items.map(item => ({
                ...item,
                agent_name: item.expand?.assigned_to?.employee_name || item.employee_name || 'Unassigned'
            })),
            totalItems: result.totalItems,
            totalPages: result.totalPages,
            currentPage: result.page,
            perPage: result.perPage
        };
    } catch (error) {
        console.error('Error fetching leads:', error);
        throw error;
    }
}

export async function fetchAgents() {
    try {
        const agents = await pb.collection('users').getFullList({
            sort: 'employee_name'
        });
        return agents;
    } catch (error) {
        console.error('Error fetching agents:', error);
        return [];
    }
}

export async function exportLeads(filters = {}) {
    try {
        const allLeads = await pb.collection('leads').getFullList({
            filter: filters.filter || '',
            sort: '-created',
            expand: 'assigned_to'
        });

        return allLeads.map(item => ({
            ...item,
            agent_name: item.expand?.assigned_to?.employee_name || item.employee_name || 'Unassigned'
        }));
    } catch (error) {
        console.error('Error exporting leads:', error);
        throw error;
    }
}
