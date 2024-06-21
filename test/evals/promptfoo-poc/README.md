# Test Driven Development of Prompts

This directory is dedicated to the systematic development of prompts for the plandex project. The prompts are developed in a test-driven manner, where the prompt is first written in a markdown file, and then the prompt is tested by running the prompt through the various evaluations. The output of the prompt is then graded and A/B tested for various metrics (see [metrics](#metrics)). The prompt is then iteratively improved until it meets the desired metrics.

We have decided to write the majority of the evals using the [promptfoo]() framework, as it is robust and contains customizations with a clear ease of setup.

## Usage

Usage will be broken down into [run](#run-evals) and [create](#create-evals) sections:

### Setup

To run or create evals, you will need to have the following installed:

- [Go](https://golang.org/)
- [Promptfoo]()

### Run Evals

To run the evaluations, you can cd into the relevant directory and use the following command:

```bash
make eval <name_of_eval_dir>
```

Or, you can run all the evaluations by running the following command:

```bash
make evals all
```

### Create Evals

To create the evaluations, you can use our `gen-*` commands. The `gen-*` commands are designed to setup the eval environment and will create the evaluations directory structure in the `evals/promptfoo` directory. You have access to the following commands:

```bash
make gen-eval # Generates the evaluation directory structure
make gen-provider # Generates a provider file based on the directory config files
```

> ![IMPORTANT]\
> Make sure to run the `gen-eval` command before running the `gen-provider` command.
> You need to have the config files filled out with your details before running the `gen-provider` command.
> Depending on the provider you use, you will need to setup an environment variable with the provider's API key.

## Metrics

The metrics we are currently tracking are:

> ![NOTE]\
> COMING SOON
