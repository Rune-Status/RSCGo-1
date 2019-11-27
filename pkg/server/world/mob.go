package world

import (
	"github.com/spkaeros/rscgo/pkg/rand"
	"github.com/spkaeros/rscgo/pkg/server/log"
	"go.uber.org/atomic"
	"sync"
	"time"
)

const (
	//MSIdle The default MobState, means doing nothing.
	MSIdle int = 0
	//MSBanking The mob is banking.
	MSBanking = 1
	//MSChatting The mob is chatting with a NPC
	MSChatting = 2
	//MSMenuChoosing The mob is in a query menu
	MSMenuChoosing = 4
	//MSTrading The mob is negotiating a trade.
	MSTrading = 8
	//MSDueling The mob is negotiating a duel.
	MSDueling = 16
	//MSFighting The mob is fighting.
	MSFighting = 32
	//MSBatching The mob is performing a skill that repeats itself an arbitrary number of times.
	MSBatching = 64
	//MSSleeping The mob is using a bed or sleeping bag, and trying to solve a CAPTCHA
	MSSleeping = 128
	//MSBusy Generic busy state
	MSBusy = 256
	//MSChangingAppearance Indicates that the mob in this state is in the player aooearance changing screen
	MSChangingAppearance = 512
)

//Mob Represents a mobile entity within the game world.
type Mob struct {
	*Entity
	TransAttrs *AttributeList
}

type MobileEntity interface {
	X() int
	Y() int
	Skills() *SkillTable
	MeleeDamage(target MobileEntity) int
	Defense(float32) float32
	Transients() *AttributeList
	IsFighting() bool
	FightTarget() MobileEntity
	SetFightTarget(MobileEntity)
	FightRound() int
	SetFightRound(int)
	ResetFighting()
	HasState(int) bool
	AddState(int)
	RemoveState(int)
	State() int
	Busy() bool
	Move()
	Remove()
	SetX(int)
	SetY(int)
	SetCoords(int, int, bool)
	Teleport(int, int)
	Direction() int
	SetDirection(int)
	Change()
	ResetMoved()
	ResetRemoved()
	ResetChanged()
	Path() *Pathway
	ResetPath()
	SetPath(*Pathway)
	TraversePath()
	UpdateSelf()
	ResetNeedsSelf()
	FinishedPath() bool
	SetLocation(Location, bool)
	UpdateLastFight()
	LastFight() time.Time
	UpdateLastRetreat()
	LastRetreat() time.Time
}

func (m *Mob) Transients() *AttributeList {
	return m.TransAttrs
}

//Busy Returns true if this mobs state is anything other than idle. otherwise returns false.
func (m *Mob) Busy() bool {
	return m.State() != MSIdle
}

func (m *Mob) IsFighting() bool {
	return m.HasState(MSFighting) && m.Transients().VarMob("fightTarget") != nil
}

func (m *Mob) FightTarget() MobileEntity {
	return m.TransAttrs.VarMob("fightTarget")
}

func (m *Mob) SetFightTarget(m2 MobileEntity) {
	m.TransAttrs.SetVar("fightTarget", m2)
}

func (m *Mob) FightRound() int {
	return m.TransAttrs.VarInt("fightRound", 0)
}

func (m *Mob) SetFightRound(i int) {
	m.TransAttrs.SetVar("fightRound", i)
}

func (m *Mob) LastRetreat() time.Time {
	return m.TransAttrs.VarTime("lastRetreat")
}

func (m *Mob) LastFight() time.Time {
	return m.TransAttrs.VarTime("lastFight")
}

func (m *Mob) UpdateLastRetreat() {
	m.TransAttrs.SetVar("lastRetreat", time.Now())
}

func (m *Mob) UpdateLastFight() {
	m.TransAttrs.SetVar("lastFight", time.Now())
}

//Direction Returns the mobs direction.
func (m *Mob) Direction() int {
	return m.TransAttrs.VarInt("direction", North)
}

