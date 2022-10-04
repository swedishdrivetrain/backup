# Joomla-backup 
[![Go](https://github.com/AnimeNL/joomla-backup/actions/workflows/go.yml/badge.svg)](https://github.com/AnimeNL/joomla-backup/actions/workflows/go.yml)

Golang script to backup data and databases on the Joomla server

## Build & Release
Build and release is automated via Github actions. Status is displayed in the above badge. 
The latest stable version of the script can be found in the releases section of this repo.

When contributing make sure you push a tag on the commit you want to release or else the build fails.

---
## Config
The script requires a config located on one of the following locations. paths starting with `./` means relative to the location of the script, `/` means from the root of the filesystem
* `./`
* `./config`
* `/etc/jbackup`

The config needs to have the `.yml` or `.yaml` extension and has the following structure and options: 
```yaml
# Global config items: 
# debug is of type BOOL
global: 
  debug: true
  dryrun: true

# database config contains credentials and a list of databases.
# credentials is of type STRING.
# databases if of type STRING in LIST format.
database:
  credentials: 
    username: example_username
    password: example_password
  databases:
    - "list"
    - "of"
    - "databases"
    - "to"
    - "be"
    - "dumped"

# path config.
# databasedumps is of type STRING and provides the location to store the dumps.
# filedumps is of type STRING in LIST format. Provide filepaths to be compressed and added to the backup.
paths:
  databasedumps: "/path/to/where/databases/need/to/be/dumped"
  filedumps: 
    - "list/of/file/paths/to/be/compressed/and/backed/up"

# sftp holds config for an SFTP endpoint to which the backup is sent.
# url is of type STRING.
# port is of type INT.
# username is of type STRING.
# password is of type STRING.
sftp:
  url: "domain.name"
  port: 22
  username: example_username
  password: example_password
```
All options in the file are **MANDATORY**
