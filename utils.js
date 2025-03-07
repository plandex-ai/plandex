/**
 * Plandex Extension Utilities
 * Contains helper functions for common tasks, error handling, storage management, and notifications
 */

// Constants
const STORAGE_KEYS = {
  AUTH_TOKEN: "plandex_auth_token",
  ORG_ID: "plandex_org_id",
  API_URL: "plandex_api_url",
  USER_INFO: "plandex_user_info",
  SELECTED_TEXT: "selectedText",
  SOURCE_TAB: "sourceTab",
  SOURCE_URL: "sourceUrl",
  HAS_SELECTED_TEXT: "hasSelectedText"
};

/**
 * Storage utilities
 */
const storage = {
  /**
   * Get a value from storage
   * @param {string} key - Storage key
   * @param {any} defaultValue - Default value if key doesn't exist
   * @returns {Promise<any>} - Value from storage or default value
   */
  async get(key, defaultValue = null) {
    try {
      const data = await chrome.storage.local.get(key);
      return data[key] !== undefined ? data[key] : defaultValue;
    } catch (error) {
      console.error(`Error getting ${key} from storage:`, error);
      return defaultValue;
    }
  },

  /**
   * Set a value in storage
   * @param {string} key - Storage key
   * @param {any} value - Value to store
   * @returns {Promise<boolean>} - Success status
   */
  async set(key, value) {
    try {
      await chrome.storage.local.set({ [key]: value });
      return true;
    } catch (error) {
      console.error(`Error setting ${key} in storage:`, error);
      return false;
    }
  },

  /**
   * Remove a value from storage
   * @param {string} key - Storage key
   * @returns {Promise<boolean>} - Success status
   */
  async remove(key) {
    try {
      await chrome.storage.local.remove(key);
      return true;
    } catch (error) {
      console.error(`Error removing ${key} from storage:`, error);
      return false;
    }
  },

  /**
   * Clear all storage
   * @returns {Promise<boolean>} - Success status
   */
  async clear() {
    try {
      await chrome.storage.local.clear();
      return true;
    } catch (error) {
      console.error('Error clearing storage:', error);
      return false;
    }
  },

  /**
   * Get authentication token from storage
   * @returns {Promise<string|null>} - Authentication token or null
   */
  async getAuthToken() {
    return await this.get(STORAGE_KEYS.AUTH_TOKEN, null);
  },

  /**
   * Get organization ID from storage
   * @returns {Promise<string|null>} - Organization ID or null
   */
  async getOrgId() {
    return await this.get(STORAGE_KEYS.ORG_ID, null);
  },

  /**
   * Get API URL from storage
   * @returns {Promise<string>} - API URL or default
   */
  async getApiUrl() {
    return await this.get(STORAGE_KEYS.API_URL, 'https://api.plandex.ai');
  },

  /**
   * Get user info from storage
   * @returns {Promise<Object|null>} - User info or null
   */
  async getUserInfo() {
    return await this.get(STORAGE_KEYS.USER_INFO, null);
  },

  /**
   * Check if user is authenticated
   * @returns {Promise<boolean>} - Authentication status
   */
  async isAuthenticated() {
    const token = await this.getAuthToken();
    const orgId = await this.getOrgId();
    return !!(token && orgId);
  },

  /**
   * Store selected text and source info
   * @param {string} text - Selected text
   * @param {number} tabId - Source tab ID
   * @param {string} url - Source URL
   * @returns {Promise<boolean>} - Success status
   */
  async storeSelectedText(text, tabId, url) {
    try {
      await chrome.storage.local.set({
        [STORAGE_KEYS.SELECTED_TEXT]: text,
        [STORAGE_KEYS.SOURCE_TAB]: tabId,
        [STORAGE_KEYS.SOURCE_URL]: url
      });
      return true;
    } catch (error) {
      console.error('Error storing selected text:', error);
      return false;
    }
  },

  /**
   * Get selected text and source info
   * @returns {Promise<Object>} - Selected text and source info
   */
  async getSelectedText() {
    try {
      const data = await chrome.storage.local.get([
        STORAGE_KEYS.SELECTED_TEXT,
        STORAGE_KEYS.SOURCE_TAB,
        STORAGE_KEYS.SOURCE_URL
      ]);
      
      return {
        text: data[STORAGE_KEYS.SELECTED_TEXT] || '',
        tabId: data[STORAGE_KEYS.SOURCE_TAB] || null,
        url: data[STORAGE_KEYS.SOURCE_URL] || ''
      };
    } catch (error) {
      console.error('Error getting selected text:', error);
      return { text: '', tabId: null, url: '' };
    }
  }
};

