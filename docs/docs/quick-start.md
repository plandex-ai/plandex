# Quick Start Guide

## Step 1: Create a New Plan

Navigate to your project's root directory and create a new plan:

```bash
plandex new
```

## Step 2: Load Context

Load any relevant files, directories, or URLs into the plan context:

```bash
plandex load component.ts action.ts reducer.ts
plandex load lib -r
plandex load https://redux.js.org/usage/writing-tests
```

## Step 3: Send a Prompt

Send a prompt to the AI to describe a task or ask a question:

```bash
plandex tell "add a new line chart showing the number of foobars over time to components/charts.tsx"
```

Follow the AI's suggestions and review the changes.
