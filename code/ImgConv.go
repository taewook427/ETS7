// test831 : ETS7 ImgConv
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/chai2010/webp"
	"github.com/taewook427/USAG-KOX/BaseUI"
)

type Page struct {
	Window    fyne.Window
	FilePath  string
	ImgWidth  int
	ImgHeight int

	// shared UI components
	pathLabel *widget.Label
	sizeLabel *widget.Label
	imgCanvas *canvas.Image

	// selected values
	effectSel string
	fmtSel    string
}

func (p *Page) Main() {
	p.Window = MainApp.NewWindow("Image Convert")
	p.Fill()
	if len(os.Args) > 1 {
		p.loadImage(os.Args[1]) // direct load
	}
	p.Window.Resize(fyne.NewSize(720*BaseUI.FyneSize, 540*BaseUI.FyneSize))
	p.Window.CenterOnScreen()
	p.Window.ShowAndRun()
}

func (p *Page) Fill() {
	// line1: image info show
	p.pathLabel = widget.NewLabel("No file loaded")
	p.sizeLabel = widget.NewLabel("(0 x 0)")
	btn1 := widget.NewButton("Load", func() {
		path, err := BaseUI.ZenityFile("Select Image")
		if path != "" && err == nil {
			p.loadImage(path)
		}
	})
	btn1.Importance = widget.HighImportance
	row1 := container.NewBorder(nil, nil, p.sizeLabel, btn1, p.pathLabel)

	// line2: effects function
	effects := []string{
		"Rotate Left", "Rotate Right", "Rotate", "Flip (X)", "Flip (Y)",
		"Add Image Left", "Add Image Right", "Add Image Top", "Add Image Bottom",
		"Gray Scale", "Invert Color", "Emphasize Red", "Emphasize Green", "Emphasize Blue",
	}
	sel2 := widget.NewSelect(effects, func(value string) { p.effectSel = value })
	sel2.SetSelected(effects[0])

	ent2 := widget.NewEntry()
	ent2.SetPlaceHolder("Intensity")

	btn2 := widget.NewButton("Apply", func() { p.applyEffect(ent2.Text) })
	btn2.Importance = widget.HighImportance
	row2 := container.NewBorder(nil, nil, nil, btn2, container.NewGridWithColumns(2, sel2, ent2))

	// line3: output config
	ent3a := widget.NewEntry()
	ent3a.SetPlaceHolder("Width")
	ent3b := widget.NewEntry()
	ent3b.SetPlaceHolder("Height")
	ent3c := widget.NewEntry()
	ent3c.SetPlaceHolder("Output Name")

	formats := []string{"JPG", "PNG", "WEBP", "WEBP(LL)", "ICO"}
	sel3 := widget.NewSelect(formats, func(value string) { p.fmtSel = value })
	sel3.SetSelected(formats[0])

	btn3 := widget.NewButton("Save", func() { p.saveImage(ent3a.Text, ent3b.Text, ent3c.Text) })
	btn3.Importance = widget.DangerImportance
	row3 := container.NewBorder(nil, nil, nil, btn3, container.NewGridWithColumns(4, ent3a, ent3b, ent3c, sel3))

	// canvas image show
	p.imgCanvas = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	p.imgCanvas.FillMode = canvas.ImageFillContain

	// main layout
	mainLayout := container.NewBorder(container.NewVBox(row1, row2, row3), nil, nil, nil, p.imgCanvas)
	p.Window.SetContent(mainLayout)
}

func (p *Page) loadImage(srcPath string) {
	// open file and decode
	file, err := os.Open(srcPath)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to open file: %v", err), p.Window)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to decode image: %v", err), p.Window)
		return
	}

	// update image info
	p.FilePath = srcPath
	bounds := img.Bounds()
	p.ImgWidth = bounds.Dx()
	p.ImgHeight = bounds.Dy()

	// UI update
	p.pathLabel.SetText(srcPath)
	p.sizeLabel.SetText(fmt.Sprintf("(%d x %d)", p.ImgWidth, p.ImgHeight))
	p.imgCanvas.Image = img
	p.imgCanvas.Refresh()
}