/**
 * Error handling utilities
 */
const errorHandler = {
  /**
   * Handle API errors
   * @param {Error} error - Error object
   * @param {string} operation - Operation description
   * @returns {Object} - Formatted error object
   */
  handleApiError(error, operation) {
    console.error(`API Error (${operation}):`, error);
    
    let message = 'An unexpected error occurred';
    let status = 500;
    
    if (error.response) {
      // Server responded with an error status
      status = error.response.status;
      
      switch (status) {
        case 401:
          message = 'Authentication failed. Please sign in again.';
          break;
        case 403:
          message = 'You do not have permission to perform this action.';
          break;
        case 404:
          message = 'The requested resource was not found.';
          break;
        case 429:
          message = 'Too many requests. Please try again later.';
          break;
        default:
          message = error.response.data?.message || 'Server error. Please try again later.';
      }
    } else if (error.request) {
      // Request was made but no response received
      message = 'No response from server. Please check your connection.';
    } else {
      // Error setting up the request
      message = error.message || 'Error setting up the request.';
    }
    
    return {
      success: false,
      status,
      message,
      operation,
      originalError: error
    };
  },

  /**
   * Handle authentication errors
   * @param {Error} error - Error object
   * @returns {Object} - Formatted error object
   */
  handleAuthError(error) {
    console.error('Authentication Error:', error);
    
    // Clear authentication data
    storage.remove(STORAGE_KEYS.AUTH_TOKEN);
    storage.remove(STORAGE_KEYS.ORG_ID);
    storage.remove(STORAGE_KEYS.USER_INFO);
    
    return this.handleApiError(error, 'authentication');
  },

  /**
   * Format error for display
   * @param {Error|string} error - Error object or message
   * @returns {string} - Formatted error message
   */
  formatErrorMessage(error) {
    if (typeof error === 'string') {
      return error;
    }
    
    if (error.message) {
      return error.message;
    }
    
    return 'An unexpected error occurred';
  }
};

/**
 * Notification utilities
 */
const notifications = {
  /**
   * Show a notification in a tab
   * @param {number} tabId - Tab ID
   * @param {string} message - Notification message
   * @param {string} type - Notification type (success, error, warning, info)
   * @param {number} duration - Duration in milliseconds
   * @returns {Promise<boolean>} - Success status
   */
  async showInTab(tabId, message, type = 'info', duration = 3000) {
    try {
      await chrome.tabs.sendMessage(tabId, {
        action: 'showNotification',
        message,
        type,
        duration
      });
      return true;
    } catch (error) {
      console.error('Error showing notification:', error);
      return false;
    }
  },

  /**
   * Show a Chrome notification
   * @param {string} title - Notification title
   * @param {string} message - Notification message
   * @param {string} type - Notification type (basic, image, list, progress)
   * @returns {Promise<string>} - Notification ID
   */
  async showChromeNotification(title, message, type = 'basic') {
    try {
      const notificationId = 'plandex-' + Date.now();
      
      await chrome.notifications.create(notificationId, {
        type: type,
        iconUrl: chrome.runtime.getURL('icons/icon48.png'),
        title: title,
        message: message
      });
      
      return notificationId;
    } catch (error) {
      console.error('Error showing Chrome notification:', error);
      return null;
    }
  }
};

