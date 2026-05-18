package main

import (
	"fmt"
	"os"

	"myai-novel-go/cmd/novel/cmd"
)

func main() {
	if err := cmd.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
