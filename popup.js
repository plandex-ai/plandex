// Constants
const STORAGE_KEYS = {
  AUTH_TOKEN: "plandex_auth_token",
  ORG_ID: "plandex_org_id",
  API_URL: "plandex_api_url",
  USER_INFO: "plandex_user_info"
};

// DOM Elements
const authForm = document.getElementById('auth-form');
const signInCode = document.getElementById('sign-in-code');
const signInBtn = document.getElementById('sign-in-btn');
const signOutBtn = document.getElementById('sign-out-btn');
const authError = document.getElementById('auth-error');
const notAuthenticatedSection = document.getElementById('not-authenticated');
const authenticatedSection = document.getElementById('authenticated');
const userName = document.getElementById('user-name');
const userEmail = document.getElementById('user-email');
const settingsForm = document.getElementById('settings-form');
const apiUrlInput = document.getElementById('api-url');
const saveSettingsBtn = document.getElementById('save-settings-btn');
const settingsMessage = document.getElementById('settings-message');
const contextMenuStatus = document.getElementById('context-menu-status');
const authStatus = document.getElementById('auth-status');

// Initialize the popup
document.addEventListener('DOMContentLoaded', async () => {
  // Load saved API URL
  loadApiUrl();
  
  // Check authentication status
  await checkAuthStatus();
  
  // Set up event listeners
  setupEventListeners();
});

// Load saved API URL
async function loadApiUrl() {
  try {
    const data = await chrome.storage.local.get(STORAGE_KEYS.API_URL);
    const apiUrl = data[STORAGE_KEYS.API_URL] || 'https://api.plandex.ai';
    apiUrlInput.value = apiUrl;
  } catch (error) {
    console.error('Error loading API URL:', error);
    apiUrlInput.value = 'https://api.plandex.ai';
  }
}

// Check authentication status
async function checkAuthStatus() {
  try {
    // Send message to background script to check auth status
    const response = await chrome.runtime.sendMessage({ action: 'checkAuth' });
    
    if (response.authenticated) {
      // Load user info
      const userInfo = await loadUserInfo();
      
      // Update UI for authenticated state
      updateAuthenticatedUI(userInfo);
    } else {
      // Update UI for not authenticated state
      updateNotAuthenticatedUI();
    }
  } catch (error) {
    console.error('Error checking auth status:', error);
    updateNotAuthenticatedUI();
  }
}

// Load user info from storage
async function loadUserInfo() {
  try {
    const data = await chrome.storage.local.get(STORAGE_KEYS.USER_INFO);
    return data[STORAGE_KEYS.USER_INFO] || { name: 'User', email: 'user@example.com' };
  } catch (error) {
    console.error('Error loading user info:', error);
    return { name: 'User', email: 'user@example.com' };
  }
}

// Update UI for authenticated state
function updateAuthenticatedUI(userInfo) {
  // Update user info
  userName.textContent = userInfo.name || 'User';
  userEmail.textContent = userInfo.email || 'user@example.com';
  
  // Show authenticated section, hide not authenticated section
  notAuthenticatedSection.classList.add('hidden');
  authenticatedSection.classList.remove('hidden');
  
  // Update status
  contextMenuStatus.textContent = 'Enabled';
  authStatus.textContent = 'Authenticated';
  
  // Clear sign-in code input
  signInCode.value = '';
  
  // Clear error message
  authError.textContent = '';
  authError.classList.add('hidden');
}

// Update UI for not authenticated state
function updateNotAuthenticatedUI() {
  // Show not authenticated section, hide authenticated section
  notAuthenticatedSection.classList.remove('hidden');
  authenticatedSection.classList.add('hidden');
  
  // Update status
  contextMenuStatus.textContent = 'Disabled';
  authStatus.textContent = 'Not authenticated';
}

// Set up event listeners
function setupEventListeners() {
  // Auth form submit
  authForm.addEventListener('submit', handleSignIn);
  
  // Sign out button click
  signOutBtn.addEventListener('click', handleSignOut);
  
  // Settings form submit
  settingsForm.addEventListener('submit', handleSaveSettings);
}

