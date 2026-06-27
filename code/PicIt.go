// test829 : ETS7 Picture It
package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strings"

	_ "image/jpeg"
	"image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	_ "github.com/chai2010/webp"
	"github.com/k-atusa/USAG-Lib/Bencode"
	"github.com/k-atusa/USAG-Lib/Bencrypt"
	"github.com/taewook427/USAG-KOX/BaseUI"
	"github.com/taewook427/USAG-KOX/TP1"
)

type Page struct {
	Window    fyne.Window
	Logs      []string
	logList   *widget.List
	isWorking bool

	// packing
	TmpList  []string
	TgtFile  string
	IsSubtle bool

	// unpacking
	TgtList []string
}

func (p *Page) Main() {
	p.Window = MainApp.NewWindow("Picture It")
	p.Fill()
	p.Window.Resize(fyne.NewSize(640*BaseUI.FyneSize, 360*BaseUI.FyneSize))
	p.Window.CenterOnScreen()
	p.Window.ShowAndRun()
}

func (p *Page) Fill() {
	// group0: pack unpack targets
	lbl0a := widget.NewLabel("No template selected")
	lbl0b := widget.NewLabel("No target selected")
	lbl0c := widget.NewLabel("No folder selected")
	btn0a := widget.NewButton("Templates", func() {
		if p.isWorking {
			return
		}
		res, err := BaseUI.ZenityMultiFiles("Select Templates")
		if len(res) == 0 || err != nil {
			return
		}
		p.TmpList = make([]string, 0)
		for _, v := range res {
			if slices.Contains([]string{".png", ".jpg", ".jpeg", ".webp"}, strings.ToLower(filepath.Ext(v))) {
				p.TmpList = append(p.TmpList, v)
			}
		}
		if len(p.TmpList) == 0 {
			lbl0a.SetText("No template selected")
		} else {
			lbl0a.SetText(fmt.Sprintf("Selected %d templates", len(p.TmpList)))
		}
	})
	btn0b := widget.NewButton("Target", func() {
		if p.isWorking {
			return
		}
		res, err := BaseUI.ZenityFile("Select target")
		if res == "" || err != nil {
			return
		}
		p.TgtFile = res
		p.TgtList = make([]string, 0)
		lbl0b.SetText(res)
		lbl0c.SetText("No folder selected")
	})
	btn0c := widget.NewButton("Folder", func() {
		if p.isWorking {
			return
		}
		res, err := BaseUI.ZenityFolder("Select folder")
		if res == "" || err != nil {
			return
		}
		p.TgtList = make([]string, 0)
		entries, err := os.ReadDir(res)
		if err != nil {
			return
		}
		for _, entry := range entries {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if slices.Contains([]string{".png", ".jpg", ".jpeg", ".webp"}, ext) {
				p.TgtList = append(p.TgtList, filepath.Join(res, entry.Name()))
			}
		}
		if len(p.TgtList) == 0 {
			lbl0c.SetText("No folder selected")
		} else {
			p.TgtFile = ""
			lbl0b.SetText("No target selected")
			lbl0c.SetText(fmt.Sprintf("Selected %d images", len(p.TgtList)))
		}
	})

	// group1: main buttons
	ent1 := widget.NewEntry()
	ent1.PlaceHolder = "Packing key"
	ent1.Text = Config.Key
	btn1 := widget.NewButton("Go", func() {
		if p.isWorking || ((len(p.TmpList) == 0 || p.TgtFile == "") && len(p.TgtList) == 0) {
			return
		}
		p.isWorking = true
		p.clearLog()
		if len(p.TgtList) == 0 {
			go p.Pack(ent1.Text)
		} else {
			go p.Unpack(ent1.Text)
		}
	})
	btn1.Importance = widget.HighImportance
	chk1 := widget.NewCheck("subtle", func(b bool) { p.IsSubtle = b })

	// log view
	p.Logs = make([]string, 0)
	p.logList = widget.NewList(
		func() int {
			return len(p.Logs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Log Message Placeholder")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < len(p.Logs) {
				item.(*widget.Label).SetText(p.Logs[id])
			}
		},
	)

	// main layout
	box1a := container.NewGridWithColumns(1, btn0a, btn0b, btn0c, btn1)
	box1b := container.NewBorder(nil, nil, nil, chk1, ent1)
	mainForm := container.NewBorder(nil, nil, box1a, nil, container.NewGridWithColumns(1, lbl0a, lbl0b, lbl0c, box1b))
	p.Window.SetContent(container.NewBorder(mainForm, nil, nil, nil, p.logList))
}

