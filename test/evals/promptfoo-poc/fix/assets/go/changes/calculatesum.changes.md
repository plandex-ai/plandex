### Subtask 1:  Add error checking for nil slice. Update the function to return 0 and an error if 'numbers' is nil.

```go
func calculateSum(numbers []int) int {
    sum := 0

    if numbers == nil {
        return 0, errors.New("Invalid input")
    }

    for _, num := range numbers {
        sum += num
    }
    return sum
}
```
