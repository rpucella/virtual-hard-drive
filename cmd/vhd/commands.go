
package main

import (
	"fmt"
	"sort"
)

func maxLength(strings []string) int {
	current := 0
	for _, s := range strings {
		if len(s) > current {
			current = len(s)
		}
	}
	return current
}

func initializeCommands() map[string]command {
	commands := make(map[string]command)
	commands["exit"] = command{0, 0, commandQuit, "exit", "Bail out"}
	commands["help"] = command{0, 0, commandHelp, "help", "List available commands"}
	commands["drive"] = command{0, 1, commandDrive, "drive [<name>]", "List or select drive"}
	return commands
}

func commandHelp(args []string, ctxt *context) error {
	keys := make([]string, 0, len(ctxt.commands))
	names := make([]string, 0, len(ctxt.commands))
	for k := range ctxt.commands {
		keys = append(keys, k)
		names = append(names, ctxt.commands[k].usage)
	}
	sort.Strings(keys)
	width := maxLength(names)
	for _, k := range keys {
		fmt.Printf("%*s   %s\n", -width, ctxt.commands[k].usage, ctxt.commands[k].help)
	}
	return nil
}

func commandQuit(args []string, ctxt *context) error {
	ctxt.exit = true
	return nil
}

func commandDrive(args []string, ctxt *context) error {
	if len(args) == 0 {
		// List available drives.
		keys := make([]string, 0, len(ctxt.drives))
		for k := range ctxt.drives {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		width := maxLength(keys)
		for _, k := range keys {
			fmt.Printf("%*s   %s://%s\n", -width, k, ctxt.drives[k].provider, ctxt.drives[k].bucket)
		}
		return nil
	}
	newName := args[0]
	newDrive, found := ctxt.drives[newName]
	if !found {
		return fmt.Errorf("Cannot find drive: %s", newName)
	}
	ctxt.drive = newDrive
	return nil
}
