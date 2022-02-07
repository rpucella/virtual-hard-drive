
package main

import (
	"fmt"
	"os"
	"strings"
	"bufio"

	"rpucella.net/virtual-hard-drive/internal/storage"
	"rpucella.net/virtual-hard-drive/internal/catalog"
)

type drive struct{
	name string
	catalog string
	storage storage.Storage
}

type command struct{
	minArgCount int
	maxArgCount int
	process func([]string, *context)error
	usage string
	help string
}

type context struct{
	commands map[string]command
	drives map[string]drive
	drive drive
	pwd catalog.Catalog
	exit bool         // Set to true to exit the main loop.
}

func main() {
	// Parse arguments if needed.
	//args := os.Args[1:]

	reader := bufio.NewReader(os.Stdin)

	commands := initializeCommands()
	drives, default_drive := initializeDrives()

	fmt.Println("------------------------------------------------------------")
	fmt.Println("                   VIRTUAL HARD DRIVE                       ")
	fmt.Println("------------------------------------------------------------")
	fmt.Print("Drives: ")
	for k, _ := range drives {
		fmt.Printf("%s ", k)
	}
	fmt.Println()

	// buckets, err := storage.ListBuckets("virtual-hard-drive")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// for _, name := range buckets {
	// 	fmt.Printf("Bucket: %v\n", name)
	// }
	
	// files, err := storage.ListFiles("vhd-7b5d41cc-86d6-11ec-a8a3-0242ac120002")
	// if err != nil {
	// 	stop(err)
	// }
	// for _, name := range files {
	// 	fmt.Printf("%v\n", name)
	// }

	cat, err := fetchCatalog(default_drive)
	if err != nil {
		stop(err)
	}

	ctxt := context{
		commands,
		drives,
		default_drive,
		cat,
		false,
	}
	
	for !ctxt.exit {
		// Keep going until we nullify the context (flag for quitting)
		fmt.Printf("\n%s:%s ", ctxt.drive.name, ctxt.pwd.Path())
		line, _ := reader.ReadString('\n')
		fields := strings.Fields(line)
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
			fmt.Println(err)
			continue
		}
	}
}

// For most errors, don't try to recover, just stop.

func stop(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func fetchCatalog(dr drive) (catalog.Catalog, error) {
	cat_uuid := dr.catalog
	path, err := storage.UUIDToPath(cat_uuid)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %w", err)
	}
	fmt.Printf("Fetching catalog for %s\n", dr.storage.Name())
	content, err := dr.storage.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch catalog: %s", err)
	}
	cat, err := catalog.NewCatalog(content)
	return cat, nil
}
