package server

import (
	"time"

	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/msgs"
)

type Experience struct {
	p *Player
	// Player stats
	MaxHp, MaxMp   int32
	ActionCooldown Cooldown

	// Things a player has to emit damage or healing
	// A container and one selected
	BaseMeleeDamage    int32
	BaseMeleeCritRange int32
	BaseMeleeCD        time.Duration

	SelectedSpell attack.Spell
	spells        [attack.SpellLen]SpellProp
	items         map[item.Item]ItemProp
	// Player progression stats
	//
	FreePoints uint16

	Agility              Agility
	ItemBuffAgility      Agility
	Vitality             Vitality
	ItemBuffVitality     Vitality
	Intelligence         Intelligence
	ItemBuffIntelligence Intelligence
	//
	Magic         ExpMagic
	ItemBuffMagic ExpMagic
	//
	Melee         ExpMelee
	ItemBuffMelee ExpMelee
}

func NewExperience(p *Player) *Experience {
	return &Experience{
		p:                  p,
		BaseMeleeDamage:    30,
		BaseMeleeCritRange: 5,
		BaseMeleeCD:        time.Millisecond * 1000,
		items:              map[item.Item]ItemProp{},
		FreePoints:         20,
	}
}

func (e *Experience) SetItemBuffs(it item.Item) {
	itemProp := GetItemProp(it)
	switch it.Type() {
	case item.TypeWeapon:
		e.Magic.Affs[itemProp.WeaponProp.MagicAff] += float32(itemProp.WeaponProp.MagicAffN)
		e.Melee.Affs[itemProp.WeaponProp.MeleeAff] += float32(itemProp.WeaponProp.MeleeAffN)
		e.ItemBuffMagic.Affs[itemProp.WeaponProp.MagicAff] += float32(itemProp.WeaponProp.MagicAffN)
		e.ItemBuffMelee.Affs[itemProp.WeaponProp.MeleeAff] += float32(itemProp.WeaponProp.MeleeAffN)
	}
}
func (e *Experience) UnsetItemBuffs(it item.Item) {
	itemProp := GetItemProp(it)
	switch it.Type() {
	case item.TypeWeapon:
		e.Magic.Affs[itemProp.WeaponProp.MagicAff] -= float32(itemProp.WeaponProp.MagicAffN)
		e.Melee.Affs[itemProp.WeaponProp.MeleeAff] -= float32(itemProp.WeaponProp.MeleeAffN)
		e.ItemBuffMagic.Affs[itemProp.WeaponProp.MagicAff] -= float32(itemProp.WeaponProp.MagicAffN)
		e.ItemBuffMelee.Affs[itemProp.WeaponProp.MeleeAff] -= float32(itemProp.WeaponProp.MeleeAffN)
	}
}

func (e *Experience) SelectSpell(s attack.Spell) {
	e.SelectedSpell = s
}

func (e *Experience) CalcWeaponProp(it item.Item) WeaponProp {
	prop := GetWeaponProp(it)

	cd := float32(prop.BaseCooldown)
	prop.ExpWeapon.CD.CD = prop.BaseCooldown - time.Duration((e.Agility.CalcMelee(cd) + e.Melee.CalcCD(prop.MeleeAff, cd)))

	prop.ExpWeapon.CritRange = e.BaseMeleeCritRange + int32(e.p.exp.Melee.CalcCritical(prop.MeleeAff, float32(prop.BaseCritRange)))

	prop.ExpWeapon.Damage = e.BaseMeleeDamage + int32(e.p.exp.Melee.CalcDamage(prop.MeleeAff, float32(prop.BaseDamage)))

	return prop
}

