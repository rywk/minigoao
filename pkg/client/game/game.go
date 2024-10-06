package game

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/player"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/client/game/typing"
	"github.com/rywk/minigoao/pkg/conc"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
	"golang.org/x/image/math/f64"
)

type Mode int

const (
	ModeLogin Mode = iota
	ModeAccount
	ModeGame
	ModeOptions
)

type YSortable interface {
	ValueY() float64
	Draw(*ebiten.Image)
}

type Login struct {
	data *msgs.EventPlayerLogin
	err  error
}

type Register struct {
	data *msgs.EventAccountLogin
	err  error
}

type Game struct {
	secureConn   bool
	debug        bool
	serverAddr   string
	start        time.Time
	memStats     *runtime.MemStats
	totalRunTime time.Duration

	// reduce mem ftp
	worldImgOp *ebiten.DrawImageOptions

	web  bool
	mode Mode

	escapePressed bool
	tabPressed    bool
	clickPressed  bool

	// register stuff
	connecting      bool
	gameLogin       chan Login
	accountResponse chan Register

	btnLogin    *Button
	btnRegister *Button
	btnEnter    *Button

	register      bool // true = create account, false = login (default)
	accountTyper  *typing.Typer
	passwordTyper *typing.Typer
	emailTyper    *typing.Typer
	typingInput   int
	account       *msgs.EventAccountLogin

	btnsCharacters   []*Button
	btnNewCharacters *Button
	creatingChar     bool
	nickTyper        *typing.Typer

	fsBtn               *Checkbox
	vsyncBtn            *Checkbox
	inputBox            *ebiten.Image
	fullscreen          bool
	vsync               bool
	connErrorColorStart int
	errorMsg            string
	loadingX            int
	loadingBar          *ebiten.Image
	// game
	mouseX, mouseY int
	latency        string
	onlines        string
	counter        int
	ms             msgs.Msgs
	world          *Map
	sessionID      uint32
	players        [constants.MaxConnCount]*player.P
	playersY       []YSortable
	player         *player.P
	outQueue       chan *GameMsg
	eventQueue     []*GameMsg
	eventLock      sync.Mutex

	client *player.ClientP
	stats  *Hud

	keys *Keys

	rankingList []msgs.RankChar

	SelectedSpell attack.Spell

	lastMove          time.Time
	leftForMove       float64 // pixels left to complete tile change
	lastDir           direction.D
	soundPrevWalk     int
	startForStep      time.Time
	lastMoveConfirmed bool

	LastAction time.Time

	lastPotionUsed item.Item

	steps []player.Step

	SoundBoard audio2d.AudioMixer

	ViewPort   f64.Vec2
	ZoomFactor float64
	Rotation   int

	LastPing    time.Time
	WaitingPong bool
}

type GameMsg struct {
	E    msgs.E
	Data interface{}
}

const ScreenWidth, ScreenHeight = 1280, 720

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

func NewGame(web bool, serverAddr string) *Game {
	start := time.Now()

	g := &Game{
		serverAddr:      serverAddr,
		secureConn:      false,
		debug:           false,
		start:           start,
		memStats:        &runtime.MemStats{},
		vsync:           true,
		mode:            ModeLogin,
		web:             web,
		gameLogin:       make(chan Login),
		accountResponse: make(chan Register),
		accountTyper:    typing.NewTyper(),
		emailTyper:      typing.NewTyper(),
		passwordTyper:   typing.NewTyper(),
		worldImgOp:      &ebiten.DrawImageOptions{},
		inputBox:        texture.Decode(img.InputBox_png),
		loadingBar:      ebiten.NewImage(400, 10),
	}
	if strings.Contains(serverAddr, ":443") {
		g.secureConn = true
	}
	g.loadingBar.Fill(color.White)
	g.fsBtn = NewCheckbox(g)
	g.vsyncBtn = NewCheckbox(g)
	g.vsyncBtn.On = false
	g.SoundBoard = audio2d.NewSoundBoard(web)

	btnImgLogin := ebiten.NewImage(120, 38)
	btnImgLogin.Fill(color.RGBA{120, 21, 88, 200})
	text.PrintBigAt(btnImgLogin, "Login", 26, 2)

	btnImgRegister := ebiten.NewImage(120, 38)
	btnImgRegister.Fill(color.RGBA{120, 21, 88, 200})
	text.PrintBigAt(btnImgRegister, "Register", 14, 2)

	btnImgEnter := ebiten.NewImage(120, 38)
	btnImgEnter.Fill(color.RGBA{96, 21, 188, 200})
	text.PrintBigAt(btnImgEnter, "Enter", 24, 2)

	g.btnLogin = NewButton(g, nil, btnImgLogin, typ.P{X: HalfScreenX - 138, Y: 120})
	g.btnRegister = NewButton(g, nil, btnImgRegister, typ.P{X: HalfScreenX + 24, Y: 120})
	g.btnEnter = NewButton(g, nil, btnImgEnter, typ.P{X: HalfScreenX - 70, Y: 470})

	return g
}

