func add(a float64, b float64) float64 {
    if a == nil || b == nil {
        return 0, errors.New("Invalid input")
    }
    return a + b
}
