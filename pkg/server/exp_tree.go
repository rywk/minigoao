package server

import (
	"log"

	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/constants/skill"
	"github.com/rywk/minigoao/pkg/msgs"
)

type Experience struct {
	p *Player
	// Player stats
	FreePoints int32
	Skills     skill.Skills
	ItemBuffs  skill.Buffs
	Stats      skill.Stats
}

func NewExperience(p *Player) *Experience {
	return &Experience{
		p:          p,
		FreePoints: TotalSkills,
		Skills:     skill.Skills{},
		ItemBuffs:  skill.Buffs{},
		Stats:      skill.Skills{}.Stats(),
	}
}

const TotalSkills = 14

// Each time the skills are updated
func (e *Experience) SetNewSkills(sk skill.Skills) {
	total := sk.Total()
	if total > TotalSkills || total < 0 {
		return
	}
	e.Skills = sk
	e.Stats = sk.Stats()
	e.FreePoints = TotalSkills - int32(total)
}

// Each time an item is equipped or unequipped
func (e *Experience) SetItemBuffs() {
	items := []item.Item{
		e.p.inv.GetHead(),
		e.p.inv.GetBody(),
		e.p.inv.GetWeapon(),
		e.p.inv.GetShield(),
	}
	e.ItemBuffs = skill.Buffs{}

	for i := range items {
		if items[i] == item.None {
			continue
		}
		itm := item.ItemProps[items[i]]
		log.Printf("item: %v, buffs: %v", itm.Type, itm.Buffs)

		e.ItemBuffs = e.ItemBuffs.Add(itm.Buffs)
	}
	log.Print(e.ItemBuffs)
}

func (e *Experience) ToMsgs() msgs.Experience {
	return msgs.Experience{
		FreePoints: e.FreePoints,
		Skills:     e.Skills,
		ItemBuffs:  e.ItemBuffs,
		Stats:      e.Stats,
	}
}
