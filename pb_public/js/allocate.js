import { checkAuth, displayUserInfo, setupLogout } from './utils/auth.js';
import { setupSidebarToggle } from './utils/ui.js';
import { renderSidebar } from './components/sidebar.js';
import pb from './utils/pb.js';

let databaseRecords = [];
let filteredRecords = [];
let selectedRecords = new Set();
let currentPage = 1;
const recordsPerPage = 50;

function showToast(message, type = 'success') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');

    const bgColor = type === 'success' ? 'bg-green-500' : 'bg-red-500';
    const icon = type === 'success' ? '✓' : '✕';

    toast.className = `${bgColor} text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-in`;
    toast.innerHTML = `
        <span class="text-lg font-bold">${icon}</span>
        <span class="text-sm">${message}</span>
    `;

    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

async function loadDatabaseRecords() {
    try {
        const records = await pb.collection('database').getFullList({
            sort: '-created',
        });

        databaseRecords = records;

        selectedRecords.clear();

        populateFilters();
        applyFilters();
        updateSelectionUI();
    } catch (error) {
        console.error('Error loading database records:', error);
        document.getElementById('databaseTableBody').innerHTML = `
            <tr><td colspan="8" class="px-4 py-8 text-center text-red-500 text-sm">Error loading records</td></tr>
        `;
    }
}

function populateFilters() {
    const dataCodes = [...new Set(databaseRecords.map(r => r.data_code).filter(Boolean))];
    const dataSubCodes = [...new Set(databaseRecords.map(r => r.data_sub_code).filter(Boolean))];
    const customCodes = [...new Set(databaseRecords.map(r => r.custom_code).filter(Boolean))];

    const dataCodeFilter = document.getElementById('dataCodeFilter');
    const dataSubCodeFilter = document.getElementById('dataSubCodeFilter');
    const customCodeFilter = document.getElementById('customCodeFilter');

    dataCodes.forEach(code => {
        const option = document.createElement('option');
        option.value = code;
        option.textContent = code;
        dataCodeFilter.appendChild(option);
    });

    dataSubCodes.forEach(code => {
        const option = document.createElement('option');
        option.value = code;
        option.textContent = code;
        dataSubCodeFilter.appendChild(option);
    });

    customCodes.forEach(code => {
        const option = document.createElement('option');
        option.value = code;
        option.textContent = code;
        customCodeFilter.appendChild(option);
    });
}

function applyFilters() {
    const searchTerm = document.getElementById('searchInput').value.toLowerCase();
    const dataCode = document.getElementById('dataCodeFilter').value;
    const dataSubCode = document.getElementById('dataSubCodeFilter').value;
    const customCode = document.getElementById('customCodeFilter').value;
    const dataStatus = document.getElementById('dataStatusFilter').value;
    const allocationCount = document.getElementById('allocationCountFilter').value;
    const employeeCount = document.getElementById('employeeCountFilter').value;

    filteredRecords = databaseRecords.filter(record => {
        const matchesSearch = !searchTerm ||
            record.customer_name?.toLowerCase().includes(searchTerm) ||
            record.mobile_no?.includes(searchTerm);
        const matchesDataCode = !dataCode || record.data_code === dataCode;
        const matchesDataSubCode = !dataSubCode || record.data_sub_code === dataSubCode;
        const matchesCustomCode = !customCode || record.custom_code === customCode;

        let matchesDataStatus = true;
        if (dataStatus === 'new') {
            matchesDataStatus = !record.data_status || record.data_status === 'new';
        } else if (dataStatus === 'used') {
            matchesDataStatus = record.data_status === 'used';
        }

        let matchesAllocationCount = true;
        if (allocationCount === '0') matchesAllocationCount = (record.allocation_count || 0) === 0;
        else if (allocationCount === '1') matchesAllocationCount = (record.allocation_count || 0) === 1;
        else if (allocationCount === '2+') matchesAllocationCount = (record.allocation_count || 0) >= 2;

        let matchesEmployeeCount = true;
        if (employeeCount === '0') matchesEmployeeCount = (record.employee_count || 0) === 0;
        else if (employeeCount === '1') matchesEmployeeCount = (record.employee_count || 0) === 1;
        else if (employeeCount === '2+') matchesEmployeeCount = (record.employee_count || 0) >= 2;

        return matchesSearch && matchesDataCode && matchesDataSubCode && matchesCustomCode &&
            matchesDataStatus && matchesAllocationCount && matchesEmployeeCount;
    });

    currentPage = 1;
    renderTable();
}

