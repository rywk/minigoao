package text

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
)

var (
	// debugPrintTextImage     *ebiten.Image
	// debugPrintTextSubImages = map[rune]*ebiten.Image{}

	bigDebugPrintTextImage     *ebiten.Image
	bigDebugPrintTextSubImages = map[rune]*ebiten.Image{}

	smallDebugPrintTextImage     *ebiten.Image
	smallDebugPrintTextSubImages = map[rune]*ebiten.Image{}

	bigBgImage   *ebiten.Image
	smallBgImage *ebiten.Image
)

const (
	bcw = 12
	bch = 32

	scw = 6
	sch = 16
)

func init() {
	bigImg, _, err := image.Decode(bytes.NewReader(img.Text_png))
	if err != nil {
		panic(err)
	}
	bigDebugPrintTextImage = ebiten.NewImageFromImage(bigImg)

	smallImg, _, err := image.Decode(bytes.NewReader(img.TextSmall_png))
	if err != nil {
		panic(err)
	}
	smallDebugPrintTextImage = ebiten.NewImageFromImage(smallImg)

	bigBgImage = ebiten.NewImage(bcw, bch+4)
	bigBgImage.Fill(color.Black)

	smallBgImage = ebiten.NewImage(scw, sch+2)
	smallBgImage.Fill(color.Black)
}

func PrintColAt(image *ebiten.Image, str string, x, y int, col color.Color) {
	drawDebugTextOpt(image, str, x, y, options{
		textimg: smallDebugPrintTextImage,
		bgimg:   smallBgImage,
		runemap: smallDebugPrintTextSubImages,
		cw:      scw, ch: sch,
		col: col,
	})
}

func PrintAt(image *ebiten.Image, str string, x, y int) {
	drawDebugTextOpt(image, str, x, y, options{
		textimg: smallDebugPrintTextImage,
		bgimg:   smallBgImage,
		runemap: smallDebugPrintTextSubImages,
		cw:      scw, ch: sch,
	})
}

func PrintAtBg(image *ebiten.Image, str string, x, y int) {
	drawDebugTextOpt(image, str, x, y, options{
		textimg: smallDebugPrintTextImage,
		bgimg:   smallBgImage,
		runemap: smallDebugPrintTextSubImages,
		cw:      scw, ch: sch,
		bg: true,
	})
}

func PrintBigAt(image *ebiten.Image, str string, x, y int) (int, int) {
	return drawDebugTextOpt(image, str, x, y, options{
		textimg: bigDebugPrintTextImage,
		bgimg:   bigBgImage,
		runemap: bigDebugPrintTextSubImages,
		cw:      bcw, ch: bch,
	})
}

func PrintBigAtBg(image *ebiten.Image, str string, x, y int) (int, int) {
	return drawDebugTextOpt(image, str, x, y, options{
		textimg: bigDebugPrintTextImage,
		bgimg:   bigBgImage,
		runemap: bigDebugPrintTextSubImages,
		cw:      bcw, ch: bch,
		bg: true,
	})
}

type options struct {
	textimg *ebiten.Image
	bgimg   *ebiten.Image
	runemap map[rune]*ebiten.Image
	cw, ch  int
	bg      bool
	col     color.Color
}

func drawDebugTextOpt(rt *ebiten.Image, str string, ox, oy int, opt options) (int, int) {
	op := &ebiten.DrawImageOptions{}
	bgOp := &ebiten.DrawImageOptions{}
	x := 0
	y := 0
	w := opt.textimg.Bounds().Dx()
	if opt.bg {
		bgOp.GeoM.Reset()
		bgOp.GeoM.Translate(float64(x), float64(y))
		bgOp.GeoM.Translate(float64(ox-opt.cw), float64(oy))
		rt.DrawImage(opt.bgimg, bgOp)
	}
	for _, c := range str {
		if c == '\n' {
			x = 0
			y += opt.ch
			continue
		}
		s, ok := opt.runemap[c]
		if !ok {
			n := w / opt.cw
			sx := (int(c) % n) * opt.cw
			sy := (int(c) / n) * opt.ch
			s = opt.textimg.SubImage(image.Rect(sx, sy, sx+opt.cw, sy+opt.ch)).(*ebiten.Image)
			opt.runemap[c] = s
		}

		if opt.bg {
			bgOp.GeoM.Reset()
			bgOp.GeoM.Translate(float64(x), float64(y))
			bgOp.GeoM.Translate(float64(ox), float64(oy))
			rt.DrawImage(opt.bgimg, bgOp)
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		op.GeoM.Translate(float64(ox), float64(oy))
		if opt.col != nil {
			op.ColorScale.Reset()
			op.ColorScale.ScaleWithColor(opt.col)
		}
		rt.DrawImage(s, op)
		x += opt.cw
	}
	if opt.bg {
		bgOp.GeoM.Reset()
		bgOp.GeoM.Translate(float64(x), float64(y))
		bgOp.GeoM.Translate(float64(ox), float64(oy))
		rt.DrawImage(opt.bgimg, bgOp)
	}
	return x, y
}
