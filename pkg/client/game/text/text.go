package text

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
)

var (
	// debugPrintTextImage     *ebiten.Image
	// debugPrintTextSubImages = map[rune]*ebiten.Image{}

	bigDebugPrintTextImage     *ebiten.Image
	bigDebugPrintTextSubImages = map[rune]*ebiten.Image{}
)

func init() {
	img, _, err := image.Decode(bytes.NewReader(img.Text_png))
	if err != nil {
		panic(err)
	}
	bigDebugPrintTextImage = ebiten.NewImageFromImage(img)
}

func PrintBigAt(image *ebiten.Image, str string, x, y int) {
	drawDebugText(image, str, x, y)
}

func PrintAt(image *ebiten.Image, str string, x, y int) {
	ebitenutil.DebugPrintAt(image, str, x, y)
}

func drawDebugText(rt *ebiten.Image, str string, ox, oy int) {
	op := &ebiten.DrawImageOptions{}
	x := 0
	y := 0
	w := bigDebugPrintTextImage.Bounds().Dx()
	for _, c := range str {
		const (
			cw = 12
			ch = 32
		)
		if c == '\n' {
			x = 0
			y += ch
			continue
		}
		s, ok := bigDebugPrintTextSubImages[c]
		if !ok {
			n := w / cw
			sx := (int(c) % n) * cw
			sy := (int(c) / n) * ch
			s = bigDebugPrintTextImage.SubImage(image.Rect(sx, sy, sx+cw, sy+ch)).(*ebiten.Image)
			bigDebugPrintTextSubImages[c] = s
		}
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		op.GeoM.Translate(float64(ox+1), float64(oy))
		rt.DrawImage(s, op)
		x += cw
	}
}
