---
sidebar_position: 2
sidebar_label: Roles
---

# Model Roles

Plandex has multiple **roles** that are used for different aspects of its functionality. Each role can have its model and settings changed independently.

## Roles

### `planner`

This is the 'main' role that replies to prompts and makes plans.

Can optionally have a 'large context fallback' set, which is the model to use when the context input limit is exceeded.

### `architect`

When auto-context is enabled, this role makes a high-level plan using the project map, then determines what context to to provide for the 'planner' role.

This role is optional. It falls back to the `planner` role if not set.

### `coder`

This role writes code to implement each step of the plan made by the `planner` role during the planning stage.

Instruction-following is important for this role as it needs to follow specific formatting rules.

This role is optional. It falls back to the `planner` role if not set.

### `summarizer`

Summarizes conversations to stay under the limit set in `max-convo-tokens`.

### `auto-continue`

Determines whether a plan is finished or should automatically continue based on the previous response.

### `builder`

Builds the proposed changes described by the `planner` role into pending file updates.

### `whole-file-builder`

Builds the proposed changes described by the `planner` role into pending file updates by writing the entire file. Used as a fallback if more targeted edits fail.

This role is optional. It falls back to the `builder` role if not set.

### `names`

Gives automatically-generated names to plans and context.

### `commit-messages`

Automatically generates commit messages for a set of pending updates.

## Fallbacks and Variants

- Roles can optionally have a 'large context fallback' set, which is an alternate model with a large context window to use when the context input limit is exceeded.

- They can also have a 'large output fallback' set, which is an alternate model with a large output window to use when the output limit is exceeded.

- They can also have a 'strong' variant set, which is an alternative model with stronger capabilities that may be used in some cases when the default model for the role is struggling.
