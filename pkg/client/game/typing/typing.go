package typing

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/rywk/minigoao/pkg/client/game/text"
)

// repeatingKeyPressed return true when key is pressed considering the repeat state.
func repeatingKeyPressed(key ebiten.Key) bool {
	const (
		delay    = 30
		interval = 3
	)
	d := inpututil.KeyPressDuration(key)
	if d == 1 {
		return true
	}
	if d >= delay && (d-delay)%interval == 0 {
		return true
	}
	return false
}

type Typer struct {
	runes   []rune
	Text    string
	counter int
}

func NewTyper(s ...string) *Typer {
	return &Typer{runes: make([]rune, 0), Text: strings.Join(s, "")}
}

func (g *Typer) Update() error {
	// Add runes that are input by the user by AppendInputChars.
	// Note that AppendInputChars result changes every frame, so you need to call this
	// every frame.
	g.runes = ebiten.AppendInputChars(g.runes[:0])
	g.Text += string(g.runes)

	// Adjust the string to be at most 10 lines.
	ss := strings.Split(g.Text, "\n")
	if len(ss) > 10 {
		g.Text = strings.Join(ss[len(ss)-10:], "\n")
	}

	// If the enter key is pressed, add a line break.
	if repeatingKeyPressed(ebiten.KeyEnter) || repeatingKeyPressed(ebiten.KeyNumpadEnter) {
		g.Text += "\n"
	}

	// If the backspace key is pressed, remove one character.
	if repeatingKeyPressed(ebiten.KeyBackspace) {
		if len(g.Text) >= 1 {
			g.Text = g.Text[:len(g.Text)-1]
		}
	}

	g.counter++
	return nil
}

func (g *Typer) String() string {
	return g.Text
}

func (g *Typer) Draw(screen *ebiten.Image, x, y int) {
	// Blink the cursor.
	t := g.Text
	if g.counter%60 < 30 {
		t += "_"
	}
	text.PrintBigAt(screen, t, x, y)
}
