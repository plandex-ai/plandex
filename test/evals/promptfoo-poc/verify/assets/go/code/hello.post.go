package main

import "fmt"

func main() {
    hello("World")
}

func hello(name string) {
    fmt.Println("Hello,", name, "!")
}
