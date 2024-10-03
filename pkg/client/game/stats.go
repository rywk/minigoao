package game

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/rywk/minigoao/pkg/client/game/assets/img"
	"github.com/rywk/minigoao/pkg/client/game/assets/img/itemimg"
	"github.com/rywk/minigoao/pkg/client/game/assets/img/spellimg"
	"github.com/rywk/minigoao/pkg/client/game/text"
	"github.com/rywk/minigoao/pkg/client/game/texture"
	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

type Checkbox struct {
	g             *Game
	Pos           typ.P
	W, H          int32
	ImgOn, ImgOff *ebiten.Image
	On            bool
	Pressed       bool
}

func NewCheckbox(g *Game) *Checkbox {
	on, off := texture.Decode(img.CheckboxOn_png), texture.Decode(img.CheckboxOff_png)
	return &Checkbox{
		g:      g,
		W:      int32(on.Bounds().Dx()),
		H:      int32(on.Bounds().Dy()),
		ImgOn:  on,
		ImgOff: off,
		On:     false,
	}
}
func (b *Checkbox) Draw(screen *ebiten.Image, x, y int) {
	b.Pos = typ.P{X: int32(x), Y: int32(y)}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(b.Pos.X), float64(b.Pos.Y))
	if b.On {
		screen.DrawImage(b.ImgOn, op)
	} else {
		screen.DrawImage(b.ImgOff, op)
	}
}

func (b *Checkbox) Update() {
	cx, cy := b.g.mouseX, b.g.mouseY
	if cx > int(b.Pos.X) && cx < int(b.Pos.X+b.W) && cy > int(b.Pos.Y) && cy < int(b.Pos.Y+b.H) {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !b.Pressed {
				b.On = !b.On
			}
			b.Pressed = true
		} else {
			b.Pressed = false
		}
	}
}

type Button struct {
	g        *Game
	Pos      typ.P
	W, H     int32
	Img      *ebiten.Image
	Icon     *ebiten.Image
	Over     bool
	pressed  bool
	pressed2 bool
}

func NewButton(g *Game, icon *ebiten.Image, bg *ebiten.Image, pos typ.P) *Button {
	return &Button{
		g:    g,
		Pos:  pos,
		W:    int32(bg.Bounds().Dx()),
		H:    int32(bg.Bounds().Dy()),
		Img:  bg,
		Icon: icon,
	}
}

func (b *Button) Draw(screen *ebiten.Image, x, y int) {

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	if b.Over {
		op.ColorScale.ScaleAlpha(.6)
	}
	screen.DrawImage(b.Img, op)
	screen.DrawImage(b.Icon, op)

}

func (b *Button) Pressed() bool {
	cx, cy := b.g.mouseX, b.g.mouseY
	v := false
	if cx > int(b.Pos.X) && cx < int(b.Pos.X+b.W) && cy > int(b.Pos.Y) && cy < int(b.Pos.Y+b.H) {
		b.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			b.pressed = true
		} else {
			if b.pressed {
				v = true
			}
			b.pressed = false
		}
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton2) {
			b.pressed2 = true
		} else {
			if b.pressed2 {
				v = true
			}
			b.pressed2 = false
		}
	} else {
		b.Over = false
	}
	return v
}

type Slider struct {
	g   *Game
	Pos image.Point

	W, H      int32
	Knob      *ebiten.Image
	Line      *ebiten.Image
	drawOp    *ebiten.DrawImageOptions
	Over      bool
	on        bool
	pressed   bool
	SliderPos image.Point

	value int
}

func NewSlider(g *Game) *Slider {
	volKnob := ebiten.NewImage(10, 22)
	s := &Slider{
		drawOp: &ebiten.DrawImageOptions{},
		g:      g,
		W:      int32(volKnob.Bounds().Dx()),
		H:      int32(volKnob.Bounds().Dy()),
		Knob:   volKnob,
		Line:   ebiten.NewImage(200, 4),
		value:  160,
	}
	s.Pos.X = int(200 - s.value)
	s.g.SoundBoard.SetVolume(float64(s.value))

	s.Knob.Fill(color.White)
	s.Line.Fill(color.White)
	return s
}
func (b *Slider) Draw(screen *ebiten.Image, x, y int) {
	b.SliderPos = image.Pt(x, y)
	b.drawOp.ColorScale.Reset()
	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(float64(b.SliderPos.X), float64(b.SliderPos.Y)+9)
	screen.DrawImage(b.Line, b.drawOp)
	if b.Over || b.on {
		b.drawOp.ColorScale.Reset()
		b.drawOp.ColorScale.ScaleAlpha(.6)
	} else {
		b.drawOp.ColorScale.Reset()
		b.drawOp.ColorScale.ScaleAlpha(1)
	}
	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(float64(x+int(b.Pos.X)), float64(y+int(b.Pos.Y)))
	screen.DrawImage(b.Knob, b.drawOp)

}
func (b *Slider) Update() {
	cx, cy := b.g.mouseX, b.g.mouseY
	knobRect := b.Knob.Bounds().
		Add(b.Pos).
		Add(b.SliderPos).
		Add(image.Pt(ScreenWidth-300, ScreenHeight-664))

	sliderRect := b.Line.Bounds().
		Add(b.SliderPos).
		Add(image.Pt(ScreenWidth-300, ScreenHeight-664))
	//	log.Printf("pos : %v ", knobRect)

	if cx > int(knobRect.Min.X) && cx < int(knobRect.Max.X) && cy > int(knobRect.Min.Y) && cy < int(knobRect.Max.Y) {
		b.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !b.pressed {
				b.on = true
			}
			b.pressed = true
		} else {
			b.pressed = false
		}
	} else {
		b.Over = false
	}
	if b.on {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			b.on = false
			b.pressed = false
		}
		b.value = int(mapValue(float64(cx), float64(sliderRect.Min.X), float64(sliderRect.Max.X), 200, 0))
		b.Pos.X = int(200 - b.value)
		b.g.SoundBoard.SetVolume(float64(b.value))
	}
}

type SkillInput struct {
	g          *Game
	Less, More *Button
	Value      *uint16
}

func NewSkillInput(g *Game, value *uint16, pos typ.P) *SkillInput {
	subIcon := texture.Decode(img.IconSubstract_png)
	plusIcon := texture.Decode(img.IconPlus_png)
	btnImg := ebiten.NewImage(24, 24)
	btnImg.Fill(color.RGBA{101, 32, 133, 0})
	return &SkillInput{
		g:     g,
		Value: value,
		Less:  NewButton(g, subIcon, btnImg, pos.Add(-40, 0)),
		More:  NewButton(g, plusIcon, btnImg, pos.Add(30, 0)),
	}
}

func (ni *SkillInput) Update() {
	if *ni.Value > 0 && ni.Less.Pressed() {
		*ni.Value--
		ni.g.stats.skills.FreePoints++
	}
	if ni.g.stats.skills.FreePoints > 0 && ni.More.Pressed() {
		*ni.Value++
		ni.g.stats.skills.FreePoints--
	}
}