//SetDirection Sets the mobs direction.
func (m *Mob) SetDirection(direction int) {
	m.Change()
	m.TransAttrs.SetVar("direction", direction)
}

//Change Sets the synchronization flag for whether this mob changed directions to true.
func (m *Mob) Change() {
	m.TransAttrs.SetVar("changed", true)
}

//Remove Sets the synchronization flag for whether this mob needs to be removed to true.
func (m *Mob) Remove() {
	m.TransAttrs.SetVar("remove", true)
}

//UpdateSelf Sets the synchronization flag for whether this mob needs to update itself to true.
func (m *Mob) UpdateSelf() {
	m.TransAttrs.SetVar("self", true)
}

//UpdateSelf Sets the synchronization flag for whether this mob has moved to true.
func (m *Mob) Move() {
	m.TransAttrs.SetVar("moved", true)
}

func (m *Mob) ResetMoved() {
	m.TransAttrs.UnsetVar("moved")
}

func (m *Mob) ResetRemoved() {
	m.TransAttrs.UnsetVar("remove")
}

func (m *Mob) ResetNeedsSelf() {
	m.TransAttrs.UnsetVar("self")
}

func (m *Mob) ResetChanged() {
	m.TransAttrs.UnsetVar("changed")
}

//SetPath Sets the mob's current pathway to path.  If path is nil, effectively resets the mobs path.
func (m *Mob) SetPath(path *Pathway) {
	m.TransAttrs.SetVar("path", path)
}

func (m *Mob) WalkTo(end Location) {
	path := MakePath(m.Location, end)
	m.SetPath(path)
}

//Path returns the path that this mob is trying to traverse.
func (m *Mob) Path() *Pathway {
	return m.TransAttrs.VarPath("path")
}

//ResetPath Sets the mobs path to nil, to stop the traversal of the path instantly
func (m *Mob) ResetPath() {
	m.ResetMoved()
	m.TransAttrs.UnsetVar("path")
}

//TraversePath If the mob has a path, calling this method will change the mobs location to the next location described by said Path data structure.  This should be called no more than once per game tick.
func (p *Player) TraversePath() {
	path := p.Path()
	if path == nil {
		return
	}
	if p.AtLocation(path.NextWaypointTile()) {
		path.CurrentWaypoint++
	}
	if p.FinishedPath() {
		p.ResetPath()
		return
	}
	dst := path.NextWaypointTile()
	x, y := p.X(), p.Y()
	next := NewLocation(x, y)
	xBlocked, yBlocked := false, false
	newXBlocked, newYBlocked := false, false
	if y > dst.Y() {
		yBlocked = IsTileBlocking(x, y, 1, true)
		newYBlocked = IsTileBlocking(x, y-1, 4, false)
		if !newYBlocked {
			next.y.Dec()
		}
	} else if y < dst.Y() {
		yBlocked = IsTileBlocking(x, y, 4, true)
		newYBlocked = IsTileBlocking(x, y+1, 1, false)
		if !newYBlocked {
			next.y.Inc()
		}
	}
	if x > dst.X() {
		xBlocked = IsTileBlocking(x, next.Y(), 2, true)
		newXBlocked = IsTileBlocking(x-1, next.Y(), 8, false)
		if !newXBlocked {
			next.x.Dec()
		}
	} else if x < dst.X() {
		xBlocked = IsTileBlocking(x, next.Y(), 8, true)
		newXBlocked = IsTileBlocking(x+1, next.Y(), 2, false)
		if !newXBlocked {
			next.x.Inc()
		}
	}

	if (xBlocked && yBlocked) || (xBlocked && y == dst.Y()) || (yBlocked && x == dst.X()) {
		p.ResetPath()
		return
	}
	if (newXBlocked && newYBlocked) || (newXBlocked && x != next.X() && y == next.Y()) || (newYBlocked && y != next.Y() && x == next.X()) {
		p.ResetPath()
		return
	}

	if next.X() > x {
		newXBlocked = IsTileBlocking(next.X(), next.Y(), 2, false)
	} else if next.X() < x {
		newXBlocked = IsTileBlocking(next.X(), next.Y(), 8, false)
	}
	if next.Y() > y {
		newYBlocked = IsTileBlocking(next.X(), next.Y(), 1, false)
	} else if next.Y() < y {
		newYBlocked = IsTileBlocking(next.X(), next.Y(), 4, false)
	}

	if (newXBlocked && newYBlocked) || (newXBlocked && y == next.Y()) || (newYBlocked && x == next.X()) {
		p.ResetPath()
		return
	}

	p.SetLocation(next, false)
}

