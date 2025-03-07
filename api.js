/**
 * Plandex API Service
 * Handles API requests, authentication, and data fetching
 */

// Default API URL
const DEFAULT_API_URL = "https://api.plandex.ai";

// Storage keys
const STORAGE_KEYS = {
  AUTH_TOKEN: "plandex_auth_token",
  ORG_ID: "plandex_org_id",
  API_URL: "plandex_api_url",
  USER_INFO: "plandex_user_info"
};

/**
 * PlandexAPI class for handling API requests
 */
class PlandexAPI {
  constructor() {
    this.apiUrl = DEFAULT_API_URL;
    this.token = null;
    this.orgId = null;
    this.initialized = false;
  }

  /**
   * Initialize the API service
   * Loads saved API URL, auth token, and org ID from storage
   */
  async initialize() {
    try {
      // Load API URL from storage
      const apiUrlData = await chrome.storage.local.get(STORAGE_KEYS.API_URL);
      this.apiUrl = apiUrlData[STORAGE_KEYS.API_URL] || DEFAULT_API_URL;

      // Load auth token from storage
      const tokenData = await chrome.storage.local.get(STORAGE_KEYS.AUTH_TOKEN);
      this.token = tokenData[STORAGE_KEYS.AUTH_TOKEN] || null;

      // Load org ID from storage
      const orgIdData = await chrome.storage.local.get(STORAGE_KEYS.ORG_ID);
      this.orgId = orgIdData[STORAGE_KEYS.ORG_ID] || null;

      this.initialized = true;
      return true;
    } catch (error) {
      console.error("Error initializing API service:", error);
      return false;
    }
  }

  /**
   * Check if the user is authenticated
   */
  isAuthenticated() {
    return !!(this.token && this.orgId);
  }

  /**
   * Get request headers with authentication
   */
  getHeaders(includeAuth = true) {
    const headers = {
      "Content-Type": "application/json"
    };

    if (includeAuth && this.token && this.orgId) {
      headers["Authorization"] = `Bearer ${this.token}`;
      headers["X-Org-Id"] = this.orgId;
    }

    return headers;
  }

  /**
   * Make an API request
   * @param {string} endpoint - API endpoint
   * @param {Object} options - Request options
   * @param {boolean} requiresAuth - Whether the request requires authentication
   */
  async request(endpoint, options = {}, requiresAuth = true) {
    if (!this.initialized) {
      await this.initialize();
    }

    if (requiresAuth && !this.isAuthenticated()) {
      throw new Error("Authentication required");
    }

    const url = `${this.apiUrl}${endpoint}`;
    
    const requestOptions = {
      ...options,
      headers: {
        ...this.getHeaders(requiresAuth),
        ...(options.headers || {})
      }
    };

    try {
      const response = await fetch(url, requestOptions);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Request failed with status ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error(`API request error (${endpoint}):`, error);
      throw error;
    }
  }

  /**
   * Authenticate with a sign-in code
   * @param {string} code - Sign-in code
   */
  async authenticate(code) {
    if (!this.initialized) {
      await this.initialize();
    }

    try {
      const response = await fetch(`${this.apiUrl}/accounts/sign_in`, {
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
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || "Authentication failed");
      }

      const data = await response.json();

      // Store authentication data
      this.token = data.token;
      this.orgId = data.orgId;

      // Save to storage
      await chrome.storage.local.set({
        [STORAGE_KEYS.AUTH_TOKEN]: data.token,
        [STORAGE_KEYS.ORG_ID]: data.orgId,
        [STORAGE_KEYS.USER_INFO]: {
          name: data.name,
          email: data.email
        }
      });

      return {
        success: true,
        user: {
          name: data.name,
          email: data.email
        }
      };
    } catch (error) {
      console.error("Authentication error:", error);
      throw error;
    }
  }

  /**
   * Logout and clear authentication data
   */
  async logout() {
    try {
      // Clear authentication data
      this.token = null;
      this.orgId = null;

      // Clear from storage
      await chrome.storage.local.remove([
        STORAGE_KEYS.AUTH_TOKEN,
        STORAGE_KEYS.ORG_ID,
        STORAGE_KEYS.USER_INFO
      ]);

      return { success: true };
    } catch (error) {
      console.error("Logout error:", error);
      throw error;
    }
  }

  /**
   * Update the API URL
   * @param {string} apiUrl - New API URL
   */
  async updateApiUrl(apiUrl) {
    try {
      this.apiUrl = apiUrl;

      // Save to storage
      await chrome.storage.local.set({
        [STORAGE_KEYS.API_URL]: apiUrl
      });

      return { success: true };
    } catch (error) {
      console.error("Error updating API URL:", error);
      throw error;
    }
  }

  /**
   * Fetch all plans
   */
  async fetchPlans() {
    try {
      const data = await this.request("/plans");
      return data.plans || [];
    } catch (error) {
      console.error("Error fetching plans:", error);
      throw error;
    }
  }

  /**
   * Fetch all projects
   */
  async fetchProjects() {
    try {
      const data = await this.request("/projects");
      return data.projects || [];
    } catch (error) {
      console.error("Error fetching projects:", error);
      throw error;
    }
  }

  /**
   * Get the current branch for a plan
   * @param {string} planId - Plan ID
   */
  async getCurrentBranch(planId) {
    try {
      const data = await this.request(`/plans/${planId}/current_branch`);
      return data.branch || "main";
    } catch (error) {
      console.error(`Error fetching current branch for plan ${planId}:`, error);
      throw error;
    }
  }

  /**
   * Send text to a plan
   * @param {string} planId - Plan ID
   * @param {string} branch - Branch name
   * @param {string} text - Text to send
   * @param {string} name - Name for the context item
   */
  async sendTextToPlan(planId, branch, text, name = "Selected text from web") {
    try {
      if (!planId) {
        throw new Error("Plan ID is required");
      }

      if (!branch) {
        branch = await this.getCurrentBranch(planId);
      }

      const contextParams = {
        contextType: "note",
        name: name,
        body: text,
        autoLoaded: false
      };

      await this.request(`/plans/${planId}/${branch}/context`, {
        method: "POST",
        body: JSON.stringify([contextParams])
      });

      return { success: true };
    } catch (error) {
      console.error("Error sending text to plan:", error);
      throw error;
    }
  }

  /**
   * Search plans by name
   * @param {string} query - Search query
   * @param {Array} plans - List of plans to search
   */
  searchPlans(query, plans) {
    if (!query || !plans || !plans.length) {
      return plans || [];
    }

    const lowerQuery = query.toLowerCase();
    return plans.filter(plan => 
      plan.name.toLowerCase().includes(lowerQuery) || 
      (plan.description && plan.description.toLowerCase().includes(lowerQuery))
    );
  }
}

// Create and export a singleton instance
const plandexAPI = new PlandexAPI();

// Initialize the API service
plandexAPI.initialize().catch(error => {
  console.error("Failed to initialize Plandex API service:", error);
});

// Export the API service
export default plandexAPI;