func (p *Page) applyEffect(intensity string) {
	// prepare images
	if p.imgCanvas.Image == nil || p.FilePath == "" {
		return
	}
	srcImg := p.imgCanvas.Image
	bounds := srcImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	var dstImg image.Image

	switch p.effectSel {
	case "Rotate Left": // counterclock 90
		rgba := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rgba.Set(y, w-1-x, srcImg.At(x, y))
			}
		}
		dstImg = rgba

	case "Rotate Right": // clockwise 90
		rgba := image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rgba.Set(h-1-y, x, srcImg.At(x, y))
			}
		}
		dstImg = rgba

	case "Rotate": // arbitrary degrees
		angle := 90.0
		if intensity != "" {
			if val, err := strconv.ParseFloat(intensity, 64); err == nil {
				angle = val
			}
		}
		if angle == 0 {
			dstImg = srcImg
			break
		}

		// convert to radian
		radians := angle * math.Pi / 180.0
		cosA := math.Cos(radians)
		sinA := math.Sin(radians)
		origCx := float64(w) / 2.0
		origCy := float64(h) / 2.0

		// calculate new canvas size
		x1, y1 := -origCx, -origCy
		x2, y2 := origCx, -origCy
		x3, y3 := -origCx, origCy
		x4, y4 := origCx, origCy

		rx1, ry1 := x1*cosA-y1*sinA, x1*sinA+y1*cosA
		rx2, ry2 := x2*cosA-y2*sinA, x2*sinA+y2*cosA
		rx3, ry3 := x3*cosA-y3*sinA, x3*sinA+y3*cosA
		rx4, ry4 := x4*cosA-y4*sinA, x4*sinA+y4*cosA

		minX := math.Min(math.Min(rx1, rx2), math.Min(rx3, rx4))
		maxX := math.Max(math.Max(rx1, rx2), math.Max(rx3, rx4))
		minY := math.Min(math.Min(ry1, ry2), math.Min(ry3, ry4))
		maxY := math.Max(math.Max(ry1, ry2), math.Max(ry3, ry4))

		// set new canvas size with minimum 1px
		newW := max(int(math.Ceil(maxX-minX)), 1)
		newH := max(int(math.Ceil(maxY-minY)), 1)
		rgba := image.NewRGBA(image.Rect(0, 0, newW, newH))
		newCx := float64(newW) / 2.0
		newCy := float64(newH) / 2.0

		// set new pixels with bilinear interpolation
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				// calculate new pixel positions
				nx := float64(x) - newCx
				ny := float64(y) - newCy
				ox := nx*cosA + ny*sinA
				oy := -nx*sinA + ny*cosA
				srcX := ox + origCx
				srcY := oy + origCy

				if srcX >= 0 && srcX < float64(w) && srcY >= 0 && srcY < float64(h) {
					// get pixel positions
					ix0 := int(math.Floor(srcX))
					iy0 := int(math.Floor(srcY))
					ix1 := ix0 + 1
					iy1 := iy0 + 1
					if ix1 >= w {
						ix1 = w - 1
					}
					if iy1 >= h {
						iy1 = h - 1
					}

					// get values of close 4 pixels
					dx := srcX - float64(ix0)
					dy := srcY - float64(iy0)
					r00, g00, b00, a00 := p.getRGBA(srcImg.At(ix0, iy0))
					r10, g10, b10, a10 := p.getRGBA(srcImg.At(ix1, iy0))
					r01, g01, b01, a01 := p.getRGBA(srcImg.At(ix0, iy1))
					r11, g11, b11, a11 := p.getRGBA(srcImg.At(ix1, iy1))

					// bilinear interpolation
					r := r00*(1-dx)*(1-dy) + r10*dx*(1-dy) + r01*(1-dx)*dy + r11*dx*dy
					g := g00*(1-dx)*(1-dy) + g10*dx*(1-dy) + g01*(1-dx)*dy + g11*dx*dy
					b := b00*(1-dx)*(1-dy) + b10*dx*(1-dy) + b01*(1-dx)*dy + b11*dx*dy
					a := a00*(1-dx)*(1-dy) + a10*dx*(1-dy) + a01*(1-dx)*dy + a11*dx*dy
					rgba.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
				}
			}
		}
		dstImg = rgba

	case "Emphasize Red", "Emphasize Green", "Emphasize Blue": // change color value
		factor := 1.0
		if intensity != "" {
			if val, err := strconv.ParseFloat(intensity, 64); err == nil {
				factor = val
			}
		}

		// loop for all pixels
		rgba := image.NewRGBA(bounds)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {

				// get rgba value
				r, g, b, a := p.getRGBA(srcImg.At(x, y))
				origSum := r + g + b
				var newR, newG, newB float64

				switch p.effectSel {
				case "Emphasize Red":
					// calculate new red value
					newR = max(min(r*factor, 255), 0)
					remSpace := origSum - newR
					if remSpace < 0 {
						remSpace = 0
					}

					// calculate new green and blue values
					origOther := g + b
					if origOther > 0 {
						newG = g * (remSpace / origOther)
						newB = b * (remSpace / origOther)
					} else {
						newG = remSpace / 2
						newB = remSpace / 2
					}

				case "Emphasize Green":
					// calculate new green value
					newG = max(min(g*factor, 255), 0)
					remSpace := origSum - newG
					if remSpace < 0 {
						remSpace = 0
					}

					// calculate new red and blue values
					origOther := r + b
					if origOther > 0 {
						newR = r * (remSpace / origOther)
						newB = b * (remSpace / origOther)
					} else {
						newR = remSpace / 2
						newB = remSpace / 2
					}

				case "Emphasize Blue":
					// calculate new blue value
					newB = max(min(b*factor, 255), 0)
					remSpace := origSum - newB
					if remSpace < 0 {
						remSpace = 0
					}

					// calculate new red and green values
					origOther := r + g
					if origOther > 0 {
						newR = r * (remSpace / origOther)
						newG = g * (remSpace / origOther)
					} else {
						newR = remSpace / 2
						newG = remSpace / 2
					}
				}

				// trim new color value
				newR = max(min(newR, 255), 0)
				newG = max(min(newG, 255), 0)
				newB = max(min(newB, 255), 0)
				rgba.Set(x, y, color.RGBA{uint8(newR), uint8(newG), uint8(newB), uint8(a)})
			}
		}
		dstImg = rgba

	case "Flip (X)": // flip horizontally
		rgba := image.NewRGBA(bounds)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rgba.Set(w-1-x, y, srcImg.At(x, y))
			}
		}
		dstImg = rgba

	case "Flip (Y)": // flip vertically
		rgba := image.NewRGBA(bounds)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				rgba.Set(x, h-1-y, srcImg.At(x, y))
			}
		}
		dstImg = rgba

	case "Gray Scale": // convert to gray scale
		rgba := image.NewRGBA(bounds)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, a := p.getRGBA(srcImg.At(x, y))
				gray := 0.299*r + 0.587*g + 0.114*b
				rgba.Set(x, y, color.RGBA{uint8(gray), uint8(gray), uint8(gray), uint8(a)})
			}
		}
		dstImg = rgba

	case "Invert Color": // invert color
		rgba := image.NewRGBA(bounds)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, a := p.getRGBA(srcImg.At(x, y))
				rgba.Set(x, y, color.RGBA{255 - uint8(r), 255 - uint8(g), 255 - uint8(b), uint8(a)})
			}
		}
		dstImg = rgba

	case "Add Image Left", "Add Image Right", "Add Image Top", "Add Image Bottom": // paste new image
		path, err := BaseUI.ZenityFile("Select Image")
		if err != nil || path == "" {
			return
		}
		f, err := os.Open(path)
		if err != nil {
			dialog.ShowError(err, p.Window)
			return
		}
		addImg, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			dialog.ShowError(err, p.Window)
			return
		}

		// get add image bounds
		addB := addImg.Bounds()
		aw, ah := addB.Dx(), addB.Dy()
		var canvasW, canvasH int
		var srcX, srcY, addX, addY int

		// set canvas size and paste position
		switch p.effectSel {
		case "Add Image Left":
			canvasW, canvasH = w+aw, max(h, ah)
			srcX, addX = aw, 0
		case "Add Image Right":
			canvasW, canvasH = w+aw, max(h, ah)
			srcX, addX = 0, w
		case "Add Image Top":
			canvasW, canvasH = max(w, aw), h+ah
			srcY, addY = ah, 0
		case "Add Image Bottom":
			canvasW, canvasH = max(w, aw), h+ah
			srcY, addY = 0, h
		}

		// set new canvas and paste images
		rgba := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
		draw.Draw(rgba, image.Rect(srcX, srcY, srcX+w, srcY+h), srcImg, bounds.Min, draw.Src)
		draw.Draw(rgba, image.Rect(addX, addY, addX+aw, addY+ah), addImg, addB.Min, draw.Src)
		dstImg = rgba
	}

	// update canvas and labels
	if dstImg != nil {
		p.imgCanvas.Image = dstImg
		p.ImgWidth = dstImg.Bounds().Dx()
		p.ImgHeight = dstImg.Bounds().Dy()
		p.sizeLabel.SetText(fmt.Sprintf("(%d x %d)", p.ImgWidth, p.ImgHeight))
		p.imgCanvas.Refresh()
	}
}