//TraversePath If the mob has a path, calling this method will change the mobs location to the next location described by said Path data structure.  This should be called no more than once per game tick.
func (n *NPC) TraversePath() {
	path := n.Path()
	if path == nil {
		return
	}
	if n.AtLocation(path.NextWaypointTile()) {
		path.CurrentWaypoint++
	}
	if n.FinishedPath() {
		n.ResetPath()
		return
	}
	dst := path.NextWaypointTile()
	x, y := n.X(), n.Y()
	next := NewLocation(x, y)
	xBlocked, yBlocked := false, false
	newXBlocked, newYBlocked := false, false
	if y > dst.Y() {
		yBlocked = IsTileBlocking(x, y, 1, true)
		newYBlocked = IsTileBlocking(x, y-1, 4, false)
		if !newYBlocked {
			next.y.Dec()
		}
	} else if y < dst.Y() {
		yBlocked = IsTileBlocking(x, y, 4, true)
		newYBlocked = IsTileBlocking(x, y+1, 1, false)
		if !newYBlocked {
			next.y.Inc()
		}
	}
	if x > dst.X() {
		xBlocked = IsTileBlocking(x, next.Y(), 2, true)
		newXBlocked = IsTileBlocking(x-1, next.Y(), 8, false)
		if !newXBlocked {
			next.x.Dec()
		}
	} else if x < dst.X() {
		xBlocked = IsTileBlocking(x, next.Y(), 8, true)
		newXBlocked = IsTileBlocking(x+1, next.Y(), 2, false)
		if !newXBlocked {
			next.x.Inc()
		}
	}

	if (xBlocked && yBlocked) || (xBlocked && y == dst.Y()) || (yBlocked && x == dst.X()) {
		n.ResetPath()
		return
	}
	if (newXBlocked && newYBlocked) || (newXBlocked && x != next.X() && y == next.Y()) || (newYBlocked && y != next.Y() && x == next.X()) {
		n.ResetPath()
		return
	}

	if next.X() > x {
		newXBlocked = IsTileBlocking(next.X(), next.Y(), 2, false)
	} else if next.X() < x {
		newXBlocked = IsTileBlocking(next.X(), next.Y(), 8, false)
	}
	if next.Y() > y {
		newYBlocked = IsTileBlocking(next.X(), next.Y(), 1, false)
	} else if next.Y() < y {
		newYBlocked = IsTileBlocking(next.X(), next.Y(), 4, false)
	}

	if (newXBlocked && newYBlocked) || (newXBlocked && y == next.Y()) || (newYBlocked && x == next.X()) {
		n.ResetPath()
		return
	}

	n.SetLocation(next, false)
}

func (p *Player) UpdateRegion(x, y int) {
	curArea := GetRegion(p.X(), p.Y())
	newArea := GetRegion(x, y)
	if newArea != curArea {
		if curArea.Players.Contains(p) {
			curArea.Players.Remove(p)
		}
		newArea.Players.Add(p)
	}
}

func (n *NPC) UpdateRegion(x, y int) {
	curArea := GetRegion(n.X(), n.Y())
	newArea := GetRegion(x, y)
	if newArea != curArea {
		if curArea.NPCs.Contains(n) {
			curArea.NPCs.Remove(n)
		}
		newArea.NPCs.Add(n)
	}
}

