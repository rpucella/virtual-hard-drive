
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
	"errors"
	"strings"

	"rpucella.net/virtual-hard-drive/internal/util"
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
	commands["exit"] = command{
		0, 0, commandQuit, "exit", "Bail out",
	}
	commands["help"] = command{
		0, 0, commandHelp, "help", "List available commands",
	}
	commands["ls"] = command{
		0, 1, commandLs, "ls [<folder>]", "List content of remote folder",
	}
	commands["cd"] = command{
		0, 1, commandCd, "cd [<folder>]", "Change working remote folder",
	}
	commands["info"] = command{
		1, 1, commandInfo, "info <file>", "Show remote file information",
	}
	commands["get"] = command{
		1, 1, commandGet, "get <file>", "Download remote file to disk",
	}
	commands["put"] = command{
		1, -1, commandPut, "put <local-file/folder> ... [<folder>]", "Upload local files to remote folder",
	}
	commands["catalog"] = command{
		0, 1, commandCatalog, "catalog [<folder>]", "Show catalog at remote folder",
	}
	commands["mkdir"] = command{
		1, 1, commandMkdir, "mkdir <folder>", "Create remote folder",
	}
	commands["hash"] = command{
		1, 1, commandHash, "hash <local-file>", "Compute CRC32C of local file",
	}
	commands["mv"] = command{
		2, 2, commandMv, "mv <folder/file> <folder/file>", "Move remote folder or file",
	}
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
		newCurr, err := virtualfs.NavigateDirectory(curr, args[0])
		if err != nil {
			return fmt.Errorf("ls: %w", err)
		}
		curr = newCurr
	}
	// Compute widths of names and sort the names list.
	width := 0
	names := make([]string, 0, len(curr.ContentList()))
	for _, k := range curr.ContentList() {
		l := len(k)
		if l > width {
			width = l
		}
		names = append(names, k)
	}
	width += 1
	sort.Strings(names)
	for _, k := range names {
		sub, _ := curr.GetContent(k)
		if dir := sub.AsDir(); dir != nil { 
			count, err := dir.CountFiles()
			if err != nil {
				return err
			}
			fmt.Printf("%*s     %6d\n", -width, dir.Name() + "/", count)
		}
	}
	for _, k := range names {
		file, _ := curr.GetContent(k)
		if file := file.AsFile(); file != nil { 
			fmt.Printf("%*s     %20s\n", -width, file.Name(), file.Updated().Format(time.RFC822))
		}
	}
	return nil
}

func commandCd(args []string, ctxt *context) error {
	path := "/"
	if len(args) > 0 {
		path = args[0]
	}
	newPwd, err := virtualfs.NavigateDirectory(ctxt.pwd, path)
	if err != nil {
		return fmt.Errorf("cd: %w", err)
	}
	ctxt.pwd = newPwd
	return nil
}

func commandCatalog(args []string, ctxt *context) error {
	curr := ctxt.pwd
	if len(args) > 0 {
		newCurr, err := virtualfs.NavigateDirectory(curr, args[0])
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
	file := fileObj.AsFile()
	if file == nil {
		return fmt.Errorf("file %s is not a file", fileObj.Name())
	}
	err = fileObj.Drive().Storage().RemoteInfo(file.UUID(), file.Metadata())
	if err != nil {
		return fmt.Errorf("remote: %w", err)
	}
	return nil
}

func isExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
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
	err = fileObj.Drive().Storage().DownloadFile(file.UUID(), file.Metadata(), fileObj.Name())
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	fmt.Printf("UUID %s downloaded to file %s\n", file.UUID(), fileObj.Name())
	return nil
}

