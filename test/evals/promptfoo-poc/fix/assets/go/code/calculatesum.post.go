pdx-1: func calculateSum(numbers []int) int {
pdx-2:     if numbers == nil {
pdx-3:         return 0, errors.New("numbers cannot be nil")
pdx-4:     }
pdx-5:     sum := 0
pdx-6:     for _, num := range numbers {
pdx-7:         sum += num
pdx-8:     }
pdx-9:     return sum
pdx-10: }
