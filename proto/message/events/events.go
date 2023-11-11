package events

import (
	"github.com/rywk/minigoao/proto/message"
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

func Bytes(m protoreflect.ProtoMessage) []byte {
	r, _ := proto.Marshal(m)
	return r
}

func Proto[K protoreflect.ProtoMessage](bs []byte, model K) K {
	proto.Unmarshal(bs, model)
	return model
}

// Event
type E = uint32

const (
	// These are client actions
	// with their respective respose.
	// Client -> Server
	// Server -> Client
	Register E = iota
	RegisterOk
	Move
	MoveOk
	CastSpell
	CastSpellOk
	RecivedSpell
	SpellHit
	CastMelee
	CastMeleeOk
	RecivedMelee
	MeleeHit

	Dir
	// Events the server sends to clients
	// These should not require a response
	// from the client.
	// PlayerAction movements/change skin/idk
	PlayerAction
	PlayerActions
	// Combat
	SpellCast // TODO
	MeleeCast // TODO

	Ping
	// Not a real event no need to handle
	// used to use arrays of this len instead of slices
	Len
)

func New(e E, id uint32, data []byte) *message.Event {
	return &message.Event{Type: e, Id: id, E: data}
}