func (ni *SkillInput) Draw(screen *ebiten.Image, x int, y int) {
	ni.Less.Draw(screen, x-40, y)
	text.PrintBigAt(screen, fmt.Sprintf("%d", *ni.Value), x, y)
	ni.More.Draw(screen, x+30, y)
}

type Inventory struct {
	g               *Game
	drawOp          *ebiten.DrawImageOptions
	ItemSize        int32
	slotBtns        [8][2]*Button
	lockButton      *Button
	unlockButton    *Button
	locked          bool
	itemEquippedImg *ebiten.Image
	items           map[item.Item]*ebiten.Image
}

func NewInventory(g *Game) *Inventory {
	btnImg := ebiten.NewImage(32, 32)
	btnBgImg := ebiten.NewImage(32, 32)
	btnBgImg.Fill(color.RGBA{79, 90, 105, 200})
	lockBtnBgImg := ebiten.NewImage(24, 24)
	lockBtnBgImg.Fill(color.RGBA{60, 60, 60, 200})
	lockedImg := texture.Decode(img.IconLockLocked_png)
	unlockedImg := texture.Decode(img.IconLockOpen_png)
	inv := &Inventory{
		g:               g,
		drawOp:          &ebiten.DrawImageOptions{},
		locked:          true,
		ItemSize:        32,
		lockButton:      NewButton(g, lockedImg, lockBtnBgImg, typ.P{X: 312, Y: ScreenHeight - 44}),
		unlockButton:    NewButton(g, unlockedImg, lockBtnBgImg, typ.P{X: 312, Y: ScreenHeight - 44}),
		items:           map[item.Item]*ebiten.Image{},
		itemEquippedImg: texture.Decode(img.EquippedItem_png),
	}

	for i := range item.ItemLen {
		inv.items[i] = texture.DecodeItem(item.Item(i), true)
	}
	for i := range inv.slotBtns {
		inv.slotBtns[i][0] = NewButton(g, btnImg, btnBgImg, typ.P{X: 344 + int32(i)*inv.ItemSize, Y: ScreenHeight - 64})
		inv.slotBtns[i][1] = NewButton(g, btnImg, btnBgImg, typ.P{X: 344 + int32(i)*inv.ItemSize, Y: ScreenHeight - 32})
	}
	return inv
}

func (inv *Inventory) Update() {
	lock, unlock := inv.lockButton.Pressed(), inv.unlockButton.Pressed()
	if inv.locked {
		if lock {
			inv.locked = false
		}
	} else {
		if unlock {
			inv.locked = true
		}
	}
	for i := range inv.slotBtns {
		if inv.slotBtns[i][0].Pressed() {
			it := inv.g.player.Inv.Slots[i][0]
			if it.Item != item.None {
				if it.Item == item.HealthPotion || it.Item == item.ManaPotion {
					if time.Since(inv.g.stats.lastHudPotion) > inv.g.stats.hudPotionCooldown {
						inv.g.stats.lastHudPotion = time.Now()
						inv.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msgs.EventUseItem{byte(i), 0}}
					}
				} else {
					inv.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msgs.EventUseItem{byte(i), 0}}
				}
			}
		}
		if inv.slotBtns[i][0].Over {
			up := inv.g.player.Inv.Slots[i][0]

			if w, ok := inv.g.player.Exp.Items[up.Item]; ok {
				switch up.Item.Type() {
				case item.TypeConsumable:
					if up.Item == item.HealthPotion {
						inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nHeals +30 HP", up.Item.Name()))
					} else {
						inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nRestores %%5 of max MP", up.Item.Name()))

					}
				case item.TypeWeapon:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\n%v\nDamage: %d\nCrit Range: %d",
						up.Item.Name(),
						w.WeaponData.Cooldown,
						w.WeaponData.Damage,
						w.WeaponData.CriticRange,
					))
				case item.TypeArmor:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						up.Item.Name(),
						w.ArmorData.PhysicalDef,
						w.ArmorData.MagicDef,
					))
				case item.TypeHelmet:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						up.Item.Name(),
						w.HelmetData.PhysicalDef,
						w.HelmetData.MagicDef,
					))
				case item.TypeShield:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						up.Item.Name(),
						w.ShieldData.PhysicalDef,
						w.ShieldData.MagicDef,
					))
				}

			}
		}
		if inv.slotBtns[i][1].Pressed() {
			it := inv.g.player.Inv.Slots[i][1]
			if it.Item != item.None {
				if it.Item == item.HealthPotion || it.Item == item.ManaPotion {
					if time.Since(inv.g.stats.lastHudPotion) > inv.g.stats.hudPotionCooldown {
						inv.g.stats.lastHudPotion = time.Now()
						inv.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msgs.EventUseItem{byte(i), 1}}
					}
				} else {
					inv.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: &msgs.EventUseItem{byte(i), 1}}
				}
			}
		}
		if inv.slotBtns[i][1].Over {
			down := inv.g.player.Inv.Slots[i][1]

			if w, ok := inv.g.player.Exp.Items[down.Item]; ok {
				switch down.Item.Type() {
				case item.TypeConsumable:
					if down.Item == item.HealthPotion {
						inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nHeals +30 HP", down.Item.Name()))
					} else {
						inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nRestores %%5 of max MP", down.Item.Name()))

					}
				case item.TypeWeapon:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\n%v\nDamage: %d\nCrit Range: %d",
						down.Item.Name(),
						w.WeaponData.Cooldown,
						w.WeaponData.Damage,
						w.WeaponData.CriticRange,
					))
				case item.TypeArmor:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						down.Item.Name(),
						w.ArmorData.PhysicalDef,
						w.ArmorData.MagicDef,
					))
				case item.TypeHelmet:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						down.Item.Name(),
						w.HelmetData.PhysicalDef,
						w.HelmetData.MagicDef,
					))
				case item.TypeShield:
					inv.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\nPhysical Def: %d\nMagic Def: %d",
						down.Item.Name(),
						w.ShieldData.PhysicalDef,
						w.ShieldData.MagicDef,
					))
				}

			}
		}
	}
}

