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

// AttacksTriggered triggers when the source creature attacks.
type AttacksTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func AttacksTrigger(effect Effect, optional bool) *AttacksTriggered {
	return &AttacksTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *AttacksTriggered) CheckEventType(et EventType) bool {
	return et == EvtDeclaredAttacker
}

func (t *AttacksTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.SourceID == t.source
}

func (t *AttacksTriggered) IsOptional() bool { return t.Optional }
func (t *AttacksTriggered) Effects() []Effect { return t.effects }
func (t *AttacksTriggered) Targets() []Target { return t.targets }

// DiesCreatureTriggered triggers when another creature you control dies.
type DiesCreatureTriggered struct {
	BaseAbility
	Optional bool
	Filter   PermanentFilter
	effects     []Effect
	targets     []Target
}

func DiesCreatureTrigger(effect Effect, optional bool, filter PermanentFilter) *DiesCreatureTriggered {
	return &DiesCreatureTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		Filter:   filter,
		effects:     []Effect{effect},
	}
}

func (t *DiesCreatureTriggered) CheckEventType(et EventType) bool {
	return et == EvtCreatureDied
}

func (t *DiesCreatureTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// The dying creature is evt.SourceID
	// The trigger source (Wraithbloom etc.) is t.source
	if evt.SourceID == t.source {
		return false // "another" creature — not itself
	}
	// Check if the dying creature was controlled by the same player
	if evt.PlayerID != t.controller {
		return false
	}
	return true
}

func (t *DiesCreatureTriggered) IsOptional() bool { return t.Optional }
func (t *DiesCreatureTriggered) Effects() []Effect { return t.effects }
func (t *DiesCreatureTriggered) Targets() []Target { return t.targets }

// ETBTriggered triggers when the source enters the battlefield.
type ETBTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func EntersBattlefieldTrigger(effect Effect, optional bool) *ETBTriggered {
	return &ETBTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *ETBTriggered) CheckEventType(et EventType) bool {
	return et == EvtEntersBattlefield
}

func (t *ETBTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.SourceID == t.source
}

func (t *ETBTriggered) IsOptional() bool { return t.Optional }
func (t *ETBTriggered) Effects() []Effect { return t.effects }
func (t *ETBTriggered) Targets() []Target { return t.targets }

// PutIntoGraveyardFromBattlefieldTriggered triggers when the source goes from battlefield to graveyard.
type PutIntoGraveyardFromBattlefieldTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func PutIntoGraveyardFromBattlefieldTrigger(effect Effect, optional bool) *PutIntoGraveyardFromBattlefieldTriggered {
	return &PutIntoGraveyardFromBattlefieldTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) CheckEventType(et EventType) bool {
	return et == EvtPutIntoGraveyardFromBattlefield
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.SourceID == t.source
}

func (t *PutIntoGraveyardFromBattlefieldTriggered) IsOptional() bool { return t.Optional }
func (t *PutIntoGraveyardFromBattlefieldTriggered) Effects() []Effect { return t.effects }
func (t *PutIntoGraveyardFromBattlefieldTriggered) Targets() []Target { return t.targets }

// BeginningOfUpkeepTriggered triggers at the beginning of your upkeep.
type BeginningOfUpkeepTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func BeginningOfUpkeepTrigger(effect Effect, optional bool) *BeginningOfUpkeepTriggered {
	return &BeginningOfUpkeepTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *BeginningOfUpkeepTriggered) CheckEventType(et EventType) bool {
	return et == EvtUpkeep
}

func (t *BeginningOfUpkeepTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Triggers for the controller's upkeep
	return evt.PlayerID == t.controller
}

func (t *BeginningOfUpkeepTriggered) IsOptional() bool { return t.Optional }
func (t *BeginningOfUpkeepTriggered) Effects() []Effect { return t.effects }
func (t *BeginningOfUpkeepTriggered) Targets() []Target { return t.targets }

