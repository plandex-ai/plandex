func calculateSum(numbers []int) int {
    if numbers == nil {
        return 0, errors.New("numbers cannot be nil")
    }
    sum := 0
    for _, num := range numbers {
        sum += num
    }
    return sum
}