func (inv *Inventory) Draw(screen *ebiten.Image) {
	if inv.locked {
		inv.lockButton.Draw(screen, 312, ScreenHeight-44)
	} else {
		inv.unlockButton.Draw(screen, 312, ScreenHeight-44)
	}
	for i := range inv.slotBtns {
		inv.slotBtns[i][0].Draw(screen, int(344+int32(i)*inv.ItemSize), ScreenHeight-64)
		inv.slotBtns[i][1].Draw(screen, int(344+int32(i)*inv.ItemSize), ScreenHeight-32)
	}
	equipped := map[msgs.InventoryPos]struct{}{}
	if inv.g.player.Inv.EquippedBody.X != 255 {
		equipped[inv.g.player.Inv.EquippedBody] = struct{}{}
	}
	if inv.g.player.Inv.EquippedHead.X != 255 {
		equipped[inv.g.player.Inv.EquippedHead] = struct{}{}
	}
	if inv.g.player.Inv.EquippedShield.X != 255 {
		equipped[inv.g.player.Inv.EquippedShield] = struct{}{}
	}
	if inv.g.player.Inv.EquippedWeapon.X != 255 {
		equipped[inv.g.player.Inv.EquippedWeapon] = struct{}{}
	}
	op := &ebiten.DrawImageOptions{}
	for i := range inv.g.player.Inv.Slots {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(344+int32(i)*inv.ItemSize), ScreenHeight-64)
		up := inv.g.player.Inv.Slots[i][0]
		if up.Item != item.None {
			if _, ok := equipped[msgs.InventoryPos{uint8(i), 0}]; ok {
				screen.DrawImage(inv.itemEquippedImg, op)
			}
			screen.DrawImage(inv.items[up.Item], op)
			text.PrintAt(screen, fmt.Sprintf("%d", up.Count), int(344+int32(i)*inv.ItemSize), ScreenHeight-64)
		}

		down := inv.g.player.Inv.Slots[i][1]
		if down.Item != item.None {
			op.GeoM.Translate(0, 32)
			if _, ok := equipped[msgs.InventoryPos{uint8(i), 1}]; ok {
				screen.DrawImage(inv.itemEquippedImg, op)
			}
			screen.DrawImage(inv.items[down.Item], op)
			text.PrintAt(screen, fmt.Sprintf("%d", down.Count), int(344+int32(i)*inv.ItemSize), ScreenHeight-32)
		}

	}

}

type Skills struct {
	g          *Game
	drawOp     *ebiten.DrawImageOptions
	W, H       int32
	Background *ebiten.Image

	updatedSkills msgs.Skills
	saveButton    *Button
	zeroButton    *Button

	FreePoints int

	Agility            *SkillInput
	Intelligence       *SkillInput
	Vitality           *SkillInput
	FireAffinity       *SkillInput
	ElectricAffinity   *SkillInput
	ClericAffinity     *SkillInput
	AssasinAffinity    *SkillInput
	WarriorAffinity    *SkillInput
	MartialArtAffinity *SkillInput
}

func NewSkills(g *Game) *Skills {
	bg := ebiten.NewImage(600, 340)
	xoff := int32(60)
	height := int32(90)
	yoff := int32(60)

	btnImg := ebiten.NewImage(32, 32)
	btnImg.Fill(color.RGBA{101, 32, 133, 0})
	windowStart := typ.P{X: 340, Y: ScreenHeight - 404}
	s := &Skills{
		drawOp:        &ebiten.DrawImageOptions{},
		g:             g,
		W:             int32(bg.Bounds().Dx()),
		H:             int32(bg.Bounds().Dy()),
		saveButton:    NewButton(g, texture.Decode(img.IconDisk_png), btnImg, windowStart.Add(400, 10)),
		zeroButton:    NewButton(g, texture.Decode(img.IconX_png), btnImg, windowStart.Add(442, 10)),
		Background:    bg,
		FreePoints:    int(g.player.Exp.Skills.FreePoints),
		updatedSkills: g.player.Exp.Skills,
	}
	s.Agility = NewSkillInput(g, &s.updatedSkills.Agility, windowStart.Add(xoff+10, yoff+yoff/2))
	s.Intelligence = NewSkillInput(g, &s.updatedSkills.Intelligence, windowStart.Add(xoff+10, yoff+height+yoff/2))
	s.Vitality = NewSkillInput(g, &s.updatedSkills.Vitality, windowStart.Add(xoff+10, yoff+height*2+yoff/2))

	s.FireAffinity = NewSkillInput(g, &s.updatedSkills.FireAffinity, windowStart.Add(xoff*4+35, yoff+yoff+yoff/2))
	s.ElectricAffinity = NewSkillInput(g, &s.updatedSkills.ElectricAffinity, windowStart.Add(xoff*6+40, yoff+yoff+yoff/2))
	s.ClericAffinity = NewSkillInput(g, &s.updatedSkills.ClericAffinity, windowStart.Add(xoff*8+40, yoff+yoff+yoff/2))

	s.AssasinAffinity = NewSkillInput(g, &s.updatedSkills.AssasinAffinity, windowStart.Add(xoff*4+35, yoff+yoff*2+yoff+yoff/2))
	s.WarriorAffinity = NewSkillInput(g, &s.updatedSkills.WarriorAffinity, windowStart.Add(xoff*6+40, yoff+yoff*2+yoff+yoff/2))
	s.MartialArtAffinity = NewSkillInput(g, &s.updatedSkills.MartialArtAffinity, windowStart.Add(xoff*8+40, yoff+yoff*2+yoff+yoff/2))
	return s
}

