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
