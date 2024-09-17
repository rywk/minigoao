package spell

type Spell uint8

const (
	Paralize Spell = iota
	RemoveParalize
	HealWounds
	Revive
	ElectricDischarge
	Explode
	None
)

var spells = [None]string{
	"Paralize",
	"RemoveParalize",
	"HealWounds",
	"Revive",
	"ElectricDischarge",
	"Explode",
}

func (s Spell) String() string {
	return spells[s]
}
