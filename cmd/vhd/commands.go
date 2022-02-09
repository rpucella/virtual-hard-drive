
package main

import (
	"fmt"
	"sort"
	"rpucella.net/virtual-hard-drive/internal/catalog"
	"github.com/google/uuid"
	"path/filepath"
)

type command struct{
	minArgCount int
	maxArgCount int
	process func([]string, *context)error
	usage string
	help string
	requireDrive bool
}

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
	commands["exit"] = command{0, 0, commandQuit, "exit", "Bail out", false}
	commands["help"] = command{0, 0, commandHelp, "help", "List available commands", false}
	commands["drive"] = command{0, 1, commandDrive, "drive [<name>]", "List or select drive", false}
	commands["ls"] = command{0, 1, commandLs, "ls [<folder>]", "List content of folder", true}
	commands["cd"] = command{0, 1, commandCd, "cd [<folder>]", "Change working folder", true}
	commands["info"] = command{1, 1, commandInfo, "info <file>", "Show file information", true}
	commands["download"] = command{1, 1, commandDownload, "download <file>", "Download file to disk", true}
	commands["upload"] = command{1, 2, commandUpload, "upload <local-file> [<folder>]", "Upload local file to drive folder", true}
	commands["catalog"] = command{0, 1, commandCatalog, "catalog [<folder>]", "Show catalog at folder", true}
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
			fmt.Printf("%*s   %s\n", -width, k, ctxt.drives[k].storage.Name())
		}
		return nil
	}
	newName := args[0]
	newDrive, found := ctxt.drives[newName]
	if !found {
		return fmt.Errorf("cannot find drive: %s", newName)
	}
	fmt.Printf("Loading catalog for %s\n", newDrive.storage.Name())
	cat, err := fetchCatalog(newDrive)
	if err != nil {
		return fmt.Errorf("cannot fetch catalog: %s", newName)
	}
	ctxt.drive = &newDrive
	ctxt.pwd = cat	
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

func commandInfo(args []string, ctxt *context) error {
	fileObj, err := catalog.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}
	fileObj.Print()
	return nil
}

func commandDownload(args []string, ctxt *context) error {
	fileObj, err := catalog.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	objectName, err := ctxt.drive.storage.UUIDToPath(fileObj.UUID())
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	err = ctxt.drive.storage.DownloadFile(objectName, fileObj.Name())
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	fmt.Printf("Object %s downloaded to %s\n", objectName, fileObj.Name())
	return nil
}

func commandUpload(args []string, ctxt *context) error {
	srcFilePath := args[0]
	srcFileName := filepath.Base(srcFilePath)
	destFolder := ctxt.pwd
	if len(args) == 2 {
		newDestFolder, err := catalog.Navigate(ctxt.pwd, args[1], false)
		if err != nil {
			return fmt.Errorf("upload: %w", err)
		}
		destFolder = newDestFolder
	}
	_, found := destFolder.Content()[srcFileName]
	if found {
		// Confirm overwrite? Or force user to delete first?
		return fmt.Errorf("upload: file %s already exists in %s", srcFileName, destFolder.Path())
	}
	newUUID := uuid.NewString()
	objectName, err := ctxt.drive.storage.UUIDToPath(newUUID)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	// Upload to storage.
	err = ctxt.drive.storage.UploadFile(srcFilePath, objectName)
	if err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	fmt.Printf("File %s uploaded to object %s\n", srcFileName, objectName)
	// Add file to catalog.
	catalog.AddFile(destFolder, srcFileName, newUUID)
	updateCatalog(*ctxt.drive, destFolder.Root())
	return nil
}
