# Evals

Evals for plandex.

## Overview

`Classification:`

- MCC
- Specificity
- Sensitivity
- Accuracy

`Regression:`

- RMSE
- R2
- MSE

## Types of Evals

🧑‍💻: Yes, promptfoo offers various methods for evaluating prompts beyond simple assertions. These include heuristics, language model-assisted metrics, and other strategies. Here are some additional methods you can use:

### Deterministic Evaluation Metrics

1. **equals**: Checks if the output matches exactly.
2. **contains**: Checks if the output contains a substring.
3. **icontains**: Case-insensitive substring check.
4. **regex**: Matches the output against a regular expression.
5. **starts-with**: Checks if the output starts with a specified string.
6. **is-json**: Validates if the output is valid JSON.
7. **contains-json**: Checks if the output contains valid JSON.
8. **is-sql**: Validates if the output is valid SQL.
9. **contains-sql**: Checks if the output contains valid SQL.
10. **javascript**: Uses a JavaScript function to validate the output.
11. **python**: Uses a Python function to validate the output.
12. **webhook**: Uses a webhook that returns `{pass: true}`.
13. **rouge-n**: Rouge-N score is above a given threshold.
14. **levenshtein**: Levenshtein distance is below a threshold.
15. **latency**: Latency is below a threshold (milliseconds).
16. **perplexity**: Perplexity is below a threshold.
17. **cost**: Cost is below a threshold (for models with cost info such as GPT).
18. **is-valid-openai-function-call**: Ensures the function call matches the function's JSON schema.
19. **is-valid-openai-tools-call**: Ensures all tool calls match the tools JSON schema.

### Model-assisted Evaluation Metrics

1. **similar**: Uses embeddings and cosine similarity to check if the output is similar to a given text.
2. **classifier**: Runs the LLM output through a classifier.
3. **llm-rubric**: Uses a Language Model to grade the output based on a rubric.
4. **answer-relevance**: Ensures the LLM output is related to the original query.
5. **context-faithfulness**: Ensures the LLM output uses the context accurately.
6. **context-recall**: Ensures the ground truth appears in the context.
7. **context-relevance**: Ensures the context is relevant to the original query.
8. **factuality**: Ensures the LLM output adheres to the given facts.
9. **model-graded-closedqa**: Uses a Closed QA method to evaluate the LLM output.
10. **moderation**: Ensures the outputs are safe.
11. **select-best**: Compares multiple outputs for a test case and picks the best one.

These methods provide a comprehensive set of tools to evaluate the quality, accuracy, and relevance of LLM outputs. You can combine these methods to create robust and thorough evaluations for your use cases.

Would you like to implement any of these advanced evaluation methods for your test cases? If so, please specify which methods you are interested in, and I can help you integrate them.

---

### Build Prompts Evaluations

1. **Syntax Check**
   - Ensure the prompt is syntactically correct and follows the required format.
2. **Completeness Check**
   - Verify that all necessary components (e.g., headers, body, footers) are included.
3. **Clarity and Precision**
   - Evaluate if the instructions are clear and unambiguous.
4. **Context Appropriateness**
   - Assess if the prompt is appropriate for the intended context and audience.
5. **Error Handling**
   - Check if there are adequate instructions for handling potential errors.
6. **Dependency Evaluation**
   - Ensure all dependencies (libraries, tools) are correctly specified.
7. **Output Validation**
   - Define criteria to validate the expected output.

### Verify Prompts Evaluations

1. **Accuracy Check**
   - Ensure the prompt's instructions lead to the correct and intended results.
2. **Validation Criteria**
   - Define and check the validation criteria for the outputs.
3. **Consistency Check**
   - Verify if the prompt maintains consistency in terminology and steps.
4. **Logic and Flow**
   - Evaluate the logical flow of the instructions to ensure they are coherent.
5. **Edge Cases Handling**
   - Assess if the prompt considers and handles edge cases effectively.
6. **User Feedback Integration**
   - Ensure there are provisions for incorporating user feedback.
7. **Performance Metrics**
   - Define and evaluate the performance metrics for the verification process.

### Fix Prompts Evaluations

1. **Error Identification**
   - Ensure the prompt accurately identifies the errors to be fixed.
2. **Correctness of Fix**
   - Verify that the proposed fix is correct and resolves the issue.
3. **Impact Analysis**
   - Assess the impact of the fix on the overall system or application.
4. **Regression Testing**
   - Ensure that the fix does not introduce new issues or regressions.
5. **Documentation Update**
   - Check if the documentation is updated to reflect the fix.
6. **Code Quality**
   - Evaluate the quality of the code after the fix (e.g., readability, maintainability).
7. **Performance Impact**
   - Assess if the fix affects the performance and ensure it remains optimal.

### Function Call Schemas Evaluations

1. **Schema Validity**
   - Ensure the schema is valid and conforms to the defined standards.
2. **Parameter Consistency**
   - Verify that the parameters are consistently defined and used.
3. **Return Type Verification**
   - Ensure the return types are correctly specified and handled.
4. **Error Handling Mechanism**
   - Assess the error handling mechanisms in the schema.
5. **Compatibility Check**
   - Check if the schema is compatible with different environments or systems.
6. **Documentation Completeness**
   - Ensure the schema is well-documented with clear explanations of each parameter and return type.
7. **Security Considerations**
   - Evaluate the schema for potential security vulnerabilities or risks.


