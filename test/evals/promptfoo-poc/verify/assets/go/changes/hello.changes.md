### Subtask 1: Update the 'main' function to add a new 'hello' function that takes a name as an argument and prints 'Hello, <name>!'. 


```go
package main

func main() {
    fmt.Println("Hello, World!")
}

func hello(name string) {
    fmt.Println("Hello,", name, "!")
}
```

### Subtask 2: Replace the existing print statement in 'main' with a call to this new function, passing a default name.

```go
package main

func main() {
    hello("World")
}

func hello(name string) {
    fmt.Println("Hello,", name, "!")
}
```
