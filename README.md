# go-unexport

[![Go Report Card](https://goreportcard.com/badge/github.com/Quasilyte/go-unexport)](https://goreportcard.com/report/github.com/Quasilyte/go-unexport)

# Overview

Tries to unexport as much symbols as possible for a given package under a current workspace.

It's mostly intended for `internal/` packages where it's simpler to change API and all your
clients are most likely reside in the same repository. In other words, it's useful for big
monoliths or command-line apps with a lot of code (which can include legacy).

This tool automatically does unexporting, the only thing you should do is to review the diff
and commit it, if it makes sense. If you would like to keep some symbols exported even though
they are only used inside the package itself, one can specify `skip` flag.

# Installation and usage (quick start)

This install `go-unexport` binary under your `$GOPATH/bin`:

```bash
go get github.com/Quasilyte/go-unexport
```

If `$GOPATH/bin` is under your system `$PATH`, `go-unexport` command should be available after that.<br>
This should print the help message:

```bash
go-unexport --help
```

To run unexporting process, do:

```bash
go-unexport -v package/import/path
```

Flag `-v` turns on verbose mode.

# Implementation notice

This tool does zero analysis on its own. I've used `go-rename` to do all the heavy lifting.

**Pros:**
* If you trust `go-rename`, you can trust `go-unexport`. It's unlikely that it will break your program.
* Maintainance cost is almost close to zero.

**Cons:**
* The execution time is slow.

# Motivation

Keep the number of exported symbols low. 

It's hard to maintain minimal exported symbol set for a big projects, so this tool can help a little bit in that regard.
