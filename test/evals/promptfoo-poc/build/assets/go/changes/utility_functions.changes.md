### Subtask 1: Add a new function `Multiply(a int, b int) int` that returns the multiplication of two integers.

```go
package utils

func Add(a int, b int) int {
    return a + b
}

func Subtract(a int, b int) int {
    return a - b
}

func Multiply(a int, b int) int {
    return a * b
}
```

### Subtask 2: Modify `Add` function to log the inputs and output.

```go
package utils

import "log"

func Add(a int, b int) int {
    log.Printf("Adding %d and %d", a, b)
    result := a + b
    log.Printf("Result: %d", result)
    return result
}

func Subtract(a int, b int) int {
    return a - b
}

func Multiply(a int, b int) int {
    return a * b
}
```

### Subtask 3: Remove the `Subtract` function.

```go
package utils

import "log"

func Add(a int, b int) int {
    log.Printf("Adding %d and %d", a, b)
    result := a + b
    log.Printf("Result: %d", result)
    return result
}

func Multiply(a int, b int) int {
    return a * b
}
```
