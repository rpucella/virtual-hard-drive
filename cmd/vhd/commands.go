
package main

import (
	"fmt"
	"sort"
	"rpucella.net/virtual-hard-drive/internal/catalog"
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
	commands["ls"] = command{0, 1, commandLs, "ls [<folder>]", "List content of folder"}
	commands["cd"] = command{0, 1, commandCd, "cd [<folder>]", "Change working folder"}
	commands["file"] = command{1, 1, commandFile, "file <file>", "Show file information"}
	commands["catalog"] = command{0, 1, commandCatalog, "catalog [<folder>]", "Show catalog at folder"}
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

func commandLs(args []string, ctxt *context) error {
	curr := ctxt.pwd
	if len(args) > 0 {
		newCurr, err := catalog.Navigate(curr, args[0], false)
		if err != nil {
			return fmt.Errorf("ls: %w", err)
		}
		curr = newCurr
	}
	dirs := make([]string, 0, len(curr.Content()))
	files := make([]string, 0, len(curr.Content()))
	for _, sub := range curr.Content() {
		if sub.IsDir() {
			dirs = append(dirs, sub.Name())
		} else {
			files = append(files, sub.Name())
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)
	for _, name := range dirs {
		fmt.Printf("%s/\n", name)
	}
	width := maxLength(files)
	for _, name := range files {
		fmt.Printf("%*s     %s\n", -width, name, curr.Content()[name].UUID())
	}
	return nil
}

func commandCd(args []string, ctxt *context) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}
	newPwd, err := catalog.Navigate(ctxt.pwd, path, false)
	if err != nil {
		return fmt.Errorf("cd: %w", err)
	}
	ctxt.pwd = newPwd
	return nil
}

func commandCatalog(args []string, ctxt *context) error {
	curr := ctxt.pwd
	if len(args) > 0 {
		newCurr, err := catalog.Navigate(curr, args[0], false)
		if err != nil {
			return fmt.Errorf("catalog: %w", err)
		}
		curr = newCurr
	}
	catalog.Print(curr)
	return nil
}

func commandFile(args []string, ctxt *context) error {
	fileObj, err := catalog.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("file: %w", err)
	}
	fileObj.Print()
	return nil
}
