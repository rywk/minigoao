package attack

type Spell uint8

const (
	SpellNone Spell = iota
	SpellParalize
	SpellRemoveParalize
	SpellHealWounds
	SpellResurrect
	SpellElectricDischarge
	SpellExplode
	SpellLen
)

var spells = [SpellLen]string{
	"SpellNone",
	"SpellParalize",
	"SpellRemoveParalize",
	"SpellHealWounds",
	"SpellResurrect",
	"SpellElectricDischarge",
	"SpellExplode",
}

func (s Spell) String() string {
	return spells[s]
}