func (p *Page) addLog(msg string) {
	p.Logs = append(p.Logs, msg)
	p.logList.Refresh()
	p.logList.ScrollToBottom()
}

func (p *Page) clearLog() {
	p.Logs = make([]string, 0)
	p.logList.Refresh()
}

func (p *Page) clearWork() {
	if r := recover(); r != nil {
		dialog.ShowError(fmt.Errorf("[PANIC] %v", r), p.Window)
	}
	p.isWorking = false
	p.addLog("finished work")
}

func (p *Page) Pack(key string) {
	defer p.clearWork()
	fi, err := os.Stat(p.TgtFile)
	if err != nil {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
		return
	}
	fileSize := int(fi.Size())

	// decode template images
	var validImgs []image.Image
	var validCaps []int
	for _, tmpPath := range p.TmpList {
		file, err := os.Open(tmpPath)
		if err != nil {
			continue
		}
		srcImg, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			continue
		}
		capacity := capImage(srcImg, p.IsSubtle)
		if capacity == 0 {
			continue
		}
		validImgs = append(validImgs, srcImg)
		validCaps = append(validCaps, capacity)
	}
	if len(validImgs) == 0 {
		p.addLog("[ERROR] No valid images found")
		return
	}

	// split target size to chunks
	var chunkSizes []int
	var imgIndices []int
	dataLeft := fileSize
	tmpIdx := 0
	for dataLeft > 0 {
		idx := tmpIdx % len(validCaps)
		pureCap := validCaps[idx] - 8 // header 8 bytes
		take := pureCap
		if dataLeft < pureCap {
			take = dataLeft
		}
		chunkSizes = append(chunkSizes, take)
		imgIndices = append(imgIndices, idx)
		dataLeft -= take
		tmpIdx++
	}
	totalChunks := len(chunkSizes)
	p.addLog(fmt.Sprintf("Target is split into %d chunks", totalChunks))

	// open target file
	f, err := os.Open(p.TgtFile)
	if err != nil {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
		return
	}
	defer f.Close()

	// generate image bits
	for i := 0; i < totalChunks; i++ {
		imgIdx := imgIndices[i]
		srcImg := validImgs[imgIdx]
		capacity := validCaps[imgIdx]
		take := chunkSizes[i]

		// NRGBA convert
		bounds := srcImg.Bounds()
		nrgbaImg := image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				nrgbaImg.Set(x, y, srcImg.At(x, y))
			}
		}

		// read chunk from stream
		chunkData := make([]byte, take)
		if _, err := f.Read(chunkData); err != nil {
			p.addLog(fmt.Sprintf("[ERROR] chunk %d: %v", i, err))
			return
		}

		// fill chunknum, totalnum, datasize
		plaintext := make([]byte, capacity)
		binary.LittleEndian.PutUint16(plaintext[0:2], uint16(i))
		binary.LittleEndian.PutUint16(plaintext[2:4], uint16(totalChunks))
		binary.LittleEndian.PutUint32(plaintext[4:8], uint32(take))
		copy(plaintext[8:8+take], chunkData)

		// image ID and set subtle flag
		id := Bencrypt.Random(16)
		if p.IsSubtle {
			id[0] |= 0x80
		} else {
			id[0] &^= 0x80
		}

		// encrypt
		targetKey := genKey(id, key)
		iv, tag, cipher, err := encrypt(plaintext, targetKey)
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] chunk %d: %v", i, err))
			return
		}

		// encode header with 2-bits
		offset := 0
		offset = writeBits(nrgbaImg.Pix, id, 2, offset)
		offset = writeBits(nrgbaImg.Pix, iv, 2, offset)
		offset = writeBits(nrgbaImg.Pix, tag, 2, offset)

		// encode data
		bitsPerSubpixel := 4
		if p.IsSubtle {
			bitsPerSubpixel = 2
		}
		writeBits(nrgbaImg.Pix, cipher, bitsPerSubpixel, offset)

		// set output name=
		outName := fmt.Sprintf("pack_%d.png", i)
		outPath := filepath.Join(filepath.Dir(p.TgtFile), outName)
		outFile, err := os.Create(outPath)
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] %s: %v", outName, err))
			return
		}

		// save image
		err = png.Encode(outFile, nrgbaImg)
		outFile.Close()
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] %s: %v", outName, err))
			return
		}
		p.addLog(fmt.Sprintf("[SUCCESS] %s", outName))
	}
}

