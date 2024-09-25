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

	//go:embed placeholder_spellbar.png
	PlaceholderSpellbar_png []byte
	//go:embed spellbar_icons.png
	SpellbarIcons_png []byte
	//go:embed spellbar_icons_2.png
	SpellbarIcons2_png []byte
	//go:embed spell_selector.png
	SpellSelector_png []byte

	// Effects

	//go:embed melee_hit.png
	MeleeHit_png []byte
	//go:embed spell_apoca.png
	SpellApoca_png []byte
	//go:embed spell_desca2.png
	SpellDesca_png []byte
	//go:embed spell_inmo.png
	SpellInmo_png []byte
	//go:embed spell_paralize.png
	SpellParalize_png []byte
	//go:embed spell_inmo_rm.png
	SpellInmoRm_png []byte
	//go:embed spell_heal_wounds.png
	SpellHealWounds_png []byte
	//go:embed spell_heal_wounds_3.png
	SpellHealWounds2_png []byte
	//go:embed heal_wounds_new.png
	SpellHealWoundsNew_png []byte
	//go:embed spell_resurrect.png
	SpellResurrect_png []byte
	//go:embed last_trial_test_alpha.png
	SpellLastTrial_png []byte

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

	//go:embed hud_bg.png
	HudBg_png []byte
	//go:embed hp_bar.png
	HpBar_png []byte
	//go:embed mp_bar.png
	MpBar_png []byte

	//go:embed blue_potion.png
	BluePotion_png []byte
	//go:embed red_potion.png
	RedPotion_png []byte

	//go:embed icon_melee.png
	IconMelee_png []byte

	//go:embed icon_resurrect64.png
	IconResurrect_png []byte
	//go:embed icon_heal64.png
	IconHeal_png []byte
	//go:embed icon_rm_paralize64.png
	IconRmParalize_png []byte
	//go:embed icon_paralize64.png
	IconParalize_png []byte
	//go:embed icon_electric_discharge64.png
	IconElectricDischarge_png []byte
	//go:embed icon_apoca64.png
	IconExplode_png []byte

	//go:embed cooldown_base.png
	CooldownBase_png []byte

	//go:embed config_icon.png
	ConfigIcon_png []byte
)
