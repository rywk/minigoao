package player

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/client/audio2d"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/assets"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
	"golang.org/x/image/math/f64"
)

// Player
type P struct {
	local       *P
	ID          uint32
	Nick        string
	NickImg     *ebiten.Image
	X, Y        int32
	DX, DY      int
	Pos         f64.Vec2
	drawOp      *ebiten.DrawImageOptions
	Dead        bool
	Inmobilized bool

	// other players use this, client uses inventory
	ArmorID, WeaponID, HelmetID, ShieldID item.Item

	Armor, Weapon, Helmet, Shield       texture.A
	NakedBody, Head, DeadBody, DeadHead texture.A
	Walking                             bool
	Direction                           direction.D

	ActiveEffects []texture.Effect
	Effect        *PEffects

	Client       *ClientP
	HPImg, MPImg *ebiten.Image
	HPMPBGImg    *ebiten.Image
	MoveSpeed    float64
	leftForMove  float64
	lastDir      direction.D
	steps        []Step

	chatMsg      string
	chatMsgStart time.Time

	Exp msgs.Experience
	Inv msgs.Inventory

	soundPrevWalk int
	soundboard    audio2d.AudioMixer
}

type ClientP struct {
	p *P
	// Stats
	HP, MP int
}

func NewClientP() *ClientP {
	return &ClientP{}
}

func (p *ClientP) DirToNewPos(d direction.D) (int, int) {
	switch d {
	case direction.Front:
		return int(p.p.X), int(p.p.Y + 1)
	case direction.Back:
		return int(p.p.X), int(p.p.Y - 1)
	case direction.Left:
		return int(p.p.X - 1), int(p.p.Y)
	case direction.Right:
		return int(p.p.X + 1), int(p.p.Y)
	}
	return 0, 0
}

func (p *P) Nil() bool { return p == nil }

func (p *P) Update(counter int) {
	if p.Dead {
		p.DeadBody.Dir(p.Direction)
		p.DeadHead.Dir(p.Direction)
	} else {

		if p.Armor != nil {
			p.Armor.Dir(p.Direction)
		} else {
			p.NakedBody.Dir(p.Direction)
		}
		p.Head.Dir(p.Direction)
		if p.Helmet != nil {
			p.Helmet.Dir(p.Direction)
		}
		if p.Weapon != nil {
			p.Weapon.Dir(p.Direction)
		}
		if p.Shield != nil {
			p.Shield.Dir(p.Direction)
		}
	}
	p.UpdateFrames(counter)
	if time.Since(p.chatMsgStart) > constants.ChatMsgTTL {
		p.chatMsg = ""
	}
}

func (p *P) SetChatMsg(msg string) {
	p.chatMsgStart = time.Now()
	p.chatMsg = msg
}

const PlayerDrawOffsetX, PlayerDrawOffsetY = 3, -14
const PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY = 4, -8

func (p *P) DrawOff(screen *ebiten.Image, offset ebiten.GeoM) {
	p.drawOp.GeoM.Reset()
	p.drawOp.GeoM.Concat(offset)
	p.drawOp.GeoM.Translate(p.Pos[0]+PlayerDrawOffsetX, p.Pos[1]+PlayerDrawOffsetY)
	if p.Dead {
		p.drawOp.GeoM.Translate(2, 5)
		screen.DrawImage(p.DeadBody.Frame(), p.drawOp)
		p.drawOp.GeoM.Translate(PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY)
		screen.DrawImage(p.DeadHead.Frame(), p.drawOp)
		p.Effect.Draw(screen, offset)
		return
	}
	if p.Direction == direction.Back || p.Direction == direction.Left {
		if p.Weapon != nil {
			screen.DrawImage(p.Weapon.Frame(), p.drawOp)
		}
	}
	if p.Direction == direction.Left || p.Direction == direction.Front {
		if p.Armor != nil {
			screen.DrawImage(p.Armor.Frame(), p.drawOp)
		} else {
			screen.DrawImage(p.NakedBody.Frame(), p.drawOp)
		}
		if p.Shield != nil {
			screen.DrawImage(p.Shield.Frame(), p.drawOp)
		}
	} else {
		if p.Shield != nil {
			screen.DrawImage(p.Shield.Frame(), p.drawOp)
		}
		if p.Armor != nil {
			screen.DrawImage(p.Armor.Frame(), p.drawOp)
		} else {
			screen.DrawImage(p.NakedBody.Frame(), p.drawOp)
		}
	}
	if p.Direction == direction.Front || p.Direction == direction.Right {
		if p.Weapon != nil {
			screen.DrawImage(p.Weapon.Frame(), p.drawOp)
		}
	}
	p.drawOp.GeoM.Translate(PlayerHeadDrawOffsetX, PlayerHeadDrawOffsetY)
	screen.DrawImage(p.Head.Frame(), p.drawOp)
	if p.Helmet != nil {
		p.drawOp.GeoM.Translate(-5, -7)
		screen.DrawImage(p.Helmet.Frame(), p.drawOp)
	}
	if p.local == nil {
		p.DrawPlayerHPMP(screen, offset)

	} else {
		p.DrawNick(screen, offset)
	}
	p.Effect.Draw(screen, offset)
	if p.chatMsg != "" {
		off := len(p.chatMsg) * 3
		text.PrintAt(screen, p.chatMsg, int(p.Pos[0])+16-off, int(p.Pos[1]-40))
	}
}

