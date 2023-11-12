package direction

type D = uint32

const (
	Still D = iota
	Front
	Back
	Left
	Right
)

var List = []D{Front, Back, Left, Right}

func S(d D) string {
	return [Right + 1]string{"Still", "Front", "Back", "Left", "Right"}[d]
}