// BeginningOfEachUpkeepTriggered triggers at each player's upkeep.
type BeginningOfEachUpkeepTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func BeginningOfEachUpkeepTrigger(effect Effect, optional bool) *BeginningOfEachUpkeepTriggered {
	return &BeginningOfEachUpkeepTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *BeginningOfEachUpkeepTriggered) CheckEventType(et EventType) bool {
	return et == EvtUpkeep
}

func (t *BeginningOfEachUpkeepTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return true // fires on every player's upkeep
}

func (t *BeginningOfEachUpkeepTriggered) IsOptional() bool { return t.Optional }
func (t *BeginningOfEachUpkeepTriggered) Effects() []Effect { return t.effects }
func (t *BeginningOfEachUpkeepTriggered) Targets() []Target { return t.targets }

// DealsDamageToOpponentTriggered triggers when the source deals damage to an opponent.
type DealsDamageToOpponentTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func DealsDamageToOpponentTrigger(effect Effect, optional bool) *DealsDamageToOpponentTriggered {
	return &DealsDamageToOpponentTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *DealsDamageToOpponentTriggered) CheckEventType(et EventType) bool {
	return et == EvtDamageDealt
}

func (t *DealsDamageToOpponentTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Source of damage is the permanent this is on
	if evt.SourceID != t.source {
		return false
	}
	// Target must be an opponent (a player who is not the controller)
	targetPlayer := g.GetPlayer(evt.TargetID)
	if targetPlayer == nil {
		return false // damage was to a creature, not a player
	}
	return targetPlayer.PlayerID() != t.controller
}

func (t *DealsDamageToOpponentTriggered) IsOptional() bool { return t.Optional }
func (t *DealsDamageToOpponentTriggered) Effects() []Effect { return t.effects }
func (t *DealsDamageToOpponentTriggered) Targets() []Target { return t.targets }

// WheneverSpellCastTriggered triggers whenever a spell of the matching type is cast.
type WheneverSpellCastTriggered struct {
	BaseAbility
	Optional  bool
	ColorFilter *Color // optional: only trigger on spells of this color
	effects      []Effect
	targets      []Target
}

