package main

import (
	"fmt"
	"strings"
)

func main() {
	// Test ANSI code preservation
	ansiCode := []rune("\x1b[38;2;215;119;87mClaude Orange\x1b[0m")
	str := string(ansiCode)
	
	fmt.Println("Testing ANSI code passthrough:")
	fmt.Printf("  Original []rune length: %d\n", len(ansiCode))
	fmt.Printf("  String length: %d\n", len(str))
	fmt.Printf("  Contains ANSI: %v\n", strings.Contains(str, "\x1b[38;2"))
	fmt.Println("  Rendered output:", str)
	
	// Test what the component is doing
	lines := []string{
		"Normal text",
		str,
		"More text",
	}
	
	output := strings.Join(lines, "\n")
	fmt.Println("\nJoined output:")
	fmt.Println(output)
}
