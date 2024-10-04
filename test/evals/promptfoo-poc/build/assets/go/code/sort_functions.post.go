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