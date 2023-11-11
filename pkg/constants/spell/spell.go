package spell

type Spell = uint32

const (
	Inmo Spell = iota
	InmoRm
	Apoca
	Desca

	None
)

var spells = [None + 1]string{
	"Inmo",
	"InmoRm",
	"Apoca",
	"Desca",

	"None",
}

func S(s Spell) string {
	return spells[s]
}