function resetFilters() {
    document.getElementById('searchInput').value = '';
    document.getElementById('dataCodeFilter').value = '';
    document.getElementById('dataSubCodeFilter').value = '';
    document.getElementById('customCodeFilter').value = '';
    document.getElementById('dataStatusFilter').value = '';
    document.getElementById('allocationCountFilter').value = '';
    document.getElementById('employeeCountFilter').value = '';
    applyFilters();
}

function renderTable() {
    const tbody = document.getElementById('databaseTableBody');
    const start = (currentPage - 1) * recordsPerPage;
    const end = start + recordsPerPage;
    const pageRecords = filteredRecords.slice(start, end);

    if (pageRecords.length === 0) {
        tbody.innerHTML = `
            <tr><td colspan="8" class="px-4 py-8 text-center text-gray-500 text-sm">No records found</td></tr>
        `;
        return;
    }

    tbody.innerHTML = pageRecords.map(record => `
        <tr class="hover:bg-gray-50 ${selectedRecords.has(record.id) ? 'bg-blue-50' : ''}">
            <td class="px-4 py-3 sticky left-0 bg-white ${selectedRecords.has(record.id) ? 'bg-blue-50' : ''}">
                <input type="checkbox" 
                    class="record-checkbox w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500" 
                    data-id="${record.id}"
                    ${selectedRecords.has(record.id) ? 'checked' : ''}
                    ${selectedRecords.size >= 100 && !selectedRecords.has(record.id) ? 'disabled' : ''}>
            </td>
            <td class="px-4 py-3 text-sm text-gray-800 whitespace-nowrap">${record.customer_name || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.mobile_no || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.city || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.employer || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.segment || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.product || '-'}</td>
            <td class="px-4 py-3 text-sm text-gray-600 whitespace-nowrap">${record.decline_reason || '-'}</td>
            <td class="px-4 py-3 text-center text-sm">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${record.data_status === 'used' ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'}">
                    ${record.data_status || 'new'}
                </span>
            </td>
            <td class="px-4 py-3 text-center text-sm whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${record.allocation_count > 0 ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-600'}">
                    ${record.allocation_count || 0}
                </span>
            </td>
            <td class="px-4 py-3 text-center text-sm whitespace-nowrap">
                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${record.employee_count > 0 ? 'bg-purple-100 text-purple-800' : 'bg-gray-100 text-gray-600'}">
                    ${record.employee_count || 0}
                </span>
            </td>
        </tr>
    `).join('');

    document.getElementById('showingCount').textContent = filteredRecords.length;
    document.getElementById('totalRecordCount').textContent = filteredRecords.length;

    const prevBtn = document.getElementById('prevPage');
    const nextBtn = document.getElementById('nextPage');

    prevBtn.disabled = currentPage === 1;
    if (currentPage === 1) {
        prevBtn.classList.add('opacity-50', 'cursor-not-allowed');
        prevBtn.classList.remove('hover:bg-gray-100');
    } else {
        prevBtn.classList.remove('opacity-50', 'cursor-not-allowed');
        prevBtn.classList.add('hover:bg-gray-100');
    }

    nextBtn.disabled = end >= filteredRecords.length;
    if (end >= filteredRecords.length) {
        nextBtn.classList.add('opacity-50', 'cursor-not-allowed');
        nextBtn.classList.remove('hover:bg-gray-100');
    } else {
        nextBtn.classList.remove('opacity-50', 'cursor-not-allowed');
        nextBtn.classList.add('hover:bg-gray-100');
    }

    setupCheckboxListeners();
}

function setupCheckboxListeners() {
    document.querySelectorAll('.record-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
            const id = e.target.dataset.id;
            if (e.target.checked) {
                if (selectedRecords.size < 100) {
                    selectedRecords.add(id);
                } else {
                    e.target.checked = false;
                }
            } else {
                selectedRecords.delete(id);
            }
            updateSelectionUI();
        });
    });
}

