# Model Settings

Plandex offers a wide range of model settings that allow you to customize the behavior of the AI.

## Viewing Model Settings

To view the current model settings, use the `plandex models` command:

```bash
plandex models
```

## Changing Model Settings

To change the model settings, use the `plandex set-model` command:

```bash
plandex set-model planner openai/gpt-4
```

## Default Model Settings

To view and change the default model settings for new plans, use the `plandex models default` and `plandex set-model default` commands:

```bash
plandex models default
plandex set-model default planner openai/gpt-4
```

## Custom Models

You can add custom models using the `plandex models add` command:

```bash
plandex models add
```

To delete a custom model, use the `plandex models delete` command:

```bash
plandex models delete
```
