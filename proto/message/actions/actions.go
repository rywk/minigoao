package actions

type A = uint32

const (
	Spawn A = iota
	Despawn
	Move
	Dir
	Died
	Revive

	Nothing
)