func (e *Experience) ApplyPlayer() {
	e.MaxHp = int32(BaseHp + e.Vitality.CalcHealth(BaseHp))
	e.MaxMp = int32(BaseMp + e.Intelligence.CalcMana(BaseMp))

	e.ActionCooldown = Cooldown{CD: ActionCD - time.Duration(e.Agility.CalcAction(float32(ActionCD)))}

	for s := range attack.SpellLen {
		sp := attack.Spell(s)
		prop := *GetSpellProp(sp)

		cd := float32(prop.BaseCooldown)
		prop.ExpSpell.CD.CD = prop.BaseCooldown - time.Duration((e.Agility.CalcSpell(cd) + e.Magic.CalcCD(prop.MagicAff, cd)))

		prop.ExpSpell.ManaCost = prop.BaseManaCost - int32(e.p.exp.Magic.CalcMana(prop.MagicAff, float32(prop.BaseManaCost)))

		prop.ExpSpell.Damage = prop.BaseDamage + int32(e.p.exp.Magic.CalcDamage(prop.MagicAff, float32(prop.BaseDamage)))

		e.spells[sp] = prop
	}
	// // apply to items in inventory
	e.p.inv.Range(func(i int, it *msgs.ItemSlot) bool {
		itemProp := GetItemProp(it.Item)

		switch itemProp.Type.Type() {

		case item.TypeWeapon:
			prop := itemProp.WeaponProp
			cd := float32(prop.BaseCooldown)
			prop.ExpWeapon.CD.CD = prop.BaseCooldown - time.Duration((e.Agility.CalcMelee(cd) + e.Melee.CalcCD(prop.MeleeAff, cd)))

			prop.ExpWeapon.CritRange = prop.BaseCritRange + int32(e.p.exp.Melee.CalcCritical(prop.MeleeAff, float32(prop.BaseCritRange)))

			prop.ExpWeapon.Damage = prop.BaseDamage + int32(e.p.exp.Melee.CalcDamage(prop.MeleeAff, float32(prop.BaseDamage)))

		case item.TypeArmor:

		case item.TypeShield:
		case item.TypeHelmet:
		}
		e.items[it.Item] = itemProp
		return true
	})

}

func (e *Experience) ApplySkills(s msgs.Skills) {
	total := 0
	total += int(s.Agility) - int(e.Agility)
	e.Agility = Agility(s.Agility)

	total += int(s.Intelligence) - int(e.Intelligence)
	e.Intelligence = Intelligence(s.Intelligence)

	total += int(s.Vitality) - int(e.Vitality)
	e.Vitality = Vitality(s.Vitality)

	total += int(s.FireAffinity) - int(e.Magic.Affs[MagicAffinityTypeFire])
	e.Magic.Affs[MagicAffinityTypeFire] = float32(s.FireAffinity)

	total += int(s.ElectricAffinity) - int(e.Magic.Affs[MagicAffinityTypeElectric])
	e.Magic.Affs[MagicAffinityTypeElectric] = float32(s.ElectricAffinity)

	total += int(s.ClericAffinity) - int(e.Magic.Affs[MagicAffinityTypeCleric])
	e.Magic.Affs[MagicAffinityTypeCleric] = float32(s.ClericAffinity)

	total += int(s.AssasinAffinity) - int(e.Melee.Affs[MeleeAffinityTypeAssasin])
	e.Melee.Affs[MeleeAffinityTypeAssasin] = float32(s.AssasinAffinity)

	total += int(s.WarriorAffinity) - int(e.Melee.Affs[MeleeAffinityTypeWarrior])
	e.Melee.Affs[MeleeAffinityTypeWarrior] = float32(s.WarriorAffinity)

	total += int(s.MartialArtAffinity) - int(e.Melee.Affs[MeleeAffinityTypeMartialArt])
	e.Melee.Affs[MeleeAffinityTypeMartialArt] = float32(s.MartialArtAffinity)

	e.FreePoints -= uint16(total)
}