// Handle sign in
async function handleSignIn(event) {
  event.preventDefault();
  
  const code = signInCode.value.trim();
  
  if (!code) {
    showAuthError('Please enter a sign-in code');
    return;
  }
  
  try {
    // Disable sign in button and show loading state
    signInBtn.disabled = true;
    signInBtn.textContent = 'Signing in...';
    
    // Send message to background script to authenticate
    const response = await chrome.runtime.sendMessage({
      action: 'authenticate',
      code: code
    });
    
    // Reset sign in button
    signInBtn.disabled = false;
    signInBtn.textContent = 'Sign In';
    
    if (response.success) {
      // Update UI for authenticated state
      updateAuthenticatedUI(response.user);
    } else {
      // Show error message
      showAuthError(response.error || 'Authentication failed');
    }
  } catch (error) {
    console.error('Error signing in:', error);
    
    // Reset sign in button
    signInBtn.disabled = false;
    signInBtn.textContent = 'Sign In';
    
    // Show error message
    showAuthError('An error occurred while signing in');
  }
}

// Show authentication error
function showAuthError(message) {
  authError.textContent = message;
  authError.classList.remove('hidden');
}

// Handle sign out
async function handleSignOut() {
  try {
    // Disable sign out button and show loading state
    signOutBtn.disabled = true;
    signOutBtn.textContent = 'Signing out...';
    
    // Send message to background script to logout
    const response = await chrome.runtime.sendMessage({ action: 'logout' });
    
    // Reset sign out button
    signOutBtn.disabled = false;
    signOutBtn.textContent = 'Sign Out';
    
    if (response.success) {
      // Update UI for not authenticated state
      updateNotAuthenticatedUI();
    } else {
      // Show error message
      console.error('Error signing out:', response.error);
    }
  } catch (error) {
    console.error('Error signing out:', error);
    
    // Reset sign out button
    signOutBtn.disabled = false;
    signOutBtn.textContent = 'Sign Out';
  }
}

// Handle save settings
async function handleSaveSettings(event) {
  event.preventDefault();
  
  const apiUrl = apiUrlInput.value.trim();
  
  if (!apiUrl) {
    showSettingsMessage('Please enter an API URL', false);
    return;
  }
  
  try {
    // Validate URL format
    new URL(apiUrl);
    
    // Disable save button and show loading state
    saveSettingsBtn.disabled = true;
    saveSettingsBtn.textContent = 'Saving...';
    
    // Send message to background script to update API URL
    const response = await chrome.runtime.sendMessage({
      action: 'updateApiUrl',
      apiUrl: apiUrl
    });
    
    // Reset save button
    saveSettingsBtn.disabled = false;
    saveSettingsBtn.textContent = 'Save Settings';
    
    if (response.success) {
      // Show success message
      showSettingsMessage('Settings saved successfully', true);
      
      // Check authentication status again
      await checkAuthStatus();
    } else {
      // Show error message
      showSettingsMessage(response.error || 'Failed to save settings', false);
    }
  } catch (error) {
    console.error('Error saving settings:', error);
    
    // Reset save button
    saveSettingsBtn.disabled = false;
    saveSettingsBtn.textContent = 'Save Settings';
    
    // Show error message
    showSettingsMessage('Please enter a valid URL', false);
  }
}

// Show settings message
function showSettingsMessage(message, isSuccess) {
  settingsMessage.textContent = message;
  settingsMessage.classList.remove('hidden');
  
  if (isSuccess) {
    settingsMessage.classList.add('success-message');
    settingsMessage.classList.remove('error-message');
  } else {
    settingsMessage.classList.add('error-message');
    settingsMessage.classList.remove('success-message');
  }
  
  // Hide message after 3 seconds
  setTimeout(() => {
    settingsMessage.classList.add('hidden');
  }, 3000);
}