func (p *P) DrawNick(screen *ebiten.Image, offset ebiten.GeoM) {
	p.drawOp.GeoM.Reset()
	p.drawOp.GeoM.Concat(offset)
	p.drawOp.GeoM.Translate(p.Pos[0]+PlayerDrawOffsetX, p.Pos[1]+PlayerDrawOffsetY)
	xoff := (len(p.Nick) * 3) - 1
	p.drawOp.GeoM.Translate(-float64(xoff-12), 40)
	screen.DrawImage(p.NickImg, p.drawOp)
}

func (p *P) DrawPlayerHPMP(screen *ebiten.Image, offset ebiten.GeoM) {
	p.drawOp.GeoM.Reset()
	p.drawOp.GeoM.Concat(offset)
	p.drawOp.GeoM.Translate(p.Pos[0]+PlayerDrawOffsetX, p.Pos[1]+PlayerDrawOffsetY)
	p.drawOp.GeoM.Translate(-2, 45)
	hpx, mpx := p.HPImg.Bounds().Max.X, p.MPImg.Bounds().Max.X
	hpx, mpx = p.Client.HP*hpx/int(p.Exp.Stats.MaxHP), p.Client.MP*mpx/int(p.Exp.Stats.MaxMP)
	hpRect := image.Rect(p.HPImg.Bounds().Min.X, p.HPImg.Bounds().Min.Y, hpx, p.HPImg.Bounds().Max.Y)
	mpRect := image.Rect(p.MPImg.Bounds().Min.X, p.MPImg.Bounds().Min.Y, mpx, p.MPImg.Bounds().Max.Y)
	p.drawOp.GeoM.Translate(-1, -1)
	screen.DrawImage(p.HPMPBGImg, p.drawOp)
	p.drawOp.GeoM.Translate(1, 1)
	screen.DrawImage(p.HPImg.SubImage(hpRect).(*ebiten.Image), p.drawOp)
	p.drawOp.GeoM.Translate(0, 5)
	screen.DrawImage(p.MPImg.SubImage(mpRect).(*ebiten.Image), p.drawOp)
	xoff := (len(p.Nick) * 3) - 1
	p.drawOp.GeoM.Translate(-float64(xoff-12), 0)
	screen.DrawImage(p.NickImg, p.drawOp)
}

func (p *P) UpdateFrames(c int) {
	if c%6 == 0 {
		if p.Walking {
			if p.Dead {
				p.DeadBody.Next(p.Direction)
				p.DeadHead.Next(p.Direction)
			} else {
				if p.Armor != nil {
					p.Armor.Next(p.Direction)
				} else {
					p.NakedBody.Next(p.Direction)
				}
				p.Head.Next(p.Direction)
				if p.Helmet != nil {
					p.Helmet.Next(p.Direction)
				}
				if p.Weapon != nil {
					p.Weapon.Next(p.Direction)
				}
				if p.Shield != nil {
					p.Shield.Next(p.Direction)
				}
			}
		} else {
			if p.Dead {
				p.DeadBody.Stopped(p.Direction)
				p.DeadHead.Stopped(p.Direction)
			} else {
				if p.Armor != nil {
					p.Armor.Stopped(p.Direction)
				} else {
					p.NakedBody.Stopped(p.Direction)
				}
				p.Head.Stopped(p.Direction)
				if p.Helmet != nil {
					p.Helmet.Stopped(p.Direction)
				}
				if p.Weapon != nil {
					p.Weapon.Stopped(p.Direction)
				}
				if p.Shield != nil {
					p.Shield.Stopped(p.Direction)
				}
			}
		}
	}
}

