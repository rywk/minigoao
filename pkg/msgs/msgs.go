package msgs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
	"unsafe"

	"github.com/rywk/minigoao/pkg/constants/attack"
	"github.com/rywk/minigoao/pkg/constants/direction"
	"github.com/rywk/minigoao/pkg/constants/item"
	"github.com/rywk/minigoao/pkg/constants/mapdef"
	"github.com/rywk/minigoao/pkg/constants/skill"
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
		log.Printf("BAD 0 byte!!!! %v", err)

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
	bsLeft := int(msgSize)
	i := 0
	for bsLeft > 0 {
		n, err := r.Read(incd.Data[i:])
		if err != nil {
			return incd, err
		}
		bsLeft -= n
		i += n
	}

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
		return nil
	}
	buf = append(buf, data...)
	bsLeft := len(buf)
	i := 0
	for bsLeft > 0 {
		n, err := w.Write(buf[i:])
		if err != nil {
			return err
		}
		bsLeft -= n
		i += n
	}
	return nil
}

// Write sends the event to the connection
func writeWithLen(w io.Writer, event E, data []byte) error {
	buff := make([]byte, 3)
	buff[0] = byte(event)
	binary.BigEndian.PutUint16(buff[1:], uint16(len(data)))
	buff = append(buff, data...)
	bsLeft := len(buff)
	i := 0
	for bsLeft > 0 {
		n, err := w.Write(buff[i:])
		if err != nil {
			return err
		}
		bsLeft -= n
		i += n
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
	ELoginCharacter
	ECreateCharacter
	ECreateAccount
	ELoginAccount
	EServerDisconnect
	EMove
	ECastSpell
	EMelee
	EUseItem
	ESendChat
	ESelectSpell
	EUpdateSkills
	EUpdateKeyConfig
	EGetRankList
	ERankList

	EAccountLoginOk
	ECharLogoutOk
	EPingOk
	EMoveOk
	ECastSpellOk
	EMeleeOk
	EUseItemOk
	EUpdateSkillsOk
	ETpTo

	EPlayerMeditating
	EPlayerChangedSkin   // A Player in the viewport changed a part of how other players view it
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

	EError
	ELen
)

func (e E) U8() uint8 { return uint8(e) }

const mapCoordinateSize = int(unsafe.Sizeof(uint32(0)))

var eventLen = [ELen]int{
	0,
	1,     // EPing
	-1,    // ELoginCharacter
	-1,    // ECreateCharacter
	-1,    // ECreateAccount
	-1,    // ELoginAccount
	0,     // EServerDisconnect
	1,     // EMove - 1 byte (uint8) to define the direction.
	4 * 2, // ECastSpell - 1 byte (uint8) to define the spell picked in the client side. x, y map coords are 2 uint32
	1,     // EMelee - signals user used the melee key
	2,     // EUseItem - 2 byte (byte) x,y inventory
	-1,    // ESendChat
	1,     // ESelectSpell 1 byte (uint8) to define spell
	-1,    // EUpdateSkills
	-1,    // EUpdateKeyConfig
	1,     // EGetRankList
	-1,    // ERankList

	-1,                // EAccountLoginOk
	1,                 // ECharLogoutOk
	2,                 // EPingOk
	2,                 // EMoveOk - 1 byte (bool) move, 1 byte (bool) direction
	1 + 2 + 4 + 4 + 1, // ECastSpellOk - 1 byte (uint8) spell, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage,  4 bytes (uint32) new mp,  1 byte (bool) killed target
	1 + 1 + 1 + 2 + 4, // EMeleeOk -  1 byte (uint8) direction,  1 byte (bool) hit/miss, 1 byte (bool) killed target, 2 bytes (uint16) to define the player id, 4 bytes (uint32) damage
	1 + 4 + 2 + 2,     // EUseItemOk - 1 byte (uint8) item, 4 byte (uint32) to define value changed (mana/health), new item count uint16, the slot used
	-1,                // EUpdateSkillsOk
	-1,                // ETpTo

	3,  // EPlayerMeditating
	6,  // EPlayerChangedSkin - 2 bytes (uint16) to define the player id 4 for the new skin
	0,  // EPlayerConnect
	-1, // EPlayerLogin - -1 dynamic size msgpack
	1,  // EPlayerLogout
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

	-1, // EError
}

var eventString = [ELen]string{
	"ENone",
	"EPing",
	"ELoginCharacter",
	"ECreateCharacter",
	"ECreateAccount",
	"ELoginAccount",
	"EServerDisconnect",
	"EMove",
	"ECastSpell",
	"EMelee",
	"EUseItem",
	"ESelectSpell",
	"EUpdateSkills",
	"EUpdateKeyConfig",
	"EGetRankList",
	"ERankList",

	"EAccountLoginOk",
	"ECharLogoutOk",
	"EPingOk",
	"EMoveOk",
	"ECastSpellOk",
	"EMeleeOk",
	"EUseItemOk",
	"EUpdateSkillsOk",
	"ETpTo",

	"EPlayerMeditating",
	"EPlayerChangedSkin",
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

	"EError",
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
	case ELoginCharacter:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventLoginCharacter)))
	case ECreateCharacter:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventCreateCharacter)))
	case ECreateAccount:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventCreateAccount)))
	case ELoginAccount:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventLoginAccount)))
	case EMove:
		return m.Write(e, []byte{msg.(uint8)})
	case ECastSpell:
		return m.Write(e, EncodeEventCastSpell(msg.(*EventCastSpell)))
	case EMelee:
		return m.Write(e, []byte{msg.(uint8)})
	case EUseItem:
		return m.Write(e, EncodeEventUseItem(msg.(*EventUseItem)))
	case ESendChat:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventSendChat)))
	case ESelectSpell:
		return m.Write(e, []byte{byte(msg.(attack.Spell))})
	case EUpdateSkills:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*skill.Skills)))
	case EUpdateKeyConfig:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*KeyConfig)))
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
	case EUpdateSkillsOk:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*Experience)))
	case EAccountLoginOk:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventAccountLogin)))
	case ETpTo:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventPlayerTp)))
	case ECharLogoutOk:
		return m.Write(e, []byte{0})
	case EPlayerChangedSkin:
		return m.Write(e, EncodeEventPlayerChangedSkin(msg.(*EventPlayerChangedSkin)))
	case EPlayerMeditating:
		return m.Write(e, EncodeEventPlayerMeditating(msg.(*EventPlayerMeditating)))
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
	case EError:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventError)))
	case EGetRankList:
		return m.Write(e, []byte{0})
	case ERankList:
		return m.WriteWithLen(e, EncodeMsgpack(msg.(*EventRankList)))

	default:
		log.Printf("unknown event %v\n", e.String())
		return fmt.Errorf("unknown event %v", e.String())
	}
}
func (m *M) EncodeAndWrite(e E, msg interface{}) error {
	return encodeAndWrite(m, e, msg)
}

