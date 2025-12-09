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
            record.mobile_no?.includes(searchTerm) ||
            record.city?.toLowerCase().includes(searchTerm) ||
            record.employer?.toLowerCase().includes(searchTerm);
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
        tbody.innerHTML = `<tr><td colspan="13" class="px-4 py-8 text-center text-gray-400 text-xs">No records found</td></tr>`;
        document.getElementById('showingCount').textContent = '0 records';
        document.getElementById('totalRecordCount').textContent = '0';
        return;
    }

    const toTitleCase = (str) => {
        if (!str) return '-';
        return str.toLowerCase().replace(/\b\w/g, c => c.toUpperCase());
    };

    tbody.innerHTML = pageRecords.map(record => {
        const isSelected = selectedRecords.has(record.id);
        const allocCount = record.allocation_count || 0;
        const empCount = record.employee_count || 0;

        return `
        <tr class="${isSelected ? 'selected' : ''}">
            <td class="px-2 py-1.5 sticky left-0 bg-inherit">
                <input type="checkbox" 
                    class="record-checkbox w-3.5 h-3.5 rounded" 
                    data-id="${record.id}"
                    ${isSelected ? 'checked' : ''}
                    ${selectedRecords.size >= 100 && !isSelected ? 'disabled' : ''}>
            </td>
            <td class="px-2 py-1.5 font-medium text-gray-800 whitespace-nowrap">${toTitleCase(record.customer_name)}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.mobile_no || '-'}</td>
            <td class="px-2 py-1.5 text-center">
                <span class="inline-block px-1.5 py-0.5 rounded text-xs font-medium ${record.data_status === 'used' ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}">
                    ${record.data_status || 'new'}
                </span>
            </td>
            <td class="px-2 py-1.5 text-center">
                <span class="text-gray-600">${allocCount}</span><span class="text-gray-300">/</span><span class="text-gray-600">${empCount}</span>
            </td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.city || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${toTitleCase(record.employer)}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.product || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.segment || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.decline_reason || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.data_code || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.data_sub_code || '-'}</td>
            <td class="px-2 py-1.5 text-gray-600 whitespace-nowrap">${record.custom_code || '-'}</td>
        </tr>`;
    }).join('');

    document.getElementById('showingCount').textContent = `${filteredRecords.length} records`;
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
            <div class="flex items-center gap-3 p-2 rounded hover:bg-gray-50 border border-gray-100">
                <input type="checkbox" 
                    class="employee-checkbox w-3.5 h-3.5 rounded" 
                    data-code="${emp.employee_code}"
                    data-name="${emp.employee_name}">
                <div class="flex-1 min-w-0 text-xs font-medium text-gray-800 truncate">${emp.employee_name}</div>
                <div class="text-xs text-green-600 font-medium w-8 text-center">${emp.new_leads_count}</div>
                <input type="number" 
                    class="allocation-count w-12 px-1 py-0.5 border border-gray-300 rounded text-xs text-center focus:border-blue-500 outline-none" 
                    data-code="${emp.employee_code}"
                    min="0" 
                    max="${selectedRecords.size}" 
                    value="0"
                    disabled>
            </div>
        `).join('');

        feather.replace();
        setupModalListeners();
    } catch (error) {
        console.error('Error loading employees:', error);
        employeeList.innerHTML = `<div class="text-center py-6 text-red-500 text-xs">Error loading employees</div>`;
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

    const mobileFilterBtn = document.getElementById('mobileFilterBtn');
    const mobileFilterPanel = document.getElementById('mobileFilterPanel');

    if (mobileFilterBtn && mobileFilterPanel) {
        mobileFilterBtn.addEventListener('click', () => {
            mobileFilterPanel.classList.toggle('hidden');
            feather.replace();
        });
    }

    const mobileFilters = ['searchInputMobile', 'dataCodeFilterMobile', 'dataSubCodeFilterMobile',
        'customCodeFilterMobile', 'dataStatusFilterMobile', 'allocationCountFilterMobile', 'employeeCountFilterMobile'];

    mobileFilters.forEach(id => {
        const el = document.getElementById(id);
        if (el) {
            el.addEventListener('input', () => {
                const desktopId = id.replace('Mobile', '');
                const desktopEl = document.getElementById(desktopId);
                if (desktopEl) desktopEl.value = el.value;
                applyFilters();
            });
            el.addEventListener('change', () => {
                const desktopId = id.replace('Mobile', '');
                const desktopEl = document.getElementById(desktopId);
                if (desktopEl) desktopEl.value = el.value;
                applyFilters();
            });
        }
    });

    const resetMobile = document.getElementById('resetFiltersMobile');
    if (resetMobile) {
        resetMobile.addEventListener('click', resetFilters);
    }

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