//FinishedPath Returns true if the mobs path is nil, the paths current waypoint exceeds the number of waypoints available, or the next tile in the path is not a valid location, implying that we have reached our destination.
func (m *Mob) FinishedPath() bool {
	path := m.Path()
	if path == nil {
		return true
	}
	return path.CurrentWaypoint >= path.CountWaypoints() || !path.NextTileFrom(m.Location).IsValid()
}

//SetLocation Sets the mobs location.
func (m *Mob) SetLocation(location Location, teleport bool) {
	m.SetCoords(location.X(), location.Y(), teleport)
}

func (p *Player) SetLocation(l Location, teleport bool) {
	p.UpdateRegion(l.X(), l.Y())
	p.Mob.SetLocation(l, teleport)
}

func (n *NPC) SetLocation(l Location, teleport bool) {
	n.UpdateRegion(l.X(), l.Y())
	n.Mob.SetLocation(l, teleport)
}

//SetCoords Sets the mobs locations coordinates.
func (m *Mob) SetCoords(x, y int, teleport bool) {
	if !teleport {
		m.SetDirection(m.DirectionTo(x, y))
		m.Move()
	} else {
		m.Remove()
	}
	m.SetX(x)
	m.SetY(y)
}

func (p *Player) SetCoords(x, y int, teleport bool) {
	p.UpdateRegion(x, y)
	p.Mob.SetCoords(x, y, teleport)
}

func (n *NPC) SetCoords(x, y int, teleport bool) {
	n.UpdateRegion(x, y)
	n.Mob.SetCoords(x, y, teleport)
}

func (p *Player) Teleport(x, y int) {
	p.SetCoords(x, y, true)
}

func (n *NPC) Teleport(x, y int) {
	n.SetCoords(x, y, true)
}

func (m *Mob) State() int {
	return m.TransAttrs.VarInt("state", 0)
}

func (m *Mob) HasState(state int) bool {
	return m.State() & state == state
}

func (m *Mob) AddState(state int) {
	if m.HasState(state) {
		log.Warning.Println("Attempted to add a Mobstate that we already have:", state)
		return
	}
	m.Transients().MaskInt("state", state)
}

func (m *Mob) RemoveState(state int) {
	if !m.HasState(state) {
		//log.Warning.Println("Attempted to remove a Mobstate that we did not add:", state)
		return
	}
	m.Transients().UnmaskInt("state", state)
}

//ResetFighting Resets melee fight related variables
func (m *Mob) ResetFighting() {
	target := m.TransAttrs.VarMob("fightTarget")
	if target != nil && target.IsFighting() {
		target.UpdateLastFight()
		target.Transients().UnsetVar("fightTarget")
		target.Transients().UnsetVar("fightRound")
		target.SetDirection(North)
		target.RemoveState(MSFighting)
	}
	if m.IsFighting() {
		m.TransAttrs.UnsetVar("fightTarget")
		m.TransAttrs.UnsetVar("fightRound")
		m.SetDirection(North)
		m.RemoveState(MSFighting)
		m.UpdateLastFight()
	}
}

//FightMode Returns the players current fight mode.
func (m *Mob) FightMode() int {
	return m.TransAttrs.VarInt("fight_mode", 0)
}

//SetFightMode Sets the players fightmode to i.  0=all,1=attack,2=defense,3=strength
func (m *Mob) SetFightMode(i int) {
	m.TransAttrs.SetVar("fight_mode", i)
}

//ArmourPoints Returns the players armour points.
func (m *Mob) ArmourPoints() int {
	return m.TransAttrs.VarInt("armour_points", 1)
}

//SetArmourPoints Sets the players armour points to i.
func (m *Mob) SetArmourPoints(i int) {
	m.TransAttrs.SetVar("armour_points", i)
}

