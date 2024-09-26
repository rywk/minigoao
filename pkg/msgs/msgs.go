package msgs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"unsafe"

	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/spell"
	"github.com/rywk/minigoao/pkg/typ"
	"github.com/vmihailenco/msgpack/v5"
)

type Msgs interface {
	IP() string
	Close()
	Read() (*IncomingData, error)
	Write(event E, data []byte) error
	WriteWithLen(event E, data []byte) error
	EncodeAndWrite(e E, msg interface{}) error
}

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

func (m *M) Close() {
	m.c.Close()
}

var ErrBadData = errors.New("bad data")

func readMsg(r io.Reader) (*IncomingData, error) {
	eventByte := make([]byte, eventTypeLen)

	_, err := r.Read(eventByte)
	if err != nil {
		log.Printf("BAD 0 byte!!!! %d", err)

		return nil, err
	}
	event := E(eventByte[0])
	if !event.Valid() {
		log.Printf("Invalid event byte!!!! %d", event)
		return nil, ErrBadData
	}

	incd := &IncomingData{Event: E(eventByte[0])}
	if event == ENone {
		log.Printf("Event none error %s %s", incd.Event.String(), err)

		return nil, nil
	}
	if incd.Event.Len() == 0 {
		log.Print("0b read")
		return incd, nil
	}
	if incd.Event.Len() != -1 {
		incd.Data = make([]byte, incd.Event.Len())
		_, err = r.Read(incd.Data)
		return incd, err
	}

	msgSizeBs := make([]byte, 2)
	_, err = r.Read(msgSizeBs)
	if err != nil {
		return nil, err
	}
	msgSize := binary.BigEndian.Uint16(msgSizeBs)
	incd.Data = make([]byte, msgSize)
	_, err = r.Read(incd.Data)
	return incd, err
}

// Read blocks until a new event is read from the tcp byte stream.
func (m *M) Read() (*IncomingData, error) {
	return readMsg(m.c)
}

// Write sends the event to the connection
func write(w io.Writer, event E, data []byte) error {
	buf := []byte{byte(event)}
	if data == nil {
		_, err := w.Write(buf)
		if err != nil {
			return err
		}
	}
	_, err := w.Write(append(buf, data...))
	if err != nil {
		return err
	}
	return nil
}

// Write sends the event to the connection
func writeWithLen(w io.Writer, event E, data []byte) error {
	pref := make([]byte, 3)
	pref[0] = byte(event)
	binary.BigEndian.PutUint16(pref[1:], uint16(len(data)))
	_, err := w.Write(append(pref, data...))
	if err != nil {
		return err
	}
	return nil
}

// Write sends the event to the connection
func (m *M) Write(event E, data []byte) error {
	return write(m.c, event, data)
}

// Write sends the event to the connection
func (m *M) WriteWithLen(event E, data []byte) error {
	return writeWithLen(m.c, event, data)
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
	ENone E = iota
	EPing
	ERegister
	EServerDisconnect
	EMove
	ECastSpell
	EMelee
	EUseItem
	ESendChat

	EPingOk
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
	EBroadcastChat       // A player chatted in viewport

	EPlayerMoved         // A Player in the viewport moved
	EPlayerSpell         // A Player in the viewport recieved a spell
	EPlayerSpellRecieved // Player recieved a spell
	EPlayerMelee         // A Player in the viewport recieved a melee
	EPlayerMeleeRecieved // Player recieved a melee

	ELen
)

const mapCoordinateSize = int(unsafe.Sizeof(uint32(0)))

