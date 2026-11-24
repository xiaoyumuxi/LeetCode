package main

import (
	"github.com/xiaoyumuxi/LeetCode/internal/plugin"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--judge-report" {
		if len(os.Args) != 3 {
			os.Exit(2)
		}
		os.Exit(plugin.PrintJudgeReport(os.Args[2], os.Stdout, os.Stderr))
	}
	os.Exit(plugin.NewBackend().Listen())
}