//PowerPoints Returns the players power points.
func (m *Mob) PowerPoints() int {
	return m.TransAttrs.VarInt("power_points", 1)
}

//SetPowerPoints Sets the players power points to i
func (m *Mob) SetPowerPoints(i int) {
	m.TransAttrs.SetVar("power_points", i)
}

//AimPoints Returns the players aim points
func (m *Mob) AimPoints() int {
	return m.TransAttrs.VarInt("aim_points", 1)
}

//SetAimPoints Sets the players aim points to i.
func (m *Mob) SetAimPoints(i int) {
	m.TransAttrs.SetVar("aim_points", i)
}

//MagicPoints Returns the players magic points
func (m *Mob) MagicPoints() int {
	return m.TransAttrs.VarInt("magic_points", 1)
}

//SetMagicPoints Sets the players magic points to i
func (m *Mob) SetMagicPoints(i int) {
	m.TransAttrs.SetVar("magic_points", i)
}

//PrayerPoints Returns the players prayer points
func (m *Mob) PrayerPoints() int {
	return m.TransAttrs.VarInt("prayer_points", 1)
}

//SetPrayerPoints Sets the players prayer points to i
func (m *Mob) SetPrayerPoints(i int) {
	m.TransAttrs.SetVar("prayer_points", i)
}

//RangedPoints Returns the players ranged points.
func (m *Mob) RangedPoints() int {
	return m.TransAttrs.VarInt("ranged_points", 1)
}

//SetRangedPoints Sets the players ranged points tp i.
func (m *Mob) SetRangedPoints(i int) {
	m.TransAttrs.SetVar("ranged_points", i)
}

func (m *Mob) Skills() *SkillTable {
	return m.TransAttrs.VarSkills("skills")
}


//AttrList A type alias for a map of strings to empty interfaces, to hold generic mob information for easy serialization and to provide dynamic insertion/deletion of new mob properties easily
type AttrList map[string]interface{}

//AttributeList A concurrency-safe collection data type for storing misc. variables by a descriptive name.
type AttributeList struct {
	Set  map[string]interface{}
	Lock sync.RWMutex
}

//Range Runs fn(key, value) for every entry in this attribute list.
func (attributes *AttributeList) Range(fn func(string, interface{})) {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	for k, v := range attributes.Set {
		fn(k, v)
	}
}

//SetVar Sets the attribute mapped at name to value in the attribute map.
func (attributes *AttributeList) SetVar(name string, value interface{}) {
	attributes.Lock.Lock()
	attributes.Set[name] = value
	attributes.Lock.Unlock()
}

//UnsetVar Removes the attribute with the key `name` from this attribute set.
func (attributes *AttributeList) UnsetVar(name string) {
	attributes.Lock.Lock()
	delete(attributes.Set, name)
	attributes.Lock.Unlock()
}

//VarInt If there is an attribute assigned to the specified name, returns it.  Otherwise, returns zero
func (attributes *AttributeList) VarInt(name string, zero int) int {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(int); !ok {
		return zero
	}

	return attributes.Set[name].(int)
}

//MaskInt Mask attribute `name` with the specified bitmask.
func (attributes *AttributeList) MaskInt(name string, mask int) {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if val, ok := attributes.Set[name].(int); ok {
		attributes.Set[name] = val | mask
		return
	}
	attributes.Set[name] = MSIdle | mask
}

//UnmaskInt Mask attribute `name` with the specified bitmask.
func (attributes *AttributeList) UnmaskInt(name string, mask int) {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if val, ok := attributes.Set[name].(int); ok {
		attributes.Set[name] = val & 0xFFFFFFFF - mask
		return
	}
	attributes.Set[name] = MSIdle
}

//CheckMask Check if a bitmask attribute has a mask set.
func (attributes *AttributeList) CheckMask(name string, mask int) bool {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if val, ok := attributes.Set[name].(int); !ok {
		return val & mask != 0
	}
	return 0 & mask != 0
}

