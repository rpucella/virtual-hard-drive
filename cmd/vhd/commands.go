
package main

import (
	"fmt"
	"sort"
	"github.com/google/uuid"
	"path/filepath"
	"time"
	"io/ioutil"
	"io"
	"os"

	"rpucella.net/virtual-hard-drive/internal/virtualfs"
)

type command struct{
	minArgCount int
	maxArgCount int
	process func([]string, *context)error
	usage string
	help string
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
	commands["exit"] = command{0, 0, commandQuit, "exit", "Bail out"}
	commands["help"] = command{0, 0, commandHelp, "help", "List available commands"}
	//commands["drive"] = command{0, 1, commandDrive, "drive [<name>]", "List or select drive"}
	commands["ls"] = command{0, 1, commandLs, "ls [<folder>]", "List content of folder"}
	commands["cd"] = command{0, 1, commandCd, "cd [<folder>]", "Change working folder"}
	commands["info"] = command{1, 1, commandInfo, "info <file>", "Show file information"}
	commands["remote"] = command{1, 1, commandRemote, "remote <file>", "Show remote file information"}
	commands["get"] = command{1, 1, commandGet, "get <file>", "Download file to disk"}
	commands["put"] = command{1, 2, commandPut, "put <local-file> [<folder>]", "Upload local file to drive folder"}
	commands["catalog"] = command{0, 1, commandCatalog, "catalog [<folder>]", "Show catalog at folder"}
	commands["mkdir"] = command{1, 1, commandMkdir, "mkdir <folder>", "Create folder"}
	commands["hash"] = command{1, 1, commandHash, "hash <local-file>", "Compute CRC32C of local file"}
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

func commandLs(args []string, ctxt *context) error {
	curr := ctxt.pwd
	if len(args) > 0 {
		newCurr, err := virtualfs.Navigate(curr, args[0])
		if err != nil {
			return fmt.Errorf("ls: %w", err)
		}
		curr = newCurr
	}
	dirs := make([]string, 0, len(curr.ContentList()))
	files := make([]string, 0, len(curr.ContentList()))
	for _, k := range curr.ContentList() {
		sub, _ := curr.GetContent(k)
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
		file, _ := curr.GetContent(name)
		if file := file.AsFile(); file != nil { 
			fmt.Printf("%*s     %-40s  %s\n", -width, name, file.UUID(), file.Updated().Format(time.RFC822))
		}
	}
	return nil
}

func commandCd(args []string, ctxt *context) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}
	newPwd, err := virtualfs.Navigate(ctxt.pwd, path)
	if err != nil {
		return fmt.Errorf("cd: %w", err)
	}
	ctxt.pwd = newPwd
	return nil
}

func commandCatalog(args []string, ctxt *context) error {
	curr := ctxt.pwd
	if len(args) > 0 {
		newCurr, err := virtualfs.Navigate(curr, args[0])
		if err != nil {
			return fmt.Errorf("catalog: %w", err)
		}
		curr = newCurr
	}
	virtualfs.Print(curr)
	return nil
}

func commandInfo(args []string, ctxt *context) error {
	fileObj, err := virtualfs.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}
	fileObj.Print()
	return nil
}

func commandRemote(args []string, ctxt *context) error {
	fileObj, err := virtualfs.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("remote: %w", err)
	}
	file := fileObj.AsFile()
	if file == nil {
		return fmt.Errorf("file %s is not a file", fileObj.Name())
	}
	objectName, err := fileObj.Drive().Storage().UUIDToPath(file.UUID())
	if err != nil {
		return fmt.Errorf("remote: %w", err)
	}
	err = fileObj.Drive().Storage().RemoteInfo(objectName)
	return nil
}

func commandGet(args []string, ctxt *context) error {
	fileObj, err := virtualfs.NavigateFile(ctxt.pwd, args[0])
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	file := fileObj.AsFile()
	if file == nil {
		return fmt.Errorf("file %s is not a file", fileObj.Name())
	}
	objectName, err := fileObj.Drive().Storage().UUIDToPath(file.UUID())
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	err = fileObj.Drive().Storage().DownloadFile(objectName, fileObj.Name())
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	fmt.Printf("File %s downloaded to %s\n", objectName, fileObj.Name())
	return nil
}

func commandPut(args []string, ctxt *context) error {
	srcFilePath := args[0]
	srcFileName := filepath.Base(srcFilePath)
	destFolder := ctxt.pwd
	if len(args) == 2 {
		newDestFolder, err := virtualfs.Navigate(ctxt.pwd, args[1])
		if err != nil {
			return fmt.Errorf("put: %w", err)
		}
		destFolder = newDestFolder
	}
	_, found := destFolder.GetContent(srcFileName)
	if found {
		// Confirm overwrite? Or force user to delete first?
		return fmt.Errorf("put: file %s already exists in %s", srcFileName, destFolder.FullPath())
	}
	newUUID := uuid.NewString()
	drive := destFolder.Drive()
	if drive == nil {
		return fmt.Errorf("no drive for folder: %s", destFolder.FullPath())
	}
	objectName, err := drive.Storage().UUIDToPath(newUUID)
	if err != nil {
		return fmt.Errorf("put: %w", err)
	}
	// Upload to storage.
	// TODO: get metadata back and put in AddFile?
	err = drive.Storage().UploadFile(srcFilePath, objectName)
	if err != nil {
		return fmt.Errorf("put: %w", err)
	}
	fmt.Printf("File %s uploaded to object %s\n", srcFileName, objectName)
	// Add file to catalog.
	virtualfs.AddFile(destFolder, srcFileName, newUUID, "")
	if err := drive.Update(); err != nil {
		// TODO: revert catalog changes?
		return fmt.Errorf("cannot update catalog: %w", err)
	}
	return nil
}

func commandHash(args []string, ctxt *context) error {
	srcFilePath := args[0]
	src, err := os.Open(srcFilePath)
	if err != nil {
		return fmt.Errorf("os.Open: %v", err)
	}
	defer src.Close()

	// ioutil.Discard is basically /dev/null
	crcw := NewCRCwriter(ioutil.Discard)
	if _, err := io.Copy(crcw, src); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	fmt.Printf("CRC32C:  %x\n", crcw.Sum())
	return nil
}

func commandMkdir(args []string, ctxt *context) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}
	newFolder, err := virtualfs.NavigateCreateLast(ctxt.pwd, path)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	drive := newFolder.Drive()
	if drive == nil {
		return fmt.Errorf("no drive for creating folder")
	}
	if err := drive.Update(); err != nil {
		// TODO: revert catalog changes?
		return fmt.Errorf("cannot update catalog: %w", err)
	}
	return nil
}

