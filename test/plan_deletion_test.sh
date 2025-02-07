# Plandex hasn't been able to get this working yet

#!/bin/bash
set -e

# Enable debug output
set -x

# Helper function to check if a plan exists
check_plan_exists() {
    local plan_name=$1
    pdxd plans | grep -q "$plan_name"
}

# Helper function to count total plans
count_plans() {
    # Add sleep to ensure plan list is updated
    sleep 1
    # Strip ANSI codes, skip help text and header, then count only the test plans
    pdxd plans | sed 's/\x1B\[[0-9;]*[mK]//g' | grep -A 1000 "^+----+" | tail -n +3 | head -n 5 | grep -E '\| (plan-config-[123]|other-plan-[12])(\.[0-9]+)? \|' | wc -l
}

# Clean up any existing test plans before starting
echo "Cleaning up any existing test plans..."
echo "y" | pdxd dp "plan-config-*" || true
echo "y" | pdxd dp "other-plan-*" || true
sleep 2

# Create test plans with verification
echo "Creating test plans..."
for plan in "plan-config-1" "plan-config-2" "plan-config-3" "other-plan-1" "other-plan-2"; do
    pdxd new -n "$plan"
    if ! check_plan_exists "$plan"; then
        echo "❌ Failed to create plan '$plan'"
        exit 1
    fi
    # Add sleep to ensure plan creation is complete
    sleep 1
done

# Verify initial plans were created
echo "Verifying plans were created..."
initial_count=$(count_plans)
echo "Found $initial_count plans"

if [ "$initial_count" -ne 5 ]; then
    echo "❌ Expected 5 plans, but found $initial_count"
    pdxd plans
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
echo "y" | pdxd dp "plan-config-*"
sleep 1

# Verify wildcard deletion worked
remaining_count=$(count_plans)
echo "Found $remaining_count plans after wildcard deletion"

if [ "$remaining_count" -ne 2 ]; then
    echo "❌ Expected 2 plans after wildcard deletion, but found $remaining_count"
    pdxd plans
    exit 1
fi

for plan in "plan-config-1" "plan-config-2" "plan-config-3"; do
    if check_plan_exists "$plan"; then
        echo "❌ Plan '$plan' should have been deleted"
        exit 1
    fi
done

# Create more plans for range deletion test
echo "Creating plans for range deletion test..."
for plan in "range-test-1" "range-test-2" "range-test-3"; do
    pdxd new -n "$plan"
    if ! check_plan_exists "$plan"; then
        echo "❌ Failed to create plan '$plan'"
        exit 1
    fi
    sleep 1
done

# Test range deletion (should delete first 3 plans)
echo "Testing range deletion..."
echo "y" | pdxd dp "1-3"
sleep 1

# Verify range deletion worked
final_count=$(count_plans)
echo "Found $final_count plans after range deletion"

if [ "$final_count" -ne 2 ]; then
    echo "❌ Expected 2 plans after range deletion, but found $final_count"
    pdxd plans
    exit 1
fi

# Clean up any remaining test plans
echo "Cleaning up remaining test plans..."
echo "y" | pdxd dp "plan-config-*" || true
echo "y" | pdxd dp "other-plan-*" || true
echo "y" | pdxd dp "range-test-*" || true
sleep 2

# Verify all plans were cleaned up
cleanup_count=$(count_plans)
if [ "$cleanup_count" -ne 0 ]; then
    echo "❌ Expected 0 plans after cleanup, but found $cleanup_count"
    pdxd plans
    exit 1
fi

echo "✅ All tests passed!"