//VarMob If there is a MobileEntity attribute assigned to the specified name, returns it.  Otherwise, returns nil
func (attributes *AttributeList) VarMob(name string) MobileEntity {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(MobileEntity); !ok {
		return nil
	}

	return attributes.Set[name].(MobileEntity)
}

//VarPlayer If there is a *Player attribute assigned to the specified name, returns it.  Otherwise, returns nil
func (attributes *AttributeList) VarPlayer(name string) *Player {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(*Player); !ok {
		return nil
	}

	return attributes.Set[name].(*Player)
}

//VarSkills If there is a *SkillTable attribute assigned to the specified name, returns it.  Otherwise, returns nil
func (attributes *AttributeList) VarSkills(name string) *SkillTable {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(*SkillTable); !ok {
		return nil
	}

	return attributes.Set[name].(*SkillTable)
}

//VarLong If there is an attribute assigned to the specified name, returns it.  Otherwise, returns zero
func (attributes *AttributeList) VarLong(name string, zero uint64) uint64 {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(uint64); !ok {
		return zero
	}

	return attributes.Set[name].(uint64)
}

//VarBool If there is an attribute assigned to the specified name, returns it.  Otherwise, returns zero
func (attributes *AttributeList) VarBool(name string, zero bool) bool {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(bool); !ok {
		return zero
	}

	return attributes.Set[name].(bool)
}

//VarTime If there is a time.Duration attribute assigned to the specified name, returns it.  Otherwise, returns zero
func (attributes *AttributeList) VarTime(name string) time.Time {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(time.Time); !ok {
		return time.Time{}
	}

	return attributes.Set[name].(time.Time)
}

//VarTime If there is a time.Duration attribute assigned to the specified name, returns it.  Otherwise, returns zero
func (attributes *AttributeList) VarPath(name string) *Pathway {
	attributes.Lock.RLock()
	defer attributes.Lock.RUnlock()
	if _, ok := attributes.Set[name].(*Pathway); !ok {
		return nil
	}

	return attributes.Set[name].(*Pathway)
}

//AppearanceTable Represents a mobs appearance.
type AppearanceTable struct {
	Head      int
	Body      int
	Legs      int
	Male      bool
	HeadColor int
	BodyColor int
	LegsColor int
	SkinColor int
}

//NewAppearanceTable Returns a reference to a new appearance table with specified parameters
func NewAppearanceTable(head, body int, male bool, hair, top, bottom, skin int) AppearanceTable {
	return AppearanceTable{head, body, 3, male, hair, top, bottom, skin}
}

const (
	StatAttack int = iota
	StatDefense
	StatStrength
	StatHits
	StatRanged
	StatPrayer
	StatMagic
	StatCooking
	StatWoodcutting
	StatFletching
	StatFishing
	StatFiremaking
	StatCrafting
	StatSmithing
	StatMining
	StatHerblaw
	StatAgility
	StatThieving
)

//SkillTable Represents a skill table for a mob.
type SkillTable struct {
	current    [18]int
	maximum    [18]int
	experience [18]int
	Lock       sync.RWMutex
}

//Current Returns the current level of the skill indicated by idx.
func (s *SkillTable) Current(idx int) int {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.current[idx]
}

func (s *SkillTable) DecreaseCur(idx, delta int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.current[idx] -= delta
}

func (s *SkillTable) IncreaseCur(idx, delta int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.current[idx] += delta
}

func (s *SkillTable) SetCur(idx, val int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.current[idx] = val
}

func (s *SkillTable) DecreaseMax(idx, delta int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.maximum[idx] -= delta
}

func (s *SkillTable) IncreaseMax(idx, delta int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.maximum[idx] += delta
}

func (s *SkillTable) SetMax(idx, val int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.maximum[idx] = val
}

