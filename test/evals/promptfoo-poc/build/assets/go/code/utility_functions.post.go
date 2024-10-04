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
