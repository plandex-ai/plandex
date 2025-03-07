// DOM Elements
const searchInput = document.getElementById('search-input');
const plansList = document.getElementById('plans-list');
const noPlansMessage = document.getElementById('no-plans');
const errorMessage = document.getElementById('error-message');
const selectedTextPreview = document.getElementById('selected-text-preview');
const sendButton = document.getElementById('send-btn');
const cancelButton = document.getElementById('cancel-btn');

// State
let plans = [];
let filteredPlans = [];
let selectedPlan = null;
let selectedText = '';
let sourceTab = null;
let sourceUrl = '';
let fuseInstance = null;

// Constants
const STORAGE_KEYS = {
  AUTH_TOKEN: "plandex_auth_token",
  ORG_ID: "plandex_org_id",
  API_URL: "plandex_api_url"
};

// Initialize the popup
document.addEventListener('DOMContentLoaded', async () => {
  // Load selected text and source info
  await loadSelectedText();
  
  // Load plans
  await loadPlans();
  
  // Set up event listeners
  setupEventListeners();
  
  // Focus search input
  searchInput.focus();
});

// Load selected text and source info from storage
async function loadSelectedText() {
  try {
    const data = await chrome.storage.local.get(['selectedText', 'sourceTab', 'sourceUrl']);
    
    selectedText = data.selectedText || '';
    sourceTab = data.sourceTab || null;
    sourceUrl = data.sourceUrl || '';
    
    // Update selected text preview
    updateSelectedTextPreview();
  } catch (error) {
    console.error('Error loading selected text:', error);
    showError('Failed to load selected text');
  }
}

// Update selected text preview
function updateSelectedTextPreview() {
  // Truncate text if too long
  const maxLength = 200;
  let previewText = selectedText;
  
  if (selectedText.length > maxLength) {
    previewText = selectedText.substring(0, maxLength) + '...';
  }
  
  selectedTextPreview.textContent = previewText || 'No text selected';
}

// Load plans from API
async function loadPlans() {
  try {
    showLoading();
    
    // Get auth token and org ID
    const tokenData = await chrome.storage.local.get(STORAGE_KEYS.AUTH_TOKEN);
    const orgIdData = await chrome.storage.local.get(STORAGE_KEYS.ORG_ID);
    const apiUrlData = await chrome.storage.local.get(STORAGE_KEYS.API_URL);
    
    const token = tokenData[STORAGE_KEYS.AUTH_TOKEN];
    const orgId = orgIdData[STORAGE_KEYS.ORG_ID];
    const apiUrl = apiUrlData[STORAGE_KEYS.API_URL] || 'https://api.plandex.ai';
    
    if (!token || !orgId) {
      throw new Error('Not authenticated');
    }
    
    // Fetch plans from API
    const response = await fetch(`${apiUrl}/plans`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-Org-Id': orgId
      }
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch plans');
    }
    
    const data = await response.json();
    plans = data.plans || [];
    
    // Initialize Fuse.js for fuzzy search
    initializeFuseSearch();
    
    // Update plans list
    filteredPlans = [...plans];
    updatePlansList();
    
    hideLoading();
  } catch (error) {
    console.error('Error loading plans:', error);
    hideLoading();
    showError('Failed to load plans: ' + error.message);
  }
}

// Initialize Fuse.js for fuzzy search
function initializeFuseSearch() {
  const options = {
    keys: ['name', 'description'],
    threshold: 0.3,
    ignoreLocation: true
  };
  
  fuseInstance = new Fuse(plans, options);
}

// Update plans list in the UI
function updatePlansList() {
  // Clear current list
  plansList.innerHTML = '';
  
  if (filteredPlans.length === 0) {
    noPlansMessage.classList.remove('hidden');
    return;
  }
  
  noPlansMessage.classList.add('hidden');
  
  // Create plan items
  filteredPlans.forEach(plan => {
    const planItem = document.createElement('div');
    planItem.className = 'plan-item';
    planItem.dataset.planId = plan.id;
    
    if (selectedPlan && selectedPlan.id === plan.id) {
      planItem.classList.add('selected');
    }
    
    const planName = document.createElement('div');
    planName.className = 'plan-name';
    planName.textContent = plan.name;
    
    const planDescription = document.createElement('div');
    planDescription.className = 'plan-description';
    planDescription.textContent = plan.description || 'No description';
    
    planItem.appendChild(planName);
    planItem.appendChild(planDescription);
    
    // Add click event
    planItem.addEventListener('click', () => {
      selectPlan(plan);
    });
    
    plansList.appendChild(planItem);
  });
}

// Select a plan
function selectPlan(plan) {
  selectedPlan = plan;
  
  // Update UI
  const planItems = plansList.querySelectorAll('.plan-item');
  planItems.forEach(item => {
    if (item.dataset.planId === plan.id) {
      item.classList.add('selected');
    } else {
      item.classList.remove('selected');
    }
  });
  
  // Enable send button
  sendButton.disabled = false;
}