func (b *Skills) Update() {
	if b.saveButton.Pressed() {
		//b.g.player.Exp.Skills = *b.updatedSkills
		b.g.outQueue <- &GameMsg{E: msgs.EUpdateSkills, Data: &b.updatedSkills}
		b.g.stats.skillsOpen = false
	}
	if b.zeroButton.Pressed() {
		total := 0
		total += int(b.updatedSkills.Agility) - int(b.g.player.Exp.ItemSkills.Agility)
		b.updatedSkills.Agility = 0 + b.g.player.Exp.ItemSkills.Agility

		total += int(b.updatedSkills.Intelligence) - int(b.g.player.Exp.ItemSkills.Intelligence)
		b.updatedSkills.Intelligence = 0 + b.g.player.Exp.ItemSkills.Intelligence

		total += int(b.updatedSkills.Vitality) - int(b.g.player.Exp.ItemSkills.Vitality)
		b.updatedSkills.Vitality = 0 + b.g.player.Exp.ItemSkills.Vitality

		total += int(b.updatedSkills.FireAffinity) - int(b.g.player.Exp.ItemSkills.FireAffinity)
		b.updatedSkills.FireAffinity = 0 + b.g.player.Exp.ItemSkills.FireAffinity

		total += int(b.updatedSkills.ElectricAffinity) - int(b.g.player.Exp.ItemSkills.ElectricAffinity)
		b.updatedSkills.ElectricAffinity = 0 + b.g.player.Exp.ItemSkills.ElectricAffinity

		total += int(b.updatedSkills.ClericAffinity) - int(b.g.player.Exp.ItemSkills.ClericAffinity)
		b.updatedSkills.ClericAffinity = 0 + b.g.player.Exp.ItemSkills.ClericAffinity

		total += int(b.updatedSkills.AssasinAffinity) - int(b.g.player.Exp.ItemSkills.AssasinAffinity)
		b.updatedSkills.AssasinAffinity = 0 + b.g.player.Exp.ItemSkills.AssasinAffinity

		total += int(b.updatedSkills.WarriorAffinity) - int(b.g.player.Exp.ItemSkills.WarriorAffinity)
		b.updatedSkills.WarriorAffinity = 0 + b.g.player.Exp.ItemSkills.WarriorAffinity

		total += int(b.updatedSkills.MartialArtAffinity) - int(b.g.player.Exp.ItemSkills.MartialArtAffinity)
		b.updatedSkills.MartialArtAffinity = 0 + b.g.player.Exp.ItemSkills.MartialArtAffinity
		if total > 0 {
			b.FreePoints = int(b.g.player.Exp.Skills.FreePoints) + total
		} else {
			//b.FreePoints = int(b.g.player.Exp.Skills.FreePoints)
			//log.Print("asd")
		}
	}
	b.Agility.Update()
	b.Intelligence.Update()
	b.Vitality.Update()

	b.FireAffinity.Update()
	b.ElectricAffinity.Update()
	b.ClericAffinity.Update()

	b.AssasinAffinity.Update()
	b.WarriorAffinity.Update()
	b.MartialArtAffinity.Update()
}
func (b *Skills) Draw(screen *ebiten.Image) {
	b.Background.Fill(color.RGBA{0, 0, 0, 170})
	b.saveButton.Draw(b.Background, 400, 10)
	b.zeroButton.Draw(b.Background, 442, 10)

	text.PrintBigAt(b.Background, "Free points", 200, 0)
	text.PrintBigAt(b.Background, fmt.Sprintf("%d", b.FreePoints), 240, 28)

	xoff := 60
	height := 90
	yoff := 60
	text.PrintBigAt(b.Background, "Agility", xoff-20, yoff)
	b.Agility.Draw(b.Background, xoff+10, yoff+yoff/2)
	text.PrintBigAt(b.Background, "Intelligence", xoff-44, yoff+height)
	b.Intelligence.Draw(b.Background, xoff+10, yoff+height+yoff/2)
	text.PrintBigAt(b.Background, "Vitality", xoff-20, yoff+height*2)
	b.Vitality.Draw(b.Background, xoff+10, yoff+height*2+yoff/2)

	text.PrintBigAt(b.Background, "Magic", xoff*3+xoff/6, yoff+yoff/2)
	//offY := 50
	text.PrintBigAt(b.Background, "Fire", xoff*4+20, yoff+yoff)
	b.FireAffinity.Draw(b.Background, xoff*4+35, yoff+yoff+yoff/2)
	text.PrintBigAt(b.Background, "Electric", xoff*6, yoff+yoff)
	b.ElectricAffinity.Draw(b.Background, xoff*6+40, yoff+yoff+yoff/2)
	text.PrintBigAt(b.Background, "Cleric", xoff*8+10, yoff+yoff)
	b.ClericAffinity.Draw(b.Background, xoff*8+40, yoff+yoff+yoff/2)

	text.PrintBigAt(b.Background, "Melee", xoff*3+xoff/6, yoff+yoff*2+yoff/2)
	text.PrintBigAt(b.Background, "Assasin", xoff*4, yoff+yoff*2+yoff)
	b.AssasinAffinity.Draw(b.Background, xoff*4+35, yoff+yoff*2+yoff+yoff/2)
	text.PrintBigAt(b.Background, "Warrior", xoff*6, yoff+yoff*2+yoff)
	b.WarriorAffinity.Draw(b.Background, xoff*6+40, yoff+yoff*2+yoff+yoff/2)
	text.PrintBigAt(b.Background, "MartialArt", xoff*8, yoff+yoff*2+yoff)
	b.MartialArtAffinity.Draw(b.Background, xoff*8+40, yoff+yoff*2+yoff+yoff/2)

	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(340, ScreenHeight-404)
	screen.DrawImage(b.Background, b.drawOp)
}

type Options struct {
	g          *Game
	drawOp     *ebiten.DrawImageOptions
	W, H       int32
	Background *ebiten.Image

	volumeSlider *Slider

	keyBinders []*NKeyBinder[*Input]
}

func NewOptions(g *Game) *Options {
	bg := ebiten.NewImage(300, 600)
	s := &Options{
		drawOp:       &ebiten.DrawImageOptions{},
		g:            g,
		W:            int32(bg.Bounds().Dx()),
		H:            int32(bg.Bounds().Dy()),
		volumeSlider: NewSlider(g),
		Background:   bg,
	}

	keyBindXStart := ScreenWidth - 210
	keyBindWidth := 200

	start := 150
	height := 38
	//meleeX := actionsBarStart
	meleeKeyBinder := NKeyBinderOpt[*Input, struct{}, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start, keyBindXStart+keyBindWidth, start+height),
		Active:     g.keys.cfg.Melee,
		actionMap:  map[*Input]struct{}{},
		pressedMap: map[*Input]bool{},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, meleeKeyBinder)

	resuKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height, keyBindXStart+keyBindWidth, start+height*2),
		Active:     g.keys.cfg.PickResurrect,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, resuKeyBinder)

	healKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*2, keyBindXStart+keyBindWidth, start+height*3),
		Active:     g.keys.cfg.PickHealWounds,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, healKeyBinder)

	remoKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*3, keyBindXStart+keyBindWidth, start+height*4),
		Active:     g.keys.cfg.PickParalizeRm,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, remoKeyBinder)

	paraKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*4, keyBindXStart+keyBindWidth, start+height*5),
		Active:     g.keys.cfg.PickParalize,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, paraKeyBinder)

	descaKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*5, keyBindXStart+keyBindWidth, start+height*6),
		Active:     g.keys.cfg.PickElectricDischarge,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, descaKeyBinder)

	apocaKeyBinder := NKeyBinderOpt[*Input, attack.Spell, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*6, keyBindXStart+keyBindWidth, start+height*7),
		Active:     g.keys.cfg.PickExplode,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, apocaKeyBinder)

	upKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*7, keyBindXStart+keyBindWidth, start+height*8),
		Active:     g.keys.cfg.Back,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, upKeyBinder)

	leftKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*8, keyBindXStart+keyBindWidth, start+height*9),
		Active:     g.keys.cfg.Left,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, leftKeyBinder)

	downKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*9, keyBindXStart+keyBindWidth, start+height*10),
		Active:     g.keys.cfg.Front,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, downKeyBinder)

	rightKeyBinder := NKeyBinderOpt[*Input, direction.D, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*10, keyBindXStart+keyBindWidth, start+height*11),
		Active:     g.keys.cfg.Right,
		actionMap:  g.keys.directionMap,
		pressedMap: g.keys.pressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, rightKeyBinder)

	redsKeyBinder := NKeyBinderOpt[*Input, item.Item, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*11, keyBindXStart+keyBindWidth, start+height*12),
		Active:     g.keys.cfg.PotionHP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, redsKeyBinder)

	bluesKeyBinder := NKeyBinderOpt[*Input, item.Item, bool]{
		g:          g,
		Rect:       image.Rect(keyBindXStart, start+height*12, keyBindXStart+keyBindWidth, start+height*13),
		Active:     g.keys.cfg.PotionMP,
		actionMap:  g.keys.potionMap,
		pressedMap: g.keys.potionPressed,
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, bluesKeyBinder)

	return s
}

