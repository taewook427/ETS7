// test000 : ETS7 FileConv
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/chai2010/webp"
	"github.com/signintech/gopdf"
	"github.com/taewook427/USAG-KOX/BaseUI"
)

type Page struct {
	Window    fyne.Window
	isWorking bool
	logList   *widget.List

	Files    []string
	Selected int
	Logs     []string
	Modes    []string
	ConvMode string
}

func (p *Page) Main() {
	p.isWorking = false
	p.Files = make([]string, 0)
	p.Selected = -1
	p.Logs = make([]string, 0)
	p.Modes = []string{"to JPG", "to PNG", "to WEBP", "to WEBP(LL)", "Merge Images(+X)", "Merge Images(+Y)", "Merge Images(PDF)", "Merge PDF"}
	p.ConvMode = p.Modes[0]

	p.Window = MainApp.NewWindow("File Convert")
	p.Fill()
	p.Window.Resize(fyne.NewSize(720*BaseUI.FyneSize, 480*BaseUI.FyneSize))
	p.Window.CenterOnScreen()
	p.Window.ShowAndRun()
}

func (p *Page) Fill() {
	// group0: file list
	list0 := widget.NewList(
		func() int {
			return len(p.Files)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("File Name Placeholder")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < len(p.Files) {
				item.(*widget.Label).SetText(p.Files[id])
			}
		},
	)
	list0.OnSelected = func(id widget.ListItemID) { p.Selected = id }
	list0.OnUnselected = func(id widget.ListItemID) { p.Selected = -1 }

	// group1: file buttons
	btn1a := widget.NewButton("Add", func() {
		if p.isWorking {
			return
		}
		BaseUI.ListAddFile(list0, &p.Files)
	})
	btn1b := widget.NewButton("Del", func() {
		if p.isWorking || p.Selected < 0 || p.Selected >= len(p.Files) {
			return
		}
		BaseUI.ListDelTgt(list0, &p.Files, p.Selected)
		p.Selected = -1
	})
	btn1c := widget.NewButton("Clear", func() {
		if p.isWorking {
			return
		}
		BaseUI.ListDelTgt(list0, &p.Files, len(p.Files))
		p.Selected = -1
	})
	btn1d := widget.NewButton("Sort", func() {
		if p.isWorking {
			return
		}
		sort.Strings(p.Files)
		list0.Refresh()
	})
	box1 := container.NewGridWithColumns(4, btn1a, btn1b, btn1c, btn1d)

	// group2: logview
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

	// group3: conv buttons
	sel3 := widget.NewSelect(p.Modes, func(value string) { p.ConvMode = value })
	sel3.SetSelected(p.ConvMode)
	btn3 := widget.NewButton("Conv", func() {
		if p.isWorking || len(p.Files) == 0 {
			return
		}
		p.isWorking = true
		p.clearLog()
		switch p.ConvMode {
		case "to JPG", "to PNG", "to WEBP", "to WEBP(LL)":
			go p.ConvImg()
		case "Merge Images(+X)", "Merge Images(+Y)":
			go p.MergeImg()
		case "Merge Images(PDF)":
			go p.ImgToPDF()
		case "Merge PDF":
			go p.MergePDF()
		}
	})
	box3 := container.NewBorder(nil, nil, nil, btn3, sel3)

	// group4: main grid
	box4 := container.NewHSplit(container.NewBorder(nil, box1, nil, nil, list0), container.NewBorder(nil, box3, nil, nil, p.logList))
	box4.Offset = 0.5
	p.Window.SetContent(box4)
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

func (p *Page) decodeImgs() ([]image.Image, []string) {
	imgs := make([]image.Image, 0)
	names := make([]string, 0)

	for i, srcPath := range p.Files {
		p.addLog(fmt.Sprintf("[%d/%d] src: %s", i+1, len(p.Files), filepath.Base(srcPath)))

		// open file
		file, err := os.Open(srcPath)
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}

		// decode image
		var img image.Image
		img, _, err = image.Decode(file)
		file.Close()
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}

		p.addLog(fmt.Sprintf("  [OK] added as index %d", len(imgs)))
		imgs = append(imgs, img)
		names = append(names, filepath.Base(srcPath))
	}
	return imgs, names
}