type EventPlayerMeditating struct {
	ID         uint16
	Meditating bool
}

func DecodeEventPlayerMeditating(data []byte) *EventPlayerMeditating {
	return &EventPlayerMeditating{
		ID:         binary.BigEndian.Uint16(data[0:2]),
		Meditating: data[2] == 1,
	}
}

func EncodeEventPlayerMeditating(c *EventPlayerMeditating) []byte {
	bs := make([]byte, EPlayerMeditating.Len())
	binary.BigEndian.PutUint16(bs[0:2], c.ID)
	if c.Meditating {
		bs[2] = 1
	}
	return bs
}

type EventRankList struct {
	Kills    []RankChar
	Arena1v1 []RankChar
	Arena2v2 []RankChar
}

type RankChar struct {
	ID     int
	Nick   string
	Kills  int
	Deaths int
}

func (c *Character) ToRankChar() RankChar {
	return RankChar{
		ID:     c.ID,
		Nick:   c.Nick,
		Kills:  c.Kills,
		Deaths: c.Deaths,
	}
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

type EventLoginCharacter struct {
	ID uint16
}
type EventCreateCharacter struct {
	AccountID uint16
	Nick      string
}
type EventCreateAccount struct {
	Account  string
	Email    string
	Password string
}
type EventLoginAccount struct {
	Account  string
	Password string
}
type EventError struct {
	Msg string
}
type EventAccountLogin struct {
	ID         uint16
	Account    string
	Email      string
	Characters []Character
}

// msgpack
type EventPlayerLogin struct {
	ID             uint16
	Nick           string
	Pos            typ.P
	MapType        mapdef.MapType
	Dir            direction.D
	Speed          uint8
	Dead           bool
	HP             int32
	MP             int32
	Inv            Inventory
	Exp            Experience
	VisiblePlayers []EventNewPlayer
	KeyConfig      KeyConfig
}
type EventPlayerTp struct {
	MapType        mapdef.MapType
	Pos            typ.P
	Dir            direction.D
	Dead           bool
	HP             int32
	MP             int32
	Inv            Inventory
	Exp            Experience
	VisiblePlayers []EventNewPlayer
}
type Inventory struct {
	HealthPotions  InventoryPos
	ManaPotions    InventoryPos
	EquippedHead   InventoryPos
	EquippedBody   InventoryPos
	EquippedWeapon InventoryPos
	EquippedShield InventoryPos
	Slots          [8][2]ItemSlot
}

func NewInvetory() *Inventory {
	return &Inventory{
		HealthPotions:  EmptyInventoryPos(),
		ManaPotions:    EmptyInventoryPos(),
		EquippedHead:   EmptyInventoryPos(),
		EquippedBody:   EmptyInventoryPos(),
		EquippedWeapon: EmptyInventoryPos(),
		EquippedShield: EmptyInventoryPos(),
	}
}

func EmptyInventoryPos() InventoryPos {
	return InventoryPos{X: 255, Y: 255}
}
func (in *Inventory) UnequipAll() {
	in.EquippedBody = EmptyInventoryPos()
	in.EquippedWeapon = EmptyInventoryPos()
	in.EquippedHead = EmptyInventoryPos()
	in.EquippedShield = EmptyInventoryPos()
}
func (in *Inventory) GetWeapon() item.Item {
	slot := in.EquippedWeapon
	if slot.X == 255 {
		return item.None
	}
	return in.Slots[slot.X][slot.Y].Item
}
func (in *Inventory) GetShield() item.Item {
	slot := in.EquippedShield
	if slot.X == 255 {
		return item.None
	}
	return in.Slots[slot.X][slot.Y].Item
}
func (in *Inventory) GetBody() item.Item {
	slot := in.EquippedBody
	if slot.X == 255 {
		return item.None
	}
	return in.Slots[slot.X][slot.Y].Item
}
func (in *Inventory) GetHead() item.Item {
	slot := in.EquippedHead
	if slot.X == 255 {
		return item.None
	}
	return in.Slots[slot.X][slot.Y].Item
}
func (in *Inventory) GetSlot(slot typ.P) ItemSlot {
	if slot.X == -1 {
		return ItemSlot{}
	}
	return in.Slots[slot.X][slot.Y]
}

func (in *Inventory) GetSlotf(slot *EventUseItem) *ItemSlot {
	return &in.Slots[slot.X][slot.Y]
}
func (in *Inventory) GetSlotv2(slot *InventoryPos) *ItemSlot {
	return &in.Slots[slot.X][slot.Y]
}
func (in *Inventory) Range(fn func(i int, it *ItemSlot) bool) {
	for i := range in.Slots {
		if !fn(i, &in.Slots[i][0]) {
			break
		}
		if !fn(i, &in.Slots[i][1]) {
			break
		}
	}
}

type InventoryPos struct {
	X, Y uint8
}
type ItemSlot struct {
	Item  item.Item
	Count uint16
}
type Experience struct {
	FreePoints int32
	Skills     skill.Skills
	ItemBuffs  skill.Buffs
	SkillBuffs skill.Buffs
	Stats      skill.Stats
}

type SpellData struct {
	//SpellType uint8
	Damage   int32
	Cooldown time.Duration
	ManaCost int32
}

type ItemData struct {
	Item       item.Item
	WeaponData WeaponData
	ArmorData  ArmorData
	ShieldData ShieldData
	HelmetData HelmetData
}
type WeaponData struct {
	//WeaponType uint8
	Damage      int32
	Cooldown    time.Duration
	CriticRange int32
}
type ArmorData struct {
	//WeaponType uint8
	PhysicalDef int32
	MagicDef    int32
}
type ShieldData struct {
	//WeaponType uint8
	PhysicalDef int32
	MagicDef    int32
}
type HelmetData struct {
	//WeaponType uint8
	PhysicalDef int32
	MagicDef    int32
}

func DecodeMsgpack[T any](data []byte, to *T) *T {
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

	Weapon item.Item
	Shield item.Item
	Body   item.Item
	Head   item.Item
}

type EventPlayerSpawned = EventNewPlayer

type EventPlayerEnterViewport = EventNewPlayer

// binary

type EventCastSpell struct {
	PX, PY uint32
}

func DecodeEventCastSpell(data []byte) *EventCastSpell {
	return &EventCastSpell{
		PX: binary.BigEndian.Uint32(data[0:4]),
		PY: binary.BigEndian.Uint32(data[4:8]),
	}
}

func EncodeEventCastSpell(c *EventCastSpell) []byte {
	bs := make([]byte, ECastSpell.Len())
	binary.BigEndian.PutUint32(bs[0:4], c.PX)
	binary.BigEndian.PutUint32(bs[4:8], c.PY)
	return bs
}

type EventCastSpellOk struct {
	ID     uint16
	Damage uint32
	NewMP  uint32
	Spell  attack.Spell
	Killed bool
}

func DecodeEventCastSpellOk(data []byte) *EventCastSpellOk {
	return &EventCastSpellOk{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Damage: binary.BigEndian.Uint32(data[2:6]),
		NewMP:  binary.BigEndian.Uint32(data[6:10]),
		Spell:  attack.Spell(data[10]),
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

type EventPlayerChangedSkin struct {
	ID     uint16
	Armor  item.Item
	Head   item.Item
	Weapon item.Item
	Shield item.Item
}

func DecodeEventPlayerChangedSkin(data []byte) *EventPlayerChangedSkin {
	return &EventPlayerChangedSkin{
		ID:     binary.BigEndian.Uint16(data[0:2]),
		Armor:  item.Item(data[2]),
		Head:   item.Item(data[3]),
		Weapon: item.Item(data[4]),
		Shield: item.Item(data[5]),
	}
}

func EncodeEventPlayerChangedSkin(c *EventPlayerChangedSkin) []byte {
	bs := make([]byte, EPlayerChangedSkin.Len())
	binary.BigEndian.PutUint16(bs[0:2], c.ID)
	bs[2] = byte(c.Armor)
	bs[3] = byte(c.Head)
	bs[4] = byte(c.Weapon)
	bs[5] = byte(c.Shield)
	return bs
}

type EventUseItemOk struct {
	Slot   InventoryPos
	Item   item.Item
	Change uint32
	Count  uint16
}

func DecodeEventUseItemOk(data []byte) *EventUseItemOk {
	return &EventUseItemOk{
		Item:   item.Item(data[0]),
		Change: binary.BigEndian.Uint32(data[1:5]),
		Count:  binary.BigEndian.Uint16(data[5:7]),
		Slot: InventoryPos{
			X: data[7],
			Y: data[8],
		},
	}
}

func EncodeEventUseItemOk(c *EventUseItemOk) []byte {
	bs := make([]byte, EUseItemOk.Len())
	bs[0] = byte(c.Item)
	binary.BigEndian.PutUint32(bs[1:5], c.Change)
	binary.BigEndian.PutUint16(bs[5:7], c.Count)
	bs[7] = c.Slot.X
	bs[8] = c.Slot.Y
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
	Spell  attack.Spell
	Killed bool
}

func DecodeEventPlayerSpell(data []byte) *EventPlayerSpell {
	return &EventPlayerSpell{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Spell:  attack.Spell(data[2]),
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
	Spell  attack.Spell
	Damage uint32
	NewHP  uint32
}

func DecodeEventPlayerSpellRecieved(data []byte) *EventPlayerSpellRecieved {
	return &EventPlayerSpellRecieved{
		ID:     binary.BigEndian.Uint16(data[:2]),
		Spell:  attack.Spell(data[2]),
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

type EventUseItem InventoryPos

func DecodeEventUseItem(data []byte) *EventUseItem {
	return &EventUseItem{
		X: data[0],
		Y: data[1],
	}
}

func EncodeEventUseItem(c *EventUseItem) []byte {
	bs := make([]byte, EUseItem.Len())
	bs[0] = c.X
	bs[1] = c.Y
	return bs
}

type KeyConfig struct {
	Front Input
	Back  Input
	Left  Input
	Right Input

	PotionHP Input
	PotionMP Input

	Melee Input

	// Spell picker
	PickParalize          Input
	PickParalizeRm        Input
	PickExplode           Input
	PickElectricDischarge Input
	PickResurrect         Input
	PickHealWounds        Input
}

type Input struct {
	Mouse    int16
	Keyboard int16
}

type Character struct {
	ID        int
	AccountID int
	Nick      string
	Dir       direction.D
	Px        int
	Py        int
	Kills     int
	Deaths    int
	WinsVOne  int
	LosesVOne int
	WinsVTwo  int
	LosesVTwo int
	Skills    skill.Skills
	Inventory Inventory
	KeyConfig KeyConfig
	LoggedIn  bool
}

func (inv *Inventory) SetTestItemsInventory() {
	inv.Slots[7][0].Item = item.HealthPotion
	inv.Slots[7][0].Count = 9999
	inv.HealthPotions = InventoryPos{X: 7, Y: 0}
	inv.Slots[6][0].Item = item.WeaponMightySword
	inv.Slots[6][0].Count = 1
	inv.Slots[5][0].Item = item.WeaponWindSword
	inv.Slots[5][0].Count = 1
	inv.Slots[4][0].Item = item.WeaponDarkDagger
	inv.Slots[4][0].Count = 1
	inv.Slots[3][0].Item = item.WeaponFireStaff
	inv.Slots[3][0].Count = 1

	inv.Slots[7][1].Item = item.ManaPotion
	inv.Slots[7][1].Count = 9999
	inv.ManaPotions = InventoryPos{X: 7, Y: 1}

	inv.Slots[4][1].Item = item.HelmetPaladin
	inv.Slots[4][1].Count = 1
	inv.Slots[1][1].Item = item.HatMage
	inv.Slots[1][1].Count = 1
	inv.Slots[2][1].Item = item.ShieldArcane
	inv.Slots[2][1].Count = 1
	inv.Slots[3][1].Item = item.ArmorShadow
	inv.Slots[3][1].Count = 1
	inv.Slots[6][1].Item = item.ArmorDark
	inv.Slots[6][1].Count = 1
	inv.Slots[5][1].Item = item.ShieldTower
	inv.Slots[5][1].Count = 1
}
