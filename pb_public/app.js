import { pb } from './js/utils/pb.js';

const loginForm = document.getElementById('loginForm');
const emailInput = document.getElementById('email');
const passwordInput = document.getElementById('password');
const loginButton = document.getElementById('loginButton');
const errorMessage = document.getElementById('errorMessage');
const buttonText = document.getElementById('buttonText');
const buttonLoader = document.getElementById('buttonLoader');

function showError(message) {
    errorMessage.textContent = message;
    errorMessage.classList.remove('hidden');

    setTimeout(() => {
        errorMessage.classList.add('hidden');
    }, 5000);
}

function setLoading(isLoading) {
    if (isLoading) {
        loginButton.disabled = true;
        buttonText.classList.add('invisible');
        buttonLoader.classList.remove('hidden');
        emailInput.disabled = true;
        passwordInput.disabled = true;
    } else {
        loginButton.disabled = false;
        buttonText.classList.remove('invisible');
        buttonLoader.classList.add('hidden');
        emailInput.disabled = false;
        passwordInput.disabled = false;
    }
}

loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const email = emailInput.value.trim();
    const password = passwordInput.value;

    if (!email || !password) {
        showError('Please enter both email and password');
        return;
    }

    setLoading(true);
    errorMessage.classList.add('hidden');

    try {
        const authData = await pb.collection('users').authWithPassword(email, password);

        const userRole = (authData.record.role || '').toLowerCase();

        if (userRole !== 'manager') {
            pb.authStore.clear();
            showError('Access denied. Manager login only.');
            setLoading(false);
            return;
        }

        window.location.href = '/dashboard.html';

    } catch (error) {
        console.error('Login error:', error);

        let errorMsg = 'Login failed. Please try again.';

        if (error.status === 400) {
            errorMsg = 'Invalid email or password';
        } else if (error.status === 0) {
            errorMsg = 'Cannot connect to server';
        } else if (error.message) {
            errorMsg = error.message;
        }

        showError(errorMsg);
        setLoading(false);
    }
});

if (pb.authStore.isValid) {
    const userRole = (pb.authStore.model.role || '').toLowerCase();
    if (userRole === 'manager') {
        window.location.href = '/dashboard.html';
    }
}
