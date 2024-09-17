package msgs

import (
	"encoding/binary"
	"log"
	"net"
	"unsafe"

	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/typ"
	"github.com/vmihailenco/msgpack/v5"
)

// M game protocol messenger
// basically plain tcp with message size prefix.
// (not anymore) using protobuf but its irrelevant.
// given the size of the messages i opted for messagepack
// also to reduce deserialization im sending
// another prefix with event type
//
// |--size-|-event-| message...
//
// 0,0,1,0,0,0,0,1, 256 byte message...
type M struct {
	// The connection
	c net.Conn
}

func New(c net.Conn) *M {
	return &M{
		c: c,
	}
}

func (m *M) IP() string {
	return m.c.RemoteAddr().String()
}

var (
	// second 4 bytes reserved for event type
	eventTypeLen = 1
)

// Read blocks until a new event is read from the tcp byte stream.
func (m *M) Read() (*IncomingData, error) {
	eventByte := make([]byte, eventTypeLen)

	_, err := m.c.Read(eventByte)
	if err != nil {
		return nil, err
	}
	incd := &IncomingData{Event: E(eventByte[0])}
	if incd.Event.Len() == 0 {
		if incd.Event == EMelee {
			log.Print("reading melee")
		}
		return incd, nil
	}
	if incd.Event.Len() != -1 {
		incd.Data = make([]byte, incd.Event.Len())
		_, err = m.c.Read(incd.Data)
		return incd, err
	}

	msgSizeBs := make([]byte, 2)
	_, err = m.c.Read(msgSizeBs)
	if err != nil {
		return nil, err
	}
	msgSize := binary.BigEndian.Uint16(msgSizeBs)
	incd.Data = make([]byte, msgSize)
	_, err = m.c.Read(incd.Data)
	return incd, err
}