func (p *P) SetSoundboard(sb audio2d.AudioMixer) {
	p.soundboard = sb
}

type Step struct {
	To        typ.P
	Dir       direction.D
	Expect    bool
	Allowed   bool
	Confirmed bool
}

func (p *P) WalkSteps(g *grid.Grid) {
	if p.leftForMove > 0 {
		vel := p.MoveSpeed
		if p.leftForMove < vel {
			vel = p.leftForMove
		}
		switch p.lastDir {
		case direction.Front:
			p.Pos[1] += vel
		case direction.Back:
			p.Pos[1] -= vel
		case direction.Left:
			p.Pos[0] -= vel
		case direction.Right:
			p.Pos[0] += vel
		}
		p.leftForMove -= vel
	}
	if p.leftForMove != 0 {
		return
	}
	if len(p.steps) == 0 {
		p.Walking = false
		return
	}
	step := p.steps[0]
	p.steps = p.steps[1:]
	if p.X == step.To.X && p.Y == step.To.Y {
		p.Direction = step.Dir
		return
	}
	if !p.Dead {
		if !p.Walking {
			p.soundboard.PlayFrom(assets.Walk1, p.local.X, p.local.Y, p.X, p.Y)
			p.soundPrevWalk = 1
		} else {
			if p.soundPrevWalk == 1 {
				p.soundboard.PlayFrom(assets.Walk2, p.local.X, p.local.Y, p.X, p.Y)
				p.soundPrevWalk = 2
			} else {
				p.soundboard.PlayFrom(assets.Walk1, p.local.X, p.local.Y, p.X, p.Y)
				p.soundPrevWalk = 1
			}
		}
	}
	p.Direction = step.Dir
	p.lastDir = step.Dir
	p.leftForMove = constants.TileSize
	p.Walking = true
	p.Pos[0] = float64(p.X * constants.TileSize)
	p.Pos[1] = float64(p.Y * constants.TileSize)
	g.Move(0, typ.P{X: int32(p.X), Y: int32(p.Y)}, step.To)
	p.X, p.Y = step.To.X, step.To.Y
}

func (p *P) AddStep(e *msgs.EventPlayerMoved) {
	p.steps = append(p.steps, Step{
		To:  e.Pos,
		Dir: e.Dir,
	})
}

func (p *P) LoadAnimations() {
	p.Armor = texture.LoadItemAninmatio(p.ArmorID)
	p.Helmet = texture.LoadItemHead(p.HelmetID)
	p.Weapon = texture.LoadItemAninmatio(p.WeaponID)
	p.Shield = texture.LoadItemAninmatio(p.ShieldID)
}

func (p *P) MaybeLoadAnimations(c *msgs.EventPlayerChangedSkin) {
	if p.ArmorID != c.Armor {
		p.ArmorID = c.Armor
		p.Armor = texture.LoadItemAninmatio(p.ArmorID)
	}
	if p.HelmetID != c.Head {
		p.HelmetID = c.Head
		p.Helmet = texture.LoadItemHead(p.HelmetID)
	}
	if p.WeaponID != c.Weapon {
		p.WeaponID = c.Weapon
		p.Weapon = texture.LoadItemAninmatio(p.WeaponID)
	}
	if p.ShieldID != c.Shield {
		p.ShieldID = c.Shield
		p.Shield = texture.LoadItemAninmatio(p.ShieldID)
	}
}