func (p *Page) Unpack(key string) {
	defer p.clearWork()
	tmpDir := "picit"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// decode images
	var totalChunks uint16 = 0
	hasTotal := false
	for _, imgPath := range p.TgtList {
		// open and decode file
		file, err := os.Open(imgPath)
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] Open fail %s: %v", filepath.Base(imgPath), err))
			continue
		}
		srcImg, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] Decode fail %s: %v", filepath.Base(imgPath), err))
			continue
		}

		// NRGBA convert
		bounds := srcImg.Bounds()
		nrgbaImg := image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				nrgbaImg.Set(x, y, srcImg.At(x, y))
			}
		}

		// decode header with 2-bits
		offset := 0
		var id, iv, tag []byte
		id, offset = readBits(nrgbaImg.Pix, 16, 2, offset)
		iv, offset = readBits(nrgbaImg.Pix, 12, 2, offset)
		tag, offset = readBits(nrgbaImg.Pix, 16, 2, offset)

		// get subtle flag and capacity
		isSubtle := (id[0] & 0x80) != 0
		capacity := capImage(srcImg, isSubtle)
		if capacity == 0 {
			p.addLog(fmt.Sprintf("[ERROR] Invalid image with %s", filepath.Base(imgPath)))
			continue
		}

		// extracy data
		bitsPerSubpixel := 4
		if isSubtle {
			bitsPerSubpixel = 2
		}
		cipher, _ := readBits(nrgbaImg.Pix, capacity, bitsPerSubpixel, offset)

		// generater key and decrypt
		targetKey := genKey(id, key)
		if targetKey == nil {
			p.addLog(fmt.Sprintf("[ERROR] Key generation failed for %s", filepath.Base(imgPath)))
			continue
		}
		plaintext, err := decrypt(iv, tag, cipher, targetKey)
		if err != nil {
			p.addLog(fmt.Sprintf("[PASS] Decryption failed for %s", filepath.Base(imgPath)))
			continue
		}

		// parse metadata
		inum := binary.LittleEndian.Uint16(plaintext[0:2])
		total := binary.LittleEndian.Uint16(plaintext[2:4])
		take := binary.LittleEndian.Uint32(plaintext[4:8])
		if 8+take > uint32(len(plaintext)) {
			p.addLog(fmt.Sprintf("[ERROR] Corrupted metadata in %s", filepath.Base(imgPath)))
			continue
		}
		if !hasTotal {
			totalChunks = total
			hasTotal = true
		}

		// save data to temp chunk
		chunkData := plaintext[8 : 8+take]
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d.tmp", inum))
		if err := os.WriteFile(chunkPath, chunkData, 0644); err != nil {
			p.addLog(fmt.Sprintf("[ERROR] Failed to save temp chunk %d: %v", inum, err))
			continue
		}

		p.addLog(fmt.Sprintf("[%d/%d] Decrypted and saved: %s", inum+1, total, filepath.Base(imgPath)))
	}

	// check integrity
	if !hasTotal || totalChunks == 0 {
		p.addLog("[ERROR] No valid chunks were recovered")
		return
	}
	for i := uint16(0); i < totalChunks; i++ {
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d.tmp", i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			p.addLog(fmt.Sprintf("[ERROR] Missing chunk index %d", i))
			return
		}
	}

	// merge chunks into output
	outPath := filepath.Join(filepath.Dir(p.TgtList[0]), "output."+Config.Format)
	outFile, err := os.Create(outPath)
	if err != nil {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
		return
	}
	defer outFile.Close()

	for i := uint16(0); i < totalChunks; i++ {
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d.tmp", i))
		cData, err := os.ReadFile(chunkPath)
		if err != nil {
			p.addLog(fmt.Sprintf("[ERROR] chunk %d: %v", i, err))
			return
		}

		if _, err := outFile.Write(cData); err != nil {
			p.addLog(fmt.Sprintf("[ERROR] chunk %d: %v", i, err))
			return
		}
	}

	p.addLog(fmt.Sprintf("[SUCCESS] Saved to %s", filepath.Base(outPath)))
}