func (g *Game) Update() error {
	switch g.mode {
	case ModeLogin:
		g.updateLogin()
	case ModeAccount:
		g.updateAccount()
	case ModeGame:
		g.updateGame()
	case ModeOptions:
	}
	if !g.debug {
		return nil
	}
	if g.counter%30 == 0 {
		runtime.ReadMemStats(g.memStats)
		g.totalRunTime = time.Since(g.start)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.mode {
	case ModeLogin:
		g.drawRegister(screen)
	case ModeAccount:
		g.drawAccount(screen)
	case ModeGame:
		g.drawGame(screen)
	case ModeOptions:
	}
	if !g.debug {
		return
	}
	ms := g.memStats
	msg := fmt.Sprintf(`TPS: %0.2f (max: %d);
Run time: %v
ticks: %d
Alloc: %s
Total: %s
Sys: %s
NextGC: %s
NumGC: %d`,
		ebiten.ActualTPS(), ebiten.TPS(),
		g.totalRunTime,
		g.counter,
		formatBytes(ms.Alloc), formatBytes(ms.TotalAlloc), formatBytes(ms.Sys),
		formatBytes(ms.NextGC), ms.NumGC,
	)
	ebitenutil.DebugPrintAt(screen, msg, 20, 64)
}

func formatBytes(b uint64) string {
	if b >= 1073741824 {
		return fmt.Sprintf("%0.2f GiB", float64(b)/1073741824)
	} else if b >= 1048576 {
		return fmt.Sprintf("%0.2f MiB", float64(b)/1048576)
	} else if b >= 1024 {
		return fmt.Sprintf("%0.2f KiB", float64(b)/1024)
	} else {
		return fmt.Sprintf("%d B", b)
	}
}
func (g *Game) updateLogin() {
	if g.connErrorColorStart > 0 {
		g.connErrorColorStart--
	}
	g.fullscreen = g.fsBtn.On
	g.fsBtn.Update()
	// g.vsync = g.vsyncBtn.On
	// g.vsyncBtn.Update()

	if g.btnLogin.Pressed() {
		g.register = false
	}

	if g.btnRegister.Pressed() {
		g.register = true
	}

	if g.register {
		g.btnRegister.Over = true
	} else {
		g.btnLogin.Over = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyTab) {
		if !g.tabPressed {
			g.typingInput++
			if !g.register {
				if g.typingInput > 1 {
					g.typingInput = 0
				}
			} else {
				if g.typingInput > 2 {
					g.typingInput = 0
				}
			}
		}
		g.tabPressed = true
	} else {
		g.tabPressed = false
	}

	switch g.typingInput {
	case 0:
		g.accountTyper.Update()
		g.emailTyper.Counter = 31
		g.passwordTyper.Counter = 31
	case 1:
		g.passwordTyper.Update()
		g.emailTyper.Counter = 31
		g.accountTyper.Counter = 31
	case 2:
		g.emailTyper.Update()
		g.accountTyper.Counter = 31
		g.passwordTyper.Counter = 31
	}

	if g.connecting {
		login, ok := conc.Check(g.accountResponse)
		if !ok {
			return
		}
		g.connecting = false
		if login.err != nil {
			log.Println(login.err)
			g.errorMsg = login.err.Error()
			g.connErrorColorStart = 255
			return
		}
		g.account = login.data
		x := 200
		w := 170
		i := 0
		g.btnsCharacters = []*Button{}
		for _, char := range g.account.Characters {
			charBtn := ebiten.NewImage(160, 178)
			charBtn.Fill(color.RGBA{120, 21, 88, 200})
			text.PrintBigAt(charBtn, char.Nick, 28, 12)
			g.btnsCharacters = append(g.btnsCharacters, NewButton(g, nil, charBtn, typ.P{X: int32(x + w*i), Y: 190}))
			i++
		}
		btnNewCharacter := ebiten.NewImage(48, 48)
		btnNewCharacter.Fill(color.RGBA{77, 6, 58, 200})
		g.btnNewCharacters = NewButton(g, texture.Decode(img.IconPlusBig_png), btnNewCharacter, typ.P{X: int32(x + w*i), Y: 190})
		g.nickTyper = typing.NewTyper()
		g.mode = ModeAccount
		return
	}
	entrePressed := g.btnEnter.Pressed()
	r := strings.NewReplacer("\n", "", " ", "")
	accountText := g.accountTyper.String()
	addressText := g.passwordTyper.String()
	g.accountTyper.Text, g.passwordTyper.Text = r.Replace(accountText), r.Replace(addressText)
	if g.register {
		emailText := g.emailTyper.String()
		g.emailTyper.Text = r.Replace(emailText)
		if !strings.HasSuffix(accountText, "\n") &&
			!strings.HasSuffix(addressText, "\n") &&
			!strings.HasSuffix(emailText, "\n") &&
			!entrePressed {
			return
		}
	} else {
		if !strings.HasSuffix(accountText, "\n") &&
			!strings.HasSuffix(addressText, "\n") &&
			!entrePressed {
			return
		}
	}

	if g.accountTyper.Text != "" && g.passwordTyper.Text != "" {
		g.connecting = true
		if g.register {
			if g.emailTyper.Text == "" {
				return
			}
			go g.CreateAccount(g.accountTyper.Text, g.passwordTyper.Text, g.emailTyper.Text)
		} else {
			go g.LoginAccount(g.accountTyper.Text, g.passwordTyper.Text)
		}
	}
}
func (g *Game) updateAccount() {
	if g.connErrorColorStart > 0 {
		g.connErrorColorStart--
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		if !g.escapePressed {
			g.mode = ModeLogin
			return
		}
		g.escapePressed = true
	} else {
		g.escapePressed = false
	}
	// in case we created a character
	if g.connecting {
		gameLogin, ok := conc.Check(g.gameLogin)
		if ok {
			g.connecting = false
			if gameLogin.err != nil {
				log.Println(gameLogin.err)
				g.errorMsg = gameLogin.err.Error()
				g.connErrorColorStart = 255
				return
			}
			g.StartGame(gameLogin.data)
			return
		}
		login, ok := conc.Check(g.accountResponse)
		if !ok {
			return
		}
		g.connecting = false
		if login.err != nil {
			log.Println(login.err)
			g.errorMsg = login.err.Error()
			g.connErrorColorStart = 255
			return
		}
		g.account = login.data
		x := 200
		w := 170
		i := 0
		g.btnsCharacters = []*Button{}
		for _, char := range g.account.Characters {
			charBtn := ebiten.NewImage(160, 178)
			charBtn.Fill(color.RGBA{120, 21, 88, 200})
			text.PrintBigAt(charBtn, char.Nick, 28, 12)
			g.btnsCharacters = append(g.btnsCharacters, NewButton(g, nil, charBtn, typ.P{X: int32(x + w*i), Y: 190}))
			i++
		}
		btnNewCharacter := ebiten.NewImage(48, 48)
		btnNewCharacter.Fill(color.RGBA{77, 6, 58, 255})
		g.btnNewCharacters = NewButton(g, texture.Decode(img.IconPlusBig_png), btnNewCharacter, typ.P{X: int32(x + w*i), Y: 190})
		g.nickTyper = typing.NewTyper()
		return
	}

	if g.creatingChar {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.nickTyper.Text = ""
			g.creatingChar = false
		}
		g.nickTyper.Update()
		r := strings.NewReplacer("\n", "", " ", "")
		nickText := g.nickTyper.String()
		g.nickTyper.Text = r.Replace(nickText)
		if !strings.HasSuffix(nickText, "\n") {
			return
		}
		g.creatingChar = false
		if g.nickTyper.Text != "" {
			g.connecting = true
			go g.CreateCharacter(0, g.nickTyper.Text)
		}
		return
	}

	for i, btn := range g.btnsCharacters {
		if btn.Pressed() {
			g.connecting = true
			go g.LoginCharacter(uint16(g.account.Characters[i].ID))
		}
	}
	if g.btnNewCharacters.Pressed() {
		g.creatingChar = true
	}

}
func (g *Game) drawAccount(screen *ebiten.Image) {
	g.mouseX, g.mouseY = ebiten.CursorPosition()

	//text.PrintBigAt(screen, fmt.Sprintf("%d", g.account.ID), 100, 100)
	text.PrintBigAt(screen, g.account.Account, 100, 100)
	text.PrintBigAt(screen, g.account.Email, 370, 100)
	for _, btn := range g.btnsCharacters {
		btn.Draw(screen, int(btn.Pos.X), int(btn.Pos.Y))
	}
	g.btnNewCharacters.Draw(screen, int(g.btnNewCharacters.Pos.X), int(g.btnNewCharacters.Pos.Y))
	if g.creatingChar {
		g.nickTyper.DrawBg(screen, int(g.btnNewCharacters.Pos.X+70), int(g.btnNewCharacters.Pos.Y+4), color.RGBA{60, 12, 57, 200})
	}
	if g.connErrorColorStart > 0 {
		text.PrintBigAtCol(screen, g.errorMsg, 80, ScreenHeight-80, color.RGBA{178, 0, 16, uint8(g.connErrorColorStart)})
	}
	if g.connecting {
		rect := image.Rect(int(float64(g.loadingX)*0.4), 0, g.loadingX+400-int(float64(g.loadingX)*0.4), 10)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(g.loadingX), 660)
		screen.DrawImage(g.loadingBar.SubImage(rect).(*ebiten.Image), op)
		g.loadingX += 3
		if g.loadingX > ScreenWidth+400 {
			g.loadingX = -400
		}
	} else {
		g.loadingX = 0
	}
}
func (g *Game) drawRegister(screen *ebiten.Image) {
	g.mouseX, g.mouseY = ebiten.CursorPosition()

	g.btnLogin.Draw(screen, HalfScreenX-138, 120)
	g.btnRegister.Draw(screen, HalfScreenX+24, 120)
	g.btnEnter.Draw(screen, HalfScreenX-70, 470)

	clicked := false
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !g.clickPressed {
			clicked = true
		}
		g.clickPressed = true
	} else {
		g.clickPressed = false
	}

	yoff := 160
	if clicked && image.Pt(g.mouseX, g.mouseY).In(g.inputBox.Bounds().Add(image.Pt(HalfScreenX-150, yoff+28))) {
		g.typingInput = 0
	}
	text.PrintBigAt(screen, "Account", HalfScreenX-144, yoff)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(HalfScreenX-150, float64(yoff+28))
	screen.DrawImage(g.inputBox, op)
	g.accountTyper.Draw(screen, HalfScreenX-130, yoff+40)

	yoff += 100
	if clicked && image.Pt(g.mouseX, g.mouseY).In(g.inputBox.Bounds().Add(image.Pt(HalfScreenX-150, yoff+28))) {
		g.typingInput = 1
	}
	text.PrintBigAt(screen, "Password", HalfScreenX-144, yoff)
	op.GeoM.Translate(0, 100)
	screen.DrawImage(g.inputBox, op)
	//g.passwordTyper.Draw(screen, HalfScreenX-130, yoff+40)
	str := strings.Repeat("#", len(g.passwordTyper.String()))
	if g.passwordTyper.Counter%40 < 20 {
		str += "_"
	}
	text.PrintBigAt(screen, str, HalfScreenX-130, yoff+40)

	if g.register {
		yoff += 100
		if clicked && image.Pt(g.mouseX, g.mouseY).In(g.inputBox.Bounds().Add(image.Pt(HalfScreenX-150, yoff+28))) {
			g.typingInput = 2
		}
		text.PrintBigAt(screen, "Email", HalfScreenX-144, yoff)
		op.GeoM.Translate(0, 100)
		screen.DrawImage(g.inputBox, op)
		g.emailTyper.Draw(screen, HalfScreenX-130, yoff+40)
	}

	text.PrintBigAt(screen, "Fullscreen", HalfScreenX-95, 561)
	g.fsBtn.Draw(screen, HalfScreenX+46, 560)

	if g.connErrorColorStart > 0 {
		text.PrintBigAtCol(screen, g.errorMsg, 80, ScreenHeight-80, color.RGBA{178, 0, 16, uint8(g.connErrorColorStart)})
	}
	if g.connecting {
		rect := image.Rect(int(float64(g.loadingX)*0.4), 0, g.loadingX+400-int(float64(g.loadingX)*0.4), 10)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(g.loadingX), 660)
		screen.DrawImage(g.loadingBar.SubImage(rect).(*ebiten.Image), op)
		g.loadingX += 3
		if g.loadingX > ScreenWidth+400 {
			g.loadingX = -400
		}
	} else {
		g.loadingX = 0
	}
}

