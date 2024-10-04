pdx-1: func add(a float64, b float64) float64 {
pdx-2:     if a == nil || b == nil {
pdx-3:         return 0, errors.New("Invalid input")
pdx-4:     }
pdx-5:     return a + b
pdx-6: }
