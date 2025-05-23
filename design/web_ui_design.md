# Plandex Web UI - UI/UX Design Document

## 1. Introduction

This document outlines the UI/UX design for the Plandex Web UI. The goal is to create an intuitive and user-friendly interface that leverages the existing `gorilla/mux` framework and graphically represents the functionalities currently available in the Plandex CLI.

## 2. Main Layout

The main layout will consist of two primary areas:

*   **Sidebar (Navigation):** A fixed sidebar on the left providing access to the main sections of the application.
*   **Main Content Area:** The central area where the content for the selected section will be displayed.

```
+------------------------------------------------------+
| Sidebar        | Main Content Area                   |
| (Navigation)   |                                     |
|                |                                     |
|                |                                     |
|                |                                     |
|                |                                     |
|                |                                     |
|                |                                     |
+------------------------------------------------------+
```

## 3. Navigation

The sidebar will contain the following navigation links:

*   **Dashboard:** Overview, quick stats, recent activity.
*   **Plans:** Manage plans (list, create, view, execute).
*   **Contexts:** Manage contexts (load, view, update, clear).
*   **Models:** Manage AI models and model packs.
*   **MCP (Multi-Channel Publishing):** Interface for MCP functionalities.
*   **Settings:** Configure Plandex settings.

## 4. Wireframes/Mockups

### 4.1. Dashboard

*   **Description:** Provides a high-level overview of the Plandex system, including key statistics and recent activities.
*   **Wireframe:**
    ```
    +------------------------------------------------------+
    | Dashboard                                            |
    +------------------------------------------------------+
    | Quick Stats          | Recent Activity               |
    | - Total Plans: X     | - Plan "ABC" executed (time)  |
    | - Active Contexts: Y | - Context "XYZ" updated (time)|
    | - Models Loaded: Z   | - ...                         |
    |                      |                               |
    | System Status        | Notifications                 |
    | - All systems green  | - Update available for model X|
    | - ...                | - ...                         |
    +------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/dashboard.png`)*

### 4.2. Plans

*   **Description:** Allows users to manage their plans.
*   **Sub-sections:**
    *   **List Plans:** Table view of existing plans with options to view details, execute, or delete.
    *   **Create New Plan:** Form to define a new plan.
    *   **View Plan Details:** Detailed view of a specific plan, including its steps, status, and history.
    *   **Manage Plan Execution:** Interface to start, stop, pause, and monitor plan execution.
*   **Wireframe (List Plans):**
    ```
    +--------------------------------------------------------------------------------------------------+
    | Plans                                                                        [Create New Plan +] |
    +--------------------------------------------------------------------------------------------------+
    | Search: [_________________] Filter: [All/Active/Completed] Sort: [Name/Date/Status]             |
    +--------------------------------------------------------------------------------------------------+
    | | Plan Name     | Status      | Last Executed | Actions                                          |
    | |---------------|-------------|---------------|--------------------------------------------------|
    | | Plan Alpha    | Completed   | 2024-07-28    | [View] [Execute] [Edit] [Delete]                 |
    | | Plan Beta     | Running     | 2024-07-29    | [View] [Pause]   [Edit] [Delete]                 |
    | | Plan Gamma    | Scheduled   | N/A           | [View] [Start]   [Edit] [Delete]                 |
    +--------------------------------------------------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/plans_list.png`)*

### 4.3. Contexts

*   **Description:** Allows users to manage contexts.
*   **Functionalities:**
    *   Load new context from various sources (files, URLs).
    *   View currently loaded contexts and their content.
    *   Update existing contexts.
    *   Clear contexts.
*   **Wireframe:**
    ```
    +--------------------------------------------------------------------------+
    | Contexts                                                 [Load Context +] |
    +--------------------------------------------------------------------------+
    | Search: [_________________]                                              |
    +--------------------------------------------------------------------------+
    | | Context Name  | Source               | Last Updated  | Actions          |
    | |---------------|----------------------|---------------|------------------|
    | | Project X Docs| file:///path/to/docs | 2024-07-28    | [View] [Update] [Clear] |
    | | API Spec Y    | https://api.spec.com | 2024-07-29    | [View] [Update] [Clear] |
    +--------------------------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/contexts.png`)*

### 4.4. Models

*   **Description:** Allows users to manage AI models and model packs.
*   **Functionalities:**
    *   List available models and model packs.
    *   View details of a specific model/pack.
    *   Load/unload models.
    *   Update model configurations.