// ===== helpers =====
func genKey(id []byte, key string) []byte {
	hm := new(Bencrypt.HashMaster)
	if err := hm.Init("arg2st", 0, 0); err != nil {
		return nil
	}
	_, k, err := hm.KDF(Bencode.NormPW(key), id)
	if err == nil {
		return k
	} else {
		return nil
	}
}

func encrypt(data []byte, key []byte) (iv []byte, tag []byte, cipher []byte, err error) {
	sm := new(Bencrypt.SymMaster)
	if err := sm.Init("gcm1", key); err != nil {
		return nil, nil, nil, err
	}
	res, err := sm.EnBin(data)
	if err == nil {
		return res[0:12], res[len(res)-16:], res[12 : len(res)-16], nil
	} else {
		return nil, nil, nil, err
	}
}

func decrypt(iv []byte, tag []byte, cipher []byte, key []byte) (data []byte, err error) {
	sm := new(Bencrypt.SymMaster)
	if err := sm.Init("gcm1", key); err != nil {
		return nil, err
	}
	temp := make([]byte, len(iv)+len(cipher)+len(tag))
	copy(temp, iv)
	copy(temp[len(iv):], cipher)
	copy(temp[len(iv)+len(cipher):], tag)
	return sm.DeBin(temp)
}

func capImage(img image.Image, is4px bool) int {
	x, y := img.Bounds().Dx(), img.Bounds().Dy()
	if x%2 != 0 || y%2 != 0 || x < 256 || y < 256 {
		return 0
	}
	if is4px {
		return 3*x*y/4 - 44
	} else {
		return 3*x*y/2 - 88
	}
}

func readBits(pix []byte, numBytes int, bitsPerSubpixel int, startSubpixel int) ([]byte, int) {
	data := make([]byte, numBytes)
	idx := startSubpixel
	if bitsPerSubpixel == 2 {
		for i := 0; i < numBytes; i++ {
			var b byte
			b |= (pix[idx] & 0x03) << 6
			idx++
			b |= (pix[idx] & 0x03) << 4
			idx++
			b |= (pix[idx] & 0x03) << 2
			idx++
			b |= (pix[idx] & 0x03)
			idx++
			data[i] = b
		}
	} else if bitsPerSubpixel == 4 {
		for i := 0; i < numBytes; i++ {
			var b byte
			b |= (pix[idx] & 0x0F) << 4
			idx++
			b |= (pix[idx] & 0x0F)
			idx++
			data[i] = b
		}
	}
	return data, idx
}

func writeBits(pix []byte, data []byte, bitsPerSubpixel int, startSubpixel int) int {
	idx := startSubpixel
	if bitsPerSubpixel == 2 {
		for _, b := range data {
			parts := [4]byte{
				(b >> 6) & 0x03,
				(b >> 4) & 0x03,
				(b >> 2) & 0x03,
				b & 0x03,
			}
			for _, p := range parts {
				pix[idx] = (pix[idx] & 0xFC) | p
				idx++
			}
		}
	} else if bitsPerSubpixel == 4 {
		for _, b := range data {
			parts := [2]byte{
				(b >> 4) & 0x0F,
				b & 0x0F,
			}
			for _, p := range parts {
				pix[idx] = (pix[idx] & 0xF0) | p
				idx++
			}
		}
	}
	return idx
}

// ===== config =====
type U1Config struct {
	Size   float32 `json:"size"`
	Format string  `json:"format"`
	Key    string  `json:"key"`
}

func (c *U1Config) Load() error {
	data, err := os.ReadFile(filepath.Join(TP1.GetPath(), "picit_cfg.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.Size = 1.0
			c.Format = "bin"
			c.Key = ""
			return c.Store()
		}
		return err
	}
	err = json.Unmarshal(data, c)
	BaseUI.FyneSize = c.Size
	return err
}

func (c *U1Config) Store() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(TP1.GetPath(), "picit_cfg.json"), data, 0644)
}

var MainApp fyne.App
var Config U1Config

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while picit.main: %v", r), 0644)
		}
	}()
	Config.Load()
	BaseUI.FyneSize = Config.Size
	MainApp = app.New()
	MainApp.Settings().SetTheme(new(BaseUI.U1Theme))
	var p Page
	p.Main()
}
