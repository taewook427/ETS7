// test000 : ETS7 ImgConv
package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ===== config =====
var TgtPath string = ""
var SrcPath string = ""
var DstPath string = ""
var UseCRC bool = false

func readArgs() {
	// read args
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&SrcPath, "src", "", "source dir to sync")
	fs.StringVar(&DstPath, "dst", "", "destination dir to sync")
	fs.BoolVar(&UseCRC, "crc", false, "enable CRC32 check")
	fs.Parse(os.Args[1:])
	if len(fs.Args()) > 0 {
		TgtPath = fs.Args()[0]
	}

	// trim args
	if strings.HasPrefix(TgtPath, "\"") && strings.HasSuffix(TgtPath, "\"") {
		TgtPath = TgtPath[1 : len(TgtPath)-1]
	}
	if strings.HasPrefix(SrcPath, "\"") && strings.HasSuffix(SrcPath, "\"") {
		SrcPath = SrcPath[1 : len(SrcPath)-1]
	}
	if strings.HasPrefix(DstPath, "\"") && strings.HasSuffix(DstPath, "\"") {
		DstPath = DstPath[1 : len(DstPath)-1]
	}
}

// ===== Hash Printer =====
func HashPrint() {
	if TgtPath == "" {
		fmt.Println("[ERROR] empty target path")
		return
	}

	// open target file
	f, err := os.Open(TgtPath)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	defer f.Close()

	// generate hash and bond together
	hCrc := crc32.NewIEEE()
	hMd5 := md5.New()
	hSha1 := sha1.New()
	hSha256 := sha256.New()
	hSha512 := sha512.New()
	hSha3_256 := sha3.New256()
	hSha3_512 := sha3.New512()
	mw := io.MultiWriter(hCrc, hMd5, hSha1, hSha256, hSha512, hSha3_256, hSha3_512)

	// streaming hash
	if _, err := io.Copy(mw, f); err != nil {
		fmt.Printf("[ERROR] failed to hash file: %v\n", err)
		return
	}
	c32Bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(c32Bytes, hCrc.Sum32())

	// print hash
	algo := []string{"CRC32(LE)", "MD5", "SHA1", "SHA2-256", "SHA2-512", "SHA3-256", "SHA3-512"}
	values := [][]byte{
		c32Bytes,
		hMd5.Sum(nil),
		hSha1.Sum(nil),
		hSha256.Sum(nil),
		hSha512.Sum(nil),
		hSha3_256.Sum(nil),
		hSha3_512.Sum(nil),
	}
	for i, a := range algo {
		fmt.Printf("[%s]\n%s\n%s\n\n", a, hex.EncodeToString(values[i]), base64.StdEncoding.EncodeToString(values[i]))
	}
}

// ===== Directory Sync =====
func copyFile(src string, dst string) error {
	// open and get metadata from src
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// create dst and copy data
	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
}

func getCRC32(filePath string) (uint32, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

func compare(src string, dst string) bool {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false
	}
	dstInfo, err := os.Stat(dst)
	if err != nil {
		return false
	}

	// check size and modtime
	if !UseCRC {
		return srcInfo.Size() == dstInfo.Size() && srcInfo.ModTime().Before(dstInfo.ModTime().Add(10*time.Second))
	}

	// check CRC32
	srcCrc, err := getCRC32(src)
	if err != nil {
		return false
	}
	dstCrc, err := getCRC32(dst)
	if err != nil {
		return false
	}
	return srcCrc == dstCrc
}

func SyncDir() {
	if SrcPath == "" || DstPath == "" {
		fmt.Println("[ERROR] both src and dst must be set")
		return
	}
	fmt.Printf("Sync from %s to %s\n", SrcPath, DstPath)
	defer fmt.Println("Sync completed")

	// sync Src has but Dst doesn't have
	err := filepath.WalkDir(SrcPath, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// get relative path
		rel, err := filepath.Rel(SrcPath, srcPath)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(DstPath, rel)

		// make directory and copy file
		if d.IsDir() {
			fmt.Printf("[Verifying Dir] %s\n", srcPath)
			if info, err := d.Info(); err == nil {
				return os.MkdirAll(dstPath, info.Mode())
			}
			return os.MkdirAll(dstPath, 0755)
		}
		if !compare(srcPath, dstPath) {
			fmt.Printf("[Copying File] %s\n", srcPath)
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("[ERROR] sync failed: %v\n", err)
		return
	}

	// sync Dst has but Src doesn't have
	err = filepath.WalkDir(DstPath, func(dstPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dstPath == DstPath {
			return nil
		}

		// get relative path
		rel, err := filepath.Rel(DstPath, dstPath)
		if err != nil {
			return err
		}
		srcPath := filepath.Join(SrcPath, rel)

		// remove file
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("[Removing] %s\n", dstPath)
			if d.IsDir() {
				if err := os.RemoveAll(dstPath); err != nil {
					return err
				}
				return filepath.SkipDir
			}
			return os.Remove(dstPath)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("[ERROR] cleanup failed: %v\n", err)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while syncit.main: %v", r), 0644)
		}
	}()
	readArgs()
	if SrcPath == "" && DstPath == "" {
		HashPrint()
	} else {
		SyncDir()
	}
}