var eventLen = [ELen]int{
	0,
	1,       // EPing
	-1,      // ERegister
	0,       // EServerDisconnect
	1,       // EMove - 1 byte (uint8) to define the direction.
	1 + 4*2, // ECastSpell - 1 byte (uint8) to define the spell picked in the client side. x, y map coords are 2 uint32
	1,       // EMelee - signals user used the melee key
	1,       // EUseItem - 1 byte (uint8) to define the item id
	-1,      // ESendChat

	2,                 // EPingOk
	2,                 // EMoveOk - 1 byte (bool) move, 1 byte (bool) direction
	1 + 2 + 4 + 4 + 1, // ECastSpellOk - 1 byte (uint8) spell, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage,  4 bytes (uint32) new mp,  1 byte (bool) killed target
	1 + 1 + 1 + 2 + 4, // EMeleeOk -  1 byte (uint8) direction,  1 byte (bool) hit/miss, 1 byte (bool) killed target, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage
	1 + 4,             // EUseItemOk - 1 byte (uint8) item, 4 byte (uint32) to define value changed (mana/health)

	0,  // EPlayerConnect
	-1, // EPlayerLogin - -1 dynamic size msgpack
	0,  // EPlayerLogout
	-1, // EPlayerSpawned - -1 dynamic size msgpack
	2,  // EPlayerDespawned - 2 bytes (uint16) to define the player id
	-1, // EPlayerEnterViewport - -1 dynamic size msgpack
	2,  // EPlayerLeaveViewport - 2 bytes (uint16) to define the player id
	-1, // EBroadcastChat

	11,                // EPlayerMoved - 1 byte (uint8) direction, 2 bytes (uint16) player id, 8 bytes (uint32, uint32) x y
	2 + 1 + 1,         // EPlayerSpell - 2 bytes (uint16) to define the target player id, 1 byte (uint8) to define the spell, 1 byte (bool) killed target
	1 + 2 + 4 + 4,     // EPlayerSpellRecieved - 1 byte (uint8) to define the spell, 2 bytes (uint16) to define the (caster) player id, 4 bytes (uint32) to define the new hp, 4 bytes (uint32) to define the damage
	1 + 1 + 1 + 2 + 2, // EPlayerMelee - 1 byte (bool) hit/miss, 1 byte (bool) killed target, 2 bytes (uint16) to define the target player id, 2 bytes (uint16) to define the attacker
	1 + 2 + 4 + 4,     // EPlayerMeleeRecieved - 2 bytes (uint16) to define the (caster) player id, 4 bytes (uint32) to define the new hp, 4 bytes (uint32) to define the damage
}

var eventString = [ELen]string{
	"ENone",
	"EPing",
	"ERegister",
	"EServerDisconnect",
	"EMove",
	"ECastSpell",
	"EMelee",
	"EUseItem",

	"EPingOk",
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

func (e E) Valid() bool {
	return e < ELen
}

func (e E) Len() int {
	return eventLen[e]
}

func (e E) String() string {
	return eventString[e]
}

func encodeAndWrite(m Msgs, e E, msg interface{}) error {
	switch e {
	case EPing:
		return m.Write(e, make([]byte, 1))
	case ERegister:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventRegister)))
	case EMove:
		return m.Write(e, []byte{msg.(uint8)})
	case ECastSpell:
		return m.Write(e, EncodeEventCastSpell(msg.(*EventCastSpell)))
	case EMelee:
		return m.Write(e, []byte{msg.(uint8)})
	case EUseItem:
		return m.Write(e, []byte{byte(msg.(Item))})
	case ESendChat:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventSendChat)))
	case EPingOk:
		return m.Write(e, binary.BigEndian.AppendUint16(make([]byte, 0, 2), msg.(uint16)))
	case EMoveOk:
		return m.Write(e, msg.([]byte))
	case ECastSpellOk:
		return m.Write(e, EncodeEventCastSpellOk(msg.(*EventCastSpellOk)))
	case EMeleeOk:
		return m.Write(e, EncodeEventMeleeOk(msg.(*EventMeleeOk)))
	case EUseItemOk:
		return m.Write(e, EncodeEventUseItemOk(msg.(*EventUseItemOk)))
	case EPlayerSpawned:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventPlayerSpawned)))
	case EPlayerDespawned:
		return m.Write(e, binary.BigEndian.AppendUint16(make([]byte, 0, 2), msg.(uint16)))
	case EPlayerEnterViewport:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventPlayerEnterViewport)))
	case EPlayerLeaveViewport:
		return m.Write(e, binary.BigEndian.AppendUint16(make([]byte, 0, 2), msg.(uint16)))
	case EBroadcastChat:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventBroadcastChat)))
	case EPlayerMoved:
		return m.Write(e, EncodeEventPlayerMoved(msg.(*EventPlayerMoved)))
	case EPlayerSpell:
		return m.Write(e, EncodeEventPlayerSpell(msg.(*EventPlayerSpell)))
	case EPlayerSpellRecieved:
		return m.Write(e, EncodeEventPlayerSpellRecieved(msg.(*EventPlayerSpellRecieved)))
	case EPlayerMelee:
		return m.Write(e, EncodeEventPlayerMelee(msg.(*EventPlayerMelee)))
	case EPlayerMeleeRecieved:
		return m.Write(e, EncodeEventPlayerMeleeRecieved(msg.(*EventPlayerMeleeRecieved)))
	default:
		log.Printf("unknown event %v\n", e.String())
		return fmt.Errorf("unknown event %v", e.String())
	}
}
func (m *M) EncodeAndWrite(e E, msg interface{}) error {
	return encodeAndWrite(m, e, msg)
}