function updateSelectionUI() {
    const count = selectedRecords.size;

    const selectedCountEl = document.getElementById('selectedCount');
    const allocateBtnEl = document.getElementById('allocateBtn');

    if (selectedCountEl) {
        selectedCountEl.textContent = count;
    }

    if (allocateBtnEl) {
        allocateBtnEl.disabled = count === 0;
        if (count === 0) {
            allocateBtnEl.classList.remove('bg-blue-600', 'hover:bg-blue-700');
            allocateBtnEl.classList.add('bg-gray-300', 'cursor-not-allowed');
        } else {
            allocateBtnEl.classList.remove('bg-gray-300', 'cursor-not-allowed');
            allocateBtnEl.classList.add('bg-blue-600', 'hover:bg-blue-700');
        }
    }

    const selectAll = document.getElementById('selectAll');
    const visibleCheckboxes = document.querySelectorAll('.record-checkbox:not([disabled])');
    const allChecked = Array.from(visibleCheckboxes).every(cb => cb.checked);
    selectAll.checked = allChecked && visibleCheckboxes.length > 0;
}

async function openAllocationModal() {
    const modal = document.getElementById('allocationModal');
    const employeeList = document.getElementById('employeeList');

    const modalSelectedCount = document.getElementById('modalSelectedCount');
    const totalSelected = document.getElementById('totalSelected');

    if (modalSelectedCount) modalSelectedCount.textContent = selectedRecords.size;
    if (totalSelected) totalSelected.textContent = selectedRecords.size;

    modal.classList.remove('hidden');

    try {
        const employees = await fetch('/api/employees/with-new-leads', {
            headers: { 'Authorization': pb.authStore.token }
        }).then(r => r.json());

        employees.sort((a, b) => a.new_leads_count - b.new_leads_count);

        employeeList.innerHTML = employees.map(emp => `
            <div class="border border-gray-200 rounded-lg p-4 hover:border-blue-300 transition-colors">
                <div class="grid grid-cols-3 gap-4">
                    <div class="col-span-1 flex items-start gap-2">
                        <input type="checkbox" 
                            class="employee-checkbox mt-1 w-4 h-4 text-blue-600 rounded focus:ring-2 focus:ring-blue-500" 
                            data-code="${emp.employee_code}"
                            data-name="${emp.employee_name}">
                        <div>
                            <div class="font-medium text-gray-800 text-sm">${emp.employee_name}</div>
                            <div class="text-xs text-gray-500">${emp.employee_code}</div>
                        </div>
                    </div>
                    
                    <div class="col-span-1 flex flex-col items-center justify-center">
                        <div class="flex items-center gap-1">
                            <input type="number" 
                                class="allocation-count w-16 px-2 py-1 border border-gray-300 rounded text-sm text-center focus:outline-none focus:ring-2 focus:ring-blue-500" 
                                data-code="${emp.employee_code}"
                                min="0" 
                                max="${selectedRecords.size}" 
                                value="0"
                                disabled>
                            <span class="text-xs text-gray-500">leads</span>
                        </div>
                    </div>
                    
                    <div class="col-span-1 flex items-center justify-end gap-4 text-sm text-gray-600">
                        <div class="text-center">
                            <div class="text-xs text-gray-500">New</div>
                            <div class="font-semibold text-green-600">${emp.new_leads_count}</div>
                        </div>
                        <div class="text-center">
                            <div class="text-xs text-gray-500">Total</div>
                            <div class="font-semibold text-blue-600">${emp.total_leads}</div>
                        </div>
                    </div>
                </div>
            </div>
        `).join('');

        feather.replace();
        setupModalListeners();
    } catch (error) {
        console.error('Error loading employees:', error);
        employeeList.innerHTML = `<div class="text-center py-8 text-red-500">Error loading employees</div>`;
    }
}

function setupModalListeners() {
    document.querySelectorAll('.employee-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', (e) => {
            const code = e.target.dataset.code;
            const input = document.querySelector(`.allocation-count[data-code="${code}"]`);
            input.disabled = !e.target.checked;
            if (!e.target.checked) {
                input.value = 0;
            }
            validateAllocation();
        });
    });

    document.querySelectorAll('.allocation-count').forEach(input => {
        input.addEventListener('input', validateAllocation);
    });
}

