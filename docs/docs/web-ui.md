# Web UI Guide

## Introduction

This guide covers the Plandex Web UI, a graphical interface for managing your Plandex projects, server instance, and accessing features like Multi-Channel Publishing (MCP). The Web UI aims to provide an intuitive and visual way to interact with Plandex's core functionalities.

## Accessing the Web UI

Once the Plandex server is running (either locally via Docker or on a self-hosted instance), the Web UI can be accessed by navigating to the `/ui` path in your web browser.

For example, if your Plandex server is running locally on port `33333`, you would access the Web UI at:
`http://localhost:33333/ui`

If accessing a remote server, replace `localhost:PORT` with your server's address and port.

## Navigating the UI

The Web UI is organized into several main sections, accessible via a sidebar navigation menu:

*   **Dashboard:** Provides a high-level overview of your Plandex system, including key statistics and recent activities.
*   **Plans:** Allows you to manage your coding plans. This includes listing existing plans, viewing their details, monitoring their execution, and potentially creating new plans or modifying existing ones.
*   **Contexts:** Interface for managing the context provided to your AI models. This may include loading files or URLs, viewing currently active contexts, and clearing context when necessary.
*   **Models:** Section for managing AI models and model packs. You can view available models, their configurations, and potentially load/unload or switch between different models or packs.
*   **MCP (Multi-Channel Publishing):** Provides tools for configuring and managing the publishing of content (e.g., generated code, documentation, summaries) to various channels.
*   **Settings:** Allows you to configure various Plandex settings related to the server, UI behavior, or default configurations.

## MCP Functionality

The Multi-Channel Publishing (MCP) section within the Web UI is designed to streamline the process of distributing content generated or managed by Plandex.

Key features (may be in development or planned):
*   **Channel Configuration:** Set up and authenticate different output channels (e.g., Git repositories, documentation platforms, blogs, social media).
*   **Content Selection:** Choose which Plandex artifacts (plans, specific files, summaries) to publish.
*   **Publishing Workflows:** Define and trigger automated workflows to push content to configured channels.
*   **Status Tracking:** Monitor the status of publishing tasks.

(More sections and detailed explanations will be added as features are fully implemented and refined.)
