package main

import (
	"archive/zip"
	"code.google.com/p/mahonia"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

func getBaseDir(s string) string {
	ext := path.Ext(s)
	return strings.Trim(s[0 : len(s)-len(ext)], ".")
}

func convertFileName(fname string) string {
	d := mahonia.NewDecoder("cp932")
	return d.ConvertString(fname)
}

func extractOne(outpath string, f *zip.File) {
	rc, err := f.Open()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rc.Close()

	w, err := os.Create(outpath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer w.Close()

	io.Copy(w, rc)
}

func extractAll(dir string, r *zip.ReadCloser) {
	var wg sync.WaitGroup
	for _, f := range r.File {
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

func smartUnzip(outdir string, zipname string) {
	// Open reader.
	r, err := zip.OpenReader(zipname)
	if err != nil {
		fmt.Println(err.Error())
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
		err = os.Rename(path.Join(dir, name), path.Join(outdir, name))
		if err == nil {
			os.Remove(dir)
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
			fmt.Println("extracted ", file)
		}(zip)
	}
	wg.Wait()
}