function validateAllocation() {
    const total = Array.from(document.querySelectorAll('.allocation-count'))
        .reduce((sum, input) => sum + (parseInt(input.value) || 0), 0);

    const selectedTotal = selectedRecords.size;
    document.getElementById('totalAllocation').innerHTML = `${total} / <span>${selectedTotal}</span>`;

    const errorDiv = document.getElementById('allocationError');
    const confirmBtn = document.getElementById('confirmAllocation');

    if (total > selectedTotal) {
        errorDiv.textContent = `Total allocation (${total}) exceeds selected records (${selectedTotal})`;
        errorDiv.classList.remove('hidden');
        confirmBtn.disabled = true;
    } else if (total === 0) {
        errorDiv.classList.add('hidden');
        confirmBtn.disabled = true;
    } else if (total < selectedTotal) {
        errorDiv.textContent = `${selectedTotal - total} records will not be allocated`;
        errorDiv.classList.remove('hidden');
        errorDiv.classList.remove('text-red-600');
        errorDiv.classList.add('text-yellow-600');
        confirmBtn.disabled = false;
    } else {
        errorDiv.classList.add('hidden');
        confirmBtn.disabled = false;
    }
}

async function confirmAllocation() {
    const allocations = [];

    document.querySelectorAll('.employee-checkbox:checked').forEach(checkbox => {
        const code = checkbox.dataset.code;
        const name = checkbox.dataset.name;
        const count = parseInt(document.querySelector(`.allocation-count[data-code="${code}"]`).value) || 0;

        if (count > 0) {
            allocations.push({ employee_code: code, employee_name: name, count });
        }
    });

    if (allocations.length === 0) return;

    const confirmBtn = document.getElementById('confirmAllocation');
    confirmBtn.disabled = true;
    confirmBtn.textContent = 'Allocating...';

    try {
        const response = await fetch('/api/allocate-leads', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': pb.authStore.token
            },
            body: JSON.stringify({
                database_record_ids: Array.from(selectedRecords),
                allocations,
                allocated_by_code: pb.authStore.record.employee_code,
                allocated_by_name: pb.authStore.record.employee_name
            })
        });

        const result = await response.json();

        if (response.ok) {
            showToast(`Allocated ${result.allocated_count} leads successfully!`, 'success');
            closeModal();
            selectedRecords.clear();
            await loadDatabaseRecords();
            setTimeout(() => updateSelectionUI(), 100);
        } else {
            showToast(result.error || 'Failed to allocate leads', 'error');
        }
    } catch (error) {
        console.error('Error allocating leads:', error);
        showToast('Network error. Please try again.', 'error');
    } finally {
        confirmBtn.disabled = false;
        confirmBtn.textContent = 'Confirm Allocation';
    }
}

function closeModal() {
    document.getElementById('allocationModal').classList.add('hidden');
}

if (checkAuth()) {
    document.getElementById('sidebarContainer').innerHTML = renderSidebar();
    displayUserInfo();
    setupSidebarToggle();
    setupLogout();

    document.getElementById('searchInput').addEventListener('input', applyFilters);
    document.getElementById('dataCodeFilter').addEventListener('change', applyFilters);
    document.getElementById('dataSubCodeFilter').addEventListener('change', applyFilters);
    document.getElementById('customCodeFilter').addEventListener('change', applyFilters);
    document.getElementById('dataStatusFilter').addEventListener('change', applyFilters);
    document.getElementById('allocationCountFilter').addEventListener('change', applyFilters);
    document.getElementById('employeeCountFilter').addEventListener('change', applyFilters);

    document.getElementById('resetFilters').addEventListener('click', resetFilters);

    document.getElementById('selectAll').addEventListener('change', (e) => {
        const checkboxes = document.querySelectorAll('.record-checkbox:not([disabled])');
        checkboxes.forEach(cb => {
            const id = cb.dataset.id;
            if (e.target.checked) {
                if (selectedRecords.size < 100) {
                    cb.checked = true;
                    selectedRecords.add(id);
                }
            } else {
                cb.checked = false;
                selectedRecords.delete(id);
            }
        });
        updateSelectionUI();
    });

    document.getElementById('prevPage').addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            renderTable();
        }
    });

    document.getElementById('nextPage').addEventListener('click', () => {
        const maxPage = Math.ceil(filteredRecords.length / recordsPerPage);
        if (currentPage < maxPage) {
            currentPage++;
            renderTable();
        }
    });

    document.getElementById('allocateBtn').addEventListener('click', openAllocationModal);
    document.getElementById('closeModal').addEventListener('click', closeModal);
    document.getElementById('cancelAllocation').addEventListener('click', closeModal);
    document.getElementById('confirmAllocation').addEventListener('click', confirmAllocation);

    loadDatabaseRecords();
    feather.replace();
}
