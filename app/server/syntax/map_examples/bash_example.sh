#!/bin/bash

# Global variables
GLOBAL_VAR="Hello World"
readonly CONSTANT_VAR="This is constant"

# Function definition
function print_message() {
    local message="$1"
    echo "$message"
}

# Function with return value
get_date() {
    echo $(date +%Y-%m-%d)
}

# Array declaration
declare -a fruits=("apple" "banana" "orange")

# Associative array
declare -A user_info=(
    ["name"]="John"
    ["age"]="30"
    ["city"]="New York"
)

# Main script execution
main() {
    print_message "$GLOBAL_VAR"
    current_date=$(get_date)
    echo "Today is: $current_date"
    
    # Loop through array
    for fruit in "${fruits[@]}"; do
        echo "Fruit: $fruit"
    done
    
    # Access associative array
    echo "User ${user_info[name]} is ${user_info[age]} years old"
}

# Call main function
main
