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
	//go:embed spell_revive.wav
	SpellRevive_wav []byte
	//go:embed spell_heal_wounds.wav
	SpellHealWounds_wav []byte
)