// Write sends the event to the connection
func (m *M) Write(event E, data []byte) error {
	buf := []byte{byte(event)}
	if data == nil {
		_, err := m.c.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err := m.c.Write(append(buf, data...))
	if err != nil {
		return err
	}
	return nil
}

// Write sends the event to the connection
func (m *M) WriteWithLen(event E, data []byte) error {
	pref := make([]byte, 3)
	pref[0] = byte(event)
	binary.BigEndian.PutUint16(pref[1:], uint16(len(data)))
	_, err := m.c.Write(append(pref, data...))
	if err != nil {
		return err
	}
	return nil
}

type IncomingData struct {
	Source int
	Event  E
	Data   []byte
}

type IncomingMsg struct {
	Source int
	Event  E
	Data   interface{}
}

type E uint8

const (
	EPing E = iota
	EServerDisconnect
	EMove
	ECastSpell
	EMelee
	EUseItem

	EMoveOk
	ECastSpellOk
	EMeleeOk
	EUseItemOk

	EPlayerConnect       // Just used internally for when the client conn starts and a nick is sent
	EPlayerLogin         // Player login response, with data about the character and the players in the viewport
	EPlayerLogout        // Just used internally for when the client conn drops
	EPlayerSpawned       // A Player spawned in the viewport
	EPlayerDespawned     // A Player despawned in the viewport
	EPlayerEnterViewport // A Player spawned at an edge of the viewport
	EPlayerLeaveViewport // A Player despawned at an edge of the viewport

	EPlayerMoved         // A Player in the viewport moved
	EPlayerSpell         // A Player in the viewport recieved a spell
	EPlayerSpellRecieved // Player recieved a spell
	EPlayerMelee         // A Player in the viewport recieved a melee
	EPlayerMeleeRecieved // Player recieved a melee

	ELen
)

const mapCoordinateSize = int(unsafe.Sizeof(uint32(0)))

var eventLen = [ELen]int{
	1,       // EPing
	0,       // EServerDisconnect
	1,       // EMove - 1 byte (uint8) to define the direction.
	1 + 4*2, // ECastSpell - 1 byte (uint8) to define the spell picked in the client side. x, y map coords are 2 uint32
	1,       // EMelee - signals user used the melee key
	1,       // EUseItem - 1 byte (uint8) to define the item id

	2,                 // EMoveOk - 1 byte (bool) move, 1 byte (bool) direction
	1 + 2 + 4 + 4 + 1, // ECastSpellOk - 1 byte (uint8) spell, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage,  4 bytes (uint32) new mp,  1 byte (bool) killed target
	1 + 1 + 2 + 4,     // EMeleeOk -   1 byte (bool) hit/miss, 1 byte (bool) killed target, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage
	1 + 4,             // EUseItemOk - 1 byte (uint8) item, 4 byte (uint32) to define value changed (mana/health)

	0,  // EPlayerConnect
	-1, // EPlayerLogin - -1 dynamic size msgpack
	0,  // EPlayerLogout
	-1, // EPlayerSpawned - -1 dynamic size msgpack
	2,  // EPlayerDespawned - 2 bytes (uint16) to define the player id
	-1, // EPlayerEnterViewport - -1 dynamic size msgpack
	2,  // EPlayerLeaveViewport - 2 bytes (uint16) to define the player id

	11,            // EPlayerMoved - 1 byte (uint8) direction, 2 bytes (uint16) player id, 8 bytes (uint32, uint32) x y
	2 + 1 + 1,     // EPlayerSpell - 2 bytes (uint16) to define the target player id, 1 byte (uint8) to define the spell, 1 byte (bool) killed target
	1 + 2 + 4 + 4, // EPlayerSpellRecieved - 1 byte (uint8) to define the spell, 2 bytes (uint16) to define the (caster) player id, 4 bytes (uint32) to define the new hp, 4 bytes (uint32) to define the damage
	1 + 1 + 2 + 2, // EPlayerMelee - 1 byte (bool) hit/miss, 1 byte (bool) killed target, 2 bytes (uint16) to define the target player id, 2 bytes (uint16) to define the attacker
	2 + 4 + 4,     // EPlayerMeleeRecieved - 2 bytes (uint16) to define the (caster) player id, 4 bytes (uint32) to define the new hp, 4 bytes (uint32) to define the damage
}

var eventString = [ELen]string{
	"EPing",
	"EMove",
	"ECastSpell",
	"EMelee",
	"EUseItem",

	"EMoveOk",
	"ECastSpellOk",
	"EMeleeOk",
	"EUseItemOk",

	"EPlayerConnect",
	"EPlayerLogin",
	"EPlayerLogout",
	"EPlayerSpawned",
	"EPlayerDespawned",
	"EPlayerEnterViewport",
	"EPlayerLeaveViewport",

	"EPlayerMoved",
	"EPlayerSpell",
	"EPlayerSpellRecieved",
	"EPlayerMelee",
	"EPlayerMeleeRecieved",
}

func (e E) Len() int {
	return eventLen[e]
}

func (e E) String() string {
	return eventString[e]
}

func (m *M) EncodeAndWrite(e E, msg interface{}) {
	switch e {
	case EPing:
		m.Write(e, nil)
	case EMove:
		m.Write(e, []byte{msg.(uint8)})
	case ECastSpell:
		m.Write(e, EncodeEventCastSpell(msg.(*EventCastSpell)))
	case EMelee:
		log.Print("writing melee")
		m.Write(e, make([]byte, EMelee.Len()))
	case EUseItem:
		m.Write(e, []byte{msg.(uint8)})
	case EMoveOk:
		m.Write(e, msg.([]byte))
	case ECastSpellOk:
		m.Write(e, EncodeEventCastSpellOk(msg.(*EventCastSpellOk)))
	case EMeleeOk:
		m.Write(e, EncodeEventMeleeOk(msg.(*EventMeleeOk)))
	case EUseItemOk:
		m.Write(e, EncodeEventUseItemOk(msg.(*EventUseItemOk)))
	case EPlayerSpawned:
		m.WriteWithLen(e, EncodeMsgpack(msg.(*EventPlayerSpawned)))
	case EPlayerDespawned:
		m.Write(e, binary.BigEndian.AppendUint16(make([]byte, 0, 2), msg.(uint16)))
	case EPlayerEnterViewport:
		m.WriteWithLen(e, EncodeMsgpack(msg.(*EventPlayerEnterViewport)))
	case EPlayerLeaveViewport:
		m.Write(e, binary.BigEndian.AppendUint16(make([]byte, 0, 2), msg.(uint16)))
	case EPlayerMoved:
		m.Write(e, EncodeEventPlayerMoved(msg.(*EventPlayerMoved)))
	case EPlayerSpell:
		m.Write(e, EncodeEventPlayerSpell(msg.(*EventPlayerSpell)))
	case EPlayerSpellRecieved:
		m.Write(e, EncodeEventPlayerSpellRecieved(msg.(*EventPlayerSpellRecieved)))
	case EPlayerMelee:
		m.Write(e, EncodeEventPlayerMelee(msg.(*EventPlayerMelee)))
	case EPlayerMeleeRecieved:
		m.Write(e, EncodeEventPlayerMeleeRecieved(msg.(*EventPlayerMeleeRecieved)))
	default:
		log.Printf("unknown event %v\n", e.String())
	}
}

func BoolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// msgpack
type EventPlayerLogin struct {
	ID             uint16
	Nick           string
	Pos            typ.P
	Dir            direction.D
	Speed          uint8
	Dead           bool
	HP             int32
	MaxHP          int32
	MP             int32
	MaxMP          int32
	VisiblePlayers []EventNewPlayer
}

func DecodeMsgpack[T any](data []byte, to *T) {
	msgpack.Unmarshal(data, to)
}

func EncodeMsgpack[T any](t *T) []byte {
	data, _ := msgpack.Marshal(t)
	return data
}

type EventNewPlayer struct {
	ID    uint16
	Nick  string
	Pos   typ.P
	Dir   direction.D
	Dead  bool
	Speed uint8
}

type EventPlayerSpawned = EventNewPlayer

type EventPlayerEnterViewport = EventNewPlayer

// binary
type Item uint8
type EventCastSpell struct {
	Spell  spell.Spell
	PX, PY uint32
}

func DecodeEventCastSpell(data []byte) *EventCastSpell {
	return &EventCastSpell{
		Spell: spell.Spell(data[0]),
		PX:    binary.BigEndian.Uint32(data[1:5]),
		PY:    binary.BigEndian.Uint32(data[5:9]),
	}
}

func EncodeEventCastSpell(c *EventCastSpell) []byte {
	bs := make([]byte, ECastSpell.Len())
	bs[0] = byte(c.Spell)
	binary.BigEndian.PutUint32(bs[1:5], c.PX)
	binary.BigEndian.PutUint32(bs[5:9], c.PY)
	return bs
}

type EventCastSpellOk struct {
	ID     uint16
	Damage uint32
	NewMP  uint32
	Spell  spell.Spell
	Killed bool
}

func DecodeEventCastSpellOk(data []byte) *EventCastSpellOk {
	return &EventCastSpellOk{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		NewMP:  binary.BigEndian.Uint32(data[6:10]),
		Spell:  spell.Spell(data[10]),
		Killed: data[11] != 0,
	}
}

func EncodeEventCastSpellOk(c *EventCastSpellOk) []byte {
	bs := make([]byte, ECastSpellOk.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	binary.BigEndian.PutUint32(bs[2:6], c.Damage)
	binary.BigEndian.PutUint32(bs[6:10], c.NewMP)
	bs[10] = byte(c.Spell)
	if c.Killed {
		bs[11] = 1
	}
	return bs
}

type EventMeleeOk struct {
	ID     uint16
	Damage uint32
	Hit    bool
	Killed bool
}

func DecodeEventMeleeOk(data []byte) *EventMeleeOk {
	return &EventMeleeOk{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		Hit:    data[6] != 0,
		Killed: data[7] != 0,
	}
}

func EncodeEventMeleeOk(c *EventMeleeOk) []byte {
	bs := make([]byte, EMeleeOk.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	binary.BigEndian.PutUint32(bs[2:6], c.Damage)
	if c.Hit {
		bs[6] = 1
	}
	if c.Killed {
		bs[7] = 1
	}
	return bs
}

type EventUseItemOk struct {
	Item   Item
	Change uint32
}

func DecodeEventUseItemOk(data []byte) *EventUseItemOk {
	return &EventUseItemOk{
		Item:   Item(data[0]),
		Change: binary.BigEndian.Uint32(data[1:5]),
	}
}

func EncodeEventUseItemOk(c *EventUseItemOk) []byte {
	bs := make([]byte, EUseItemOk.Len())
	bs[0] = byte(c.Item)
	binary.BigEndian.PutUint32(bs[1:5], c.Change)
	return bs
}

type EventPlayerMoved struct {
	Dir direction.D
	ID  uint16
	Pos typ.P
}

func DecodeEventPlayerMoved(data []byte) *EventPlayerMoved {
	return &EventPlayerMoved{
		Dir: direction.D(data[0]),
		ID:  binary.BigEndian.Uint16(data[1:3]),
		Pos: typ.P{
			X: int32(binary.BigEndian.Uint32(data[3:7])),
			Y: int32(binary.BigEndian.Uint32(data[7:11])),
		},
	}
}

func EncodeEventPlayerMoved(c *EventPlayerMoved) []byte {
	bs := make([]byte, EPlayerMoved.Len())
	bs[0] = byte(c.Dir)
	binary.BigEndian.PutUint16(bs[1:3], c.ID)
	binary.BigEndian.PutUint32(bs[3:7], uint32(c.Pos.X))
	binary.BigEndian.PutUint32(bs[7:11], uint32(c.Pos.Y))
	return bs
}

type EventPlayerSpell struct {
	ID     uint16
	Spell  spell.Spell
	Killed bool
}

func DecodeEventPlayerSpell(data []byte) *EventPlayerSpell {
	return &EventPlayerSpell{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Spell:  spell.Spell(data[2]),
		Killed: data[3] != 0,
	}
}

func EncodeEventPlayerSpell(c *EventPlayerSpell) []byte {
	bs := make([]byte, EPlayerSpell.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	bs[2] = byte(c.Spell)
	if c.Killed {
		bs[3] = 1
	}
	return bs
}

type EventPlayerMelee struct {
	From   uint16
	ID     uint16
	Hit    bool
	Killed bool
}

func DecodeEventPlayerMelee(data []byte) *EventPlayerMelee {
	return &EventPlayerMelee{
		From:   binary.BigEndian.Uint16(data[:2]),
		ID:     binary.BigEndian.Uint16(data[2:4]),
		Hit:    data[4] != 0,
		Killed: data[5] != 0,
	}
}

func EncodeEventPlayerMelee(c *EventPlayerMelee) []byte {
	bs := make([]byte, EPlayerMelee.Len())
	binary.BigEndian.PutUint16(bs[:2], c.From)
	binary.BigEndian.PutUint16(bs[2:4], c.ID)
	if c.Hit {
		bs[4] = 1
	}
	if c.Killed {
		bs[5] = 1
	}
	return bs
}

type EventPlayerSpellRecieved struct {
	ID     uint16
	Spell  spell.Spell
	Damage uint32
	NewHP  uint32
}

func DecodeEventPlayerSpellRecieved(data []byte) *EventPlayerSpellRecieved {
	return &EventPlayerSpellRecieved{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Spell:  spell.Spell(data[2]),
		Damage: binary.BigEndian.Uint32(data[3:7]),
		NewHP:  binary.BigEndian.Uint32(data[7:11]),
	}
}

func EncodeEventPlayerSpellRecieved(c *EventPlayerSpellRecieved) []byte {
	bs := make([]byte, EPlayerSpellRecieved.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	bs[2] = byte(c.Spell)
	binary.BigEndian.PutUint32(bs[3:7], c.Damage)
	binary.BigEndian.PutUint32(bs[7:11], c.NewHP)
	return bs
}

type EventPlayerMeleeRecieved struct {
	ID     uint16
	Damage uint32
	NewHP  uint32
}

func DecodeEventPlayerMeleeRecieved(data []byte) *EventPlayerMeleeRecieved {
	return &EventPlayerMeleeRecieved{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		NewHP:  binary.BigEndian.Uint32(data[6:10]),
	}
}

func EncodeEventPlayerMeleeRecieved(c *EventPlayerMeleeRecieved) []byte {
	bs := make([]byte, EPlayerMeleeRecieved.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	binary.BigEndian.PutUint32(bs[2:6], c.Damage)
	binary.BigEndian.PutUint32(bs[6:10], c.NewHP)
	return bs
}