func (b *Options) Update() {
	for _, kb := range b.keyBinders {
		kb.Update()
	}
	b.volumeSlider.Update()

}
func (b *Options) Draw(screen *ebiten.Image) {
	b.Background.Fill(color.Black)
	start := 18
	height := 38
	text.PrintBigAt(b.Background, "Vol", 18, start-2)
	text.PrintBigAt(b.Background, "Keys", 18, start+height)
	keyX := 24
	text.PrintBigAt(b.Background, "Piña", keyX, start+height*2)
	text.PrintBigAt(b.Background, "Resu", keyX, start+height*3)
	text.PrintBigAt(b.Background, "Cura", keyX, start+height*4)
	text.PrintBigAt(b.Background, "Remo", keyX, start+height*5)
	text.PrintBigAt(b.Background, "Para", keyX, start+height*6)
	text.PrintBigAt(b.Background, "Desca", keyX, start+height*7)
	text.PrintBigAt(b.Background, "Apoca", keyX, start+height*8)
	text.PrintBigAt(b.Background, "Up", keyX, start+height*9)
	text.PrintBigAt(b.Background, "Left", keyX, start+height*10)
	text.PrintBigAt(b.Background, "Down", keyX, start+height*11)
	text.PrintBigAt(b.Background, "Right", keyX, start+height*12)
	text.PrintBigAt(b.Background, "Rojas", keyX, start+height*13)
	text.PrintBigAt(b.Background, "Azules", keyX, start+height*14)

	inputX := keyX + 40
	b.volumeSlider.Draw(b.Background, inputX+8, start)

	b.drawOp.GeoM.Reset()
	b.drawOp.GeoM.Translate(ScreenWidth-300, ScreenHeight-664)

	screen.DrawImage(b.Background, b.drawOp)

	for _, kb := range b.keyBinders {
		kb.Draw(screen)
	}
}

type Hud struct {
	x, y float64
	g    *Game

	modeSwitchCooldown time.Duration
	lastSwitch         time.Time

	barOffsetStart, barOffsetEnd int
	hpBar, mpBar                 *ebiten.Image
	hpBarRect, mpBarRect         image.Rectangle

	hudBg *ebiten.Image

	//spellIconImgs    *ebiten.Image
	selectedSpellImg *ebiten.Image

	bluePotionImg, redPotionImg *ebiten.Image

	manaPotionSignalImg   *ebiten.Image
	healthPotionSignalImg *ebiten.Image
	potionAlpha           float64

	optionsOpen   bool
	options       *Options
	optionsButton *Button

	skillsOpen   bool
	skills       *Skills
	skillsButton *Button

	inventory *Inventory

	keyBinders []*KeyBinder[*Input]

	lastHudPotion     time.Time
	hudPotionCooldown time.Duration

	infoTooltipData string
	infoTooltipBg   *ebiten.Image
	infoTooltipSet  bool

	meleeCooldownInfo *Cooldown
}

func (h *Hud) ChangeSetTooltip(s string) {
	h.infoTooltipSet = true
	h.infoTooltipData = s
}

func (h *Hud) RefreshTooltip() {
	if !h.infoTooltipSet {
		h.infoTooltipData = ""
	}
	h.infoTooltipSet = false
}
func NewHud(g *Game) *Hud {
	btnImg := ebiten.NewImage(32, 32)
	btnImg.Fill(color.RGBA{101, 32, 133, 0})

	s := &Hud{
		g:                  g,
		lastSwitch:         time.Now(),
		modeSwitchCooldown: time.Millisecond * 700,
		barOffsetStart:     32,
		barOffsetEnd:       6,

		optionsOpen: false,

		options: NewOptions(g),

		skillsOpen: false,
		skills:     NewSkills(g),

		hpBar: texture.Decode(img.HpBar_png),
		mpBar: texture.Decode(img.MpBar_png),

		hudBg: texture.Decode(img.HudBg_png),
		//spellIconImgs:    texture.Decode(img.SpellbarIcons2_png),
		selectedSpellImg: texture.Decode(img.SpellSelector_png),
		bluePotionImg:    texture.Decode(itemimg.ManaPotion_png),
		redPotionImg:     texture.Decode(itemimg.HealthPotion_png),

		manaPotionSignalImg:   ebiten.NewImage(32, 32),
		healthPotionSignalImg: ebiten.NewImage(32, 32),
		potionAlpha:           0,

		inventory: NewInventory(g),

		lastHudPotion:     time.Now(),
		hudPotionCooldown: time.Millisecond * 250,

		infoTooltipData: "",
		infoTooltipBg:   ebiten.NewImage(210, 160),
	}
	s.infoTooltipBg.Fill(color.RGBA{43, 33, 74, 255})
	s.x = 0
	s.y = float64(ScreenHeight - s.hudBg.Bounds().Dy())
	s.optionsButton = NewButton(g, texture.Decode(img.ConfigIcon_png), btnImg, typ.P{X: ScreenWidth - 64, Y: int32(s.y) + 16})
	s.skillsButton = NewButton(g, texture.Decode(img.Icon_png), btnImg, typ.P{X: ScreenWidth - 128, Y: int32(s.y) + 16})

	cooldownBarImg := texture.Decode(img.CooldownBase_png)
	s.meleeCooldownInfo = &Cooldown{
		g:          g,
		BaseImg:    cooldownBarImg,
		Weapon:     item.None,
		Last:       &g.keys.LastMelee,
		GlobalLast: &g.keys.LastAction,
	}

	actionsBarStart := 400
	meleeX := actionsBarStart
	meleeX = meleeX + SpellIconWidth
	meleeX = meleeX + SpellIconWidth
	meleeX = meleeX + SpellIconWidth

	iconX := meleeX + SpellIconWidth*2
	resurrectSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Resu",
		IconImg:    texture.Decode(spellimg.IconResurrect_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickResurrect,
		Spell:      attack.SpellResurrect,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellResurrect,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellResurrect],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, resurrectSpellKeyBinder)

	iconX += SpellIconWidth
	healSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Cura",
		IconImg:    texture.Decode(spellimg.IconHeal_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickHealWounds,
		Spell:      attack.SpellHealWounds,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellHealWounds,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellHealWounds],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, healSpellKeyBinder)

	iconX += SpellIconWidth
	rmParalizeSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Remo",
		IconImg:    texture.Decode(spellimg.IconRmParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalizeRm,
		Spell:      attack.SpellRemoveParalize,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellRemoveParalize,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellRemoveParalize],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, rmParalizeSpellKeyBinder)

	iconX += SpellIconWidth
	paralizeSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Para",
		IconImg:    texture.Decode(spellimg.IconParalize_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickParalize,
		Spell:      attack.SpellParalize,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellParalize,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellParalize],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, paralizeSpellKeyBinder)

	iconX += SpellIconWidth
	electricDischargeSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Desca",
		IconImg:    texture.Decode(spellimg.IconElectricDischarge_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickElectricDischarge,
		Spell:      attack.SpellElectricDischarge,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellElectricDischarge,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellElectricDischarge],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, electricDischargeSpellKeyBinder)

	iconX += SpellIconWidth
	explodeSpellKeyBinder := KeyBinderOpt[*Input, attack.Spell, bool]{
		Desc:       "Apoca",
		IconImg:    texture.Decode(spellimg.IconExplode_png),
		Rect:       image.Rect(iconX, int(s.y), iconX+SpellIconWidth, ScreenHeight),
		Active:     g.keys.cfg.PickExplode,
		Spell:      attack.SpellExplode,
		Exp:        &g.player.Exp,
		actionMap:  g.keys.spellMap,
		pressedMap: g.keys.spellPressed,
		CooldownInfo: &Cooldown{
			g:          g,
			Spell:      attack.SpellExplode,
			BaseImg:    cooldownBarImg,
			Last:       &g.keys.LastSpells[attack.SpellExplode],
			GlobalLast: &g.keys.LastAction,
		},
	}.NewKeyBinder(EmptyInput())
	s.keyBinders = append(s.keyBinders, explodeSpellKeyBinder)

	for _, kb := range s.keyBinders {
		kb.g = g
	}

	s.healthPotionSignalImg.Fill(RedAlpha(uint8(60)))
	s.manaPotionSignalImg.Fill(BlueAlpha(uint8(90)))

	return s
}
func (s *Hud) SetCooldowns() {
}
func RedAlpha(a uint8) color.Color {
	return color.RGBA{168, 0, 16, a}
}
func BlueAlpha(a uint8) color.Color {
	return color.RGBA{0, 18, 174, a}
}