func (e *Experience) ToMsgs() msgs.Experience {
	exp := msgs.Experience{
		MaxHp:          e.MaxHp,
		MaxMp:          e.MaxMp,
		ActionCooldown: e.ActionCooldown.CD,
		SelectedSpell:  e.SelectedSpell,
		Skills: msgs.Skills{
			FreePoints:         e.FreePoints,
			Agility:            uint16(e.Agility),
			Intelligence:       uint16(e.Intelligence),
			Vitality:           uint16(e.Vitality),
			FireAffinity:       uint16(e.Magic.Affs[MagicAffinityTypeFire]),
			ElectricAffinity:   uint16(e.Magic.Affs[MagicAffinityTypeElectric]),
			ClericAffinity:     uint16(e.Magic.Affs[MagicAffinityTypeCleric]),
			WarriorAffinity:    uint16(e.Melee.Affs[MeleeAffinityTypeWarrior]),
			AssasinAffinity:    uint16(e.Melee.Affs[MeleeAffinityTypeAssasin]),
			MartialArtAffinity: uint16(e.Melee.Affs[MeleeAffinityTypeMartialArt]),
		},
		ItemSkills: msgs.Skills{
			FreePoints:         e.FreePoints,
			Agility:            uint16(e.ItemBuffAgility),
			Intelligence:       uint16(e.ItemBuffIntelligence),
			Vitality:           uint16(e.ItemBuffVitality),
			FireAffinity:       uint16(e.ItemBuffMagic.Affs[MagicAffinityTypeFire]),
			ElectricAffinity:   uint16(e.ItemBuffMagic.Affs[MagicAffinityTypeElectric]),
			ClericAffinity:     uint16(e.ItemBuffMagic.Affs[MagicAffinityTypeCleric]),
			WarriorAffinity:    uint16(e.ItemBuffMelee.Affs[MeleeAffinityTypeWarrior]),
			AssasinAffinity:    uint16(e.ItemBuffMelee.Affs[MeleeAffinityTypeAssasin]),
			MartialArtAffinity: uint16(e.ItemBuffMelee.Affs[MeleeAffinityTypeMartialArt]),
		},
		Items: map[item.Item]msgs.ItemData{},
	}
	for s := range attack.SpellLen {
		exp.Spells[s] = msgs.SpellData{
			Damage:   e.spells[s].ExpSpell.Damage,
			ManaCost: e.spells[s].ExpSpell.ManaCost,
			Cooldown: e.spells[s].ExpSpell.CD.CD,
		}
	}
	e.p.inv.Range(func(i int, it *msgs.ItemSlot) bool {
		itemProp := GetItemProp(it.Item)
		switch itemProp.Type.Type() {
		case item.TypeConsumable:
			exp.Items[it.Item] = msgs.ItemData{Item: it.Item}
		case item.TypeWeapon:
			exp.Items[it.Item] = msgs.ItemData{
				Item: it.Item,
				WeaponData: msgs.WeaponData{
					Damage:      e.items[it.Item].WeaponProp.ExpWeapon.Damage,
					CriticRange: e.items[it.Item].WeaponProp.ExpWeapon.CritRange,
					Cooldown:    e.items[it.Item].WeaponProp.ExpWeapon.CD.CD,
				}}

		case item.TypeArmor:
			exp.Items[it.Item] = msgs.ItemData{
				Item:      it.Item,
				ArmorData: msgs.ArmorData{}}
		case item.TypeShield:
			exp.Items[it.Item] = msgs.ItemData{
				Item:       it.Item,
				ShieldData: msgs.ShieldData{}}
		case item.TypeHelmet:
			exp.Items[it.Item] = msgs.ItemData{
				Item:       it.Item,
				HelmetData: msgs.HelmetData{}}

		}
		return true
	})

	return exp
}

type Agility float32

const ActionCD = time.Millisecond * 600

var actionCDreducer Agility = 0.012
var spellCDreducer Agility = 0.001
var meleeCDreducer Agility = 0.001