func (p *P) RefreshEquipped() {
	if p.Inv.EquippedBody.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedBody)
		p.Armor = texture.LoadItemAninmatio(is.Item)
	}
	if p.Inv.EquippedHead.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedHead)
		p.Armor = texture.LoadItemHead(is.Item)
	}
	if p.Inv.EquippedWeapon.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedWeapon)
		p.Armor = texture.LoadItemAninmatio(is.Item)
	}
	if p.Inv.EquippedShield.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedShield)
		p.Armor = texture.LoadItemAninmatio(is.Item)
	}
}
func (p *P) RefreshBody() {
	if p.Inv.EquippedBody.X == 255 {
		p.Armor = nil
	}
	is := p.Inv.GetSlotv2(&p.Inv.EquippedBody)
	p.Armor = texture.LoadItemAninmatio(is.Item)
}
func (p *P) RefreshHead() {
	if p.Inv.EquippedHead.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedHead)
		p.Helmet = texture.LoadItemHead(is.Item)
	}
}
func (p *P) RefreshWeapon() {
	if p.Inv.EquippedWeapon.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedWeapon)
		p.Weapon = texture.LoadItemAninmatio(is.Item)
	}
}
func (p *P) RefreshShield() {
	if p.Inv.EquippedShield.X != 255 {
		is := p.Inv.GetSlotv2(&p.Inv.EquippedShield)
		p.Shield = texture.LoadItemAninmatio(is.Item)
	}
}
func NewLogin(e *msgs.EventPlayerLogin) *P {
	p := Create(e)
	p.Client = NewClientP()
	p.Client.HP = int(e.HP)
	p.Client.MP = int(e.MP)
	p.Client.p = p
	p.HPImg = ebiten.NewImage(30, 3)
	p.MPImg = ebiten.NewImage(30, 3)
	p.HPMPBGImg = ebiten.NewImage(32, 10)
	p.HPImg.Fill(color.RGBA{255, 60, 60, 255})
	p.MPImg.Fill(color.RGBA{40, 130, 250, 255})
	p.HPMPBGImg.Fill(color.RGBA{0, 0, 0, 200})
	p.Inv = e.Inv
	return p
}

func Create(a *msgs.EventPlayerLogin) *P {
	p := &P{
		drawOp:    &ebiten.DrawImageOptions{},
		ID:        uint32(a.ID),
		X:         a.Pos.X,
		Y:         a.Pos.Y,
		Direction: a.Dir,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.Pos.X) * constants.TileSize,
			float64(a.Pos.Y) * constants.TileSize},
		Nick:      a.Nick,
		NickImg:   text.PrintImg(a.Nick),
		Dead:      a.Dead,
		Exp:       a.Exp,
		MoveSpeed: float64(a.Speed),
		ArmorID:   a.Inv.GetBody(),
		HelmetID:  a.Inv.GetHead(),
		WeaponID:  a.Inv.GetWeapon(),
		ShieldID:  a.Inv.GetShield(),
		NakedBody: texture.LoadAnimation(assets.NakedBody),
		Head:      texture.LoadStill(assets.Head),
		DeadBody:  texture.LoadAnimation(assets.DeadBody),
		DeadHead:  texture.LoadStill(assets.DeadHead),
	}
	p.Effect = &PEffects{
		drawOp: &ebiten.DrawImageOptions{},
		p:      p,
		active: make([]texture.Effect, 0),
	}
	p.LoadAnimations()
	return p
}

func CreatePlayerSpawned(local *P, a *msgs.EventPlayerSpawned) *P {
	p := &P{
		drawOp:    &ebiten.DrawImageOptions{},
		local:     local,
		ID:        uint32(a.ID),
		X:         a.Pos.X,
		Y:         a.Pos.Y,
		Direction: a.Dir,
		Pos: f64.Vec2{ // pixel value of position
			float64(a.Pos.X) * constants.TileSize,
			float64(a.Pos.Y) * constants.TileSize},
		Nick:      a.Nick,
		NickImg:   text.PrintImg(a.Nick),
		Dead:      a.Dead,
		MoveSpeed: float64(a.Speed),
		ArmorID:   a.Body,
		HelmetID:  a.Head,
		WeaponID:  a.Weapon,
		ShieldID:  a.Shield,
		NakedBody: texture.LoadAnimation(assets.NakedBody),
		Head:      texture.LoadStill(assets.Head),
		DeadBody:  texture.LoadAnimation(assets.DeadBody),
		DeadHead:  texture.LoadStill(assets.DeadHead),
	}
	p.Effect = &PEffects{
		drawOp: &ebiten.DrawImageOptions{},
		p:      p,
		active: make([]texture.Effect, 0),
	}
	p.LoadAnimations()
	return p
}

