package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	marker   = "&*^!@#$%\n\n\n"
)

func main() {
	if len(os.Args) < 2 {
		println("What are you searching for?")
		os.Exit(1)
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	cacheDir := openDir(dir)
	f := openCached(cacheDir)
	defer f.Close()

	lines, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	nbytes := 0
	for _, line := range bytes.Split(lines, []byte("\n")) {
		nbytes++ // Add one for the newline we remove from Split.
		nbytes += len(line)
		if !bytes.Contains(line, []byte(os.Args[1])) {
			continue
		}
		midx := bytes.LastIndex(lines[:nbytes], []byte(marker))
		if midx < 0 {
			panic("The cached documents have been malformed")
		}
		fmt.Printf("Found in %s\n\t%s\n",
			lines[bytes.LastIndex(lines[:midx], []byte("\n"))+1:midx],
			line)
	}
}

func cacheDocs(f *os.File) {
	listing := exec.Command("go", "list", "...")
	var buf bytes.Buffer
	listing.Stdout = &buf
	if err := listing.Run(); err != nil {
		panic(err)
	}

	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		if has := func(b []byte) bool { return bytes.Contains(line, b) }; has([]byte("internal")) ||
			has([]byte("cmd")) {
			continue
		}
		doc := exec.Command("go", "doc", string(line))
		doc.Stdout = f
		f.Write(append(append([]byte("\n\n"), line...), []byte(marker)...))
		if err := doc.Run(); err != nil {
			panic(err)
		}
	}
	f.Seek(0, io.SeekStart)
}

func openCached(cacheDir string) *os.File {
	mpath := exec.Command("go", "env", "GOMOD")
	var b strings.Builder
	mpath.Stdout = &b
	if err := mpath.Run(); err != nil {
		panic(err)
	}
	root := path.Dir(b.String())
	fileName := path.Join(cacheDir, root[strings.LastIndex(root, "/")+1:]+".txt")
	f, err := os.OpenFile(fileName, os.O_RDWR, 0o644)
	var pe *os.PathError
	if errors.As(err, &pe) && strings.Contains(pe.Error(), "no such file") {
		f, err = os.Create(fileName)
		if err != nil {
			panic(err)
		}
		cacheDocs(f)
	}
	return f
}

func openDir(dir string) string {
	cacheDir := path.Join(dir, "go-grepdocs")
	err := os.Mkdir(cacheDir, 0o755)
	switch {
	case err != nil && errors.Is(err, os.ErrExist):
		return cacheDir
	case err != nil:
		panic(err)
	}
	return cacheDir
}