func commandPut(args []string, ctxt *context) error {
	destFolder := ctxt.pwd
	lastArg := len(args)

	// Global to control whether to show a separating line above the upload info.
	first := true
	failures := 0

	var process func(string, virtualfs.VirtualFS) error
	process = func(srcFilePath string, destFolder virtualfs.VirtualFS) error {
		srcName := filepath.Base(srcFilePath)
		_, found := destFolder.GetContent(srcName)
		if found {
			// Confirm overwrite? Or force user to delete first?
			return fmt.Errorf("file %s already exists in %s", srcName, destFolder.Path())
		}
		isDir, err := isDirectory(srcFilePath)
		if err != nil {
			return err
		}
		if isDir {
			if destFolder.IsRoot() {
				return fmt.Errorf("cannot create drive")
			}
			if err := virtualfs.ValidateName(srcName); err != nil {
				return err
			}
			dirObj, err := virtualfs.CreateDirectory(destFolder, srcName)
			if err != nil {
				return err
			}
			files, err := ioutil.ReadDir(srcFilePath)
			if err != nil {
				return err
			}
			for _, f := range files {
				if strings.HasPrefix(f.Name(), ".") {
					// Skip hidden files.
					continue
				}
				if err := process(filepath.Join(srcFilePath, f.Name()), dirObj); err != nil {
					failures += 1
					fmt.Println(fmt.Errorf("Upload SKIPPED - %w\n", err))
				}
			}
		} else { 
			newUUID := uuid.NewString()
			drive := destFolder.Drive()
			if drive == nil {
				return fmt.Errorf("no drive for folder: %s", destFolder.Path())
			}
			// Upload to storage.
			if first {
				first = false
			} else {
				fmt.Println("----------------------------------------")
			}
			metadata, err := drive.Storage().UploadFile(srcFilePath, newUUID)
			if err != nil {
				return fmt.Errorf("put: %w", err)
			}
			fmt.Printf("Uploaded to UUID %s\n", newUUID)
			// Add file to catalog.
			if _, err := virtualfs.CreateFile(destFolder, srcName, newUUID, metadata); err != nil {
				return fmt.Errorf("put: %w", err)
			}
		}
		return nil
	}

	if len(args) > 1 {
		// Check the last argument. Does it describe a local file/folder or a remote folder?
		if exists, _ := isExists(args[lastArg - 1]); exists {
			// Last argument is a local file - is is also a remote folder?
			if _, err := virtualfs.NavigateDirectory(ctxt.pwd, args[lastArg - 1]); err == nil {
				// It's also a folder - ambiguous command.
				return fmt.Errorf("put: last arg is a local file/folder and a remote folder")
			}
			// It's not a folder, so all arguments are local.
		} else {
			// Last argument is not local - it better be a remote folder
			newDestFolder, err := virtualfs.NavigateDirectory(ctxt.pwd, args[lastArg - 1])
			if err != nil {
				// It's not a folder either - fail.
				return fmt.Errorf("put: %w", err)
			}
			destFolder = newDestFolder
			lastArg = lastArg - 1
		}
	}
	for i := 0; i < lastArg; i++ {
		if err := process(args[i], destFolder); err != nil {
			failures += 1
			fmt.Println(fmt.Errorf("Upload SKIPPED - %w\n", err))
		}
	}
	if failures > 0 {
		fmt.Printf("\nNumber of failures: %d\n", failures)
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
	crcw := util.NewCRCWriter(ioutil.Discard)
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
	parentObj, name, err := virtualfs.NavigateParent(ctxt.pwd, path)
	if err != nil { 
		return fmt.Errorf("mkdir: %w", err)
	}
	if parentObj.IsRoot() {
		return fmt.Errorf("mkdir: cannot create drive")
	}
	if err := virtualfs.ValidateName(name); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if _, err := virtualfs.CreateDirectory(parentObj, name); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return nil
}

func commandMv(args []string, ctxt *context) error {
	srcPath := args[0]
	tgtPath := args[1]

	srcObj, err := virtualfs.NavigatePath(ctxt.pwd, srcPath)
	if err != nil {
		return fmt.Errorf("mv: %w", err)
	}
	// Check destination path.
	tgtObj, err := virtualfs.CheckPath(ctxt.pwd, tgtPath)
	if err != nil {
		return fmt.Errorf("mv: %w", err)
	}
	if tgtObj != nil && tgtObj.IsDir() {
		// Target exists, and it's a directory.
		if err := srcObj.Move(tgtObj, srcObj.Name()); err != nil {
			return fmt.Errorf("mv: %w", err)
		}
		return nil
	}
	// Target name doesn't exist. Move away.
	tgtParent, tgtName, err := virtualfs.NavigateParent(ctxt.pwd, tgtPath)
	if err != nil {
		return fmt.Errorf("mv: %w", err)
	}
	if err := srcObj.Move(tgtParent, tgtName); err != nil {
		return fmt.Errorf("mv: %w", err)
	}
	return nil
}
