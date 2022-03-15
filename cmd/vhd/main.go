
package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	"unicode"
	
	"rpucella.net/virtual-hard-drive/internal/virtualfs"
	"rpucella.net/virtual-hard-drive/internal/catalog"
)

type context struct{
	commands map[string]command
	root virtualfs.Root
	//drive catalog.Drive
	pwd virtualfs.VirtualFS
	exit bool         // Set to true to exit the main loop.
}

func main() {
	args := os.Args[1:]

	commands := initializeCommands()
	catalog, err := catalog.Load()
	if err != nil {
		panic(err)
	}
	root, err := virtualfs.NewRoot(catalog)
	if err != nil {
		panic(err)
	}

	ctxt := &context{
		commands,
		root,
		root.AsVirtualFS(),
		false,
	}
	
	if len(args) > 0 {
		if err := processCommand(ctxt, args[0], args[1:]); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	} else {
		loop(ctxt)
	}	
}

func loop(ctxt *context) {
	
	// https://patorjk.com/software/taag/#p=display&f=Colossal&t=Virtual%20HD
	
	fmt.Print(`
888     888 d8b         888                      888      888    888 8888888b.  
888     888 Y8P         888                      888      888    888 888  "Y88b 
888     888             888                      888      888    888 888    888 
Y88b   d88P 888 888d888 888888 888  888  8888b.  888      8888888888 888    888 
 Y88b d88P  888 888P"   888    888  888     "88b 888      888    888 888    888 
  Y88o88P   888 888     888    888  888 .d888888 888      888    888 888    888 
   Y888P    888 888     Y88b.  Y88b 888 888  888 888      888    888 888  .d88P 
    Y8P     888 888      "Y888  "Y88888 "Y888888 888      888    888 8888888P"

`)

	reader := bufio.NewReader(os.Stdin)

	for !ctxt.exit {
		// Keep going until we nullify the context (flag for quitting)
		// if ctxt.drive == nil {
		// 	fmt.Printf("\n(no drive) ")
		// } else {
		// 	fmt.Printf("\n%s:%s ", ctxt.drive.Name(), ctxt.pwd.Path())
		// }
		path := ctxt.pwd.Path()
		if len(path) > 0 {
			// strip off final /
			path = path[:len(path) - 1]
		}
		fmt.Printf("\u001b[1m%s>\u001b[0m ", path)
		line, _ := reader.ReadString('\n')
		fields := split(line) // strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		comm := fields[0]
		args := fields[1:]
		if err := processCommand(ctxt, comm, args); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
	}
}

func processCommand(ctxt *context, comm string, args []string) error {
	commObj, ok := ctxt.commands[comm]
	if !ok {
		return fmt.Errorf("Unknown command: %s", comm)
	}
	if len(args) < commObj.minArgCount {
		return fmt.Errorf("Too few arguments (expected %d): %s", commObj.minArgCount, comm)
	}
	if commObj.maxArgCount >= 0 && len(args) > commObj.maxArgCount {
		return fmt.Errorf("Too many arguments (expected %d): %s", commObj.maxArgCount, comm)
	}
	err := commObj.process(args, ctxt)
	return err
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
