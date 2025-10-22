package main

import (
	"fmt"
	"github.com/muesli/termenv"
)

func main() {
	// Get the output
	output := termenv.DefaultOutput()
	
	// Create a string with RGB color (215, 119, 87 - Claude orange)
	str := output.String("Claude Orange Text")
	styled := str.Foreground(output.Color("#d77757")) // RGB to hex

	fmt.Println("Profile:", output.Profile)
	fmt.Println("Styled string:", styled)
	
	// Test with raw ANSI
	rawANSI := "\x1b[38;2;215;119;87mClaude Orange (raw ANSI)\x1b[0m"
	fmt.Println("Raw ANSI:", rawANSI)
}