func (p *Page) ConvImg() {
	defer p.clearWork()
	for i, srcPath := range p.Files {
		p.addLog(fmt.Sprintf("[%d/%d] src: %s", i+1, len(p.Files), filepath.Base(srcPath)))

		// open and decode target file
		file, err := os.Open(srcPath)
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}
		var img image.Image
		img, _, err = image.Decode(file)
		file.Close()
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}

		// set destination extension
		var ext string
		switch p.ConvMode {
		case "to JPG":
			ext = ".jpg"
		case "to PNG":
			ext = ".png"
		case "to WEBP", "to WEBP(LL)":
			ext = ".webp"
		default:
			p.addLog(fmt.Sprintf("  [ERROR] invalid mode %s", p.ConvMode))
			continue
		}
		dstPath := strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ext

		// create output file
		outFile, err := os.Create(dstPath)
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}
		switch p.ConvMode {
		case "to JPG":
			err = jpeg.Encode(outFile, img, &jpeg.Options{Quality: 90})
		case "to PNG":
			err = png.Encode(outFile, img)
		case "to WEBP":
			err = webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: 80})
		case "to WEBP(LL)":
			err = webp.Encode(outFile, img, &webp.Options{Lossless: true, Quality: 100})
		}
		outFile.Close()
		if err == nil {
			p.addLog(fmt.Sprintf("  [OK] dst: %s", filepath.Base(dstPath)))
		} else {
			os.Remove(dstPath)
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
		}
	}
}

func (p *Page) MergeImg() {
	defer p.clearWork()

	// decode images
	imgs, names := p.decodeImgs()
	if len(imgs) == 0 {
		p.addLog("[ERROR] No valid images")
		return
	}

	// generate whole canvas
	var tWidth, tHeight int
	for _, img := range imgs {
		bounds := img.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()
		if p.ConvMode == "Merge Images(+X)" {
			tWidth += w
			tHeight = max(tHeight, h)
		} else {
			tHeight += h
			tWidth = max(tWidth, w)
		}
	}
	canvas := image.NewRGBA(image.Rect(0, 0, tWidth, tHeight))

	// draw images to canvas
	var curX, curY int
	for i, img := range imgs {
		bounds := img.Bounds()
		w := bounds.Dx()
		h := bounds.Dy()

		drawRect := image.Rect(curX, curY, curX+w, curY+h)
		draw.Draw(canvas, drawRect, img, bounds.Min, draw.Src)
		p.addLog(fmt.Sprintf("[OK] src: %s -> loc: (%d, %d)", names[i], curX, curY))

		if p.ConvMode == "Merge Images(+X)" {
			curX += w
		} else {
			curY += h
		}
	}

	// save to one webp file
	dstPath := filepath.Join(filepath.Dir(p.Files[0]), fmt.Sprintf("merged_%d.webp", len(names)))
	outFile, err := os.Create(dstPath)
	if err != nil {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
		return
	}
	err = webp.Encode(outFile, canvas, &webp.Options{Lossless: false, Quality: 80})
	outFile.Close()
	if err == nil {
		p.addLog(fmt.Sprintf("[SUCCESS] dst: %s", filepath.Base(dstPath)))
	} else {
		os.Remove(dstPath)
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
	}
}

