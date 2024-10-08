package img

import (
	_ "embed"
)

var (
	//go:embed icon.png
	Icon_png []byte

	//go:embed icon_x.png
	IconX_png []byte
	//go:embed icon_disk.png
	IconDisk_png []byte
	//go:embed icon_plus.png
	IconPlus_png []byte
	//go:embed icon_plus_big.png
	IconPlusBig_png []byte
	//go:embed icon_substract.png
	IconSubstract_png []byte
	//go:embed icon_blood.png
	IconBlood_png []byte
	//go:embed equipped_item.png
	EquippedItem_png []byte

	//go:embed icon_locked.png
	IconLockLocked_png []byte
	//go:embed icon_lock_open.png
	IconLockOpen_png []byte
	// Textures

	//go:embed brick_patch.png
	BrickPatches_png []byte
	//go:embed grass_patches.png
	GrassPatches_png []byte
	//go:embed rock.png
	Rock_png []byte
	//go:embed ongo.png
	Ongo_png []byte
	//go:embed excalibur.png
	Excalibur_png []byte

	// Spell Icons

	//go:embed placeholder_spellbar.png
	PlaceholderSpellbar_png []byte
	//go:embed spellbar_icons.png
	SpellbarIcons_png []byte
	//go:embed spellbar_icons_2.png
	SpellbarIcons2_png []byte
	//go:embed spell_selector.png
	SpellSelector_png []byte

	//go:embed checkbox-off.png
	CheckboxOff_png []byte
	//go:embed checkbox-on.png
	CheckboxOn_png []byte

	//go:embed input_box.png
	InputBox_png []byte

	//go:embed text.png
	Text_png []byte
	//go:embed text_small.png
	TextSmall_png []byte
	//go:embed damage_numbers.png
	DamageNumbers_png []byte

	//go:embed hud_bg2.png
	HudBg_png []byte
	//go:embed hp_bar.png
	HpBar_png []byte
	//go:embed mp_bar.png
	MpBar_png []byte

	//go:embed cooldown_base.png
	CooldownBase_png []byte

	//go:embed config_icon.png
	ConfigIcon_png []byte
)
