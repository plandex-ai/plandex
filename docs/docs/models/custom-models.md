# Custom Models

Plandex allows you to add and manage custom models to tailor the AI's behavior to your specific needs.

## Adding a Custom Model

To add a custom model, use the `plandex models add` command:

```bash
plandex models add
```

Follow the prompts to provide the necessary details for the custom model, such as the provider, model name, base URL, API key environment variable, and other settings.

## Deleting a Custom Model

To delete a custom model, use the `plandex models delete` command:

```bash
plandex models delete model-name
```

You can also delete a custom model by its index in the list of custom models:

```bash
plandex models delete 1
```

## Listing Custom Models

To list all custom models, use the `plandex models available --custom` command:

```bash
plandex models available --custom
```

This will display a list of all custom models that you have added.

## Using Custom Models

Once you have added a custom model, you can use it in your plans by setting it as the model for a specific role. For example, to set a custom model as the planner model, use the `plandex set-model` command:

```bash
plandex set-model planner custom-provider/model-name
```

You can also set custom models for other roles, such as builder, namer, and verifier, using the same command.
