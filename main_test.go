package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// flag vars
var (
	update = flag.Bool("update", false, "update .golden files")
	keep   = flag.Bool("keep", false, "keep test output files")
)

var testFiles = []string{"helloworld.go", "example.json", "empty.json"}

func Map(arr []string, f func(string) string) []string {
	mapped := make([]string, len(arr))
	for i, v := range arr {
		mapped[i] = f(v)
	}
	return mapped
}

func TestBundler(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		prefix   string
		useConst bool
		varFunc  func(string, string) (string, error)
	}{
		{
			name:     "var",
			pkg:      "",
			prefix:   "gen",
			useConst: false,
			varFunc:  makeVarNameFromBaseName,
		},
		{
			name:     "const",
			pkg:      "somepackage",
			prefix:   "",
			useConst: true,
			varFunc:  makeVarNameFromFileName,
		},
	}

	Verbose = testing.Verbose()

	tmpDir, err := ioutil.TempDir("", "bundler")
	if err != nil {
		t.Fatal("error generating temp directory:", err)
	}
	defer func() {
		if !*keep {
			os.RemoveAll(tmpDir)
		} else {
			t.Log("kept test output in", tmpDir)
		}
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outFile := filepath.Join(tmpDir, test.name+"_out.go")
			goldenFile := filepath.Join("testdata", test.name+"_golden.go")

			bundler := NewBundler(outFile, test.pkg, test.prefix, test.useConst, test.varFunc)
			err = bundler.ProcessFiles(Map(
				testFiles,
				func(fn string) string {
					return filepath.Join("testdata", fn)
				})...,
			)
			if err != nil {
				t.Fatal("error generating file:", err)
			}

			current, err := ioutil.ReadFile(outFile)
			if err != nil {
				t.Fatal("can't open test file:", err)
			}

			if *update {
				ioutil.WriteFile(goldenFile, current, 0644)
			}

			expected, err := ioutil.ReadFile(goldenFile)
			if err != nil {
				t.Fatal("can't open golden test file:", err)
			}

			if !bytes.Equal(current, expected) {
				t.Error("current output differs from golden test file", goldenFile)
			}
		})
	}
}
