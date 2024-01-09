# doppel
An utility to find files with different names but same contents

# Build

To build the main executable:

```shell
go build cmd/doppel
```

# Usage

Basic usage is to run with the directories you want to search as arguments:

```shell
./doppel dir1/ dir2/
```

The directories will be recursively searched according to the rules of Golang's
[`Walkdir`](https://pkg.go.dev/path/filepath#WalkDir).

The format of the output is:

```
HASH_HEX
[TAB]filepath_1
[TAB]filepath_2
HASH_HEX_2
[TAB]filepath_1
[TAB]filepath_2
[TAB]filepath_3
...
```

where the `HASH_HEX`es are the sum of the files (currently BLAKE3-256). If two
or more files share the same hash, they are considered doppelgangers.

There's also an "uniques" mode, where files that *didn't* have a doppelganger are
printed. For that, the paths are simply printed:

```
filepath_1
filepath_2
filepath_3
...
```

Run `./doppel -h` for other options.