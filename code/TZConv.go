// test833 : ETS7 TZConv
package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func compress(dirPath string) error {
	// get absolute path and base name
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	baseName := filepath.Base(absPath)
	outputName := baseName + ".tar.zst"

	numCPU := max(runtime.NumCPU(), 2)
	fmt.Printf("[COMPRESS] %s -> %s (%d cores)\n", absPath, outputName, numCPU)
	outFile, err := os.Create(outputName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// zstd config, make TAR writer
	zw, err := zstd.NewWriter(outFile,
		zstd.WithEncoderLevel(zstd.SpeedBetterCompression),
		zstd.WithEncoderConcurrency(numCPU),
	)
	if err != nil {
		return err
	}
	defer zw.Close()
	tw := tar.NewWriter(zw)
	defer tw.Close()

	parentDir := filepath.Dir(absPath)
	err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, e error) error {
		// get relative path
		if e != nil {
			return e
		}
		relPath, err := filepath.Rel(parentDir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}

		// make TAR header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)
		fmt.Printf("compressing %s\n", header.Name)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// write file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	})
	return err
}

func decompress(filePath string) error {
	// open target file
	numCPU := max(runtime.NumCPU(), 2)
	fmt.Printf("[DECOMPRESS] %s (%d cores)\n", filePath, numCPU)
	inFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	// config zstd, make TAR reader
	zr, err := zstd.NewReader(inFile, zstd.WithDecoderConcurrency(numCPU))
	if err != nil {
		return err
	}
	defer zr.Close()
	tr := tar.NewReader(zr)

	for {
		// read header, trim abspath
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Clean(header.Name)
		if strings.HasPrefix(target, "..") || filepath.IsAbs(target) {
			continue
		}

		// write file
		fmt.Printf("decompressing %s\n", target)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while tzconv.main: %v", r), 0644)
		}
	}()

	// check args
	if len(os.Args) < 2 {
		fmt.Println("[ERROR] requires target directory or *.tar.zst file")
		os.Exit(1)
	}
	target := os.Args[1]
	if strings.HasPrefix(target, "\"") && strings.HasSuffix(target, "\"") {
		target = target[1 : len(target)-1]
	}
	fi, err := os.Stat(target)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		os.Exit(1)
	}

	// compress or decompress
	if fi.IsDir() {
		err = compress(target)
	} else if strings.HasSuffix(strings.ToLower(target), ".tar.zst") {
		err = decompress(target)
	} else {
		fmt.Println("[ERROR] requires target directory or *.tar.zst file")
		os.Exit(1)
	}

	// check error
	if err != nil {
		fmt.Printf("\n[ERROR] %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\n[SUCCESS] process completed")
}
