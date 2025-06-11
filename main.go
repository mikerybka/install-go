package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func write(tr *tar.Reader, destDir string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar entry: %w", err)
		}

		targetPath := filepath.Join(destDir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		case tar.TypeLink:
			linkTarget := filepath.Join(destDir, hdr.Linkname)
			if err := os.Link(linkTarget, targetPath); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
		case tar.TypeChar:
			fallthrough
		case tar.TypeBlock:
			fallthrough
		case tar.TypeFifo:
			mode := uint32(hdr.Mode)
			if err := syscall.Mknod(targetPath, mode, int(hdr.Devmajor)<<8|int(hdr.Devminor)); err != nil {
				return fmt.Errorf("failed to create special file: %w", err)
			}
		default:
			fmt.Printf("skipping unsupported file type: %s\n", hdr.Name)
		}
	}
	return nil
}

func main() {
	v := version()
	err := os.RemoveAll("/usr/local/go")
	if err != nil {
		panic(err)
	}
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
	destDir := "/usr/local"
	err = write(tr, destDir)
	if err != nil {
		panic(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "/root"
	}

	// Write to .bashrc if not already
	txt := fmt.Sprintf("\n# Go\nexport PATH=/usr/local/go/bin:%s/go/bin:$PATH\n", home)
	path := filepath.Join(home, ".bashrc")
	f, err := os.Open(path)
	if err != nil {
		f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(txt); err != nil {
			panic(err)
		}
		return
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
	if !strings.Contains(string(b), txt) {
		f, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err := f.WriteString(txt); err != nil {
			panic(err)
		}
		return
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