func (p *Page) ImgToPDF() {
	defer p.clearWork()

	// decode images
	imgs, names := p.decodeImgs()
	if len(imgs) == 0 {
		p.addLog("[ERROR] No valid images")
		return
	}

	// generate new PDF document
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	for i, img := range imgs {
		p.addLog(fmt.Sprintf("[%d/%d] Encoding to PDF: %s", i+1, len(imgs), names[i]))

		// in-memory JPEG encoding
		buf := new(bytes.Buffer)
		err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}

		// gopdf image holder binding
		imgHolder, err := gopdf.ImageHolderByBytes(buf.Bytes())
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}

		// create new PDF page, calculate layout to fit A4 page
		pdf.AddPage()
		bounds := img.Bounds()
		imgW := float64(bounds.Dx())
		imgH := float64(bounds.Dy())

		a4W := gopdf.PageSizeA4.W
		a4H := gopdf.PageSizeA4.H
		scale := a4W / imgW
		if imgH*scale > a4H {
			scale = a4H / imgH
		}
		finalW := imgW * scale
		finalH := imgH * scale
		offsetX := (a4W - finalW) / 2
		offsetY := (a4H - finalH) / 2

		// draw image
		err = pdf.ImageByHolder(imgHolder, offsetX, offsetY, &gopdf.Rect{W: finalW, H: finalH})
		if err != nil {
			p.addLog(fmt.Sprintf("  [ERROR] %v", err))
			continue
		}
	}

	// save to one pdf file
	dstPath := filepath.Join(filepath.Dir(p.Files[0]), fmt.Sprintf("merged_%d.pdf", len(names)))
	err := pdf.WritePdf(dstPath)
	if err == nil {
		p.addLog(fmt.Sprintf("[SUCCESS] dst: %s", filepath.Base(dstPath)))
	} else {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
	}
}

func (p *Page) MergePDF() {
	defer p.clearWork()

	// generate new PDF document
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	merged := 0

	// warm up decoding
	func() {
		tmpPath := filepath.Join(os.TempDir(), "fileconv_warmup.pdf")
		defer os.Remove(tmpPath)

		// create a valid PDF to trigger the first-call
		warmupPdf := gopdf.GoPdf{}
		warmupPdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
		warmupPdf.AddPage()
		warmupPdf.WritePdf(tmpPath)

		defer func() { recover() }()
		pdf.ImportPage(tmpPath, 1, "/MediaBox")
	}()

	// decode and merge PDF
	for i, srcPath := range p.Files {
		p.addLog(fmt.Sprintf("[%d/%d] Merging PDF: %s", i+1, len(p.Files), filepath.Base(srcPath)))

		pageNo := 1
		for {
			var tpl int
			var isEnded bool

			// cover decodeing
			func() {
				defer func() {
					if r := recover(); r != nil {
						isEnded = true
					}
				}()
				tpl = pdf.ImportPage(srcPath, pageNo, "/MediaBox")
			}()
			if isEnded || tpl <= 0 {
				break
			}

			// make new page and import
			pdf.AddPage()
			pdf.UseImportedTemplate(tpl, 0, 0, gopdf.PageSizeA4.W, gopdf.PageSizeA4.H)
			pageNo++
			merged++
		}

		if pageNo > 1 {
			p.addLog(fmt.Sprintf("  [OK] imported %d pages", pageNo-1))
		} else {
			p.addLog("  [ERROR] No vaild pages found")
		}
	}

	// save to one pdf file
	if merged == 0 {
		p.addLog("[ERROR] No valid PDF pages imported")
		return
	}
	dstPath := filepath.Join(filepath.Dir(p.Files[0]), fmt.Sprintf("merged_%d.pdf", len(p.Files)))
	err := pdf.WritePdf(dstPath)
	if err == nil {
		p.addLog(fmt.Sprintf("[SUCCESS] dst: %s", filepath.Base(dstPath)))
	} else {
		p.addLog(fmt.Sprintf("[ERROR] %v", err))
	}
}

var MainApp fyne.App

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while fileconv.main: %v", r), 0644)
		}
	}()

	// init fynesize
	exePath, _ := os.Executable()
	realPath, _ := filepath.EvalSymlinks(exePath)
	cfgPath := filepath.Join(filepath.Dir(realPath), "fileconv_size.txt")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		os.WriteFile(cfgPath, []byte("1"), 0644)
	} else {
		data, _ := os.ReadFile(cfgPath)
		var val float32
		fmt.Sscanf(string(data), "%f", &val)
		if val > 0 {
			BaseUI.FyneSize = val
		}
	}

	// start app
	MainApp = app.New()
	MainApp.Settings().SetTheme(new(BaseUI.U1Theme))
	var p Page
	p.Main()
}
