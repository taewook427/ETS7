// test834 : ETS7 RepairIt
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ===== config =====
const CHUNK_SIZE = 256 * 1024

var TgtPath string = ""
var PrtPath string = ""
var CheckOnly bool = false

func readArgs() {
	// read args
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&PrtPath, "prt", "", "parity storage dir")
	fs.BoolVar(&CheckOnly, "check", false, "check only mode")
	fs.Parse(os.Args[1:])
	if len(fs.Args()) > 0 {
		TgtPath = fs.Args()[0]
	}

	// trim args
	if strings.HasPrefix(TgtPath, "\"") && strings.HasSuffix(TgtPath, "\"") {
		TgtPath = TgtPath[1 : len(TgtPath)-1]
	}
	if strings.HasPrefix(PrtPath, "\"") && strings.HasSuffix(PrtPath, "\"") {
		PrtPath = PrtPath[1 : len(PrtPath)-1]
	}
}

// ===== Parity Repair =====
type Task struct {
	RelPath string
	Action  rune // 'R': repair, 'G': generate
}

// conduct repair or generation
func worker(taskChan <-chan Task, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[CRITICAL] %v\n", r)
		}
	}()
	for task := range taskChan {
		switch task.Action {
		case 'R':
			procRepair(task.RelPath)
		case 'G':
			procGen(task.RelPath)
		}
	}
}

// check integrity and repair
func procRepair(relPath string) {
	tgtFullPath := filepath.Join(TgtPath, relPath)
	prtFullPath := filepath.Join(PrtPath, relPath)

	// open parity file
	pInfo, err := os.Stat(prtFullPath)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	if pInfo.IsDir() {
		return
	}
	pf, err := os.Open(prtFullPath)
	if err != nil {
		return
	}
	defer pf.Close()

	// setup teereader for master crc
	masterHash := crc32.NewIEEE()
	tr := io.TeeReader(pf, masterHash)

	// 8B size reading
	var savedSize int64
	if err := binary.Read(tr, binary.LittleEndian, &savedSize); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}

	// open target file
	tf, err := os.OpenFile(tgtFullPath, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	defer tf.Close()

	// check file size
	tInfo, _ := tf.Stat()
	if tInfo.Size() != savedSize {
		fmt.Printf("[ERROR] filesize not match %s (tgt %d, prt %d)\n", relPath, tInfo.Size(), savedSize)
		return
	}

	// calculate chunk count and load CRC array
	numChunks := max(int((savedSize+CHUNK_SIZE-1)/CHUNK_SIZE), 1)
	savedCRCs := make([]uint32, numChunks)
	for i := range numChunks {
		binary.Read(tr, binary.LittleEndian, &savedCRCs[i])
	}

	// read parity block
	var savedParity [CHUNK_SIZE]byte
	pLen := min(CHUNK_SIZE, savedSize)
	if pLen > 0 {
		if _, err := io.ReadFull(tr, savedParity[:pLen]); err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			return
		}
	}

	// verify master crc
	var savedMasterCRC uint32
	if err := binary.Read(pf, binary.LittleEndian, &savedMasterCRC); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	if masterHash.Sum32() != savedMasterCRC {
		fmt.Printf("[CRITICAL] parity file corrupted %s\n", relPath)
		return
	}

	// scan target file
	brokenChunkIdx := -1
	brokenCount := 0
	var buf [CHUNK_SIZE]byte
	for i := 0; i < numChunks; i++ {
		n, err := io.ReadFull(tf, buf[:])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			fmt.Printf("[ERROR] %v\n", err)
			return
		}
		if n > 0 {
			for j := n; j < CHUNK_SIZE; j++ {
				buf[j] = 0 // zero-padding
			}
		}
		currentCRC := crc32.ChecksumIEEE(buf[:])
		if currentCRC != savedCRCs[i] {
			brokenCount++
			brokenChunkIdx = i
		}
	}

	// check integrity result
	if brokenCount == 0 {
		return // ok
	}
	if brokenCount > 1 {
		fmt.Printf("[CRITICAL] multiple bit-rot %s (%d chunks broken)\n", relPath, brokenCount)
		return
	}
	fmt.Printf("  [REPAIR] single bit-rot %s (chunk idx %d)\n", relPath, brokenChunkIdx)
	if CheckOnly {
		return
	}

	// repair = parity ^ others
	var reconstructed [CHUNK_SIZE]byte
	copy(reconstructed[:], savedParity[:])
	if _, err = tf.Seek(0, io.SeekStart); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	for i := range numChunks {
		n, err := io.ReadFull(tf, buf[:])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			fmt.Printf("[ERROR] %v\n", err)
			return
		}
		if i == brokenChunkIdx {
			continue // skip broken chunk
		}
		if n > 0 {
			for j := n; j < CHUNK_SIZE; j++ {
				buf[j] = 0 // zero-padding
			}
		}
		for j := range CHUNK_SIZE {
			reconstructed[j] ^= buf[j]
		}
	}

	// write reconstructed chunk
	if _, err = tf.Seek(int64(brokenChunkIdx)*CHUNK_SIZE, io.SeekStart); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	writeSize := CHUNK_SIZE
	if int64(brokenChunkIdx) == int64(numChunks-1) {
		remainder := int(savedSize % CHUNK_SIZE)
		if remainder != 0 {
			writeSize = remainder
		}
	}
	if _, err = tf.Write(reconstructed[:writeSize]); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	fmt.Printf("  [REPAIR] %s repair done!\n", relPath)
}