func WheneverSpellCastTrigger(effect Effect, optional bool, colorFilter *Color) *WheneverSpellCastTriggered {
	return &WheneverSpellCastTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional:    optional,
		ColorFilter: colorFilter,
		effects:        []Effect{effect},
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
func (t *WheneverSpellCastTriggered) Effects() []Effect { return t.effects }
func (t *WheneverSpellCastTriggered) Targets() []Target { return t.targets }

// WheneverEnchantmentCastTriggered triggers whenever an enchantment spell is cast by the controller.
type WheneverEnchantmentCastTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func WheneverEnchantmentCastTrigger(effect Effect, optional bool) *WheneverEnchantmentCastTriggered {
	return &WheneverEnchantmentCastTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *WheneverEnchantmentCastTriggered) CheckEventType(et EventType) bool {
	return et == EvtSpellCast
}

func (t *WheneverEnchantmentCastTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Only trigger for spells cast by the controller
	if evt.PlayerID != t.controller {
		return false
	}
	// Check if the cast card is an enchantment
	card := g.FindCardAnywhere(evt.SourceID)
	if card == nil {
		return false
	}
	return card.HasType(TypeEnchantment)
}

func (t *WheneverEnchantmentCastTriggered) IsOptional() bool { return t.Optional }
func (t *WheneverEnchantmentCastTriggered) Effects() []Effect { return t.effects }
func (t *WheneverEnchantmentCastTriggered) Targets() []Target { return t.targets }

// WhenDamageDealtToThisTriggered triggers when damage is dealt to this creature.
type WhenDamageDealtToThisTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func WhenDamageDealtToThisTrigger(effect Effect, optional bool) *WhenDamageDealtToThisTriggered {
	return &WhenDamageDealtToThisTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *WhenDamageDealtToThisTriggered) CheckEventType(et EventType) bool {
	return et == EvtDamageDealt
}

func (t *WhenDamageDealtToThisTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return evt.TargetID == t.source
}

func (t *WhenDamageDealtToThisTriggered) IsOptional() bool { return t.Optional }
func (t *WhenDamageDealtToThisTriggered) Effects() []Effect { return t.effects }
func (t *WhenDamageDealtToThisTriggered) Targets() []Target { return t.targets }

// BeginningOfAttachedControllerUpkeepTriggered triggers at the beginning of
// the upkeep of the player who controls the permanent this aura is attached to.
type BeginningOfAttachedControllerUpkeepTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func BeginningOfAttachedControllerUpkeepTrigger(effect Effect, optional bool) *BeginningOfAttachedControllerUpkeepTriggered {
	return &BeginningOfAttachedControllerUpkeepTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *BeginningOfAttachedControllerUpkeepTriggered) CheckEventType(et EventType) bool {
	return et == EvtUpkeep
}

func (t *BeginningOfAttachedControllerUpkeepTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// Find the source permanent (the aura)
	src := g.FindPermanent(t.source)
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
func (t *BeginningOfAttachedControllerUpkeepTriggered) Effects() []Effect { return t.effects }
func (t *BeginningOfAttachedControllerUpkeepTriggered) Targets() []Target { return t.targets }

// WheneverLandEntersBattlefieldTriggered triggers whenever a land enters the battlefield.
type WheneverLandEntersBattlefieldTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func WheneverLandEntersBattlefieldTrigger(effect Effect, optional bool) *WheneverLandEntersBattlefieldTriggered {
	return &WheneverLandEntersBattlefieldTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
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
func (t *WheneverLandEntersBattlefieldTriggered) Effects() []Effect { return t.effects }
func (t *WheneverLandEntersBattlefieldTriggered) Targets() []Target { return t.targets }

// AnyCreatureDiesTriggered triggers when any creature dies.
type AnyCreatureDiesTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func AnyCreatureDiesTrigger(effect Effect, optional bool) *AnyCreatureDiesTriggered {
	return &AnyCreatureDiesTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *AnyCreatureDiesTriggered) CheckEventType(et EventType) bool {
	return et == EvtCreatureDied
}

func (t *AnyCreatureDiesTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	return true // any creature dying triggers this
}

func (t *AnyCreatureDiesTriggered) IsOptional() bool { return t.Optional }
func (t *AnyCreatureDiesTriggered) Effects() []Effect { return t.effects }
func (t *AnyCreatureDiesTriggered) Targets() []Target { return t.targets }

// CreatureDealtDamageBySourceDiesTriggered triggers when a creature that was
// dealt damage by the source permanent this turn dies.
type CreatureDealtDamageBySourceDiesTriggered struct {
	BaseAbility
	Optional bool
	effects     []Effect
	targets     []Target
}

func CreatureDealtDamageBySourceDiesTrigger(effect Effect, optional bool) *CreatureDealtDamageBySourceDiesTriggered {
	return &CreatureDealtDamageBySourceDiesTriggered{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilityTriggered,
		},
		Optional: optional,
		effects:     []Effect{effect},
	}
}

func (t *CreatureDealtDamageBySourceDiesTriggered) CheckEventType(et EventType) bool {
	return et == EvtCreatureDied
}

func (t *CreatureDealtDamageBySourceDiesTriggered) CheckTrigger(evt *GameEvent, g *Game) bool {
	// evt.SourceID is the dying creature's ID
	// t.source is the permanent with this trigger (e.g. Sengir Vampire)
	// Check if the source dealt damage to the dying creature this turn
	sources := g.DamageDealtBy[evt.SourceID]
	return sources != nil && sources[t.source]
}

func (t *CreatureDealtDamageBySourceDiesTriggered) IsOptional() bool { return t.Optional }
func (t *CreatureDealtDamageBySourceDiesTriggered) Effects() []Effect { return t.effects }
func (t *CreatureDealtDamageBySourceDiesTriggered) Targets() []Target { return t.targets }