func (agility Agility) CalcAction(v float32) float32 {
	return float32(agility*actionCDreducer) * v
}
func (agility Agility) CalcSpell(v float32) float32 {
	return float32(agility*spellCDreducer) * v
}
func (agility Agility) CalcMelee(v float32) float32 {
	return float32(agility*meleeCDreducer) * v
}

type Intelligence float32

var manaReducer Intelligence = 0.20

func (intel Intelligence) CalcMana(v float32) float32 {
	return float32(intel*manaReducer) * v
}

type Vitality float32

var healthReducer Vitality = 0.16

func (vita Vitality) CalcHealth(v float32) float32 {
	return float32(vita*healthReducer) * v
}

type MagicAffinityType uint8

const (
	MagicAffinityTypeNone MagicAffinityType = iota
	MagicAffinityTypeFire
	MagicAffinityTypeElectric
	MagicAffinityTypeCleric
	MagicAffinityTypeLen
)

type ExpMagic struct {
	Affs [MagicAffinityTypeLen]float32
}

type MagicAffinity struct {
	DamageMult    float32
	ManaCostReduc float32
	SpellCDReduc  float32
}

var magicReducers = [MagicAffinityTypeLen]MagicAffinity{
	{}, // None
	{ // MagicAffinityTypeFire
		DamageMult:    0.033,
		ManaCostReduc: 0.014,
		SpellCDReduc:  0.016,
	},
	{ // MagicAffinityTypeElectric
		DamageMult:    0.028,
		ManaCostReduc: 0.019,
		SpellCDReduc:  0.01,
	},
	{ // MagicAffinityTypeCleric
		DamageMult:    0.09,
		ManaCostReduc: 0.017,
		SpellCDReduc:  0.022,
	},
}

func (expMagic ExpMagic) CalcDamage(aff MagicAffinityType, v float32) float32 {
	return (expMagic.Affs[aff] * magicReducers[aff].DamageMult) * v
}
func (expMagic ExpMagic) CalcMana(aff MagicAffinityType, v float32) float32 {
	return (expMagic.Affs[aff] * magicReducers[aff].ManaCostReduc) * v
}
func (expMagic ExpMagic) CalcCD(aff MagicAffinityType, v float32) float32 {
	return (expMagic.Affs[aff] * magicReducers[aff].SpellCDReduc) * v
}

type MeleeAffinityType uint8

const (
	MeleeAffinityTypeNone MeleeAffinityType = iota
	MeleeAffinityTypeAssasin
	MeleeAffinityTypeWarrior
	MeleeAffinityTypeMartialArt
	MeleeAffinityTypeLen
)

type ExpMelee struct {
	Affs [MeleeAffinityTypeLen]float32
}
type MeleeAffinity struct {
	DamageMult      float32
	CriticalHitMult float32
	MeleeCDReduc    float32
}

var meleeReducers = [MeleeAffinityTypeLen]MeleeAffinity{
	{}, // None
	{ // MeleeAffinityTypeAssasin 18
		DamageMult:      0.08,
		CriticalHitMult: 0.11,
		MeleeCDReduc:    0.02,
	},
	{ // MeleeAffinityTypeWarrior -- 18
		DamageMult:      0.09,
		CriticalHitMult: 0.09,
		MeleeCDReduc:    0.02,
	},
	{ // MeleeAffinityTypeMartialArt
		DamageMult:      0.09,
		CriticalHitMult: 0.09,
		MeleeCDReduc:    0.025,
	},
}

func (expMelee ExpMelee) CalcDamage(aff MeleeAffinityType, v float32) float32 {
	return (expMelee.Affs[aff] * meleeReducers[aff].DamageMult) * v
}
func (expMelee ExpMelee) CalcCritical(aff MeleeAffinityType, v float32) float32 {
	return (expMelee.Affs[aff] * meleeReducers[aff].CriticalHitMult) * v
}
func (expMelee ExpMelee) CalcCD(aff MeleeAffinityType, v float32) float32 {
	return (expMelee.Affs[aff] * meleeReducers[aff].MeleeCDReduc) * v
}
