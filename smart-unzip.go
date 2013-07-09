package main

import (
	"archive/zip"
	"code.google.com/p/mahonia"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

func getBaseDir(s string) string {
	ext := path.Ext(s)
	return strings.Trim(s[0:len(s)-len(ext)], ".")
}

func convertFileName(fname string) string {
	d := mahonia.NewDecoder("cp932")
	return d.ConvertString(fname)
}

func extractOne(outpath string, f *zip.File) {
	rc, err := f.Open()
	if err != nil {
		fmt.Println("Can't open a zip entry:", err)
		return
	}
	defer rc.Close()

	w, err := os.Create(outpath)
	if err != nil {
		fmt.Println("Can't open a file to write:", err)
		return
	}
	defer w.Close()

	io.Copy(w, rc)
}

func extractAll(dir string, r *zip.ReadCloser) {
	var wg sync.WaitGroup
	for _, f := range r.File {
		if f.Mode().IsDir() {
			continue
		}
		wg.Add(1)
		name := convertFileName(f.FileHeader.Name)
		outpath := path.Join(dir, name)
		os.MkdirAll(path.Dir(outpath), 0755)
		go func(file *zip.File) {
			defer wg.Done()
			extractOne(outpath, file)
		}(f)
	}
	wg.Wait()
}

func isMeaningful(name string) (retval bool) {
	match, err := regexp.MatchString("^[0-9A-Za-z]{1,12}$", name)
	if err != nil {
		fmt.Println("Failed meaningful check:", err)
		retval = true
	} else if !match {
		retval = true
	}
	return
}

func isNameMeaningful(name, other string) (retval bool) {
	if isMeaningful(name) || !isMeaningful(other) {
		retval = true
	}
	return
}

func moveContents(src, dst string) (err error) {
	ls, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}
	for _, f := range ls {
		name := f.Name()
		err = os.Rename(path.Join(src, name), path.Join(dst, name))
		if err != nil {
			break
		}
	}
	return
}

func stripOneDir(target string) (err error) {
	middle, name := path.Split(target)
	root, midname := path.Split(path.Clean(middle))
	if isNameMeaningful(name, midname) {
		err = os.Rename(target, path.Join(root, name))
		if err == nil {
			os.Remove(middle)
		}
	} else {
		err = moveContents(target, middle)
		if err == nil {
			os.Remove(target)
		}
	}
	return
}

func smartUnzip(outdir, zipname string) {
	// Open reader.
	r, err := zip.OpenReader(zipname)
	if err != nil {
		fmt.Println("Can't open a zip file:", err)
		return
	}
	defer r.Close()

	// Extract all files.
	dir := path.Join(outdir, getBaseDir(zipname))
	extractAll(dir, r)

	// Check dir contents.
	ls, err := ioutil.ReadDir(dir)
	if len(ls) == 1 && ls[0].IsDir() {
		name := ls[0].Name()
		err = stripOneDir(path.Join(dir, name))
		if err != nil {
			fmt.Println("Did't remove dir:", err)
		}
	}
}

func main() {
	var wg sync.WaitGroup
	outdir := "outdir"
	for _, zip := range os.Args[1:] {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			smartUnzip(outdir, file)
			fmt.Println("Extracted:", file)
		}(zip)
	}
	wg.Wait()
}
