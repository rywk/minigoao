package server

import (
	"log"

	"github.com/rywk/minigoao/pkg/constants"
	"github.com/rywk/minigoao/pkg/constants/mapdef"
	"github.com/rywk/minigoao/pkg/grid"
	"github.com/rywk/minigoao/pkg/msgs"
	"github.com/rywk/minigoao/pkg/typ"
)

func (g *Game) EndPvP1v1(p1, p2 *Player) {
	if p1.dead {
		g.db.AddCharacterLossVOne(p1.characterID)
		g.db.AddCharacterWinVOne(p2.characterID)
	} else if p2.dead {
		g.db.AddCharacterLossVOne(p2.characterID)
		g.db.AddCharacterWinVOne(p1.characterID)
	}

	p1.space = g.space
	p1.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
	g.space.SetSlot(mapdef.Players.Int(), p1.pos, p1.id)
	p1.mapType = mapdef.MapLobby

	p2.space = g.space
	p2.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
	g.space.SetSlot(mapdef.Players.Int(), p2.pos, p2.id)
	p2.mapType = mapdef.MapLobby

	p1TpEv := p1.ToEventPlayerTp()
	p1.obs = grid.NewObserverRange(p1.space, p1.pos,
		constants.GridViewportX, constants.GridViewportY,
		func(t *grid.Tile) {
			if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == p1.id {
				return
			}
			vp := g.players[t.Layers[mapdef.Players]]
			p1TpEv.VisiblePlayers = append(p1TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
		},
	)

	p2TpEv := p2.ToEventPlayerTp()
	p2.obs = grid.NewObserverRange(p2.space, p2.pos,
		constants.GridViewportX, constants.GridViewportY,
		func(t *grid.Tile) {
			if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == p2.id {
				return
			}
			vp := g.players[t.Layers[mapdef.Players]]
			p2TpEv.VisiblePlayers = append(p2TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
		},
	)

	p1.space.Notify(p1.pos, msgs.EPlayerSpawned.U8(), (*msgs.EventPlayerSpawned)(p1.ToEventNewPlayer()), uint16(p1.id), uint16(p2.id))
	p2.space.Notify(p2.pos, msgs.EPlayerSpawned.U8(), (*msgs.EventPlayerSpawned)(p2.ToEventNewPlayer()), uint16(p1.id), uint16(p2.id))
	p1.Send <- OutMsg{Event: msgs.ETpTo, Data: p1TpEv}
	p2.Send <- OutMsg{Event: msgs.ETpTo, Data: p2TpEv}
}

func (g *Game) StartPvP1v1(p1, p2 *Player) {
	p1.space.Unset(mapdef.Players.Int(), p1.pos)
	p2.space.Unset(mapdef.Players.Int(), p2.pos)
	p1.space.Notify(p1.pos, msgs.EPlayerLeaveViewport.U8(), p1.id, p1.id, p2.id)
	p2.space.Notify(p2.pos, msgs.EPlayerLeaveViewport.U8(), p2.id, p1.id, p2.id)

	pvpSpace := grid.NewGrid(int32(mapdef.Arena1v1.Dx()), int32(mapdef.Arena1v1.Dy()), uint8(mapdef.LayerTypes))
	g.AddObjectsToSpace(pvpSpace, mapdef.MapPvP1v1)

	p1.space = pvpSpace
	p1.pos = typ.P{X: 1, Y: 1}
	pvpSpace.SetSlot(mapdef.Players.Int(), p1.pos, p1.id)
	p1.mapType = mapdef.MapPvP1v1
	p1.hp = p1.exp.Stats.MaxHP
	p1.mp = p1.exp.Stats.MaxMP
	p1.obs = grid.NewObserver(p1.space, p1.pos,
		constants.GridViewportX, constants.GridViewportY)

	p2.space = pvpSpace
	p2.pos = typ.P{X: 7, Y: 7}
	pvpSpace.SetSlot(mapdef.Players.Int(), p2.pos, p2.id)
	p2.mapType = mapdef.MapPvP1v1
	p2.hp = p2.exp.Stats.MaxHP
	p2.mp = p2.exp.Stats.MaxMP
	p2.obs = grid.NewObserver(p2.space, p2.pos,
		constants.GridViewportX, constants.GridViewportY)

	p1Tp := p1.ToEventPlayerTp()
	p1Tp.VisiblePlayers = []msgs.EventNewPlayer{*p2.ToEventNewPlayer()}
	p1.Send <- OutMsg{Event: msgs.ETpTo, Data: p1Tp}

	p2Tp := p2.ToEventPlayerTp()
	p2Tp.VisiblePlayers = []msgs.EventNewPlayer{*p1.ToEventNewPlayer()}
	p2.Send <- OutMsg{Event: msgs.ETpTo, Data: p2Tp}
}

func (g *Game) EndPvP2v2(p1, p2 *Player) {
	var winner1 *Player
	var winner2 *Player
	var loser1 *Player
	var loser2 *Player

	if !p1.dead {
		winner1 = p1
	} else if !p2.dead {
		winner1 = p2
	}

	if winner1 == nil {
		return
	}

	if len(winner1.team) > 0 {
		winner2 = g.players[winner1.team[0]]
	}
	if len(winner1.enemys) > 1 {
		loser1 = g.players[winner1.enemys[0]]
		loser2 = g.players[winner1.enemys[1]]
	}
	if loser1 != nil && loser2 != nil && (!loser1.dead || !loser2.dead) {
		return
	}

	notifyExclude := []uint16{winner1.id}

	winner1.space = g.space
	winner1.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
	g.space.SetSlot(mapdef.Players.Int(), winner1.pos, winner1.id)
	winner1.mapType = mapdef.MapLobby
	winner1.team = []uint16{}
	winner1.enemys = []uint16{}
	if err := g.db.AddCharacterWinVTwo(winner1.characterID); err != nil {
		log.Printf("error AddCharacterWinVTwo %v", err)
	}
	if winner2 != nil {
		notifyExclude = append(notifyExclude, winner2.id)
		winner2.space = g.space
		winner2.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
		g.space.SetSlot(mapdef.Players.Int(), winner2.pos, winner2.id)
		winner2.mapType = mapdef.MapLobby
		winner2.team = []uint16{}
		winner2.enemys = []uint16{}
		if err := g.db.AddCharacterWinVTwo(winner2.characterID); err != nil {
			log.Printf("error AddCharacterWinVTwo %v", err)
		}
	}
	if loser1 != nil {
		notifyExclude = append(notifyExclude, loser1.id)
		loser1.space = g.space
		loser1.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
		g.space.SetSlot(mapdef.Players.Int(), loser1.pos, loser1.id)
		loser1.mapType = mapdef.MapLobby
		loser1.team = []uint16{}
		loser1.enemys = []uint16{}
		if err := g.db.AddCharacterLossVTwo(loser1.characterID); err != nil {
			log.Printf("error AddCharacterLossVTwo %v", err)
		}
	}
	if loser2 != nil {
		notifyExclude = append(notifyExclude, loser2.id)
		loser2.space = g.space
		loser2.pos = checkSpawn(g.space, typ.P{X: 10, Y: 35})
		g.space.SetSlot(mapdef.Players.Int(), loser2.pos, loser2.id)
		loser2.mapType = mapdef.MapLobby
		loser2.team = []uint16{}
		loser2.enemys = []uint16{}
		if err := g.db.AddCharacterLossVTwo(loser2.characterID); err != nil {
			log.Printf("error AddCharacterLossVTwo %v", err)
		}
	}

	winner1TpEv := winner1.ToEventPlayerTp()
	winner1.obs = grid.NewObserverRange(winner1.space, winner1.pos,
		constants.GridViewportX, constants.GridViewportY,
		func(t *grid.Tile) {
			if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == winner1.id {
				return
			}
			vp := g.players[t.Layers[mapdef.Players]]
			winner1TpEv.VisiblePlayers = append(winner1TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
		},
	)
	winner1.space.Notify(winner1.pos, msgs.EPlayerSpawned.U8(),
		(*msgs.EventPlayerSpawned)(winner1.ToEventNewPlayer()), notifyExclude...)
	winner1.Send <- OutMsg{Event: msgs.ETpTo, Data: winner1TpEv}

	if winner2 != nil {
		winner2TpEv := winner2.ToEventPlayerTp()
		winner2.obs = grid.NewObserverRange(winner2.space, winner2.pos,
			constants.GridViewportX, constants.GridViewportY,
			func(t *grid.Tile) {
				if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == winner2.id {
					return
				}
				vp := g.players[t.Layers[mapdef.Players]]
				winner2TpEv.VisiblePlayers = append(winner2TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
			},
		)
		winner2.space.Notify(winner2.pos, msgs.EPlayerSpawned.U8(),
			(*msgs.EventPlayerSpawned)(winner2.ToEventNewPlayer()), notifyExclude...)
		winner2.Send <- OutMsg{Event: msgs.ETpTo, Data: winner2TpEv}
	}
	if loser1 != nil {
		loser1TpEv := loser1.ToEventPlayerTp()
		loser1.obs = grid.NewObserverRange(loser1.space, loser1.pos,
			constants.GridViewportX, constants.GridViewportY,
			func(t *grid.Tile) {
				if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == loser1.id {
					return
				}
				vp := g.players[t.Layers[mapdef.Players]]
				loser1TpEv.VisiblePlayers = append(loser1TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
			},
		)
		loser1.space.Notify(loser1.pos, msgs.EPlayerSpawned.U8(),
			(*msgs.EventPlayerSpawned)(loser1.ToEventNewPlayer()), notifyExclude...)
		loser1.Send <- OutMsg{Event: msgs.ETpTo, Data: loser1TpEv}
	}
	if loser2 != nil {
		loser2TpEv := loser2.ToEventPlayerTp()
		loser2.obs = grid.NewObserverRange(loser2.space, loser2.pos,
			constants.GridViewportX, constants.GridViewportY,
			func(t *grid.Tile) {
				if t.Layers[mapdef.Players] == 0 || t.Layers[mapdef.Players] == loser2.id {
					return
				}
				vp := g.players[t.Layers[mapdef.Players]]
				loser2TpEv.VisiblePlayers = append(loser2TpEv.VisiblePlayers, *vp.ToEventNewPlayer())
			},
		)
		loser2.space.Notify(loser2.pos, msgs.EPlayerSpawned.U8(),
			(*msgs.EventPlayerSpawned)(loser2.ToEventNewPlayer()), notifyExclude...)
		loser2.Send <- OutMsg{Event: msgs.ETpTo, Data: loser2TpEv}
	}
}

func (g *Game) StartPvP2v2(p1, p2, p3, p4 *Player) {
	p1.space.Unset(mapdef.Players.Int(), p1.pos)
	p2.space.Unset(mapdef.Players.Int(), p2.pos)
	p3.space.Unset(mapdef.Players.Int(), p3.pos)
	p4.space.Unset(mapdef.Players.Int(), p4.pos)
	p1.space.Notify(p1.pos, msgs.EPlayerLeaveViewport.U8(), p1.id, p1.id, p2.id, p3.id, p4.id)
	p2.space.Notify(p2.pos, msgs.EPlayerLeaveViewport.U8(), p1.id, p1.id, p2.id, p3.id, p4.id)
	p3.space.Notify(p3.pos, msgs.EPlayerLeaveViewport.U8(), p1.id, p1.id, p2.id, p3.id, p4.id)
	p4.space.Notify(p4.pos, msgs.EPlayerLeaveViewport.U8(), p1.id, p1.id, p2.id, p3.id, p4.id)

	pvpSpace := grid.NewGrid(int32(mapdef.Arena2v2.Dx()), int32(mapdef.Arena2v2.Dy()), uint8(mapdef.LayerTypes))
	g.AddObjectsToSpace(pvpSpace, mapdef.MapPvP2v2)

	// team 1
	p1.space = pvpSpace
	p1.pos = typ.P{X: 1, Y: 1}
	pvpSpace.SetSlot(mapdef.Players.Int(), p1.pos, p1.id)
	p1.mapType = mapdef.MapPvP2v2
	p1.hp = p1.exp.Stats.MaxHP
	p1.mp = p1.exp.Stats.MaxMP
	p1.obs = grid.NewObserver(p1.space, p1.pos, constants.GridViewportX, constants.GridViewportY)
	p1np := *p1.ToEventNewPlayer()

	p2.space = pvpSpace
	p2.pos = typ.P{X: 2, Y: 1}
	pvpSpace.SetSlot(mapdef.Players.Int(), p2.pos, p2.id)
	p2.mapType = mapdef.MapPvP2v2
	p2.hp = p2.exp.Stats.MaxHP
	p2.mp = p2.exp.Stats.MaxMP
	p2.obs = grid.NewObserver(p2.space, p2.pos, constants.GridViewportX, constants.GridViewportY)
	p2np := *p2.ToEventNewPlayer()

	p1.team = append(p1.team, p2.id)
	p2.team = append(p2.team, p1.id)
	p1.enemys = append(p1.enemys, p3.id, p4.id)
	p2.enemys = append(p2.enemys, p3.id, p4.id)

	// team 2
	p3.space = pvpSpace
	p3.pos = typ.P{X: 14, Y: 10}
	pvpSpace.SetSlot(mapdef.Players.Int(), p3.pos, p3.id)
	p3.mapType = mapdef.MapPvP2v2
	p3.hp = p3.exp.Stats.MaxHP
	p3.mp = p3.exp.Stats.MaxMP
	p3.obs = grid.NewObserver(p3.space, p3.pos, constants.GridViewportX, constants.GridViewportY)
	p3np := *p3.ToEventNewPlayer()

	p4.space = pvpSpace
	p4.pos = typ.P{X: 13, Y: 10}
	pvpSpace.SetSlot(mapdef.Players.Int(), p4.pos, p4.id)
	p4.mapType = mapdef.MapPvP2v2
	p4.hp = p4.exp.Stats.MaxHP
	p4.mp = p4.exp.Stats.MaxMP
	p4.obs = grid.NewObserver(p4.space, p4.pos, constants.GridViewportX, constants.GridViewportY)
	p4np := *p4.ToEventNewPlayer()

	p3.team = append(p3.team, p4.id)
	p4.team = append(p4.team, p3.id)
	p3.enemys = append(p3.enemys, p1.id, p2.id)
	p4.enemys = append(p4.enemys, p1.id, p2.id)

	p1Tp := p1.ToEventPlayerTp()
	p2Tp := p2.ToEventPlayerTp()
	p3Tp := p3.ToEventPlayerTp()
	p4Tp := p4.ToEventPlayerTp()

	p1Tp.VisiblePlayers = []msgs.EventNewPlayer{p2np, p3np, p4np}
	p2Tp.VisiblePlayers = []msgs.EventNewPlayer{p1np, p3np, p4np}
	p3Tp.VisiblePlayers = []msgs.EventNewPlayer{p1np, p2np, p4np}
	p4Tp.VisiblePlayers = []msgs.EventNewPlayer{p1np, p2np, p3np}

	p1.Send <- OutMsg{Event: msgs.ETpTo, Data: p1Tp}
	p2.Send <- OutMsg{Event: msgs.ETpTo, Data: p2Tp}
	p3.Send <- OutMsg{Event: msgs.ETpTo, Data: p3Tp}
	p4.Send <- OutMsg{Event: msgs.ETpTo, Data: p4Tp}
}

func (p *Player) ToEventNewPlayer() *msgs.EventNewPlayer {
	return &msgs.EventNewPlayer{
		ID:         uint16(p.id),
		Nick:       p.nick,
		Pos:        p.pos,
		Dir:        p.dir,
		Speed:      uint8(p.speedPxXFrame),
		Weapon:     p.inv.GetWeapon(),
		Shield:     p.inv.GetShield(),
		Head:       p.inv.GetHead(),
		Body:       p.inv.GetBody(),
		Dead:       p.dead,
		Meditating: p.meditating,
	}
}

func (p *Player) ToEventPlayerTp() *msgs.EventPlayerTp {
	return &msgs.EventPlayerTp{
		MapType: p.mapType,
		Pos:     p.pos,
		Dir:     p.dir,
		Dead:    p.dead,
		HP:      p.hp,
		MP:      p.mp,
		Inv:     *p.inv,
		Exp:     p.exp.ToMsgs(),
	}
}
