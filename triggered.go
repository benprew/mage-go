package mage

import "github.com/google/uuid"

// TriggeredAbility checks events and produces effects.
type TriggeredAbility interface {
	Ability
	CheckEventType(EventType) bool
	CheckTrigger(*GameEvent, *Game) bool
	IsOptional() bool
	Effects() []Effect
	Targets() []Target
}

// DiesCreatureTriggered triggers when another creature you control dies.
type DiesCreatureTriggered struct {
	BaseAbility
	Optional bool
	Filter   PermanentFilter
	Effs     []Effect
	Tgts     []Target
}

func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *DiesCreatureTriggered {
	return &DiesCreatureTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Filter:   filter,
		Effs:     []Effect{effect},
	}
}

func (t *DiesCreatureTriggered) AddEffect(e Effect) *DiesCreatureTriggered {
	t.Effs = append(t.Effs, e)
	return t
}

func (t *DiesCreatureTriggered) AddTarget(tgt Target) *DiesCreatureTriggered {
	t.Tgts = append(t.Tgts, tgt)
	return t
}

func (t *DiesCreatureTriggered) CheckEventType(et EventType) bool {
	return et == EvtCreatureDied
}

func (t *DiesCreatureTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// The dying creature is evt.SourceID
	// The trigger source (Wraithbloom etc.) is t.Source_
	if evt.SourceID == t.Source_ {
		return false // "another" creature — not itself
	}
	// Check if the dying creature was controlled by the same player
	if evt.PlayerID != t.Controller_ {
		return false
	}
	return true
}

func (t *DiesCreatureTriggered) IsOptional() bool { return t.Optional }
func (t *DiesCreatureTriggered) Effects() []Effect { return t.Effs }
func (t *DiesCreatureTriggered) Targets() []Target { return t.Tgts }

// ETBTriggered triggers when the source enters the battlefield.
type ETBTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func EntersBattlefieldTrigger(effect Effect, optional bool) *ETBTriggered {
	return &ETBTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *ETBTriggered) AddEffect(e Effect) *ETBTriggered {
	t.Effs = append(t.Effs, e)
	return t
}

func (t *ETBTriggered) AddTarget(tgt Target) *ETBTriggered {
	t.Tgts = append(t.Tgts, tgt)
	return t
}

func (t *ETBTriggered) CheckEventType(et EventType) bool {
	return et == EvtEntersBattlefield
}

func (t *ETBTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.SourceID == t.Source_
}

func (t *ETBTriggered) IsOptional() bool { return t.Optional }
func (t *ETBTriggered) Effects() []Effect { return t.Effs }
func (t *ETBTriggered) Targets() []Target { return t.Tgts }

// PutIntoGraveyardFromBattlefieldTriggered triggers when the source goes from battlefield to graveyard.
type PutIntoGraveyardFromBattlefieldTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *PutIntoGraveyardFromBattlefieldTriggered {
	return &PutIntoGraveyardFromBattlefieldTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) AddEffect(e Effect) *PutIntoGraveyardFromBattlefieldTriggered {
	t.Effs = append(t.Effs, e)
	return t
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) CheckEventType(et EventType) bool {
	return et == EvtPutIntoGraveyardFromBattlefield
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.SourceID == t.Source_
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) IsOptional() bool { return t.Optional }
func (t *PutIntoGraveyardFromBattlefieldTriggered) Effects() []Effect { return t.Effs }
func (t *PutIntoGraveyardFromBattlefieldTriggered) Targets() []Target { return t.Tgts }

// BeginningOfUpkeepTriggered triggers at the beginning of your upkeep.
type BeginningOfUpkeepTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func BeginningOfUpkeepTrigger(effect Effect, optional bool) *BeginningOfUpkeepTriggered {
	return &BeginningOfUpkeepTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *BeginningOfUpkeepTriggered) AddEffect(e Effect) *BeginningOfUpkeepTriggered {
	t.Effs = append(t.Effs, e)
	return t
}

func (t *BeginningOfUpkeepTriggered) CheckEventType(et EventType) bool {
	return et == EvtUpkeep
}

func (t *BeginningOfUpkeepTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Triggers for the controller's upkeep
	return evt.PlayerID == t.Controller_
}

func (t *BeginningOfUpkeepTriggered) IsOptional() bool { return t.Optional }
func (t *BeginningOfUpkeepTriggered) Effects() []Effect { return t.Effs }
func (t *BeginningOfUpkeepTriggered) Targets() []Target { return t.Tgts }

// DealsDamageToOpponentTriggered triggers when the source deals damage to an opponent.
type DealsDamageToOpponentTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func DealsDamageToOpponentTrigger(effect Effect, optional bool) *DealsDamageToOpponentTriggered {
	return &DealsDamageToOpponentTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *DealsDamageToOpponentTriggered) CheckEventType(et EventType) bool {
	return et == EvtDamageDealt
}

func (t *DealsDamageToOpponentTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Source of damage is the permanent this is on
	if evt.SourceID != t.Source_ {
		return false
	}
	// Target must be an opponent (a player who is not the controller)
	targetPlayer := g.GetPlayer(evt.TargetID)
	if targetPlayer == nil {
		return false // damage was to a creature, not a player
	}
	return targetPlayer.PlayerID() != t.Controller_
}

