// Constants
const API_URL = "https://api.plandex.ai";
const CONTEXT_MENU_ID = "send-to-plandex";
const STORAGE_KEYS = {
  AUTH_TOKEN: "plandex_auth_token",
  ORG_ID: "plandex_org_id",
  API_URL: "plandex_api_url",
  USER_INFO: "plandex_user_info"
};

// Initialize the extension
function initializeExtension() {
  // Create context menu item
  createContextMenu();
  
  // Set up message listeners
  setupMessageListeners();
  
  // Check authentication status and update badge
  checkAuthStatus();
}

// Create the context menu item
function createContextMenu() {
  chrome.contextMenus.create({
    id: CONTEXT_MENU_ID,
    title: "Send to Plandex",
    contexts: ["selection"],
    visible: false // Hide until authenticated
  });
  
  // Add listener for context menu clicks
  chrome.contextMenus.onClicked.addListener((info, tab) => {
    if (info.menuItemId === CONTEXT_MENU_ID) {
      handleContextMenuClick(info, tab);
    }
  });
}

// Handle context menu click
async function handleContextMenuClick(info, tab) {
  if (info.selectionText) {
    // Open plan selector popup
    chrome.windows.create({
      url: chrome.runtime.getURL("planSelector.html"),
      type: "popup",
      width: 400,
      height: 500,
      focused: true
    }, (window) => {
      // Store the selected text and tab info for later use
      chrome.storage.local.set({
        "selectedText": info.selectionText,
        "sourceTab": tab.id,
        "sourceUrl": tab.url
      });
    });
  }
}

// Set up message listeners for communication between different parts of the extension
function setupMessageListeners() {
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    switch (message.action) {
      case "authenticate":
        handleAuthentication(message.code)
          .then(result => sendResponse(result))
          .catch(error => sendResponse({ success: false, error: error.message }));
        return true; // Indicates async response
        
      case "checkAuth":
        checkAuthStatus()
          .then(status => sendResponse(status))
          .catch(error => sendResponse({ authenticated: false, error: error.message }));
        return true;
        
      case "logout":
        handleLogout()
          .then(() => sendResponse({ success: true }))
          .catch(error => sendResponse({ success: false, error: error.message }));
        return true;
        
      case "sendToPlan":
        sendTextToPlan(message.planId, message.branch, message.text)
          .then(result => sendResponse(result))
          .catch(error => sendResponse({ success: false, error: error.message }));
        return true;
        
      case "updateApiUrl":
        updateApiUrl(message.apiUrl)
          .then(() => sendResponse({ success: true }))
          .catch(error => sendResponse({ success: false, error: error.message }));
        return true;
    }
  });
}

// Handle authentication with sign-in code
async function handleAuthentication(code) {
  try {
    const apiUrl = await getApiUrl();
    
    const response = await fetch(`${apiUrl}/accounts/sign_in`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        pin: code,
        isSignInCode: true
      })
    });
    
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || "Authentication failed");
    }
    
    const data = await response.json();
    
    // Store authentication data
    await chrome.storage.local.set({
      [STORAGE_KEYS.AUTH_TOKEN]: data.token,
      [STORAGE_KEYS.ORG_ID]: data.orgId,
      [STORAGE_KEYS.USER_INFO]: {
        name: data.name,
        email: data.email
      }
    });
    
    // Update context menu visibility
    chrome.contextMenus.update(CONTEXT_MENU_ID, {
      visible: true
    });
    
    // Update badge
    chrome.action.setBadgeText({ text: "✓" });
    chrome.action.setBadgeBackgroundColor({ color: "#4CAF50" });
    
    return { success: true, user: data };
  } catch (error) {
    console.error("Authentication error:", error);
    return { success: false, error: error.message };
  }
}

// Check authentication status
async function checkAuthStatus() {
  try {
    const token = await chrome.storage.local.get(STORAGE_KEYS.AUTH_TOKEN);
    const orgId = await chrome.storage.local.get(STORAGE_KEYS.ORG_ID);
    
    const isAuthenticated = token[STORAGE_KEYS.AUTH_TOKEN] && orgId[STORAGE_KEYS.ORG_ID];
    
    // Update context menu visibility
    chrome.contextMenus.update(CONTEXT_MENU_ID, {
      visible: isAuthenticated
    });
    
    // Update badge
    if (isAuthenticated) {
      chrome.action.setBadgeText({ text: "✓" });
      chrome.action.setBadgeBackgroundColor({ color: "#4CAF50" });
    } else {
      chrome.action.setBadgeText({ text: "×" });
      chrome.action.setBadgeBackgroundColor({ color: "#F44336" });
    }
    
    return { authenticated: isAuthenticated };
  } catch (error) {
    console.error("Error checking auth status:", error);
    return { authenticated: false, error: error.message };
  }
}

// Handle logout
async function handleLogout() {
  try {
    // Clear authentication data
    await chrome.storage.local.remove([
      STORAGE_KEYS.AUTH_TOKEN,
      STORAGE_KEYS.ORG_ID,
      STORAGE_KEYS.USER_INFO
    ]);
    
    // Update context menu visibility
    chrome.contextMenus.update(CONTEXT_MENU_ID, {
      visible: false
    });
    
    // Update badge
    chrome.action.setBadgeText({ text: "×" });
    chrome.action.setBadgeBackgroundColor({ color: "#F44336" });
    
    return { success: true };
  } catch (error) {
    console.error("Logout error:", error);
    return { success: false, error: error.message };
  }
}

// Send text to a Plandex plan
async function sendTextToPlan(planId, branch, text) {
  try {
    const token = await getAuthToken();
    const orgId = await getOrgId();
    const apiUrl = await getApiUrl();
    
    if (!token || !orgId) {
      throw new Error("Not authenticated");
    }
    
    const contextParams = {
      contextType: "note",
      name: "Selected text from web",
      body: text,
      autoLoaded: false
    };
    
    const response = await fetch(`${apiUrl}/plans/${planId}/${branch}/context`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${token}`,
        "X-Org-Id": orgId
      },
      body: JSON.stringify([contextParams])
    });
    
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.message || "Failed to send text to plan");
    }
    
    return { success: true };
  } catch (error) {
    console.error("Error sending text to plan:", error);
    return { success: false, error: error.message };
  }
}

// Update API URL
async function updateApiUrl(apiUrl) {
  try {
    await chrome.storage.local.set({
      [STORAGE_KEYS.API_URL]: apiUrl
    });
    
    return { success: true };
  } catch (error) {
    console.error("Error updating API URL:", error);
    return { success: false, error: error.message };
  }
}

// Helper function to get auth token
async function getAuthToken() {
  const data = await chrome.storage.local.get(STORAGE_KEYS.AUTH_TOKEN);
  return data[STORAGE_KEYS.AUTH_TOKEN];
}

// Helper function to get org ID
async function getOrgId() {
  const data = await chrome.storage.local.get(STORAGE_KEYS.ORG_ID);
  return data[STORAGE_KEYS.ORG_ID];
}

// Helper function to get API URL
async function getApiUrl() {
  const data = await chrome.storage.local.get(STORAGE_KEYS.API_URL);
  return data[STORAGE_KEYS.API_URL] || API_URL;
}

// Initialize the extension when the service worker starts
initializeExtension();