func (s *Hud) Update() {
	s.inventory.Update()
	for _, kb := range s.keyBinders {
		kb.Update()
	}
	s.meleeCooldownInfo.UpdateImage()
	if s.potionAlpha > 0 {
		s.potionAlpha -= .04
	}

	hp := mapValue(float64(s.g.client.HP), 0, float64(s.g.player.Exp.MaxHp), float64(s.barOffsetStart), float64(s.hpBar.Bounds().Max.X-s.barOffsetEnd))
	s.hpBarRect = image.Rect(s.hpBar.Bounds().Min.X, s.hpBar.Bounds().Min.Y, int(hp), s.hpBar.Bounds().Max.Y)

	mp := mapValue(float64(s.g.client.MP), 0, float64(s.g.player.Exp.MaxMp), float64(s.barOffsetStart), float64(s.mpBar.Bounds().Max.X-s.barOffsetEnd))
	s.mpBarRect = image.Rect(s.mpBar.Bounds().Min.X, s.mpBar.Bounds().Min.Y, int(mp), s.mpBar.Bounds().Max.Y)
	if s.optionsButton.Pressed() {
		s.optionsOpen = !s.optionsOpen
		if s.skillsOpen {
			s.skillsOpen = false
		}
	}
	if s.optionsOpen {
		s.options.Update()
		if (s.g.mouseX < ScreenWidth-300 || s.g.mouseY < ScreenHeight-664) &&
			ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			s.optionsOpen = false
		}
	}

	if s.skillsButton.Pressed() {
		if s.skillsOpen {
			s.skills.FreePoints = int(s.g.player.Exp.Skills.FreePoints)
			s.skills.updatedSkills = s.g.player.Exp.Skills
		} else {
			s.skills.FreePoints = int(s.g.player.Exp.Skills.FreePoints)
			s.skills.updatedSkills = s.g.player.Exp.Skills
			//s.g.player.Exp.Skills.FreePoints = uint16(s.skills.FreePoints)
		}
		s.skillsOpen = !s.skillsOpen
		if s.optionsOpen {
			s.optionsOpen = false
		}
	}
	if s.skillsOpen {
		s.skills.Update()
		if (s.g.mouseX < 340 || s.g.mouseY < ScreenHeight-404 || s.g.mouseX > 940) &&
			ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			s.skillsOpen = false
			s.skills.FreePoints = int(s.g.player.Exp.Skills.FreePoints)
			s.skills.updatedSkills = s.g.player.Exp.Skills
		}
	}
	s.RefreshTooltip()
}

func (s *Hud) Draw(screen *ebiten.Image) {

	if s.infoTooltipData != "" {
		y := strings.Count(s.infoTooltipData, "\n")
		spl := strings.SplitN(s.infoTooltipData, "\n", 2)
		y += 2
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(ScreenHeight-64-(y*18)))
		screen.DrawImage(s.infoTooltipBg, op)
		y--
		text.PrintBigAt(screen, spl[0], 20, (ScreenHeight - 64 - (y * 22)))
		y--
		text.PrintAt(screen, spl[1], 20, (ScreenHeight - 64 - (y * 18)))
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.hudBg, op)
	s.optionsButton.Draw(screen, ScreenWidth-64, int(s.y)+16)
	s.skillsButton.Draw(screen, ScreenWidth-128, int(s.y)+16)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(s.x, s.y)
	screen.DrawImage(s.hpBar.SubImage(s.hpBarRect).(*ebiten.Image), op)
	screen.DrawImage(s.mpBar.SubImage(s.mpBarRect).(*ebiten.Image), op)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v/%v", s.g.client.HP, s.g.player.Exp.MaxHp), int(s.x)+230, int(s.y)+10)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v/%v", s.g.client.MP, s.g.player.Exp.MaxMp), int(s.x)+230, int(s.y)+38)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.X), int(s.x)+7, int(s.y)+12)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%v", s.g.player.Y), int(s.x)+7, int(s.y)+35)

	s.ShowSpellPicker(screen)
	// if s.potionAlpha > 0 {
	// 	if s.g.lastPotionUsed == item.ManaPotion {
	// 		op := &ebiten.DrawImageOptions{}
	// 		op.GeoM.Translate(s.x+304, s.y+32)
	// 		op.ColorScale.ScaleAlpha(float32(s.potionAlpha))
	// 		screen.DrawImage(s.manaPotionSignalImg, op)
	// 	} else if s.g.lastPotionUsed == item.HealthPotion {
	// 		op := &ebiten.DrawImageOptions{}
	// 		op.GeoM.Translate(s.x+304, s.y)
	// 		op.ColorScale.ScaleAlpha(float32(s.potionAlpha))
	// 		screen.DrawImage(s.healthPotionSignalImg, op)
	// 	}
	// }
	// op = &ebiten.DrawImageOptions{}
	// op.GeoM.Translate(s.x+304, s.y+1)
	// screen.DrawImage(s.redPotionImg, op)
	// op = &ebiten.DrawImageOptions{}
	// op.GeoM.Translate(s.x+304, s.y+31)
	// screen.DrawImage(s.bluePotionImg, op)

	text.PrintAt(screen, fmt.Sprintf("Action\n%v", s.g.player.Exp.ActionCooldown.String()), int(s.x+990), int(s.y+16))

	for _, kb := range s.keyBinders {
		kb.Draw(screen)
	}
	for _, kb := range s.keyBinders {
		kb.DrawTooltips(screen, s.g.mouseX, s.g.mouseY)
	}
	if s.optionsOpen {
		s.options.Draw(screen)
	}
	if s.skillsOpen {
		s.skills.Draw(screen)
	}
	s.inventory.Draw(screen)

}

const (
	SpellIconWidth = 50
)