*   **Wireframe:**
    ```
    +--------------------------------------------------------------------------+
    | Models                                                  [Add Model Pack +] |
    +--------------------------------------------------------------------------+
    | Search: [_________________] Filter: [All/Loaded]                         |
    +--------------------------------------------------------------------------+
    | | Model Name/Pack | Type        | Status      | Actions                  |
    | |-----------------|-------------|-------------|--------------------------|
    | | GPT-4           | LLM         | Loaded      | [View] [Unload] [Config] |
    | | Local Llama3    | LLM         | Not Loaded  | [View] [Load]   [Config] |
    | | Embedding Model | Embedding   | Loaded      | [View] [Unload] [Config] |
    +--------------------------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/models.png`)*

### 4.5. MCP (Multi-Channel Publishing)

*   **Description:** Interface for Multi-Channel Publishing functionalities. (This is a placeholder design as specific requirements may not be fully defined).
*   **Wireframe:**
    ```
    +------------------------------------------------------+
    | MCP (Multi-Channel Publishing)                       |
    +------------------------------------------------------+
    | This section will provide tools to manage and        |
    | publish content across various channels.             |
    |                                                      |
    | [Channel Configuration] [Content Staging] [Analytics]|
    |                                                      |
    | (Further details to be added based on requirements)  |
    +------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/mcp.png`)*

### 4.6. Settings

*   **Description:** Allows users to configure various Plandex settings.
*   **Sections:**
    *   General Settings (API keys, default model, etc.)
    *   UI Preferences (theme, layout options)
    *   Data Management (backup, export)
*   **Wireframe:**
    ```
    +------------------------------------------------------+
    | Settings                                             |
    +------------------------------------------------------+
    | [General] [UI Preferences] [Data Management]         |
    +------------------------------------------------------+
    | General Settings:                                    |
    |   OpenAI API Key: [*******************] [Edit]       |
    |   Default Model:  [GPT-4            ▼] [Save]       |
    |   ...                                                |
    |                                                      |
    | UI Preferences:                                      |
    |   Theme:          [Light / Dark     ▼] [Save]       |
    |   ...                                                |
    +------------------------------------------------------+
    ```
    *(Placeholder for actual image: `design/wireframes/settings.png`)*

## 5. Reusable UI Components

This section describes common UI components that will be used across the application to ensure consistency.

### 5.1. Tables

*   **Description:** Used for displaying lists of data (e.g., plans, contexts, models).
*   **Features:**
    *   Sortable columns.
    *   Pagination for large datasets.
    *   Search/filter functionality.
    *   Action buttons per row (e.g., View, Edit, Delete).

### 5.2. Forms

*   **Description:** Used for creating or editing data (e.g., new plan, model configuration).
*   **Features:**
    *   Standard input fields (text, textarea, select, checkbox, radio).
    *   Validation messages.
    *   Submit and Cancel buttons.

### 5.3. Buttons

*   **Description:** Used for triggering actions.
*   **Types:**
    *   Primary (e.g., Create, Save).
    *   Secondary (e.g., Cancel, View Details).
    *   Destructive (e.g., Delete, Clear).
    *   Icon buttons (for actions in tables or compact areas).

### 5.4. Modals

*   **Description:** Used for displaying information or forms in an overlay without navigating away from the current page.
*   **Use Cases:**
    *   Confirmation dialogs (e.g., "Are you sure you want to delete?").
    *   Quick edit forms.
    *   Displaying detailed information.

## 6. User Experience (UX) Considerations

*   **Intuitive Navigation:** The sidebar provides clear and consistent navigation.
*   **Graphical Representation:** Where possible, CLI functionalities will be represented graphically. For example, plan execution can show progress bars and status indicators.
*   **Feedback:** The system will provide clear feedback to user actions (e.g., success messages, error notifications, loading indicators).
*   **Consistency:** Reusable components and consistent design patterns will be used throughout the application.
*   **Accessibility:** Basic accessibility principles will be considered (e.g., keyboard navigation, sufficient color contrast).

## 7. Technology Stack (Reminder)

*   **Backend:** `gorilla/mux` (as specified)
*   **Frontend:** Standard HTML, CSS, and JavaScript. A lightweight JS framework (e.g., Alpine.js, htmx, or vanilla JS) might be considered for dynamic interactions to keep things simple and integrate well with a Go backend. The choice of a specific JS tool will be made during implementation.

## 8. Future Considerations

*   Real-time updates for plan execution and system status.
*   User roles and permissions.
*   Advanced analytics and reporting.
```
