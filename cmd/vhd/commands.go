
package main

import (
	"fmt"
	"sort"
)

func initializeCommands() {
	commands = make(map[string]command)
	commands["exit"] = command{0, commandQuit, "Bail out"}
	commands["help"] = command{0, commandHelp, "List available commands"}
}

func commandHelp(args []string, ctxt repl_context) (repl_context, bool) {
	fmt.Println("Available commands:")
	keys := make([]string, 0, len(commands))
	for k := range commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf(" %-20s\t%s\n", k, commands[k].help)
	}
	return ctxt, false
}

func commandQuit(args []string, ctxt repl_context) (repl_context, bool) {
	return ctxt, true
}
