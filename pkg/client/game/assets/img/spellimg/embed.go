package spellimg

import _ "embed"

var (
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
	//go:embed icon_apoca264.png
	IconExplode_png []byte

	// Effects

	//go:embed melee_hit.png
	MeleeHit_png []byte
	//go:embed spell_apoca.png
	SpellApoca_png []byte
	//go:embed spell_apoca22.png
	SpellApoca2_png []byte
	//go:embed spell_desca2.png
	SpellDesca_png []byte
	//go:embed spell_paralize.png
	SpellParalize_png []byte
	//go:embed spell_inmo_rm.png
	SpellInmoRm_png []byte

	//go:embed heal_wounds_new.png
	SpellHealWoundsNew_png []byte
	//go:embed spell_resurrect.png
	SpellResurrect_png []byte
	// //go:embed last_trial_test_alpha.png
	// SpellLastTrial_png []byte
)
