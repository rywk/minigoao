package server

import "github.com/rywk/minigoao/pkg/msgs"

// Potions
type Item struct {
	Type msgs.Item
	Use  func(p *Player) uint32
}

var items = [msgs.ItemLen]Item{
	{Type: msgs.ItemNone},
	{
		Type: msgs.ItemManaPotion,
		Use: func(p *Player) uint32 {
			p.mp = p.mp + int32(float32(p.maxMp)*0.05)
			if p.mp > p.maxMp {
				p.mp = p.maxMp
			}
			return uint32(p.mp)
		},
	}, {
		Type: msgs.ItemHealthPotion,
		Use: func(p *Player) uint32 {
			p.hp = p.hp + 30
			if p.hp > p.maxHp {
				p.hp = p.maxHp
			}
			return uint32(p.hp)
		},
	},
}

func UseItem(item msgs.Item, p *Player) uint32 {
	return items[item].Use(p)
}
