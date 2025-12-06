export function renderImportModal() {
    return `
        <div id="importModal" class="hidden fixed inset-0 bg-black bg-opacity-50 z-50 flex items-start justify-center p-4 pt-8 overflow-y-auto">
            <div class="bg-white rounded-xl shadow-2xl w-full max-w-4xl flex flex-col" style="max-height: 85vh;">
                <div class="flex items-center justify-between p-4 border-b border-gray-200 flex-shrink-0">
                    <div class="flex items-center space-x-2">
                        <i data-feather="upload" class="w-5 h-5 text-blue-600"></i>
                        <h2 class="text-lg font-bold text-gray-900">Import Data to Database</h2>
                    </div>
                    <button id="closeImportModal" class="p-1 hover:bg-gray-100 rounded-lg transition-colors">
                        <i data-feather="x" class="w-5 h-5 text-gray-500"></i>
                    </button>
                </div>

                <div class="flex-1 overflow-y-auto p-4">
                    <div id="uploadStep">
                        <h3 class="text-sm font-semibold text-gray-900 mb-2">Step 1: Upload Excel File</h3>
                        <p class="text-xs text-gray-600 mb-3">Select an Excel file (.xlsx, .xls) to import</p>

                        <div id="dropZone" class="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center hover:border-blue-400 transition-colors cursor-pointer">
                            <i data-feather="upload-cloud" class="w-12 h-12 text-gray-400 mx-auto mb-3"></i>
                            <p class="text-sm font-medium text-gray-700">Click to upload or drag and drop</p>
                            <p class="text-xs text-gray-500 mt-1">Excel files only (.xlsx, .xls)</p>
                            <input type="file" id="fileInput" accept=".xlsx,.xls" class="hidden">
                        </div>

                        <div id="fileInfo" class="hidden mt-3 p-3 bg-blue-50 rounded-lg flex items-center justify-between">
                            <div class="flex items-center space-x-2">
                                <i data-feather="file" class="w-4 h-4 text-blue-600"></i>
                                <span class="text-sm font-medium text-blue-900" id="fileName"></span>
                                <span class="text-xs text-blue-600" id="rowCount"></span>
                            </div>
                        </div>

                        <button id="downloadTemplate" class="mt-3 text-xs text-blue-600 hover:text-blue-700 font-medium flex items-center space-x-1">
                            <i data-feather="download" class="w-3.5 h-3.5"></i>
                            <span>Download Template</span>
                        </button>
                    </div>

                    <div id="mappingStep" class="hidden">
                        <h3 class="text-sm font-semibold text-gray-900 mb-2">Step 2: Map Fields</h3>
                        <p class="text-xs text-gray-600 mb-3">Match Excel columns to PocketBase fields</p>
                        <div class="bg-gray-50 rounded-lg p-3 space-y-2" id="fieldMappings"></div>
                    </div>

                    <div id="previewSection" class="hidden">
                        <h3 class="text-sm font-semibold text-gray-900 mb-2">Preview (First 5 Rows)</h3>
                        <div class="overflow-x-auto border border-gray-200 rounded-lg">
                            <table class="min-w-full divide-y divide-gray-200 text-xs" id="previewTable"></table>
                        </div>
                    </div>
                </div>

                <div class="flex items-center justify-between p-4 border-t border-gray-200 bg-gray-50 flex-shrink-0">
                    <div class="flex items-center space-x-2">
                        <button id="cancelImport" class="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors">Cancel</button>
                    </div>
                    <div class="flex items-center space-x-2">
                        <button id="previewBtn" class="hidden px-4 py-2 text-sm font-medium text-blue-600 hover:bg-blue-50 rounded-lg transition-colors">Preview Data</button>
                        <button id="importDataBtn" class="hidden px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors">
                            <span id="importBtnText">Import Leads</span>
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
}
