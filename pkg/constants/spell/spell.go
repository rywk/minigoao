package spell

type Spell uint8

const (
	None Spell = iota
	Paralize
	RemoveParalize
	HealWounds
	Resurrect
	ElectricDischarge
	Explode
	Len
)

var spells = [Len]string{
	"None",
	"Paralize",
	"RemoveParalize",
	"HealWounds",
	"Resurrect",
	"ElectricDischarge",
	"Explode",
}

func (s Spell) String() string {
	return spells[s]
}
