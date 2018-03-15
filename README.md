[![GoDoc](https://godoc.org/github.com/wvh/bundle?status.svg)](https://godoc.org/github.com/wvh/bundle)
[![Build Status](https://travis-ci.org/wvh/bundle.svg?branch=master)](https://travis-ci.org/wvh/bundle)
[![Go Report Card](https://goreportcard.com/badge/github.com/wvh/bundle)](https://goreportcard.com/report/github.com/wvh/bundle)

# bundle

This command generates a Go code file that bundles external (text) files into variables or constants. For use with `go:generate`.

You could use this to compile JSON or HTML templates into Go binaries without having to deal with all sorts of string-inside-a-string problems or clumsy editing.