// Search plans
function searchPlans(query) {
  if (!query) {
    filteredPlans = [...plans];
  } else {
    const results = fuseInstance.search(query);
    filteredPlans = results.map(result => result.item);
  }
  
  updatePlansList();
}

// Show loading indicator
function showLoading() {
  plansList.innerHTML = `
    <div class="loading-indicator">
      <div class="spinner"></div>
      <p>Loading plans...</p>
    </div>
  `;
  noPlansMessage.classList.add('hidden');
  errorMessage.classList.add('hidden');
}

// Hide loading indicator
function hideLoading() {
  const loadingIndicator = plansList.querySelector('.loading-indicator');
  if (loadingIndicator) {
    loadingIndicator.remove();
  }
}

// Show error message
function showError(message) {
  errorMessage.textContent = message;
  errorMessage.classList.remove('hidden');
}

// Send selected text to plan
async function sendTextToPlan() {
  if (!selectedPlan || !selectedText) {
    return;
  }
  
  try {
    // Disable send button and show loading state
    sendButton.disabled = true;
    sendButton.textContent = 'Sending...';
    
    // Get current branch for the plan
    const tokenData = await chrome.storage.local.get(STORAGE_KEYS.AUTH_TOKEN);
    const orgIdData = await chrome.storage.local.get(STORAGE_KEYS.ORG_ID);
    const apiUrlData = await chrome.storage.local.get(STORAGE_KEYS.API_URL);
    
    const token = tokenData[STORAGE_KEYS.AUTH_TOKEN];
    const orgId = orgIdData[STORAGE_KEYS.ORG_ID];
    const apiUrl = apiUrlData[STORAGE_KEYS.API_URL] || 'https://api.plandex.ai';
    
    if (!token || !orgId) {
      throw new Error('Not authenticated');
    }
    
    // Get current branch
    const branchResponse = await fetch(`${apiUrl}/plans/${selectedPlan.id}/current_branch`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'X-Org-Id': orgId
      }
    });
    
    if (!branchResponse.ok) {
      throw new Error('Failed to fetch current branch');
    }
    
    const branchData = await branchResponse.json();
    const branch = branchData.branch || 'main';
    
    // Create context name with source URL if available
    let contextName = 'Selected text from web';
    if (sourceUrl) {
      contextName = `Selected text from ${new URL(sourceUrl).hostname}`;
    }
    
    // Send text to plan
    const response = await chrome.runtime.sendMessage({
      action: 'sendToPlan',
      planId: selectedPlan.id,
      branch: branch,
      text: selectedText,
      name: contextName
    });
    
    if (!response.success) {
      throw new Error(response.error || 'Failed to send text to plan');
    }
    
    // Show success message in the source tab
    if (sourceTab) {
      chrome.tabs.sendMessage(sourceTab, {
        action: 'showNotification',
        type: 'success',
        message: `Text sent to plan: ${selectedPlan.name}`
      });
    }
    
    // Close the popup
    window.close();
  } catch (error) {
    console.error('Error sending text to plan:', error);
    
    // Reset send button
    sendButton.disabled = false;
    sendButton.textContent = 'Send to Plan';
    
    // Show error message
    showError('Failed to send text: ' + error.message);
    
    // Show error notification in the source tab
    if (sourceTab) {
      chrome.tabs.sendMessage(sourceTab, {
        action: 'showNotification',
        type: 'error',
        message: 'Failed to send text to plan'
      });
    }
  }
}

// Set up event listeners
function setupEventListeners() {
  // Search input
  searchInput.addEventListener('input', () => {
    searchPlans(searchInput.value.trim());
  });
  
  // Send button
  sendButton.addEventListener('click', sendTextToPlan);
  
  // Cancel button
  cancelButton.addEventListener('click', () => {
    window.close();
  });
  
  // Keyboard navigation
  document.addEventListener('keydown', (event) => {
    // Enter key to send
    if (event.key === 'Enter' && selectedPlan) {
      sendTextToPlan();
    }
    
    // Escape key to cancel
    if (event.key === 'Escape') {
      window.close();
    }
    
    // Arrow keys to navigate plans
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      
      if (filteredPlans.length === 0) {
        return;
      }
      
      const currentIndex = selectedPlan 
        ? filteredPlans.findIndex(plan => plan.id === selectedPlan.id) 
        : -1;
      
      let newIndex;
      
      if (event.key === 'ArrowDown') {
        newIndex = currentIndex < filteredPlans.length - 1 ? currentIndex + 1 : 0;
      } else {
        newIndex = currentIndex > 0 ? currentIndex - 1 : filteredPlans.length - 1;
      }
      
      selectPlan(filteredPlans[newIndex]);
      
      // Scroll to selected plan
      const selectedElement = plansList.querySelector('.selected');
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest' });
      }
    }
  });
}
