# Virtual Hard Drive


## Installation

Uses `go 1.16`.

To compile, run:

    make

You may need `CGO_ENABLED` set to install the `go-sqlite3` package.


## Creating initial database

Create the database in `~/.vhd`:

    sqlite3 ~/.vhd/catalog.db < schema.sql
    

## Adding a new virtual drive

    sqlite3 ~/.vhd/catalog.db
    INSERT INTO drives (name, description, host, address) values ('test-drive', 'A test drive', 'gcs', 'bucket');
    
Allowed values for `host` are:
- `gcs` for Google Cloud Storage, and `address` is the bucket name (requires authentication using google cloud SDK)
- `local` for a local file system, and `address` is an absolute path to the host folder
