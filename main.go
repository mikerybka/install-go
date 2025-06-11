package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func main() {
	v := version()
	filename := fmt.Sprintf("%s.%s-%s.tar.gz", v, runtime.GOOS, runtime.GOARCH)
	u := fmt.Sprintf("https://go.dev/dl/%s", filename)
	res, err := http.Get(u)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	gzr, err := gzip.NewReader(res.Body)
	if err != nil {
		panic(err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			panic(err)
		}
		if header.Name == "go/bin/go" {
			f, err := os.Create("/usr/bin/go")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			_, err = io.Copy(f, tr)
			if err != nil {
				panic(err)
			}
		}
		if header.Name == "go/bin/gofmt" {
			f, err := os.Create("/usr/bin/gofmt")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			_, err = io.Copy(f, tr)
			if err != nil {
				panic(err)
			}
		}
	}
}

func version() string {
	res, err := http.Get("https://go.dev/VERSION?m=text")
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(b), "\n")
	return lines[0]
}
