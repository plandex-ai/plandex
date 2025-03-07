/**
 * Plandex Extension Content Script
 * Handles selected text and notifications
 */

// Create a notification container if it doesn't exist
let notificationContainer = document.getElementById('plandex-notification-container');
if (!notificationContainer) {
  notificationContainer = document.createElement('div');
  notificationContainer.id = 'plandex-notification-container';
  notificationContainer.style.cssText = `
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 9999;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  `;
  document.body.appendChild(notificationContainer);
}

// Create and show a notification
function showNotification(message, type = 'info', duration = 3000) {
  // Create notification element
  const notification = document.createElement('div');
  notification.className = `plandex-notification plandex-notification-${type}`;
  notification.style.cssText = `
    padding: 12px 16px;
    margin-bottom: 10px;
    border-radius: 4px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    font-size: 14px;
    line-height: 1.5;
    display: flex;
    align-items: center;
    transition: opacity 0.3s ease-in-out, transform 0.3s ease-in-out;
    opacity: 0;
    transform: translateY(20px);
    max-width: 300px;
    word-break: break-word;
  `;

  // Set background color based on notification type
  if (type === 'success') {
    notification.style.backgroundColor = '#e6f4ea';
    notification.style.color = '#137333';
    notification.style.borderLeft = '4px solid #137333';
  } else if (type === 'error') {
    notification.style.backgroundColor = '#fdecea';
    notification.style.color = '#d93025';
    notification.style.borderLeft = '4px solid #d93025';
  } else if (type === 'warning') {
    notification.style.backgroundColor = '#fef7e0';
    notification.style.color = '#b06000';
    notification.style.borderLeft = '4px solid #b06000';
  } else {
    notification.style.backgroundColor = '#e8f0fe';
    notification.style.color = '#1a73e8';
    notification.style.borderLeft = '4px solid #1a73e8';
  }

  // Create icon based on notification type
  const icon = document.createElement('div');
  icon.style.cssText = `
    margin-right: 10px;
    width: 20px;
    height: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
  `;

  // Set icon content based on notification type
  if (type === 'success') {
    icon.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
        <polyline points="22 4 12 14.01 9 11.01"></polyline>
      </svg>
    `;
  } else if (type === 'error') {
    icon.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="15" y1="9" x2="9" y2="15"></line>
        <line x1="9" y1="9" x2="15" y2="15"></line>
      </svg>
    `;
  } else if (type === 'warning') {
    icon.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
        <line x1="12" y1="9" x2="12" y2="13"></line>
        <line x1="12" y1="17" x2="12.01" y2="17"></line>
      </svg>
    `;
  } else {
    icon.innerHTML = `
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="12" y1="8" x2="12" y2="12"></line>
        <line x1="12" y1="16" x2="12.01" y2="16"></line>
      </svg>
    `;
  }

  // Create message element
  const messageElement = document.createElement('div');
  messageElement.textContent = message;

  // Create close button
  const closeButton = document.createElement('div');
  closeButton.style.cssText = `
    margin-left: 10px;
    cursor: pointer;
    opacity: 0.7;
    transition: opacity 0.2s;
  `;
  closeButton.innerHTML = `
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <line x1="18" y1="6" x2="6" y2="18"></line>
      <line x1="6" y1="6" x2="18" y2="18"></line>
    </svg>
  `;
  closeButton.addEventListener('mouseover', () => {
    closeButton.style.opacity = '1';
  });
  closeButton.addEventListener('mouseout', () => {
    closeButton.style.opacity = '0.7';
  });
  closeButton.addEventListener('click', () => {
    removeNotification(notification);
  });

  // Append elements to notification
  notification.appendChild(icon);
  notification.appendChild(messageElement);
  notification.appendChild(closeButton);

  // Append notification to container
  notificationContainer.appendChild(notification);

  // Show notification with animation
  setTimeout(() => {
    notification.style.opacity = '1';
    notification.style.transform = 'translateY(0)';
  }, 10);

  // Remove notification after duration
  if (duration > 0) {
    setTimeout(() => {
      removeNotification(notification);
    }, duration);
  }

  return notification;
}

// Remove a notification with animation
function removeNotification(notification) {
  notification.style.opacity = '0';
  notification.style.transform = 'translateY(20px)';
  
  setTimeout(() => {
    if (notification.parentNode) {
      notification.parentNode.removeChild(notification);
    }
  }, 300);
}

// Listen for messages from the background script or popup
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'showNotification') {
    showNotification(message.message, message.type || 'info', message.duration || 3000);
    sendResponse({ success: true });
    return true;
  }
});

// Function to get selected text
function getSelectedText() {
  return window.getSelection().toString().trim();
}

// Function to check if text is selected
function hasSelectedText() {
  return getSelectedText().length > 0;
}

// Listen for context menu clicks
document.addEventListener('contextmenu', (event) => {
  // Check if text is selected
  const selectedText = getSelectedText();
  
  // Store selected text in local storage for context menu handler
  if (selectedText) {
    chrome.storage.local.set({ 'tempSelectedText': selectedText });
  } else {
    chrome.storage.local.remove('tempSelectedText');
  }
});

// Listen for keyboard shortcuts
document.addEventListener('keydown', (event) => {
  // Check for Ctrl+Shift+P (Windows/Linux) or Command+Shift+P (Mac)
  if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === 'p') {
    const selectedText = getSelectedText();
    
    if (selectedText) {
      // Store selected text
      chrome.storage.local.set({
        'selectedText': selectedText,
        'sourceUrl': window.location.href
      });
      
      // Show notification
      showNotification('Opening Plandex plan selector...', 'info');
      
      // Send message to background script to open plan selector
      chrome.runtime.sendMessage({
        action: 'openPlanSelector',
        selectedText: selectedText,
        sourceUrl: window.location.href
      });
    } else {
      showNotification('No text selected', 'warning');
    }
  }
});

// Function to handle text selection changes
function handleSelectionChange() {
  // This function can be used to update UI elements based on selection
  // For now, we'll just use it to update the context menu visibility
  const hasSelection = hasSelectedText();
  
  // We can't directly update the context menu from here,
  // but we can store the selection state for the background script
  chrome.storage.local.set({ 'hasSelectedText': hasSelection });
}

// Listen for selection changes
document.addEventListener('selectionchange', handleSelectionChange);

// Initialize
console.log('Plandex content script loaded');