func (g *Game) drawGame(screen *ebiten.Image) {

	g.mouseX, g.mouseY = ebiten.CursorPosition()
	g.world.Draw(typ.P{X: g.player.X, Y: g.player.Y})
	for _, p := range g.playersY {
		if p == nil {
			continue
		}
		p.Draw(g.world.Image())
	}
	g.keys.DrawChat(g.world.Image(), int(g.player.Pos[0]+16), int(g.player.Pos[1]-40))
	g.Render(g.world.Image(), screen)
	g.stats.Draw(screen)
	text.PrintAt(screen, fmt.Sprintf("%vFPS\n%v", int(ebiten.ActualFPS()), g.latency), 0, 0)
	text.PrintAt(screen, fmt.Sprintf("Online: %v", g.onlines), 50, 0)
}

func (g *Game) updateGame() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.escapePressed = true
		g.Clear()
		g.accountTyper = typing.NewTyper()
		g.passwordTyper = typing.NewTyper()
		g.emailTyper = typing.NewTyper()
		g.mode = ModeAccount
		ebiten.SetFullscreen(false)
		g.ms.Write(msgs.EPlayerLogout, []byte{0})
		// g.ms.Close()
		// g.ms = nil
		return errors.New("esc exit")
	}
	if err := g.ProcessEventQueue(); err != nil {
		g.Clear()
		g.accountTyper = typing.NewTyper()
		g.passwordTyper = typing.NewTyper()
		g.emailTyper = typing.NewTyper()
		g.mode = ModeLogin
		ebiten.SetFullscreen(false)
		return err
	}
	g.pingServer()
	g.SendChat()
	g.UpdateGamePos()
	g.ListenInputs()
	g.world.Update()
	g.player.Update(g.counter)
	g.player.Effect.Update(g.counter)
	for _, p := range g.players {
		if p == nil {
			continue
		}
		p.WalkSteps(g.world.Space)
		p.Update(g.counter)
		p.Effect.Update(g.counter)
	}
	sort.Slice(g.playersY, func(i, j int) bool {
		return g.playersY[i].ValueY() < g.playersY[j].ValueY()
	})
	g.stats.Update()
	g.counter++
	return nil
}
func (g *Game) CreateAccount(account string, password string, email string) {
	var err error
	if g.ms == nil {
		g.ms, err = msgs.DialServer(g.serverAddr, g.web, g.secureConn)
		if err != nil {
			g.accountResponse <- Register{data: nil, err: err}
			return
		}
	}
	createAcc := &msgs.EventCreateAccount{
		Account:  account,
		Password: password,
		Email:    email,
	}

	err = g.ms.EncodeAndWrite(msgs.ECreateAccount, createAcc)
	if err != nil {
		g.accountResponse <- Register{data: nil, err: err}
		return
	}

	im, err := g.ms.Read()
	if err != nil {
		log.Printf("read error: %v", err)
		g.accountResponse <- Register{data: nil, err: err}
		return
	}

	if im.Event == msgs.EError {
		errstr := msgs.DecodeMsgpack(im.Data, &msgs.EventError{}).Msg
		log.Printf("game err: %v", errstr)
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("%v", errstr)}
		return
	}
	if im.Event != msgs.EAccountLoginOk {
		log.Printf("not login response")
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("not login response")}
		return
	}
	resp := msgs.DecodeMsgpack(im.Data, &msgs.EventAccountLogin{})
	g.accountResponse <- Register{data: resp, err: nil}
}

