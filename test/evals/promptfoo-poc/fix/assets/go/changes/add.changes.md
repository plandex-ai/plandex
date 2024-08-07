### Subtask 1:  Add error checking to the add function. Convert it to handle float types instead of int.

```go
func add(a float64, b float64) float64 {
    if a == nil || b == nil {
        return 0, errors.New("Invalid input")
    }
    return a + b
}
```