func (s *SkillTable) SetExp(idx, val int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.experience[idx] = val
}

func (s *SkillTable) IncExp(idx, val int) {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	s.experience[idx] += val
}

//Maximum Returns the maximum level of the skill indicated by idx.
func (s *SkillTable) Maximum(idx int) int {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.maximum[idx]
}

//Experience Returns the current level of the skill indicated by idx.
func (s *SkillTable) Experience(idx int) int {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	return s.experience[idx]
}

//CombatLevel Calculates and returns the combat level for this skill table.
func (s *SkillTable) CombatLevel() int {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	aggressiveTotal := float32(s.maximum[0] + s.maximum[2])
	defensiveTotal := float32(s.maximum[1] + s.maximum[3])
	spiritualTotal := float32((s.maximum[5] + s.maximum[6]) / 8)
	ranged := float32(s.maximum[4])
	if aggressiveTotal < ranged*1.5 {
		return int((defensiveTotal / 4) + (ranged * 0.375) + spiritualTotal)
	}
	return int((aggressiveTotal / 4) + (defensiveTotal / 4) + spiritualTotal)
}

//NpcDefinition This represents a single definition for a single NPC in the game.
type NpcDefinition struct {
	ID          int
	Name        string
	Description string
	Command     string
	Hits        int
	Attack      int
	Strength    int
	Defense     int
	Attackable  bool
}

//NpcDefs This holds the defining characteristics for all of the game's NPCs, ordered by ID.
var NpcDefs []NpcDefinition

//NpcCounter Counts the number of total NPCs within the world.
var NpcCounter = atomic.NewUint32(0)

//Npcs A collection of every NPC in the game, sorted by index
var Npcs []*NPC
var npcsLock sync.RWMutex

//NPC Represents a single non-playable character within the game world.
type NPC struct {
	*Mob
	ID          int
	Boundaries  [2]Location
	StartPoint  Location
	ChatMessage string
	ChatTarget  int
}

//NewNpc Creates a new NPC and returns a reference to it
func NewNpc(id int, startX int, startY int, minX, maxX, minY, maxY int) *NPC {
	n := &NPC{ID: id, Mob: &Mob{Entity: &Entity{Index: int(NpcCounter.Swap(NpcCounter.Load() + 1)), Location: NewLocation(startX, startY)}, TransAttrs: &AttributeList{Set: make(map[string]interface{})}}, ChatTarget: -1, ChatMessage: ""}
	n.Transients().SetVar("skills", &SkillTable{})
	n.Boundaries[0] = NewLocation(minX, minY)
	n.Boundaries[1] = NewLocation(maxX, maxY)
	n.StartPoint = NewLocation(startX, startY)
	if id < 794 {
		n.Skills().current[0] = NpcDefs[id].Attack
		n.Skills().current[1] = NpcDefs[id].Defense
		n.Skills().current[2] = NpcDefs[id].Strength
		n.Skills().current[3] = NpcDefs[id].Hits
		n.Skills().maximum[0] = NpcDefs[id].Attack
		n.Skills().maximum[1] = NpcDefs[id].Defense
		n.Skills().maximum[2] = NpcDefs[id].Strength
		n.Skills().maximum[3] = NpcDefs[id].Hits
	}
	npcsLock.Lock()
	Npcs = append(Npcs, n)
	npcsLock.Unlock()
	return n
}

//UpdateNPCPositions Loops through the global NPC list and, if they are by a player, updates their path to a new path every so often,
// within their boundaries, and traverses each NPC along said path if necessary.
func UpdateNPCPositions() {
	npcsLock.RLock()
	for _, n := range Npcs {
		if n.Busy() || n.IsFighting() || n.Equals(DeathPoint) {
			continue
		}
		if n.TransAttrs.VarTime("nextMove").Before(time.Now()) {
			for _, r := range SurroundingRegions(n.X(), n.Y()) {
				r.Players.lock.RLock()
				if len(r.Players.List) > 0 {
					r.Players.lock.RUnlock()
					n.TransAttrs.SetVar("nextMove", time.Now().Add(time.Second*time.Duration(rand.Int31N(5, 15))))
					go n.WalkTo(NewRandomLocation(n.Boundaries))
					break
				}
				r.Players.lock.RUnlock()
			}
		}
		n.TraversePath()
	}
	npcsLock.RUnlock()
}

