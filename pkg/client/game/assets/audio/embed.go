package audio

import (
	_ "embed"
)

var (
	//go:embed walk_1.wav
	Walk1_wav []byte
	//go:embed walk_2.wav
	Walk2_wav []byte

	//go:embed melee_air.wav
	MeleeAir_wav []byte
	//go:embed melee_hit.wav
	MeleeHit_wav []byte

	//go:embed death.wav
	Death_wav []byte
	//go:embed kill_bell.wav
	KillBell_wav []byte

	//go:embed potion.wav
	Potion_wav []byte

	//go:embed spawn.wav
	Spawn_wav []byte

	//go:embed spell_apoca.wav
	SpellApoca_wav []byte
	//go:embed spell_inmo.wav
	SpellInmo_wav []byte
	//go:embed spell_inmo_rm.wav
	SpellInmoRm_wav []byte
	//go:embed spell_desca.wav
	SpellDesca_wav []byte
	//go:embed spell_resurrect.wav
	SpellResurrect_wav []byte
	//go:embed spell_heal_wounds.wav
	SpellHealWounds_wav []byte

	//go:embed walk_1_low.ogg
	Walk1Low_ogg []byte
	//go:embed walk_2_low.ogg
	Walk2Low_ogg []byte

	//go:embed melee_air_low.ogg
	MeleeAirLow_ogg []byte
	//go:embed melee_hit_low.ogg
	MeleeHitLow_ogg []byte

	//go:embed death.ogg
	Death_ogg []byte
	//go:embed kill_bell.ogg
	KillBell_ogg []byte

	//go:embed potion_low.ogg
	PotionLow_ogg []byte

	//go:embed spawn_low.ogg
	SpawnLow_ogg []byte

	//go:embed spell_apoca_low.ogg
	SpellApocaLow_ogg []byte
	//go:embed spell_inmo_low.ogg
	SpellInmoLow_ogg []byte
	//go:embed spell_inmo_rm_low.ogg
	SpellInmoRmLow_ogg []byte
	//go:embed spell_desca_low.ogg
	SpellDescaLow_ogg []byte
	//go:embed spell_resurrect_low.ogg
	SpellResurrectLow_ogg []byte
	//go:embed spell_heal_wounds_low.ogg
	SpellHealWoundsLow_ogg []byte
)