func (p *Page) saveImage(xsize string, ysize string, name string) {
	if p.imgCanvas.Image == nil || p.FilePath == "" {
		return
	}
	img := p.imgCanvas.Image
	outW, _ := strconv.Atoi(xsize)
	outH, _ := strconv.Atoi(ysize)

	// resize when both values are valid
	if outW > 0 && outH > 0 && (outW != p.ImgWidth || outH != p.ImgHeight) {
		resized := image.NewRGBA(image.Rect(0, 0, outW, outH))
		scaleX := float64(p.ImgWidth) / float64(outW)
		scaleY := float64(p.ImgHeight) / float64(outH)

		for y := 0; y < outH; y++ {
			for x := 0; x < outW; x++ {
				// get original cords
				srcX := float64(x) * scaleX
				srcY := float64(y) * scaleY

				// get close pixels
				x0 := int(srcX)
				y0 := int(srcY)
				x1 := x0 + 1
				y1 := y0 + 1

				// clamp to bound
				if x1 >= p.ImgWidth {
					x1 = p.ImgWidth - 1
				}
				if y1 >= p.ImgHeight {
					y1 = p.ImgHeight - 1
				}

				// calculate weight
				dx := srcX - float64(x0)
				dy := srcY - float64(y0)

				// get 4 close pixels
				r00, g00, b00, a00 := p.getRGBA(img.At(x0, y0))
				r10, g10, b10, a10 := p.getRGBA(img.At(x1, y0))
				r01, g01, b01, a01 := p.getRGBA(img.At(x0, y1))
				r11, g11, b11, a11 := p.getRGBA(img.At(x1, y1))

				// calculate weight
				r := r00*(1-dx)*(1-dy) + r10*dx*(1-dy) + r01*(1-dx)*dy + r11*dx*dy
				g := g00*(1-dx)*(1-dy) + g10*dx*(1-dy) + g01*(1-dx)*dy + g11*dx*dy
				b := b00*(1-dx)*(1-dy) + b10*dx*(1-dy) + b01*(1-dx)*dy + b11*dx*dy
				a := a00*(1-dx)*(1-dy) + a10*dx*(1-dy) + a01*(1-dx)*dy + a11*dx*dy
				resized.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)})
			}
		}
		img = resized
	}

	// output name
	outName := strings.TrimSpace(name)
	if outName == "" {
		outName = strings.TrimSuffix(filepath.Base(p.FilePath), filepath.Ext(p.FilePath)) + "_conv"
	}
	ext := "." + strings.ToLower(strings.Split(p.fmtSel, "(")[0])
	dstPath := filepath.Join(filepath.Dir(p.FilePath), outName+ext)

	// make output file
	outFile, err := os.Create(dstPath)
	if err != nil {
		dialog.ShowError(err, p.Window)
		return
	}
	defer outFile.Close()

	// encode to each formats
	switch p.fmtSel {
	case "JPG":
		err = jpeg.Encode(outFile, img, &jpeg.Options{Quality: 90})
	case "PNG":
		err = png.Encode(outFile, img)
	case "WEBP":
		err = webp.Encode(outFile, img, &webp.Options{Lossless: false, Quality: 80})
	case "WEBP(LL)":
		err = webp.Encode(outFile, img, &webp.Options{Lossless: true, Quality: 100})
	case "ICO":
		err = p.encodeICO(outFile, img)
	}
	if err != nil {
		dialog.ShowError(err, p.Window)
	} else {
		dialog.ShowInformation("Success", fmt.Sprintf("Saved to %s", filepath.Base(dstPath)), p.Window)
	}
}

