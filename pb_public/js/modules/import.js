import pb from '../utils/pb.js';
import { fetchLeadsStats } from './leads.js';

let excelData = [];
let excelColumns = [];
let fieldMapping = {};

const pbFields = [
    { name: 'old_arn_no', label: 'Old ARN Number', required: false },
    { name: 'segment', label: 'Segment', required: false },
    { name: 'customer_name', label: 'Customer Name', required: true },
    { name: 'mobile_no', label: 'Mobile Number', required: true },
    { name: 'old_decision_date', label: 'Old Decision Date', required: false, type: 'date' },
    { name: 'city', label: 'City', required: false },
    { name: 'promo_code', label: 'Promo Code', required: false },
    { name: 'product', label: 'Product', required: false },
    { name: 'employer', label: 'Employer', required: false },
    { name: 'decline_reason', label: 'Decline Reason', required: false },
    { name: 'data_code', label: 'Data Code', required: false },
    { name: 'data_sub_code', label: 'Data Sub Code', required: false },
    { name: 'custom_code', label: 'Custom Code', required: false }
];

export function setupImportModal() {
    const importBtn = document.getElementById('importBtn');
    const importModal = document.getElementById('importModal');
    const closeModal = document.getElementById('closeImportModal');
    const cancelImport = document.getElementById('cancelImport');
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('fileInput');
    const previewBtn = document.getElementById('previewBtn');
    const importDataBtn = document.getElementById('importDataBtn');
    const downloadTemplate = document.getElementById('downloadTemplate');

    importBtn.addEventListener('click', () => {
        importModal.classList.remove('hidden');
        setTimeout(() => feather.replace(), 100);
    });

    closeModal.addEventListener('click', closeImportModal);
    cancelImport.addEventListener('click', closeImportModal);

    downloadTemplate.addEventListener('click', () => {
        const headers = pbFields.map(f => f.label);
        const ws = XLSX.utils.aoa_to_sheet([headers]);
        const wb = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(wb, ws, 'Template');
        XLSX.writeFile(wb, 'database_import_template.xlsx');
    });

    dropZone.addEventListener('click', () => fileInput.click());

    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('border-blue-400');
    });

    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('border-blue-400');
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('border-blue-400');
        const file = e.dataTransfer.files[0];
        if (file) handleFile(file);
    });

    fileInput.addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) handleFile(file);
    });

    previewBtn.addEventListener('click', showPreview);
    importDataBtn.addEventListener('click', importData);
}

function closeImportModal() {
    document.getElementById('importModal').classList.add('hidden');
    document.getElementById('uploadStep').classList.remove('hidden');
    document.getElementById('mappingStep').classList.add('hidden');
    document.getElementById('fileInfo').classList.add('hidden');
    document.getElementById('previewBtn').classList.add('hidden');
    document.getElementById('importDataBtn').classList.add('hidden');
    document.getElementById('fileInput').value = '';
    excelData = [];
    excelColumns = [];
    fieldMapping = {};
}

function handleFile(file) {
    const reader = new FileReader();

    reader.onload = (e) => {
        try {
            const data = new Uint8Array(e.target.result);
            const workbook = XLSX.read(data, { type: 'array' });
            const sheetName = workbook.SheetNames[0];
            const sheet = workbook.Sheets[sheetName];
            const jsonData = XLSX.utils.sheet_to_json(sheet);

            if (jsonData.length === 0) {
                alert('Excel file is empty!');
                return;
            }

            excelData = jsonData;
            excelColumns = Object.keys(jsonData[0]);

            document.getElementById('fileName').textContent = file.name;
            document.getElementById('rowCount').textContent = `(${jsonData.length} rows)`;
            document.getElementById('fileInfo').classList.remove('hidden');

            showMappingStep();

        } catch (error) {
            console.error('Error parsing file:', error);
            alert('Error reading file. Please check the format.');
        }
    };

    reader.readAsArrayBuffer(file);
}

