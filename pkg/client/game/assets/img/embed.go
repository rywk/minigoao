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
	//go:embed spell_selector.png
	SpellSelector_png []byte

	// Stats

	//go:embed hp_bar_stats_big.png
	BigHPBar_png []byte
	//go:embed mp_bar_stats_big.png
	BigMPBar_png []byte
	//go:embed placeholder_stats.png
	PlaceholderStats_png []byte
	//go:embed hp_bar_stats_mini.png
	MiniHPBar_png []byte
	//go:embed mp_bar_stats_mini.png
	MiniMPBar_png []byte
	//go:embed mini_placeholder_stats.png
	MiniPlaceholderStats_png []byte

	// Effects

	//go:embed melee_hit.png
	MeleeHit_png []byte
	//go:embed spell_apoca.png
	SpellApoca_png []byte
	//go:embed spell_desca.png
	SpellDesca_png []byte
	//go:embed spell_inmo.png
	SpellInmo_png []byte
	//go:embed spell_paralize.png
	SpellParalize_png []byte
	//go:embed spell_inmo_rm.png
	SpellInmoRm_png []byte
	//go:embed spell_heal_wounds.png
	SpellHealWounds_png []byte
	//go:embed spell_heal_wounds_2.png
	SpellHealWounds2_png []byte
	//go:embed spell_revive.png
	SpellRevive_png []byte
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
)