func (g *Game) LoginAccount(account string, password string) {
	var err error
	if g.ms == nil {
		g.ms, err = msgs.DialServer(g.serverAddr, g.web, g.secureConn)
		if err != nil {
			g.accountResponse <- Register{data: nil, err: err}
			return
		}
	}
	loginAcc := &msgs.EventLoginAccount{
		Account:  account,
		Password: password,
	}
	err = g.ms.EncodeAndWrite(msgs.ELoginAccount, loginAcc)
	if err != nil {
		g.accountResponse <- Register{data: nil, err: err}
		return
	}

	im, err := g.ms.Read()
	if err != nil {
		log.Printf("read err %v", err)
		g.accountResponse <- Register{data: nil, err: err}
		return
	}

	if im.Event == msgs.EError {
		errstr := msgs.DecodeMsgpack(im.Data, &msgs.EventError{}).Msg
		log.Printf("game err: %v", errstr)
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("%v", errstr)}
		return
	}

	if im.Event != msgs.EAccountLoginOk {
		log.Printf("not login response")
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("not login response")}
		return
	}

	resp := msgs.DecodeMsgpack(im.Data, &msgs.EventAccountLogin{})
	g.accountResponse <- Register{data: resp, err: nil}
}

func (g *Game) LoginCharacter(id uint16) {
	var err error
	if g.ms == nil {
		g.ms, err = msgs.DialServer(g.serverAddr, g.web, g.secureConn)
		if err != nil {
			g.gameLogin <- Login{data: nil, err: err}
			return
		}
	}
	loginChar := &msgs.EventLoginCharacter{
		ID: id,
	}
	err = g.ms.EncodeAndWrite(msgs.ELoginCharacter, loginChar)
	if err != nil {
		g.gameLogin <- Login{data: nil, err: err}
		return
	}

	im, err := g.ms.Read()
	if err != nil {
		g.gameLogin <- Login{data: nil, err: err}
		return
	}

	if im.Event == msgs.EError {
		errstr := msgs.DecodeMsgpack(im.Data, &msgs.EventError{}).Msg
		log.Printf("game err: %v", errstr)
		g.gameLogin <- Login{data: nil, err: fmt.Errorf("%v", errstr)}
		return
	}

	if im.Event != msgs.EPlayerLogin {
		log.Printf("not login response")
		g.gameLogin <- Login{data: nil, err: fmt.Errorf("not login response")}
		return
	}

	g.gameLogin <- Login{data: msgs.DecodeMsgpack(im.Data, &msgs.EventPlayerLogin{}), err: nil}
}