func (p *Page) encodeICO(w *os.File, img image.Image) error {
	// encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	pngBytes := buf.Bytes()

	// ICO supports up to 256
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > 256 {
		width = 0
	}
	if height > 256 {
		height = 0
	}

	// 1. ICO Header (6 bytes)
	binary.Write(w, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(w, binary.LittleEndian, uint16(1)) // Type (1 = Icon)
	binary.Write(w, binary.LittleEndian, uint16(1)) // Image Count

	// 2. Icon Directory Entry (16 bytes)
	w.Write([]byte{byte(width), byte(height), 0, 0})            // W, H, ColorCount, Reserved
	binary.Write(w, binary.LittleEndian, uint16(1))             // Color Planes
	binary.Write(w, binary.LittleEndian, uint16(32))            // Bits per pixel
	binary.Write(w, binary.LittleEndian, uint32(len(pngBytes))) // Image Size
	binary.Write(w, binary.LittleEndian, uint32(22))            // Image Offset (6 + 16)

	// 3. Image Data (PNG Payload)
	_, err := w.Write(pngBytes)
	return err
}

func (p *Page) getRGBA(c color.Color) (float64, float64, float64, float64) {
	r, g, b, a := color.RGBAModel.Convert(c).RGBA()
	return float64(r >> 8), float64(g >> 8), float64(b >> 8), float64(a >> 8)
}

var MainApp fyne.App

func main() {
	defer func() {
		if r := recover(); r != nil {
			os.WriteFile("panic-log.txt", fmt.Appendf(nil, "panic while imgconv.main: %v", r), 0644)
		}
	}()

	// init fynesize
	exePath, _ := os.Executable()
	realPath, _ := filepath.EvalSymlinks(exePath)
	cfgPath := filepath.Join(filepath.Dir(realPath), "imgconv_size.txt")
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
