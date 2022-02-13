
package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	"unicode"
	
	"rpucella.net/virtual-hard-drive/internal/virtualfs"
)

type context struct{
	commands map[string]command
	root virtualfs.Root
	//drive catalog.Drive
	pwd virtualfs.VirtualFS
	exit bool         // Set to true to exit the main loop.
}

func main() {
	// Parse arguments if needed.
	//args := os.Args[1:]

	reader := bufio.NewReader(os.Stdin)

	commands := initializeCommands()
	root, err := virtualfs.NewRoot()
	if err != nil {
		panic(err)
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Println("                   VIRTUAL HARD DRIVE                       ")
	fmt.Println("------------------------------------------------------------")

	ctxt := context{
		commands,
		root,
		root.AsVirtualFS(),
		false,
	}
	
	for !ctxt.exit {
		// Keep going until we nullify the context (flag for quitting)
		// if ctxt.drive == nil {
		// 	fmt.Printf("\n(no drive) ")
		// } else {
		// 	fmt.Printf("\n%s:%s ", ctxt.drive.Name(), ctxt.pwd.Path())
		// }
		fmt.Printf("\n%s ", ctxt.pwd.Path())
		line, _ := reader.ReadString('\n')
		fields := split(line) // strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		comm := fields[0]
		args := fields[1:]
		commObj, ok := commands[comm]
		if !ok {
			fmt.Printf("Unknown command: %s\n", comm)
			continue
		}
		if len(args) < commObj.minArgCount {
			fmt.Printf("Too few arguments (expected %d): %s\n", commObj.minArgCount, comm)
			continue
		}
		if commObj.maxArgCount >= 0 && len(args) > commObj.maxArgCount {
			fmt.Printf("Too many arguments (expected %d): %s\n", commObj.maxArgCount, comm)
			continue
		}
		err := commObj.process(args, &ctxt)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			continue
		}
	}
}

// Split a line into fields at spaces.
// Do not split within double quotes "...".
//
func split(s string) []string {
	result := []string{}
	sb := &strings.Builder{}
	quoted := false
	started := false
	for _, r := range s {
		if r == '"' {
			quoted = !quoted
		} else if !quoted && unicode.IsSpace(r) {
			if started {
				result = append(result, sb.String())
				sb.Reset()
			}
			started = false
		} else {
			started = true
			sb.WriteRune(r)
		}
	}
	if sb.Len() > 0 {
		result = append(result, sb.String())
	}
	return result
}