func (g *Game) CreateCharacter(accountID int, nick string) {
	var err error
	if g.ms == nil {
		g.ms, err = msgs.DialServer(g.serverAddr, g.web, g.secureConn)
		if err != nil {
			g.accountResponse <- Register{data: nil, err: err}
			return
		}
	}
	createChar := &msgs.EventCreateCharacter{
		AccountID: uint16(accountID),
		Nick:      nick,
	}
	err = g.ms.EncodeAndWrite(msgs.ECreateCharacter, createChar)
	if err != nil {
		g.accountResponse <- Register{data: nil, err: err}
		return
	}
	im, err := g.ms.Read()
	if err != nil {
		g.accountResponse <- Register{data: nil, err: err}
		return
	}
	if im.Event == msgs.EError {
		errstr := msgs.DecodeMsgpack(im.Data, &msgs.EventError{}).Msg
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("%v", errstr)}
		return
	}
	if im.Event != msgs.EAccountLoginOk {
		g.accountResponse <- Register{data: nil, err: fmt.Errorf("not login response")}
		return
	}

	g.accountResponse <- Register{data: msgs.DecodeMsgpack(im.Data, &msgs.EventAccountLogin{}), err: nil}
}

func (g *Game) StartGame(login *msgs.EventPlayerLogin) {
	if g.web {
		g.vsync = true
	}
	ebiten.SetVsyncEnabled(g.vsync)
	ebiten.SetFullscreen(g.fullscreen)
	g.mode = ModeGame
	g.Login(login)
	g.ViewPort = f64.Vec2{ScreenWidth, ScreenHeight}
	//g.ZoomFactor = 1
	g.lastMove = time.Now()
	g.lastMoveConfirmed = true
	var keyConfig *KeyConfig
	if login.KeyConfig.Back.Keyboard != 0 || login.KeyConfig.Back.Mouse != 0 {
		keyConfig = &KeyConfig{}
		keyConfig.FromMsgs(login.KeyConfig)
	}
	g.keys = NewKeys(g, keyConfig)
	g.keys.enterDown = true
	g.playersY = append(g.playersY, g.player)

	g.stats = NewHud(g)

	g.eventQueue = make([]*GameMsg, 0, 100)
	g.outQueue = make(chan *GameMsg, 100)
	g.eventLock = sync.Mutex{}
	go g.WriteEventQueue()
	go g.WriteToServer()

	g.SoundBoard.Play(assets.Spawn)

}

func (g *Game) Login(e *msgs.EventPlayerLogin) {
	g.sessionID = uint32(e.ID)
	g.world = NewMap(MapConfigFromPlayerLogin(e))
	g.player = player.NewLogin(e)
	g.client = g.player.Client
	for _, p := range e.VisiblePlayers {
		g.AddToGame(&p)
	}
}

func (g *Game) WriteToServer() {
	for m := range g.outQueue {
		g.ms.EncodeAndWrite(m.E, m.Data)
	}
}

