package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/rywk/minigoao/pkg/direction"
)

type KeyConfig struct {
	Front ebiten.Key
	Back  ebiten.Key
	Left  ebiten.Key
	Right ebiten.Key
}

var DefaultConfig = KeyConfig{
	Front: ebiten.KeyS,
	Back:  ebiten.KeyW,
	Left:  ebiten.KeyA,
	Right: ebiten.KeyD,
}

type Keys struct {
	cfg     *KeyConfig
	last    ebiten.Key
	pressed map[ebiten.Key]bool

	directionMap map[ebiten.Key]direction.D
}

func NewKeys(cfg *KeyConfig) *Keys {
	if cfg == nil {
		cfg = &DefaultConfig
	}
	k := &Keys{
		cfg: cfg,
		pressed: map[ebiten.Key]bool{
			cfg.Front: false,
			cfg.Back:  false,
			cfg.Left:  false,
			cfg.Right: false,
		},
		directionMap: map[ebiten.Key]direction.D{
			cfg.Front: direction.Front,
			cfg.Back:  direction.Back,
			cfg.Left:  direction.Left,
			cfg.Right: direction.Right,
			-1:        direction.Still,
		},
	}
	return k
}

func (k *Keys) ListenMovement() {
	front, back, left, right := ebiten.IsKeyPressed(k.cfg.Front),
		ebiten.IsKeyPressed(k.cfg.Back),
		ebiten.IsKeyPressed(k.cfg.Left),
		ebiten.IsKeyPressed(k.cfg.Right)

	if front && !k.pressed[k.cfg.Front] {
		k.pressed[k.cfg.Front] = true
		k.last = k.cfg.Front
	} else if front && !k.pressed[k.last] {
		k.last = k.cfg.Front
	} else if !front && k.pressed[k.cfg.Front] {
		k.pressed[k.cfg.Front] = false

	}

	if back && !k.pressed[k.cfg.Back] {
		k.pressed[k.cfg.Back] = true
		k.last = k.cfg.Back
	} else if back && !k.pressed[k.last] {
		k.last = k.cfg.Back
	} else if !back && k.pressed[k.cfg.Back] {
		k.pressed[k.cfg.Back] = false
	}

	if left && !k.pressed[k.cfg.Left] {
		k.pressed[k.cfg.Left] = true
		k.last = k.cfg.Left
	} else if left && !k.pressed[k.last] {
		k.last = k.cfg.Left
	} else if !left && k.pressed[k.cfg.Left] {
		k.pressed[k.cfg.Left] = false
	}

	if right && !k.pressed[k.cfg.Right] {
		k.pressed[k.cfg.Right] = true
		k.last = k.cfg.Right
	} else if right && !k.pressed[k.last] {
		k.last = k.cfg.Right
	} else if !right && k.pressed[k.cfg.Right] {
		k.pressed[k.cfg.Right] = false
	}

	if !front && !back && !left && !right {
		k.last = -1
	}
}

func (k *Keys) MovingTo() direction.D {
	return k.directionMap[k.last]
}