func (s *Hud) ShowSpellPicker(screen *ebiten.Image) {
	opselect := &ebiten.DrawImageOptions{}
	weaponsX, wY := float64(400), s.y
	s.meleeCooldownInfo.Draw(screen, int(weaponsX)+SpellIconWidth*4, int(s.y))
	w := s.g.player.Inv.GetWeapon()
	if w != item.None {
		opselect.GeoM.Translate(float64(weaponsX)+SpellIconWidth*4+10, float64(wY)+16)
		screen.DrawImage(s.inventory.items[w], opselect)
	}
	//op.GeoM.Translate(weaponsX, weaponsY)
	// switch s.g.player.Exp.SelectedWeapon {
	// case item.WeaponMightySword:
	// 	opselect.GeoM.Translate(float64(weaponsX), float64(weaponsY))
	// 	screen.DrawImage(s.selectedSpellImg, opselect)
	// case item.WeaponDarkDagger:
	// 	opselect.GeoM.Translate(float64(weaponsX+SpellIconWidth), float64(weaponsY))
	// 	screen.DrawImage(s.selectedSpellImg, opselect)
	// case item.WeaponFireStaff:
	// 	opselect.GeoM.Translate(float64(weaponsX+SpellIconWidth*2), float64(weaponsY))
	// 	screen.DrawImage(s.selectedSpellImg, opselect)
	// case item.WeaponGoldAxe:
	// 	opselect.GeoM.Translate(float64(weaponsX+SpellIconWidth*3), float64(weaponsY))
	// 	screen.DrawImage(s.selectedSpellImg, opselect)
	// }
	spellsX, spellsY := weaponsX+SpellIconWidth*5, s.y
	opselect.GeoM.Reset()
	switch s.g.player.Exp.SelectedSpell {
	case attack.SpellResurrect:
		opselect.GeoM.Translate(float64(spellsX), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case attack.SpellHealWounds:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case attack.SpellRemoveParalize:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*2), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case attack.SpellParalize:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*3), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case attack.SpellElectricDischarge:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*4), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	case attack.SpellExplode:
		opselect.GeoM.Translate(float64(spellsX+SpellIconWidth*5), float64(spellsY))
		screen.DrawImage(s.selectedSpellImg, opselect)
	}
}

func mapValue(v, start1, stop1, start2, stop2 float64) float64 {
	newval := (v-start1)/(stop1-start1)*(stop2-start2) + start2
	if start2 < stop2 {
		if newval > stop2 {
			newval = stop2
		} else if newval < start2 {
			newval = start2
		}
	} else {
		if newval > start2 {
			newval = start2
		} else if newval < stop2 {
			newval = stop2
		}
	}
	return newval
}
func mapIntValue(v, start1, stop1, start2, stop2 int64) int64 {
	val := (stop1-start1)*(stop2-start2) + start2
	if val == 0 {
		return (v - start1)
	}
	newval := (v - start1) / val
	if start2 < stop2 {
		if newval > stop2 {
			newval = stop2
		} else if newval < start2 {
			newval = start2
		}
	} else {
		if newval > start2 {
			newval = start2
		} else if newval < stop2 {
			newval = stop2
		}
	}
	return newval
}

type KeyBinder[A KBind] struct {
	g            *Game
	counter      int
	Img          *ebiten.Image
	IconImg      *ebiten.Image
	CooldownInfo *Cooldown
	Over         bool
	Rect         image.Rectangle
	Active       A
	Spell        attack.Spell
	Weapon       item.Item
	Exp          *msgs.Experience
	Item         item.Item
	Desc         string
	Change       func(old, new A)
	Open         bool
	Clicked      bool
	Selected     A
}

type Cooldown struct {
	g          *Game
	Spell      attack.Spell
	Weapon     item.Item
	BaseImg    *ebiten.Image
	Img        *ebiten.Image
	Last       *time.Time
	GlobalLast *time.Time
}

const (
	iconX = 50
	iconY = 64
)

func (cd *Cooldown) UpdateImage() {
	if cd == nil {
		return
	}
	var val time.Duration
	if cd.Spell != attack.SpellNone {
		val = cd.g.player.Exp.Spells[cd.Spell].Cooldown
	} else if cd.Weapon != item.None {
		val = cd.g.player.Exp.Items[cd.Weapon].WeaponData.Cooldown
	} else if cd.Weapon == item.None {
		val = cd.g.player.Exp.ActionCooldown
	}
	now := time.Now()
	dt := now.Sub(*cd.Last)
	v := mapValue(float64(dt), float64(val), 0, 0, iconY)
	gdt := now.Sub(*cd.GlobalLast)
	gv := mapValue(float64(gdt), float64(cd.g.player.Exp.ActionCooldown), 0, 0, iconY)

	if gv > (v) {
		if gv < 1 {
			cd.Img = nil
			return
		}
		cd.Img = cd.BaseImg.SubImage(image.Rect(0, 0, iconX, int(gv))).(*ebiten.Image)
	} else {
		if v < 1 {
			cd.Img = nil
			return
		}
		//log.Printf("%v %v", iconX, v)
		cd.Img = cd.BaseImg.SubImage(image.Rect(0, 0, iconX, int(v))).(*ebiten.Image)
	}
}

func (cd *Cooldown) Draw(screen *ebiten.Image, x, y int) {
	if cd == nil {
		return
	}
	if cd.Img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	//ny := iconY - cd.Img.Bounds().Dy()
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(cd.Img, op)
}

type KeyBinderOpt[A KBind, B any, C any] struct {
	Spell  attack.Spell
	Exp    *msgs.Experience
	Weapon item.Item

	Item         item.Item
	Desc         string
	CooldownInfo *Cooldown
	IconImg      *ebiten.Image
	Rect         image.Rectangle
	Active       A
	actionMap    map[A]B
	pressedMap   map[A]C
}

func (opt KeyBinderOpt[A, B, C]) NewKeyBinder(selected A) *KeyBinder[A] {
	kb := &KeyBinder[A]{
		Spell:        opt.Spell,
		Weapon:       opt.Weapon,
		Exp:          opt.Exp,
		Item:         opt.Item,
		IconImg:      opt.IconImg,
		Desc:         opt.Desc,
		Active:       opt.Active,
		Selected:     selected,
		Rect:         opt.Rect,
		CooldownInfo: opt.CooldownInfo,
		Change: func(old, new A) {
			Replace(opt.actionMap, new, old)
			Replace(opt.pressedMap, new, old)
		},
		Img: ebiten.NewImage(opt.Rect.Dx(), opt.Rect.Dy()),
	}
	kb.Img.Fill(color.RGBA{60, 60, 60, 200})
	return kb
}