func (t *DealsDamageToOpponentTriggered) IsOptional() bool { return t.Optional }
func (t *DealsDamageToOpponentTriggered) Effects() []Effect { return t.Effs }
func (t *DealsDamageToOpponentTriggered) Targets() []Target { return t.Tgts }

// WheneverSpellCastTriggered triggers whenever a spell of the matching type is cast.
type WheneverSpellCastTriggered struct {
	BaseAbility
	Optional  bool
	ColorFilter *Color // optional: only trigger on spells of this color
	Effs      []Effect
	Tgts      []Target
}

func WheneverSpellCastTrigger(effect Effect, optional bool, colorFilter *Color) *WheneverSpellCastTriggered {
	return &WheneverSpellCastTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional:    optional,
		ColorFilter: colorFilter,
		Effs:        []Effect{effect},
	}
}

func (t *WheneverSpellCastTriggered) CheckEventType(et EventType) bool {
	return et == EvtSpellCast
}

func (t *WheneverSpellCastTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	if t.ColorFilter == nil {
		return true
	}
	// Find the spell card to check its color
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return false
	}
	for _, c := range card.ManaCost().Colors() {
		if c == *t.ColorFilter {
			return true
		}
	}
	return false
}

func (t *WheneverSpellCastTriggered) IsOptional() bool { return t.Optional }
func (t *WheneverSpellCastTriggered) Effects() []Effect { return t.Effs }
func (t *WheneverSpellCastTriggered) Targets() []Target { return t.Tgts }

// WhenDamageDealtToThisTriggered triggers when damage is dealt to this creature.
type WhenDamageDealtToThisTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func WhenDamageDealtToThisTrigger(effect Effect, optional bool) *WhenDamageDealtToThisTriggered {
	return &WhenDamageDealtToThisTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *WhenDamageDealtToThisTriggered) CheckEventType(et EventType) bool {
	return et == EvtDamageDealt
}

func (t *WhenDamageDealtToThisTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.TargetID == t.Source_
}

func (t *WhenDamageDealtToThisTriggered) IsOptional() bool { return t.Optional }
func (t *WhenDamageDealtToThisTriggered) Effects() []Effect { return t.Effs }
func (t *WhenDamageDealtToThisTriggered) Targets() []Target { return t.Tgts }

// BeginningOfAttachedControllerUpkeepTriggered triggers at the beginning of
// the upkeep of the player who controls the permanent this aura is attached to.
type BeginningOfAttachedControllerUpkeepTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func BeginningOfAttachedControllerUpkeepTrigger(effect Effect, optional bool) *BeginningOfAttachedControllerUpkeepTriggered {
	return &BeginningOfAttachedControllerUpkeepTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *BeginningOfAttachedControllerUpkeepTriggered) CheckEventType(et EventType) bool {
	return et == EvtUpkeep
}

func (t *BeginningOfAttachedControllerUpkeepTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Find the source permanent (the aura)
	src := g.FindPermanent(t.Source_)
	if src == nil || !src.IsAttached() {
		return false
	}
	// Find the host permanent
	host := g.FindPermanent(src.AttachedTo)
	if host == nil {
		return false
	}
	// Trigger if the upkeep belongs to the controller of the host
	return evt.PlayerID == host.Controller
}

func (t *BeginningOfAttachedControllerUpkeepTriggered) IsOptional() bool { return t.Optional }
func (t *BeginningOfAttachedControllerUpkeepTriggered) Effects() []Effect { return t.Effs }
func (t *BeginningOfAttachedControllerUpkeepTriggered) Targets() []Target { return t.Tgts }

// WheneverLandEntersBattlefieldTriggered triggers whenever a land enters the battlefield.
type WheneverLandEntersBattlefieldTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *WheneverLandEntersBattlefieldTriggered {
	return &WheneverLandEntersBattlefieldTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *WheneverLandEntersBattlefieldTriggered) CheckEventType(et EventType) bool {
	return et == EvtEntersBattlefield
}

func (t *WheneverLandEntersBattlefieldTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Check if the entering permanent is a land
	perm := g.FindPermanent(evt.SourceID)
	if perm == nil {
		return false
	}
	return perm.HasType(TypeLand)
}

func (t *WheneverLandEntersBattlefieldTriggered) IsOptional() bool { return t.Optional }
func (t *WheneverLandEntersBattlefieldTriggered) Effects() []Effect { return t.Effs }
func (t *WheneverLandEntersBattlefieldTriggered) Targets() []Target { return t.Tgts }

// AnyCreatureDiesTriggered triggers when any creature dies.
type AnyCreatureDiesTriggered struct {
	BaseAbility
	Optional bool
	Effs     []Effect
	Tgts     []Target
}

func AnyCreatureDiesTrigger(effect Effect, optional bool) *AnyCreatureDiesTriggered {
	return &AnyCreatureDiesTriggered{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilityTriggered,
		},
		Optional: optional,
		Effs:     []Effect{effect},
	}
}

func (t *AnyCreatureDiesTriggered) CheckEventType(et EventType) bool {
	return et == EvtCreatureDied
}

func (t *AnyCreatureDiesTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return true // any creature dying triggers this
}

func (t *AnyCreatureDiesTriggered) IsOptional() bool { return t.Optional }
func (t *AnyCreatureDiesTriggered) Effects() []Effect { return t.Effs }
func (t *AnyCreatureDiesTriggered) Targets() []Target { return t.Tgts }