type EventSendChat struct {
	Msg string
}

type EventBroadcastChat struct {
	ID  uint16
	Msg string
}

func BoolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

type EventRegister struct {
	Nick string
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

func DecodeMsgpack[T any](data []byte, to *T) *T {
	log.Print(len(data))
	err := msgpack.Unmarshal(data, to)
	if err != nil {
		panic(err)
	}
	return to
}

func EncodeMsgpack[T any](t *T) []byte {
	data, err := msgpack.Marshal(t)
	if err != nil {
		panic(err)
	}
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

const (
	ItemNone Item = iota
	ItemManaPotion
	ItemHealthPotion
	ItemLen
)

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
	Dir    direction.D
}

func DecodeEventMeleeOk(data []byte) *EventMeleeOk {
	return &EventMeleeOk{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		Hit:    data[6] != 0,
		Killed: data[7] != 0,
		Dir:    data[8],
	}
}

func EncodeEventMeleeOk(c *EventMeleeOk) []byte {
	bs := make([]byte, EMeleeOk.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	binary.BigEndian.PutUint32(bs[2:6], c.Damage)
	bs[6] = BoolByte(c.Hit)
	bs[7] = BoolByte(c.Killed)
	bs[8] = c.Dir
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
	Dir    direction.D
}

func DecodeEventPlayerMelee(data []byte) *EventPlayerMelee {
	return &EventPlayerMelee{
		From:   binary.BigEndian.Uint16(data[:2]),
		ID:     binary.BigEndian.Uint16(data[2:4]),
		Hit:    data[4] != 0,
		Killed: data[5] != 0,
		Dir:    data[6],
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
	bs[6] = c.Dir
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
	Dir    direction.D
}

func DecodeEventPlayerMeleeRecieved(data []byte) *EventPlayerMeleeRecieved {
	return &EventPlayerMeleeRecieved{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		NewHP:  binary.BigEndian.Uint32(data[6:10]),
		Dir:    data[10],
	}
}

func EncodeEventPlayerMeleeRecieved(c *EventPlayerMeleeRecieved) []byte {
	bs := make([]byte, EPlayerMeleeRecieved.Len())
	binary.BigEndian.PutUint16(bs[:2], c.ID)
	binary.BigEndian.PutUint32(bs[2:6], c.Damage)
	binary.BigEndian.PutUint32(bs[6:10], c.NewHP)
	bs[10] = c.Dir
	return bs
}
