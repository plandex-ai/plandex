package main

import (
	"context"
	"fmt"
	"os"
	"plandex-server/syntax"
)

func main() {
	ctx := context.Background()

	// file path is first arg
	if len(os.Args) < 2 {
		fmt.Println("usage: mapper <file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("error reading file: %v\n", err)
		os.Exit(1)
	}

	m, err := syntax.MapFile(ctx, filename, content)
	if err != nil {
		fmt.Printf("error mapping file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(m.String())
}