/**
 * URL utilities
 */
const urlUtils = {
  /**
   * Get domain from URL
   * @param {string} url - URL
   * @returns {string} - Domain
   */
  getDomain(url) {
    try {
      const urlObj = new URL(url);
      return urlObj.hostname;
    } catch (error) {
      console.error('Error parsing URL:', error);
      return '';
    }
  },

  /**
   * Check if URL is valid
   * @param {string} url - URL
   * @returns {boolean} - Validity status
   */
  isValidUrl(url) {
    try {
      new URL(url);
      return true;
    } catch (error) {
      return false;
    }
  },

  /**
   * Ensure URL has protocol
   * @param {string} url - URL
   * @returns {string} - URL with protocol
   */
  ensureProtocol(url) {
    if (!url) return '';
    
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
      return 'https://' + url;
    }
    
    return url;
  }
};

/**
 * Text utilities
 */
const textUtils = {
  /**
   * Truncate text
   * @param {string} text - Text to truncate
   * @param {number} maxLength - Maximum length
   * @param {string} suffix - Suffix to add
   * @returns {string} - Truncated text
   */
  truncate(text, maxLength = 100, suffix = '...') {
    if (!text) return '';
    
    if (text.length <= maxLength) {
      return text;
    }
    
    return text.substring(0, maxLength) + suffix;
  },

  /**
   * Create context name from URL
   * @param {string} url - URL
   * @returns {string} - Context name
   */
  createContextName(url) {
    if (!url) {
      return 'Selected text from web';
    }
    
    try {
      const domain = urlUtils.getDomain(url);
      return `Selected text from ${domain}`;
    } catch (error) {
      return 'Selected text from web';
    }
  },

  /**
   * Sanitize text for display
   * @param {string} text - Text to sanitize
   * @returns {string} - Sanitized text
   */
  sanitize(text) {
    if (!text) return '';
    
    // Replace HTML tags with their entities
    return text
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }
};

/**
 * DOM utilities
 */
