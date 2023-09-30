package lib

import "fmt"

func ClearScreen() {
	fmt.Printf("\033[H\033[J") // Move to top-left and clear from cursor to end of screen
}
