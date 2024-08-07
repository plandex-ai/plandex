package main

import (
    "fmt"
    "time"
)

func main() {
    current_time := time.Now().Format("2006-01-02")
    if _, err := fmt.Println("Hello, World! Current date: ", current_time); err != nil {
        panic(err)
    }
}