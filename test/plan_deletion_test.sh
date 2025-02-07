
#!/bin/bash
set -e

# Helper function to check if a plan exists
check_plan_exists() {
    local plan_name=$1
    pdxd plans | grep -q "$plan_name"
}

# Helper function to count total plans
count_plans() {
    pdxd plans | grep -v "^[[:space:]]*$" | grep -v "^[[:space:]]*\\.*" | grep -v "^[[:space:]]*+" | grep -v "^[[:space:]]*\\.*" | grep -v "^[[:space:]]*$" | wc -l
}

# Create test plans
echo "Creating test plans..."
pdxd new -n "plan-config-1"
pdxd new -n "plan-config-2" 
pdxd new -n "plan-config-3"
pdxd new -n "other-plan-1"
pdxd new -n "other-plan-2"

# Verify initial plans were created
echo "Verifying plans were created..."
initial_count=$(count_plans)
if [ "$initial_count" -ne 5 ]; then
    echo "❌ Expected 5 plans, but found $initial_count"
    exit 1
fi

for plan in "plan-config-1" "plan-config-2" "plan-config-3" "other-plan-1" "other-plan-2"; do
    if ! check_plan_exists "$plan"; then
        echo "❌ Plan '$plan' was not created successfully"
        exit 1
    fi
done

# Test wildcard deletion
echo "Testing wildcard deletion..."
pdxd dp "plan-config-*" <<< "y"

# Verify wildcard deletion worked
remaining_count=$(count_plans)
if [ "$remaining_count" -ne 2 ]; then
    echo "❌ Expected 2 plans after wildcard deletion, but found $remaining_count"
    exit 1
fi

for plan in "plan-config-1" "plan-config-2" "plan-config-3"; do
    if check_plan_exists "$plan"; then
        echo "❌ Plan '$plan' should have been deleted"
        exit 1
    fi
done

# Create more plans for range deletion test
pdxd new -n "range-test-1"
pdxd new -n "range-test-2"
pdxd new -n "range-test-3"

# Test range deletion (should delete first 3 plans)
echo "Testing range deletion..."
pdxd dp "1-3" <<< "y"

# Verify range deletion worked
final_count=$(count_plans)
if [ "$final_count" -ne 2 ]; then
    echo "❌ Expected 2 plans after range deletion, but found $final_count"
    exit 1
fi

# Clean up any remaining plans
echo "Cleaning up remaining plans..."
pdxd dp --all <<< "y"

echo "✅ All tests passed!"