const domUtils = {
  /**
   * Create element with attributes and children
   * @param {string} tag - Element tag
   * @param {Object} attributes - Element attributes
   * @param {Array|string|Node} children - Element children
   * @returns {HTMLElement} - Created element
   */
  createElement(tag, attributes = {}, children = []) {
    const element = document.createElement(tag);
    
    // Set attributes
    Object.entries(attributes).forEach(([key, value]) => {
      if (key === 'style' && typeof value === 'object') {
        Object.entries(value).forEach(([styleKey, styleValue]) => {
          element.style[styleKey] = styleValue;
        });
      } else if (key === 'className') {
        element.className = value;
      } else if (key === 'dataset' && typeof value === 'object') {
        Object.entries(value).forEach(([dataKey, dataValue]) => {
          element.dataset[dataKey] = dataValue;
        });
      } else if (key.startsWith('on') && typeof value === 'function') {
        const eventName = key.substring(2).toLowerCase();
        element.addEventListener(eventName, value);
      } else {
        element.setAttribute(key, value);
      }
    });
    
    // Add children
    if (children) {
      if (Array.isArray(children)) {
        children.forEach(child => {
          if (child) {
            if (typeof child === 'string') {
              element.appendChild(document.createTextNode(child));
            } else {
              element.appendChild(child);
            }
          }
        });
      } else if (typeof children === 'string') {
        element.textContent = children;
      } else {
        element.appendChild(children);
      }
    }
    
    return element;
  },

  /**
   * Show element
   * @param {HTMLElement} element - Element to show
   */
  showElement(element) {
    if (element) {
      element.classList.remove('hidden');
    }
  },

  /**
   * Hide element
   * @param {HTMLElement} element - Element to hide
   */
  hideElement(element) {
    if (element) {
      element.classList.add('hidden');
    }
  },

  /**
   * Toggle element visibility
   * @param {HTMLElement} element - Element to toggle
   * @param {boolean} show - Show or hide
   */
  toggleElement(element, show) {
    if (element) {
      if (show) {
        this.showElement(element);
      } else {
        this.hideElement(element);
      }
    }
  },

  /**
   * Create a notification element
   * @param {string} message - Notification message
   * @param {string} type - Notification type (success, error, warning, info)
   * @param {number} duration - Duration in milliseconds
   * @returns {HTMLElement} - Notification element
   */
  createNotification(message, type = 'info', duration = 3000) {
    // Create container if it doesn't exist
    let container = document.getElementById('plandex-notification-container');
    if (!container) {
      container = this.createElement('div', {
        id: 'plandex-notification-container',
        style: {
          position: 'fixed',
          bottom: '20px',
          right: '20px',
          zIndex: '9999',
          fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif'
        }
      });
      document.body.appendChild(container);
    }
    
    // Create notification
    const notification = this.createElement('div', {
      className: `plandex-notification plandex-notification-${type}`,
      style: {
        padding: '12px 16px',
        marginBottom: '10px',
        borderRadius: '4px',
        boxShadow: '0 2px 10px rgba(0, 0, 0, 0.1)',
        fontSize: '14px',
        lineHeight: '1.5',
        display: 'flex',
        alignItems: 'center',
        transition: 'opacity 0.3s ease-in-out, transform 0.3s ease-in-out',
        opacity: '0',
        transform: 'translateY(20px)',
        maxWidth: '300px',
        wordBreak: 'break-word'
      }
    });
    
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
    
    // Create icon
    let iconSvg = '';
    if (type === 'success') {
      iconSvg = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
          <polyline points="22 4 12 14.01 9 11.01"></polyline>
        </svg>
      `;
    } else if (type === 'error') {
      iconSvg = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="15" y1="9" x2="9" y2="15"></line>
          <line x1="9" y1="9" x2="15" y2="15"></line>
        </svg>
      `;
    } else if (type === 'warning') {
      iconSvg = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path>
          <line x1="12" y1="9" x2="12" y2="13"></line>
          <line x1="12" y1="17" x2="12.01" y2="17"></line>
        </svg>
      `;
    } else {
      iconSvg = `
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
      `;
    }
    
    const icon = this.createElement('div', {
      style: {
        marginRight: '10px'
      },
      innerHTML: iconSvg
    });
    
    // Create message
    const messageElement = this.createElement('div', {}, message);
    
    // Create close button
    const closeButton = this.createElement('div', {
      style: {
        marginLeft: '10px',
        cursor: 'pointer',
        opacity: '0.7',
        transition: 'opacity 0.2s'
      },
      innerHTML: `
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      `,
      onmouseover: () => { closeButton.style.opacity = '1'; },
      onmouseout: () => { closeButton.style.opacity = '0.7'; },
      onclick: () => { this.removeNotification(notification); }
    });
    
    // Append elements
    notification.appendChild(icon);
    notification.appendChild(messageElement);
    notification.appendChild(closeButton);
    
    // Append to container
    container.appendChild(notification);
    
    // Show notification with animation
    setTimeout(() => {
      notification.style.opacity = '1';
      notification.style.transform = 'translateY(0)';
    }, 10);
    
    // Remove notification after duration
    if (duration > 0) {
      setTimeout(() => {
        this.removeNotification(notification);
      }, duration);
    }
    
    return notification;
  },

  /**
   * Remove a notification with animation
   * @param {HTMLElement} notification - Notification element
   */
  removeNotification(notification) {
    if (notification) {
      notification.style.opacity = '0';
      notification.style.transform = 'translateY(20px)';
      
      setTimeout(() => {
        if (notification.parentNode) {
          notification.parentNode.removeChild(notification);
        }
      }, 300);
    }
  }
};

// Export utilities
export {
  STORAGE_KEYS,
  storage,
  errorHandler,
  notifications,
  urlUtils,
  textUtils,
  domUtils
};