function showMappingStep() {
    document.getElementById('uploadStep').classList.add('hidden');
    document.getElementById('mappingStep').classList.remove('hidden');
    document.getElementById('previewBtn').classList.remove('hidden');

    const mappingsContainer = document.getElementById('fieldMappings');
    mappingsContainer.innerHTML = '';

    pbFields.forEach(field => {
        const row = document.createElement('div');
        row.className = 'flex items-center space-x-3 text-sm';

        row.innerHTML = `
            <div class="w-1/2">
                <label class="font-medium text-gray-900">
                    ${field.label}
                    ${field.required ? '<span class="text-red-500">*</span>' : ''}
                </label>
            </div>
            <div class="w-1/2">
                <select class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent" data-field="${field.name}">
                    <option value="">-- Select Column --</option>
                    ${excelColumns.map(col => `<option value="${col}">${col}</option>`).join('')}
                </select>
            </div>
        `;

        mappingsContainer.appendChild(row);
    });

    const selects = mappingsContainer.querySelectorAll('select');
    selects.forEach(select => {
        select.addEventListener('change', (e) => {
            const fieldName = e.target.dataset.field;
            const excelCol = e.target.value;
            if (excelCol) {
                fieldMapping[fieldName] = excelCol;
            } else {
                delete fieldMapping[fieldName];
            }
            updateImportButton();
        });

        const fieldDef = pbFields.find(f => f.name === select.dataset.field);
        if (fieldDef) {
            const fieldLabel = fieldDef.label.toLowerCase();
            const fieldName = select.dataset.field.toLowerCase();

            const autoMatch = excelColumns.find(col => {
                const colLower = col.toLowerCase();
                return colLower === fieldLabel ||
                    colLower === fieldName ||
                    colLower.replace(/\s+/g, '_') === fieldName ||
                    colLower.replace(/\s+/g, '') === fieldName.replace(/_/g, '');
            });

            if (autoMatch) {
                select.value = autoMatch;
                fieldMapping[select.dataset.field] = autoMatch;
            }
        }
    });

    updateImportButton();
}

function updateImportButton() {
    const requiredFields = pbFields.filter(f => f.required).map(f => f.name);
    const allRequiredMapped = requiredFields.every(field => fieldMapping[field]);

    if (allRequiredMapped) {
        document.getElementById('importDataBtn').classList.remove('hidden');
    } else {
        document.getElementById('importDataBtn').classList.add('hidden');
    }
}

function showPreview() {
    const previewSection = document.getElementById('previewSection');
    const previewTable = document.getElementById('previewTable');

    const mappedFields = Object.keys(fieldMapping);
    const previewData = excelData.slice(0, 5);

    let tableHTML = '<thead class="bg-gray-50"><tr>';
    mappedFields.forEach(field => {
        const fieldLabel = pbFields.find(f => f.name === field)?.label || field;
        tableHTML += `<th class="px-3 py-2 text-left font-semibold text-gray-900">${fieldLabel}</th>`;
    });
    tableHTML += '</tr></thead><tbody class="divide-y divide-gray-200">';

    previewData.forEach(row => {
        tableHTML += '<tr>';
        mappedFields.forEach(field => {
            const excelCol = fieldMapping[field];
            const value = row[excelCol] || '';
            tableHTML += `<td class="px-3 py-2 text-gray-700">${value}</td>`;
        });
        tableHTML += '</tr>';
    });

    tableHTML += '</tbody>';
    previewTable.innerHTML = tableHTML;
    previewSection.classList.remove('hidden');
}

function convertToUTC(dateValue) {
    if (!dateValue) return null;

    let date;
    if (typeof dateValue === 'number') {
        date = new Date((dateValue - 25569) * 86400 * 1000);
    } else if (typeof dateValue === 'string') {
        date = new Date(dateValue);
    } else if (dateValue instanceof Date) {
        date = dateValue;
    } else {
        return null;
    }

    if (isNaN(date.getTime())) return null;

    date.setHours(12, 0, 0, 0);
    return date.toISOString();
}

async function importData() {
    const importBtn = document.getElementById('importDataBtn');
    const importBtnText = document.getElementById('importBtnText');

    importBtn.disabled = true;
    importBtnText.textContent = 'Importing...';

    try {
        let successCount = 0;
        let errorCount = 0;
        let skipCount = 0;
        const now = new Date().toISOString();

        for (let i = 0; i < excelData.length; i++) {
            const row = excelData[i];
            const recordData = {
                import_date: now
            };

            Object.keys(fieldMapping).forEach(field => {
                const excelCol = fieldMapping[field];
                let value = row[excelCol] || '';

                const fieldDef = pbFields.find(f => f.name === field);
                if (fieldDef && fieldDef.type === 'date') {
                    value = convertToUTC(value);
                }

                recordData[field] = value;
            });

            try {
                await pb.collection('database').create(recordData);
                successCount++;
                importBtnText.textContent = `Importing... (${successCount + skipCount}/${excelData.length})`;
            } catch (error) {
                if (error.message && error.message.includes('Mobile number already exists')) {
                    skipCount++;
                    importBtnText.textContent = `Importing... (${successCount + skipCount}/${excelData.length})`;
                } else {
                    console.error('Error importing row:', error);
                    errorCount++;
                }
            }
        }

        alert(`Import complete!\n✅ Success: ${successCount}\n⏭️ Skipped (Duplicate): ${skipCount}\n❌ Errors: ${errorCount}`);
        closeImportModal();
        fetchLeadsStats();

    } catch (error) {
        console.error('Import error:', error);
        alert('Error during import. Please try again.');
    } finally {
        importBtn.disabled = false;
        importBtnText.textContent = 'Import Leads';
    }
}
