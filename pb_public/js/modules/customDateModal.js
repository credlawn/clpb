let currentModalCallback = null;

export function setupCustomDateModal() {
    const modal = document.getElementById('customDateModal');
    const closeBtn = document.getElementById('closeCustomDateModal');
    const cancelBtn = document.getElementById('cancelCustomDate');
    const applyBtn = document.getElementById('applyCustomDate');
    const startDateInput = document.getElementById('customStartDate');
    const endDateInput = document.getElementById('customEndDate');

    if (!modal) return;

    const closeModal = () => {
        modal.classList.add('hidden');
        startDateInput.value = '';
        endDateInput.value = '';
        currentModalCallback = null;
    };

    closeBtn.addEventListener('click', closeModal);
    cancelBtn.addEventListener('click', closeModal);

    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            closeModal();
        }
    });

    applyBtn.addEventListener('click', () => {
        const startDate = startDateInput.value;
        const endDate = endDateInput.value;

        if (startDate && endDate) {
            if (currentModalCallback) {
                currentModalCallback(startDate, endDate);
            }
            closeModal();
        } else {
            alert('Please select both start and end dates');
        }
    });
}

export function openCustomDateModal(callback) {
    const modal = document.getElementById('customDateModal');
    if (modal) {
        currentModalCallback = callback;
        modal.classList.remove('hidden');
    }
}
