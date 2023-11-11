package img

import (
	_ "embed"
)

var (
	// Textures

	//go:embed grass_patches.png
	GrassPatches_png []byte
	//go:embed ongo.png
	Ongo_png []byte
	//go:embed tiletest.png
	Tiletest_png []byte

	// Spell Icons

	//go:embed spell_icon_placeholder.png
	PlaceholderSpellIcon_png []byte
	//go:embed icon_apoca.png
	IconSpellApoca_png []byte
	//go:embed icon_inmo.png
	IconSpellInmo_png []byte
	//go:embed icon_inmo_rm.png
	IconSpellInmoRm_png []byte

	// Effects

	//go:embed melee_hit.png
	MeleeHit_png []byte
	//go:embed spell_apoca.png
	SpellApoca_png []byte
	//go:embed spell_inmo.png
	SpellInmo_png []byte

	// Body

	//go:embed body_naked.png
	BodyNaked_png []byte
	//go:embed head.png
	Head_png []byte
	//go:embed dead_body.png
	DeadBody_png []byte
	//go:embed dead_head.png
	DeadHead_png []byte

	// Helmets

	//go:embed hat_pro.png
	HatPro_png []byte

	// Armors

	//go:embed dark_armor.png
	DarkArmor_png []byte

	// Weapons

	//go:embed axe.png
	Axe_png []byte
	//go:embed sword_special.png
	SwordSpecial_png []byte

	// Shields

	//go:embed shield_tower.png
	ShieldTower_png []byte
	//go:embed shield_silver.png
	ShieldSilver_png []byte
)