func (g *Game) WriteEventQueue() {
	for {
		im, err := g.ms.Read()
		if err != nil {
			g.eventLock.Lock()
			g.eventQueue = append(g.eventQueue, &GameMsg{E: msgs.EServerDisconnect})
			g.eventLock.Unlock()
			return
		}
		dim := GameMsg{E: im.Event}
		switch im.Event {
		case msgs.EPingOk:
			dim.Data = binary.BigEndian.Uint16(im.Data[:2])
		case msgs.EMoveOk:
			dim.Data = im.Data
		case msgs.EMeleeOk:
			dim.Data = msgs.DecodeEventMeleeOk(im.Data)
		case msgs.ECastSpellOk:
			dim.Data = msgs.DecodeEventCastSpellOk(im.Data)
		case msgs.EUseItemOk:
			dim.Data = msgs.DecodeEventUseItemOk(im.Data)
		case msgs.EPlayerMoved:
			dim.Data = msgs.DecodeEventPlayerMoved(im.Data)
		case msgs.EPlayerMelee:
			dim.Data = msgs.DecodeEventPlayerMelee(im.Data)
		case msgs.EPlayerSpell:
			dim.Data = msgs.DecodeEventPlayerSpell(im.Data)
		case msgs.EPlayerMeleeRecieved:
			dim.Data = msgs.DecodeEventPlayerMeleeRecieved(im.Data)
		case msgs.EPlayerSpellRecieved:
			dim.Data = msgs.DecodeEventPlayerSpellRecieved(im.Data)
		case msgs.EPlayerChangedSkin:
			dim.Data = msgs.DecodeEventPlayerChangedSkin(im.Data)
		case msgs.EUpdateSkillsOk:
			dim.Data = msgs.DecodeMsgpack(im.Data, &msgs.Experience{})
		case msgs.EPlayerSpawned:
			msg := &msgs.EventPlayerSpawned{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.EPlayerLeaveViewport:
			dim.Data = binary.BigEndian.Uint16(im.Data)
		case msgs.EPlayerEnterViewport:
			msg := &msgs.EventPlayerSpawned{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.EPlayerDespawned:
			dim.Data = binary.BigEndian.Uint16(im.Data)
		case msgs.EBroadcastChat:
			msg := &msgs.EventBroadcastChat{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.EError:
			msg := &msgs.EventError{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.ERankList:
			msg := &msgs.EventRankList{}
			msgs.DecodeMsgpack(im.Data, msg)
			dim.Data = msg
		case msgs.ECharLogoutOk:
			//log.Println("stopped client game read")
			return
		}
		g.eventLock.Lock()
		g.eventQueue = append(g.eventQueue, &dim)
		g.eventLock.Unlock()
	}
}

func (g *Game) Clear() {
	g.sessionID = 0
	g.player = nil
	g.players = [50]*player.P{}
	g.playersY = []YSortable{}
}

func (g *Game) AddToGame(event *msgs.EventNewPlayer) {
	g.world.Space.SetSlot(0, event.Pos, event.ID)
	g.players[event.ID] = player.CreatePlayerSpawned(g.player, event)
	g.players[event.ID].SetSoundboard(g.SoundBoard)
	g.playersY = append(g.playersY, g.players[event.ID])
}

func (g *Game) ProcessEventQueue() error {
	g.eventLock.Lock()
	for _, ev := range g.eventQueue {
		switch ev.E {
		case msgs.ERankList:
			g.rankingList = ev.Data.(*msgs.EventRankList).Characters
		case msgs.EServerDisconnect:
			g.ms.Close()
			g.ms = nil
			log.Printf("Server disconnected\n")
			return errors.New("server disconnected")
		case msgs.EPingOk:
			g.WaitingPong = false
			g.onlines = fmt.Sprintf("%d", ev.Data.(uint16))
			g.latency = fmt.Sprintf("%dms", time.Since(g.LastPing).Milliseconds()/2)
		case msgs.EPlayerDespawned:
			pid := ev.Data.(uint16)
			g.DespawnPlayer(pid)
			log.Printf("Player [%v] despawned\n", pid)
		case msgs.EPlayerSpawned:
			event := ev.Data.(*msgs.EventPlayerSpawned)
			log.Printf("Player [%v] spawned %v\n", event.ID, event.Nick)
			g.AddToGame(event)
		case msgs.EPlayerLeaveViewport:
			pid := ev.Data.(uint16)
			g.DespawnPlayer(pid)
			log.Printf("Player [%v] left viewport\n", pid)
		case msgs.EPlayerEnterViewport:
			event := ev.Data.(*msgs.EventPlayerSpawned)
			log.Printf("Player [%v] entered viewport %v\n", event.ID, event.Nick)
			g.AddToGame(event)
		case msgs.EPlayerMoved:
			event := ev.Data.(*msgs.EventPlayerMoved)
			//log.Printf("Player [%v] moved\n", event.ID)
			g.players[event.ID].AddStep(event)
		case msgs.EMoveOk:
			g.MovementResponse(ev.Data.([]byte))
		case msgs.EMeleeOk:
			event := ev.Data.(*msgs.EventMeleeOk)
			//log.Printf("CastMeleeOk m: %#v\n", event)
			if g.player.Dead {
				break
			}
			if !event.Hit {
				g.SoundBoard.Play(assets.MeleeAir)
				g.player.Direction = event.Dir
				break
			}
			g.player.Direction = event.Dir
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewAttackNumber(int(event.Damage), false)
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.EPlayerMeleeRecieved:
			event := ev.Data.(*msgs.EventPlayerMeleeRecieved)
			g.SoundBoard.Play(assets.MeleeBlood)
			g.player.Effect.NewMeleeHit()
			//log.Printf("RecivedMelee m: %#v\n", event)
			g.players[event.ID].Effect.NewAttackNumber(int(event.Damage), false)
			g.players[event.ID].Direction = event.Dir
			g.player.Client.HP = int(event.NewHP)

			if g.player.Client.HP == 0 {
				g.player.Dead = true
				g.player.Inmobilized = false
			}
		case msgs.EPlayerMelee:
			event := ev.Data.(*msgs.EventPlayerMelee)
			//log.Printf("EPlayerMelee m: %#v\n", event)
			if !event.Hit {
				g.players[event.From].Direction = event.Dir
				g.SoundBoard.PlayFrom(assets.MeleeAir, g.player.X, g.player.Y, g.players[event.From].X, g.players[event.From].Y)
				break
			}
			g.players[event.ID].Direction = event.Dir
			g.SoundBoard.PlayFrom(assets.MeleeBlood, g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
			g.players[event.ID].Effect.NewMeleeHit()
			g.players[event.ID].Dead = event.Killed
		case msgs.ECastSpellOk:
			event := ev.Data.(*msgs.EventCastSpellOk)
			//log.Printf("CastSpellOk m: %#v\n", event)
			g.player.Client.MP = int(event.NewMP)
			if uint32(event.ID) != g.sessionID {
				g.player.Effect.NewAttackNumber(int(event.Damage), event.Spell == attack.SpellHealWounds)
				g.players[event.ID].Effect.NewSpellHit(event.Spell)
				g.SoundBoard.PlayFrom(assets.SoundFromSpell(event.Spell), g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
				g.players[event.ID].Dead = event.Killed
			}
		case msgs.EPlayerSpellRecieved:
			event := ev.Data.(*msgs.EventPlayerSpellRecieved)
			//log.Printf("RecivedSpell m: %#v\n", event)
			switch event.Spell {
			case attack.SpellParalize:
				g.player.Inmobilized = true
			case attack.SpellRemoveParalize:
				g.player.Inmobilized = false
			case attack.SpellResurrect:
				g.player.Dead = false
			}
			caster := g.players[event.ID]
			if event.ID == uint16(g.sessionID) {
				caster = g.player
			}
			g.SoundBoard.Play(assets.SoundFromSpell(event.Spell))
			g.player.Effect.NewSpellHit(event.Spell)
			caster.Effect.NewAttackNumber(int(event.Damage), event.Spell == attack.SpellHealWounds)
			g.player.Client.HP = int(event.NewHP)
			if g.player.Client.HP == 0 {
				g.player.Inmobilized = false
				g.player.Dead = true
			}
		case msgs.EPlayerSpell:
			event := ev.Data.(*msgs.EventPlayerSpell)
			//log.Printf("SpellHit m: %#v\n", event)
			g.SoundBoard.PlayFrom(assets.SoundFromSpell(event.Spell), g.player.X, g.player.Y, g.players[event.ID].X, g.players[event.ID].Y)
			g.players[event.ID].Effect.NewSpellHit(event.Spell)
			g.players[event.ID].Dead = event.Killed
		case msgs.EUseItemOk:
			event := ev.Data.(*msgs.EventUseItemOk)
			if event.Item.Type() == item.TypeConsumable {

				//log.Printf("UsePotionOk m: %#v\n", event)
				switch event.Item {
				case item.ManaPotion:
					g.player.Client.MP = int(event.Change)
				case item.HealthPotion:
					g.player.Client.HP = int(event.Change)
				}
				g.player.Inv.Slots[event.Slot.X][event.Slot.Y].Count = event.Count
				if event.Count == 0 {
					g.player.Inv.Slots[event.Slot.X][event.Slot.Y].Item = item.None
				}
				g.stats.potionAlpha = 1
				g.lastPotionUsed = event.Item
				g.SoundBoard.Play(assets.Potion)
				continue
			}

			switch event.Item.Type() {
			case item.TypeArmor:
				if event.Change == 0 {
					g.player.Inv.EquippedBody.X = 255
					g.player.Armor = nil
					continue
				}
				g.player.Inv.EquippedBody.X = event.Slot.X
				g.player.Inv.EquippedBody.Y = event.Slot.Y
				g.player.RefreshBody()
			case item.TypeHelmet:
				if event.Change == 0 {
					g.player.Inv.EquippedHead.X = 255
					g.player.Helmet = nil
					continue
				}
				g.player.Inv.EquippedHead.X = event.Slot.X
				g.player.Inv.EquippedHead.Y = event.Slot.Y
				g.player.RefreshHead()

			case item.TypeShield:
				if event.Change == 0 {
					g.player.Inv.EquippedShield.X = 255
					g.player.Shield = nil
					continue
				}
				g.player.Inv.EquippedShield.X = event.Slot.X
				g.player.Inv.EquippedShield.Y = event.Slot.Y
				g.player.RefreshShield()

			case item.TypeWeapon:
				if event.Change == 0 {
					g.player.Inv.EquippedWeapon.X = 255
					g.player.Weapon = nil
					continue
				}
				g.player.Inv.EquippedWeapon.X = event.Slot.X
				g.player.Inv.EquippedWeapon.Y = event.Slot.Y
				g.player.RefreshWeapon()
				g.stats.meleeCooldownInfo.Weapon = g.player.Inv.GetWeapon()
			}
			//g.player.RefreshEquipped()
		case msgs.EBroadcastChat:
			event := ev.Data.(*msgs.EventBroadcastChat)
			g.players[event.ID].SetChatMsg(event.Msg)
		case msgs.EPlayerChangedSkin:
			event := ev.Data.(*msgs.EventPlayerChangedSkin)
			g.players[event.ID].MaybeLoadAnimations(event)
		case msgs.EUpdateSkillsOk:
			event := ev.Data.(*msgs.Experience)
			g.player.Exp = *event
			g.stats.skills.updatedSkills = g.player.Exp.Skills
			g.stats.skills.FreePoints = int(g.player.Exp.FreePoints)
			if g.player.Client.HP > int(g.player.Exp.Stats.MaxHP) {
				g.player.Client.HP = int(g.player.Exp.Stats.MaxHP)
			}
			if g.player.Client.MP > int(g.player.Exp.Stats.MaxMP) {
				g.player.Client.MP = int(g.player.Exp.Stats.MaxMP)
			}
		}
	}
	g.eventQueue = g.eventQueue[:0]
	g.eventLock.Unlock()
	return nil
}

func (g *Game) pingServer() {
	if g.WaitingPong || g.counter%240 != 0 {
		return
	}
	g.outQueue <- &GameMsg{E: msgs.EPing}
	g.WaitingPong = true
	g.LastPing = time.Now()
}

func (g *Game) DespawnPlayer(pid uint16) {
	p := g.players[pid]
	g.players[pid] = nil
	if !p.Dead {
		g.world.Space.Set(0, typ.P{X: int32(p.X), Y: int32(p.Y)}, 0)
	}
	for iy, ys := range g.playersY {
		if py, ok := ys.(*player.P); ok && pid == uint16(py.ID) {
			g.playersY[iy] = g.playersY[len(g.playersY)-1]
			g.playersY = g.playersY[:len(g.playersY)-1]
			break
		}
	}
}
func (g *Game) SendChat() {
	if msg := g.keys.ChatMessage(); msg != "" {
		g.outQueue <- &GameMsg{
			E:    msgs.ESendChat,
			Data: &msgs.EventSendChat{Msg: msg},
		}
	}
}
func (g *Game) GameTooltip() {

}
func DirToNewPos(p *player.P, d direction.D) typ.P {
	switch d {
	case direction.Front:
		return typ.P{X: p.X, Y: p.Y + 1}
	case direction.Back:
		return typ.P{X: p.X, Y: p.Y - 1}
	case direction.Left:
		return typ.P{X: p.X - 1, Y: p.Y}
	case direction.Right:
		return typ.P{X: p.X + 1, Y: p.Y}
	}
	return typ.P{X: p.X, Y: p.Y}
}

func (g *Game) MovementResponse(data []byte) {

	g.lastMoveConfirmed = true
	allowed := data[0] != 0
	dir := direction.D(data[1])
	//log.Printf("MovementResponse %v  %v\n", allowed, dir)
	if len(g.steps) == 0 {
		g.player.Walking = false
		log.Printf("move ok arrived without having a step sent\n")
		return
	}
	step := g.steps[0]
	g.steps = g.steps[1:]
	//log.Printf("MOVE:[%v][%v] %v %vms\n", allowed, step.Expect, direction.S(g.player.Direction), time.Since(g.startForStep).Milliseconds())

	if allowed == step.Expect {
		// if it was expected, we already did it
		return
	}
	if !allowed && step.Expect {
		// if it wasnt expected to fail we need to go back
		goBackPx := float64(constants.TileSize - g.leftForMove)
		log.Println("move not allowed, go back", goBackPx, "px", *g.player)
		switch dir {
		case direction.Front:
			g.player.Y--
			g.player.Pos[1] -= goBackPx
		case direction.Back:
			g.player.Y++
			g.player.Pos[1] += goBackPx
		case direction.Left:
			g.player.X++
			g.player.Pos[0] += goBackPx
		case direction.Right:
			g.player.X--
			g.player.Pos[0] -= goBackPx
		}
		g.player.Walking = false
		g.leftForMove = 0
		return
	}

	if allowed && !step.Expect {
		g.player.Inmobilized = false // if were allowed to move then we can change this already in any case
		g.lastDir = dir
		g.Move(dir, step.To)
		return
	}

}

func (g *Game) TryMove(d direction.D, np typ.P, confident bool) {
	g.lastMoveConfirmed = false
	g.outQueue <- &GameMsg{E: msgs.EMove, Data: d}
	g.steps = append(g.steps, player.Step{
		To:     np,
		Dir:    d,
		Expect: confident,
	})
	if confident {
		g.Move(d, np)
	}
}

func (g *Game) Move(d direction.D, np typ.P) {
	g.leftForMove += constants.TileSize
	g.player.Walking = true
	g.player.X = np.X
	g.player.Y = np.Y
	if g.player.Dead {
		return
	}
	if !g.player.Walking {
		g.SoundBoard.Play(assets.Walk1)
		g.soundPrevWalk = 1
	} else {
		if g.soundPrevWalk == 1 {
			g.SoundBoard.Play(assets.Walk2)
			g.soundPrevWalk = 2
		} else {
			g.SoundBoard.Play(assets.Walk1)
			g.soundPrevWalk = 1
		}
	}
}

func (g *Game) AddGameStep(d direction.D) {
	g.startForStep = time.Now()
	g.player.Direction = d
	np := DirToNewPos(g.player, d)
	outWorld := np.Out(g.world.Space.Rect)
	if outWorld {
		return
	}
	stuff := g.world.Space.GetSlot(1, np)
	if stuff != 0 && d == g.lastDir {
		return
	}

	playerLayer := g.world.Space.GetSlot(0, np)

	confident := !outWorld && !g.player.Inmobilized && playerLayer == 0 && stuff == 0
	g.lastDir = d
	g.TryMove(d, np, confident)
}

func (g *Game) UpdateGamePos() {
	if g.leftForMove > 0 {
		vel := g.player.MoveSpeed
		if g.leftForMove <= vel {
			vel = g.leftForMove
		}
		switch g.lastDir {
		case direction.Front:
			g.player.Pos[1] += vel
		case direction.Back:
			g.player.Pos[1] -= vel
		case direction.Left:
			g.player.Pos[0] -= vel
		case direction.Right:
			g.player.Pos[0] += vel
		}
		g.leftForMove -= vel
	} else {
		g.player.Walking = false
	}
}

func (g *Game) ListenInputs() {
	g.keys.ListenMovement()
	spellSelected := g.keys.ListenSpell()
	if spellSelected != attack.SpellNone && spellSelected != g.SelectedSpell {
		g.outQueue <- &GameMsg{E: msgs.ESelectSpell, Data: spellSelected}
		g.SelectedSpell = spellSelected
	}

	d := g.keys.MovingTo()

	if d == direction.Still {
		goto COMBAT
	}
	if g.lastMoveConfirmed && g.leftForMove == 0 && (time.Since(g.startForStep).Milliseconds() > 60) {
		g.AddGameStep(d)
	}
COMBAT:
	{
		if g.keys.MeleeHit() {
			g.outQueue <- &GameMsg{E: msgs.EMelee, Data: d}
		}
		if ok, x, y := g.keys.CastSpell(); ok {
			worldX, worldY := g.ScreenToWorld(x, y)
			g.outQueue <- &GameMsg{E: msgs.ECastSpell, Data: &msgs.EventCastSpell{
				PX: uint32(worldX),
				PY: uint32(worldY),
			}}
		}
		// HARDCODING POTION SLOTS
		if pressedPotion := g.keys.PressedPotion(); pressedPotion != item.None {
			if pressedPotion == item.HealthPotion {
				if g.player.Inv.HealthPotions.X != 255 {
					msg := msgs.EventUseItem(g.player.Inv.HealthPotions)
					g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msg}
				}
			} else {
				if g.player.Inv.ManaPotions.X != 255 {
					msg := msgs.EventUseItem(g.player.Inv.ManaPotions)
					g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msg}
				}
			}
		}

	}
}

const HalfScreenX, HalfScreenY = ScreenWidth / 2, ScreenHeight / 2

func (g *Game) updateWorldMatrix() {
	g.worldImgOp.GeoM.Reset()
	g.worldImgOp.GeoM.Translate(-g.player.Pos[0]+HalfScreenX-16, -g.player.Pos[1]+HalfScreenY-48)
}

func (g *Game) Render(world, screen *ebiten.Image) {
	g.updateWorldMatrix()

	screen.DrawImage(world, g.worldImgOp)
}

func (g *Game) ScreenToWorld(posX, posY int) (float64, float64) {
	g.updateWorldMatrix()
	if g.worldImgOp.GeoM.IsInvertible() {
		g.worldImgOp.GeoM.Invert()
		return g.worldImgOp.GeoM.Apply(float64(posX), float64(posY))
	} else {
		// When scaling it can happened that matrix is not invertable
		return math.NaN(), math.NaN()
	}
}