// generate new parity file
func procGen(relPath string) {
	fmt.Printf("  [GEN] new file detected %s\n", relPath)
	if CheckOnly {
		return
	}
	tgtFullPath := filepath.Join(TgtPath, relPath)
	prtFullPath := filepath.Join(PrtPath, relPath)

	// open target file
	tf, err := os.Open(tgtFullPath)
	if err != nil {
		return
	}
	defer tf.Close()
	tInfo, _ := tf.Stat()
	fileSize := tInfo.Size()

	// make folder, new parity file
	os.MkdirAll(filepath.Dir(prtFullPath), 0755)
	pf, err := os.Create(prtFullPath)
	if err != nil {
		return
	}
	defer pf.Close()

	// write filesize, get chunk number
	binary.Write(pf, binary.LittleEndian, fileSize)
	numChunks := max(int((fileSize+CHUNK_SIZE-1)/CHUNK_SIZE), 1)

	// calc chunk crc, make parity block
	crcs := make([]uint32, numChunks)
	var parityBlock [CHUNK_SIZE]byte
	var buf [CHUNK_SIZE]byte
	for i := range numChunks {
		n, err := io.ReadFull(tf, buf[:])
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			fmt.Printf("[ERROR] %v\n", err)
			return
		}
		if n > 0 {
			for j := n; j < CHUNK_SIZE; j++ {
				buf[j] = 0 // zero-padding
			}
		}
		crcs[i] = crc32.ChecksumIEEE(buf[:])
		for j := range CHUNK_SIZE {
			parityBlock[j] ^= buf[j]
		}
	}

	// write crcs and parity
	for i := range numChunks {
		binary.Write(pf, binary.LittleEndian, crcs[i])
	}
	pLen := min(CHUNK_SIZE, fileSize)
	if pLen > 0 {
		if _, err := pf.Write(parityBlock[:pLen]); err != nil {
			fmt.Printf("[ERROR] %v\n", err)
			return
		}
	}

	// write master hash
	if _, err := pf.Seek(0, io.SeekStart); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	masterHash := crc32.NewIEEE()
	if _, err := io.Copy(masterHash, pf); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	if err := binary.Write(pf, binary.LittleEndian, masterHash.Sum32()); err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return
	}
	fmt.Printf("  [GEN] %s parity generated!\n", relPath)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while repairit.main: %v", r), 0644)
		}
	}()
	readArgs()
	if TgtPath == "" || PrtPath == "" {
		fmt.Println("Usage: repairit [-prt <parity_storage>] [-check] <target_dir>")
		return
	}

	// task chan, worker pool
	taskChan := make(chan Task, 100)
	var wg sync.WaitGroup
	numWorkers := max(runtime.NumCPU(), 2)
	for range numWorkers {
		wg.Add(1)
		go worker(taskChan, &wg)
	}

	// scan target directory
	filepath.WalkDir(TgtPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("[ERROR] %v at %s\n", err, path)
			return nil
		}
		if path == TgtPath || d.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(TgtPath, path)
		prtFullPath := filepath.Join(PrtPath, relPath)
		if _, err := os.Stat(prtFullPath); err == nil {
			taskChan <- Task{RelPath: relPath, Action: 'R'}
		} else {
			taskChan <- Task{RelPath: relPath, Action: 'G'}
		}
		return nil
	})
	close(taskChan)
	wg.Wait()

	// clean orphan parity files and empty folders
	filepath.WalkDir(PrtPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("[ERROR] %v at %s\n", err, path)
			return nil
		}
		if path == PrtPath {
			return nil
		}
		relPath, _ := filepath.Rel(PrtPath, path)
		tgtFullPath := filepath.Join(TgtPath, relPath)

		if _, err := os.Stat(tgtFullPath); os.IsNotExist(err) {
			fmt.Printf("  [DEL] deleted item detected %s\n", relPath)
			if !CheckOnly {
				if d.IsDir() {
					os.RemoveAll(path)
					return filepath.SkipDir
				} else {
					os.Remove(path)
				}
			}
		}
		return nil
	})

	fmt.Println("[OK] All tasks done!")
}
