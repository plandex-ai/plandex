### Subtask 1:  Correct the sorting logic in `sortIntegers` function to actually sort the integers.

```go
package utils

func sortIntegers(input []int) {
  // Sort the input array
  for i := 0; i < len(input); i++ {
      for j := i + 1; j < len(input); j++ {
      if input[i] > input[j] {
          input[i], input[j] = input[j], input[i]
      }
      }
  }
}

func main() {
  values := []int{2, 3, 1, 4}
  sortIntegers(values)
   // Output should be a sorted array
}
```

### Subtask 2:  Add a new function `printValues` to print the sorted array.

```go
package utils

import "fmt"

func sortIntegers(input *[]int) []int {
  // Sort the input array
  for i := 0; i < len(input); i++ {
      for j := i + 1; j < len(input); j++ {
        if input[i] > input[j] {
            input[i], input[j] = input[j], input[i]
        }
      }
  }
}

func printValues(input []int) {
  // Print the sorted array
  for i := 0; i < len(input); i++ {
      fmt.Println(input[i])
  }
}

func main() {
  values := []int{2, 3, 1, 4}
  sortIntegers(values)
   // Output should be a sorted array
}
```

### Subtask 3:  Update `main` function to call `printValues` after sorting.

```go
package utils

import "fmt"

func sortIntegers(input []int) {
  // Sort the input array
  for i := 0; i < len(input); i++ {
      for j := i + 1; j < len(input); j++ {
      if input[i] > input[j] {
          input[i], input[j] = input[j], input[i]
      }
      }
  }
}

func printValues(input []int) {
  // Print the sorted array
  for i := 0; i < len(input); i++ {
      fmt.Println(input[i])
  }
}

func main() {
  values := []int{2, 3, 1, 4}
  sortIntegers(values)
  printValues(values)
}
```