//ResetNpcUpdateFlags Resets the synchronization update flags for all NPCs in the game world.
func ResetNpcUpdateFlags() {
	npcsLock.RLock()
	for _, n := range Npcs {
		n.TransAttrs.UnsetVar("changed")
		n.TransAttrs.UnsetVar("moved")
		n.TransAttrs.UnsetVar("remove")
	}
	npcsLock.RUnlock()
}

func (m *Mob) StyleBonus(stat int) int {
	mode := m.TransAttrs.VarInt("fight_mode", 0)
	if mode == 0 {
		return 1
	} else if (mode == 2 && stat == 0) || (mode == 1 && stat == 2) || (mode == 3 && stat == 1) {
		return 3
	}
	return 0
}

//MaxHit Calculates and returns the current max hit for this mob.
func (m *Mob) MaxHit() int {
	prayer := float32(1.0)
	newStr := (float32(m.Skills().Current(2)) * prayer) + float32(m.StyleBonus(2))
	return int((newStr*((float32(m.TransAttrs.VarInt("power_points", 1))*0.00175)+0.1) + 1.05) * 0.95)
}

func (m *Mob) Accuracy(npcMul float32) float32 {
	styleBonus := float32(m.StyleBonus(0))
	prayer := float32(1.0)
	attackLvl := (float32(m.Skills().Current(0)) * prayer) + styleBonus + 8
	multiplier := float32(m.TransAttrs.VarInt("aim_points", 1) + 64)
	multiplier *= npcMul
	return attackLvl * multiplier
}

func (m *Mob) Defense(npcMul float32) float32 {
	styleBonus := float32(m.StyleBonus(1))
	prayer := float32(1.0)
	defenseLvl := (float32(m.Skills().Current(1)) * prayer) + styleBonus + 8
	multiplier := float32(m.TransAttrs.VarInt("armour_points", 1) + 64)
	multiplier *= npcMul
	return defenseLvl * multiplier
}

func (n *NPC) MeleeDamage(target MobileEntity) int {
	att := n.Accuracy(0.9)
	mul := float32(1.0)
	if _, ok := target.(*NPC); ok {
		mul = 0.9
	}
	def := target.Defense(mul)
	max := n.MaxHit()
	if att*10 < def {
		return 0
	}

	finalAtt := int((att / (2.0 * (def + 1.0))) * 10000.0)

	if att > def {
		finalAtt = int((1.0 - ((def + 2.0) / (2.0 * (att + 1.0)))) * 10000.0)
	}

	roll := rand.Int31N(0, 10000)
	//	log.Info.Println(finalAtt, roll, att, def, max)
	if finalAtt > roll {
		return rand.Int31N(0, max)
	}
	return 0
}

func (p *Player) MeleeDamage(target MobileEntity) int {
	att := p.Accuracy(1.0)
	mul := float32(1.0)
	if _, ok := target.(*NPC); ok {
		mul = 0.9
	}
	def := target.Defense(mul)
	max := p.MaxHit()
	if att*10 < def {
		return 0
	}

	finalAtt := int((att / (2.0 * (def + 1.0))) * 10000.0)

	if att > def {
		finalAtt = int((1.0 - ((def + 2.0) / (2.0 * (att + 1.0)))) * 10000.0)
	}

	roll := rand.Int31N(0, 10000)
	//	log.Info.Println(finalAtt, roll, att, def, max)
	if finalAtt > roll {
		return rand.Int31N(0, max)
	}
	return 0
}