func (kb *KeyBinder[A]) Mouse() {
	x, y := kb.g.mouseX, kb.g.mouseY
	if x > kb.Rect.Min.X && x < kb.Rect.Max.X && y > kb.Rect.Min.Y && y < kb.Rect.Max.Y {
		kb.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Spell != attack.SpellNone && kb.Spell != kb.g.player.Exp.SelectedSpell {
					kb.g.player.Exp.SelectedSpell = kb.Spell
					kb.g.outQueue <- &GameMsg{E: msgs.ESelectSpell, Data: kb.Spell}
				} else if kb.Item != item.None {
					if time.Since(kb.g.stats.lastHudPotion) > kb.g.stats.hudPotionCooldown {
						kb.g.outQueue <- &GameMsg{E: msgs.EUseItem, Data: kb.Item}
						kb.g.stats.lastHudPotion = time.Now()
					}
				} else if kb.Weapon != item.None && kb.Weapon != kb.g.player.Exp.SelectedWeapon {
					// kb.g.player.Exp.SelectedWeapon = kb.Weapon
					// kb.g.outQueue <- &GameMsg{E: msgs.ESelectWeapon, Data: kb.Weapon}
					// kb.g.player.Weapon = texture.LoadAnimation(texture.AssetFromWeaponItem(kb.Weapon))
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else {
		kb.Over = false
	}
}

func (kb *KeyBinder[A]) Update() {
	if kb.CooldownInfo != nil {
		kb.CooldownInfo.UpdateImage()
	}
	kb.Mouse()
	kb.counter++
}

func (kb *KeyBinder[A]) Draw(screen *ebiten.Image) {
	kb.CooldownInfo.Draw(screen, kb.Rect.Min.X, kb.Rect.Min.Y)
	if kb.IconImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		screen.DrawImage(kb.IconImg, op)
	}
}

func (kb *KeyBinder[A]) DrawTooltips(screen *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	if x > ScreenWidth-300 {
		x -= 180
	}
	if kb.Over {
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		op.ColorScale.ScaleAlpha(0.3)
		screen.DrawImage(kb.Img, op)
		//text.PrintAtBg(screen, kb.Active.String(), x+16, y-48)
		//text.PrintAt(screen, kb.Desc, x+15, y-30)
		if kb.Spell != attack.SpellNone {
			kb.g.stats.ChangeSetTooltip(fmt.Sprintf("%v\n%v\nDamage: %d\nMana: %d",
				kb.Desc,
				roundDuration(kb.Exp.Spells[kb.Spell].Cooldown, 1).String(),
				kb.Exp.Spells[kb.Spell].Damage,
				kb.Exp.Spells[kb.Spell].ManaCost))
			// text.PrintAt(screen, kb.Exp.Spells[kb.Spell].Cooldown.String(), x+15, y-15)
			// text.PrintAt(screen, fmt.Sprintf("Damage: %d", kb.Exp.Spells[kb.Spell].Damage), x+15, y)
			// text.PrintAt(screen, fmt.Sprintf("Mana: %d", kb.Exp.Spells[kb.Spell].ManaCost), x+15, y+15)
		}
		//  else if kb.Weapon != item.None {
		// 	text.PrintAt(screen, kb.Exp.Weapons[kb.Weapon].Cooldown.String(), x+15, y-15)
		// 	text.PrintAt(screen, fmt.Sprintf("Damage: %d", kb.Exp.Weapons[kb.Weapon].Damage), x+15, y)
		// 	text.PrintAt(screen, fmt.Sprintf("Crit range: %d", kb.Exp.Weapons[kb.Weapon].CriticRange), x+15, y+15)
		// }
	}
}

var divs = []time.Duration{
	time.Duration(1), time.Duration(10), time.Duration(100), time.Duration(1000)}

func roundDuration(d time.Duration, digits int) time.Duration {
	switch {
	case d > time.Second:
		d = d.Round(time.Second / divs[digits])
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / divs[digits])
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / divs[digits])
	}
	return d
}

type NKeyBinder[A KBind] struct {
	g        *Game
	counter  int
	Img      *ebiten.Image
	Over     bool
	Rect     image.Rectangle
	Active   A
	Change   func(old, new A)
	Open     bool
	Clicked  bool
	Selected A
}

type NKeyBinderOpt[A KBind, B any, C any] struct {
	g          *Game
	Rect       image.Rectangle
	Active     A
	actionMap  map[A]B
	pressedMap map[A]C
}

func (opt NKeyBinderOpt[A, B, C]) NewKeyBinder(selected A) *NKeyBinder[A] {
	kb := &NKeyBinder[A]{
		g:        opt.g,
		Active:   opt.Active,
		Selected: selected,
		Rect:     opt.Rect,
		Change: func(old, new A) {
			Replace(opt.actionMap, new, old)
			Replace(opt.pressedMap, new, old)
		},
		Img: ebiten.NewImage(opt.Rect.Dx(), opt.Rect.Dy()),
	}
	kb.Img.Fill(color.RGBA{60, 60, 60, 200})
	return kb
}

func (kb *NKeyBinder[A]) Mouse() {
	x, y := kb.g.mouseX, kb.g.mouseY
	if x > kb.Rect.Min.X && x < kb.Rect.Max.X && y > kb.Rect.Min.Y && y < kb.Rect.Max.Y {
		kb.Over = true
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Open {
					if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
						kb.Active.Set(kb.Selected.V())
					}
					kb.Selected.Set(NoInput)
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else if kb.Open {
		kb.Over = false
		if ebiten.IsMouseButtonPressed(ebiten.MouseButton0) {
			if !kb.Clicked {
				kb.Clicked = true
				if kb.Open {
					if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
						kb.Active.Set(kb.Selected.V())
					}
					kb.Selected.Set(NoInput)
				}
				kb.Open = !kb.Open
			}
		} else {
			kb.Clicked = false
		}
	} else {
		kb.Over = false
	}
}

func (kb *NKeyBinder[A]) GetInput() {
	input := NoInput
	keys := inpututil.AppendPressedKeys([]ebiten.Key{})
	if len(keys) != 0 {
		for i, key := range keys {
			if key != ebiten.KeyEnter {
				input = NewInput(keys[i])
				break
			}
		}
	}
	rmb := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if rmb {
		input = NewInput(ebiten.MouseButtonRight)
	}
	mmb := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	if mmb {
		input = NewInput(ebiten.MouseButtonMiddle)
	}
	mb3 := ebiten.IsMouseButtonPressed(ebiten.MouseButton3)
	if mb3 {
		input = NewInput(ebiten.MouseButton3)
	}
	mb4 := ebiten.IsMouseButtonPressed(ebiten.MouseButton4)
	if mb4 {
		input = NewInput(ebiten.MouseButton4)
	}
	if !input.Empty() {
		kb.Selected.Set(input)
	}
	if ebiten.IsKeyPressed(ebiten.KeyEnter) {
		if kb.Selected.V() != kb.Active.V() && !kb.Selected.Empty() {
			kb.Active.Set(kb.Selected.V())
		}
		kb.Open = false
		kb.Selected.Set(NoInput)
	}
}

func (kb *NKeyBinder[A]) Update() {
	kb.Mouse()
	if kb.Open {
		kb.GetInput()
	}
	kb.counter++
}

func (kb *NKeyBinder[A]) Draw(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	if kb.Over && !kb.Open {
		op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
		op.ColorScale.ScaleAlpha(0.3)
		screen.DrawImage(kb.Img, op)
	}
	if !kb.Open {
		text.PrintBigAtBg(screen, kb.Active.String(), kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
		return
	}
	op.GeoM.Translate(float64(kb.Rect.Min.X), float64(kb.Rect.Min.Y))
	screen.DrawImage(kb.Img, op)
	if !kb.Selected.Empty() {
		text.PrintBigAtBg(screen, kb.Selected.String(), kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
	} else {
		str := " "
		if kb.counter%60 < 30 {
			str = "_"
		}
		text.PrintBigAtBg(screen, str, kb.Rect.Min.X+20, kb.Rect.Min.Y+1)
	}
}

func Replace[A comparable, B any, T map[A]B](m T, new, old A) {
	if m == nil {
		return
	}
	m[new] = m[old]
	delete(m, old)
}