// maps.YSortable
func (p *P) ValueY() float64 { return p.Pos[1] }

type PEffects struct {
	p      *P
	active []texture.Effect
	drawOp *ebiten.DrawImageOptions
}

func (pfx *PEffects) NewMeleeHit() {
	pfx.active = append(pfx.active, texture.LoadEffect(assets.MeleeHit))
}

func (pfx *PEffects) NewSpellHit(s attack.Spell) {
	a := texture.AssetFromSpell(s)
	if a != assets.Nothing {
		pfx.active = append(pfx.active, NewSpellOffset(a))
	}
}

func (pfx *PEffects) NewAttackNumber(dmg int, heal bool) {
	if dmg > 0 {
		dmgs := ""
		if dmg >= 150 {
			dmgs = fmt.Sprintf("¡%d!", dmg)
		} else {
			dmgs = strconv.FormatInt(int64(dmg), 10)
		}
		offx := len(dmgs) * 4
		pfx.active = append(pfx.active, &AtkDmgFxTxt{img: ebiten.NewImage(40, 60), dmg: dmgs, heal: heal, offx: offx})
	}
}

type SpellOffset struct {
	x, y int
	fx   texture.Effect
}

var spellOffsets = map[assets.Image]struct{ x, y int }{
	assets.SpellApoca:      {-20, -40},
	assets.SpellInmo:       {-30, -55},
	assets.SpellInmoRm:     {-20, -30},
	assets.SpellDesca:      {-6, -178},
	assets.SpellHealWounds: {-40, -56},
	assets.SpellResurrect:  {-24, -36},
}

func NewSpellOffset(a assets.Image) *SpellOffset {
	off := spellOffsets[a]
	return &SpellOffset{off.x, off.y, texture.LoadEffect(a)}
}

func (as *SpellOffset) Play() bool {
	return as.fx.Play()
}

func (as *SpellOffset) EffectFrame() *ebiten.Image {
	return as.fx.EffectFrame()
}

func (as *SpellOffset) EffectOpt(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	op.GeoM.Translate(float64(as.x), float64(as.y))
	return op
}

type AtkDmgFxTxt struct {
	dmg  string
	heal bool
	img  *ebiten.Image
	y    int
	c    int
	offx int
}

func (adt *AtkDmgFxTxt) Play() bool {
	adt.img.Clear()
	col := color.RGBA{194, 6, 6, 255}
	if adt.heal {
		col = color.RGBA{6, 153, 194, 255}
	}

	text.DrawNumbers(adt.img, adt.dmg, 0, 46-adt.y, col)
	if adt.y == 46 {
		adt.y = 0
		return false
	}
	//if adt.c%2 == 0 {
	adt.y++
	//}
	adt.c++

	return true
}

func (adt *AtkDmgFxTxt) EffectFrame() *ebiten.Image {
	return adt.img
}

func (a *AtkDmgFxTxt) EffectOpt(op *ebiten.DrawImageOptions) *ebiten.DrawImageOptions {
	op.GeoM.Translate(float64(18-a.offx), -70)
	return op
}

func (pfx *PEffects) Update(counter int) {
	if counter%2 == 0 {
		i := 0
		for _, fx := range pfx.active {
			if fx.Play() {
				pfx.active[i] = fx
				i++
			}
		}
		for j := i; j < len(pfx.active); j++ {
			pfx.active[j] = nil
		}
		pfx.active = pfx.active[:i]
	}
}

func (pfx *PEffects) Draw(screen *ebiten.Image, offset ebiten.GeoM) {
	for _, fx := range pfx.active {
		pfx.drawOp.GeoM.Reset()
		pfx.drawOp.GeoM.Concat(offset)
		pfx.drawOp.GeoM.Translate(pfx.p.Pos[0], pfx.p.Pos[1])
		screen.DrawImage(fx.EffectFrame(), fx.EffectOpt(pfx.drawOp))
	}
}
