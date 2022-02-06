
package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"

	"rpucella.net/virtual-hard-drive/internal/storage"
)

func main() {
	// Parse arguments if needed.
	//args := os.Args[1:]

	reader := bufio.NewReader(os.Stdin)

	// We need to put this here to break the initialization loop for command.

	fmt.Println("------------------------------------------------------------")
	fmt.Println("                   VIRTUAL HARD DRIVE")
	fmt.Println("------------------------------------------------------------")
	fmt.Print("Drives: ")
	for k, _ := range drives {
		fmt.Printf("%s ", k)
	}
	fmt.Println()

	initializeCommands()

	// buckets, err := storage.ListBuckets("virtual-hard-drive")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// for _, name := range buckets {
	// 	fmt.Printf("Bucket: %v\n", name)
	// }
	files, err := storage.ListFiles("vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002")
	if err != nil {
		fmt.Println(err)
	}
	for _, name := range files {
		fmt.Printf("%v\n", name)
	}

	// Fetch the catalog for the default drive.
	default_drive := "test"
	default_catalog := "7b5d41cc-86d6-11ec-a8a3-0242ac120002"
	path, err := storage.UIDToPath("7b5d41cc-86d6-11ec-a8a3-0242ac120002")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Path for %s: %s\n", default_catalog, path)

	content, err := storage.ReadFile(drives[default_drive].bucket, path)
	catalog := string(content)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Content: %s\n", catalog)

	ctxt := repl_context{default_drive, "/", "", ""}
	done := false
	for !done {
		fmt.Printf("\n%s:%s> ", ctxt.drive, ctxt.pwd)
		line, _ := reader.ReadString('\n')
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		comm := fields[0]
		args := fields[1:]
		commObj, ok := commands[comm]
		if !ok {
			fmt.Printf("Unknown command - %s\n", comm)
			continue
		}
		if len(args) != commObj.argCount {
			fmt.Printf("Wrong number of arguments to %s - expected %d\n", comm, commObj.argCount)
			continue
		}
		ctxt, done = commObj.process(args, ctxt)	
	}
	
}

// For most errors, don't try to recover, just stop.

type command struct{
	argCount int
	process func([]string, repl_context)(repl_context, bool)
	help string
}

type repl_context struct{
	drive string
	pwd string
	catalog string    // Should be a DirTree pointer.
	current string    // Should be a pointer in the DirTree.
}

var commands map[string]command

func stop(message string) {
	fmt.Println(message)
	os.Exit(1)
}

type drive struct{
	provider string
	bucket string
	catalog string
}

var drives = map[string]drive {
	"test": drive{"gcs", "vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002", "7b5d41cc-86d6-11ec-a8a3-0242ac120002"},
}
