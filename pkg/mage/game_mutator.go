package mage

import (
	"maps"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// PlayerReader provides read-only access to players.
type PlayerReader interface {
	GetPlayer(uuid.UUID) Player
	GetOpponent(uuid.UUID) Player
	ActivePlayerObj() Player
	NonActivePlayerObj() Player
	AllPlayers() []Player
}

// BattlefieldReader provides read-only queries against permanents on the battlefield.
type BattlefieldReader interface {
	FindPermanent(uuid.UUID) *Permanent
	FindPermanentByName(string, uuid.UUID) *Permanent
	FindPermanentIncludingPhased(uuid.UUID) *Permanent
	AnyBattlefield(PermanentFilter) bool
	FilterBattlefield(PermanentFilter) []*Permanent
	CountBattlefield(PermanentFilter) int
	AllBattlefield() []*Permanent
}

// ResolutionReader provides read-only queries for transient resolution context.
type ResolutionReader interface {
	XValue() int
	ModeValue() int
	EventAmount() int
	EventSourceID() uuid.UUID
	GetResolvingCard() Card
	ResolvingCastZone() Zone
	ResolvingCastContext() *CastContext
	GetResolvingTargets() []uuid.UUID
	LastExiledCard() Card
}

// TurnReader provides read-only queries for turn and active player progression.
type TurnReader interface {
	CombatWindowOpen(PhaseStep, bool) bool
	UpkeepHasEnded() bool
	GetStep() PhaseStep
	CurrentTurn() int
	MostRecentTurnStart(uuid.UUID) int
	ActivePlayerIndex() int
}

// CombatReader provides read-only queries for combat state.
type CombatReader interface {
	CombatGroups() []*CombatGroup
	CombatGroupFor(uuid.UUID) *CombatGroup
	IsAttackingInCombat(uuid.UUID) bool
	IsAttackingIncarnation(uuid.UUID, uuid.UUID) bool
	IsBlockingInCombat(uuid.UUID) bool
	IsBlockingIncarnation(uuid.UUID, uuid.UUID) bool
	CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int
}

// TrackerReader provides read-only queries for turn-scoped and duel-scoped observations.
type TrackerReader interface {
	DamageTakenByPlayer(uuid.UUID) int
	HasDealtDamageToPlayerThisTurn(sourceID, incarnation, playerID uuid.UUID) bool
	PlayerLifeGainedThisTurn(uuid.UUID) int
	HasAttackedThisTurn(uuid.UUID) bool
	HasAttackedThisTurnIncarnation(uuid.UUID, uuid.UUID) bool
	AttackedDuringLastTurnIncarnation(uuid.UUID, uuid.UUID, uuid.UUID) bool
	HasAttackedOrBlockedThisTurnIncarnation(uuid.UUID, uuid.UUID) bool
	HasAttackedOrBlockedThisCombatIncarnation(uuid.UUID, uuid.UUID) bool
	CreatureDeaths() int
	GetDamageSources(uuid.UUID) map[uuid.UUID]bool
	GetBlockedThisTurn(uuid.UUID) []uuid.UUID
	GetInstantsCastThisTurn(uuid.UUID) int
	UntappedLandsAtTurnStart(uuid.UUID) int
	GetSorceriesCastThisTurn(uuid.UUID) int
	GetInstantOrSorceryCastThisTurn(uuid.UUID) int
	TimesTargetedThisTurn(uuid.UUID) int
	PlayerCardsDrawnThisTurn(uuid.UUID) int
	PlayerCardsLeftGraveyardThisTurn(uuid.UUID) int
	PlayerHadCardLeaveGraveyardThisTurn(uuid.UUID) bool
	CardsPutIntoExileThisTurn() int
}

// StackReader provides read-only queries for the stack.
type StackReader interface {
	FindStackObject(uuid.UUID) *StackObject
}

// GameReader is the composed read-only view of Game used by ValueSource,
// PlayerSelector, filters, and conditions. It composes focused domain readers.
// *Game satisfies this interface and each sub-interface.
type GameReader interface {
	PlayerReader
	BattlefieldReader
	ResolutionReader
	TurnReader
	CombatReader
	TrackerReader
	StackReader

	FindCardAnywhere(uuid.UUID) Card
	GetArtifactUntapMax() int
	HypotheticalMana(uuid.UUID) int
}

// Compile-time check that *Game satisfies GameReader and all sub-interfaces.
var (
	_ GameReader        = (*Game)(nil)
	_ PlayerReader      = (*Game)(nil)
	_ BattlefieldReader = (*Game)(nil)
	_ ResolutionReader  = (*Game)(nil)
	_ TurnReader        = (*Game)(nil)
	_ CombatReader      = (*Game)(nil)
	_ TrackerReader     = (*Game)(nil)
	_ StackReader       = (*Game)(nil)
)

// --- GameReader proxy methods on *Game ---

// LastExiledCard returns the most recently exiled card recorded during cost payment or effect resolution.
func (g *Game) LastExiledCard() Card { return g.resolution.LastExiledCard() }

// SetLastExiledCard sets the most recently exiled card.
func (g *Game) SetLastExiledCard(c Card) { g.resolution.SetLastExiledCard(c) }

// AllPlayers returns all players in the game.
func (g *Game) AllPlayers() []Player { return g.players }

// XValue returns the current X value for the resolving spell/ability.
func (g *Game) XValue() int { return g.resolution.X() }

// ModeValue returns the current chosen mode for the resolving modal spell.
func (g *Game) ModeValue() int { return g.resolution.Mode() }

// EventAmount returns the amount from the triggering event (e.g. damage dealt).
func (g *Game) EventAmount() int { return g.resolution.EventAmount() }

// EventSourceID returns the SourceID of the event that triggered the
// currently-resolving triggered ability. For an EvtDamageDealt trigger,
// this is the damager's permanent ID. Returns uuid.Nil when there is no
// trigger context (e.g. spell resolution).
func (g *Game) EventSourceID() uuid.UUID { return g.resolution.EventSourceID() }

// GetResolvingCard returns the card currently being resolved from the stack.
func (g *Game) GetResolvingCard() Card { return g.resolution.ResolvingCard() }

// ResolvingCastZone returns the zone the resolving spell was cast from
// (CR 601.2a). Returns ZoneAny when no spell is resolving or the resolving
// stack object is an ability rather than a spell. Read by triggers expressing
// "if you cast it from your hand"/"from your graveyard" conditions, including
// ETB triggers that fire while PutOnBattlefield is in flight.
func (g *Game) ResolvingCastZone() Zone { return g.resolution.ResolvingCastZone() }

// ResolvingCastContext returns the cast-time snapshot of the spell currently
// being resolved (CR 608.2g). Returns nil when there is no resolving spell
// or the resolving stack object is an ability. Effects that reference
// cast-time state ("as you cast this spell") should consult this rather
// than re-querying live state.
func (g *Game) ResolvingCastContext() *CastContext { return g.resolution.ResolvingCastContext() }

// ResolvingDamageDistribution returns the divided damage distribution for the resolving object.
func (g *Game) ResolvingDamageDistribution() map[uuid.UUID]int {
	return g.resolution.DamageDistribution()
}

// ResolvingCounterDistribution returns the counter distribution for the resolving object.
func (g *Game) ResolvingCounterDistribution() map[uuid.UUID]int {
	return g.resolution.CounterDistribution()
}

// Resolution returns the ResolutionState subsystem.
func (g *Game) Resolution() *ResolutionState { return &g.resolution }

// snapshotCastContext builds a CastContext for a spell about to be pushed
// onto the stack by the controller with playerID. Captures the controller's
// permanent subtypes (CR 608.2g) and consumes any pending lastCostReveal
// recorded by additional costs paid earlier in the cast pipeline.
//
// Must be called AFTER additional costs have been paid so that reveal-style
// costs are visible in the snapshot.
func (g *Game) snapshotCastContext(playerID uuid.UUID) *CastContext {
	ctx := &CastContext{
		ControllerSubtypesAtCast:         make(map[string]bool),
		ControllerCreatureSubtypesAtCast: make(map[string]bool),
	}
	for _, perm := range g.AllBattlefield() {
		if perm == nil || perm.ControllerID() != playerID || perm.FaceDown || perm.PhasedOut {
			continue
		}
		for _, st := range perm.computedSubtypes() {
			ctx.ControllerSubtypesAtCast[st] = true
			if perm.HasType(TypeCreature) {
				ctx.ControllerCreatureSubtypesAtCast[st] = true
			}
		}
	}
	if lastReveal := g.resolution.LastCostReveal(); lastReveal != nil {
		ctx.RevealedAtCast = append(ctx.RevealedAtCast, copyCardForCastContext(lastReveal))
	}
	if pl := g.GetPlayer(playerID); pl != nil {
		if drained := pl.ManaPool().LastDrainedColors; len(drained) > 0 {
			ctx.ColorsSpent = make(map[Color]int, len(drained))
			maps.Copy(ctx.ColorsSpent, drained)
		}
	}
	return ctx
}

// FindStackObject finds a stack object by its source card ID.
func (g *Game) FindStackObject(id uuid.UUID) *StackObject {
	return g.stack.FindBySourceID(id)
}

// CombatGroups returns the current combat groups (attacker/blocker pairings).
// Returns nil if combat has not been initialized.
func (g *Game) CombatGroups() []*CombatGroup {
	if g.combat == nil {
		return nil
	}
	return g.combat.Groups
}

// CombatGroupFor returns the combat group for the given attacker, or nil.
func (g *Game) CombatGroupFor(attackerID uuid.UUID) *CombatGroup {
	if g.combat == nil {
		return nil
	}
	return g.combat.GroupFor(attackerID)
}

// IsAttackingInCombat reports whether the permanent is currently an attacking creature in combat.
func (g *Game) IsAttackingInCombat(permID uuid.UUID) bool {
	if g.combat == nil {
		return false
	}
	if p := g.FindPermanent(permID); p == nil || p.PhasedOut {
		return false
	}
	return g.combat.IsAttacking(permID)
}

// IsAttackingIncarnation reports whether the battlefield object is attacking.
func (g *Game) IsAttackingIncarnation(permID, incarnation uuid.UUID) bool {
	return g.combat != nil && g.combat.IsAttackingIncarnation(permID, incarnation)
}

// IsBlockingInCombat reports whether the permanent is currently a blocking creature in combat.
func (g *Game) IsBlockingInCombat(permID uuid.UUID) bool {
	if g.combat == nil {
		return false
	}
	if p := g.FindPermanent(permID); p == nil || p.PhasedOut {
		return false
	}
	return g.combat.IsBlocking(permID)
}

func (g *Game) IsBlockingIncarnation(permID, incarnation uuid.UUID) bool {
	return g.combat != nil && g.combat.IsBlockingIncarnation(permID, incarnation)
}

// DamageTakenByPlayer returns the total damage the given player has taken this turn.
func (g *Game) DamageTakenByPlayer(playerID uuid.UUID) int {
	return g.trackers.Turn.DamageTaken(playerID)
}

// HasAttackedThisTurn reports whether the permanent with the given ID attacked this turn.
func (g *Game) HasAttackedThisTurn(permID uuid.UUID) bool {
	return g.trackers.Turn.Attacked(permID)
}

func (g *Game) HasAttackedThisTurnIncarnation(permID, incarnation uuid.UUID) bool {
	return g.trackers.Turn.AttackedIncarnation(permID, incarnation)
}

// CreatureDeaths returns the number of creatures that died this turn.
func (g *Game) CreatureDeaths() int {
	return g.trackers.Turn.CreatureDeaths()
}

func (g *Game) CurrentTurn() int {
	return g.turns.Turn()
}

// GetDamageSources returns the set of permanent IDs that dealt damage to the
// given permanent this turn. Returns nil if nothing dealt damage.
func (g *Game) GetDamageSources(permID uuid.UUID) map[uuid.UUID]bool {
	return g.damage.DamageDealtBy(permID)
}

// GetBlockedThisTurn returns the list of attacker IDs that the given blocker
// blocked this turn. Returns nil if it didn't block anything.
func (g *Game) GetBlockedThisTurn(blockerID uuid.UUID) []uuid.UUID {
	return g.trackers.Turn.Blocked(blockerID)
}

// GetInstantsCastThisTurn returns the number of instants the given player has cast this turn.
func (g *Game) GetInstantsCastThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.InstantsCast(playerID)
}

// UntappedLandsAtTurnStart returns the number of untapped lands the given
// player controlled at the start of the current turn (snapshot taken before
// the untap step). Used by Power Surge.
func (g *Game) UntappedLandsAtTurnStart(playerID uuid.UUID) int {
	return g.trackers.Turn.UntappedLandsAtTurnStart(playerID)
}

// GetSorceriesCastThisTurn returns the number of sorceries the given player
// has cast this turn.
func (g *Game) GetSorceriesCastThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.SorceriesCast(playerID)
}

// GetInstantOrSorceryCastThisTurn returns the total instants and sorceries
// the given player has cast this turn (CR 117 — combined predicate used by
// many cards that ask "if you've cast an instant or sorcery spell this turn").
func (g *Game) GetInstantOrSorceryCastThisTurn(playerID uuid.UUID) int {
	return g.trackers.Turn.InstantsCast(playerID) + g.trackers.Turn.SorceriesCast(playerID)
}

// --- Mutation methods on *Game ---

// GrantExtraTurn gives the specified player an extra turn after the current one.
func (g *Game) GrantExtraTurn(playerID uuid.UUID) {
	g.turns.GrantExtraTurn(playerID)
}

// RemoveFromCombat removes a permanent from combat by ID.
func (g *Game) RemoveFromCombat(id uuid.UUID) {
	g.combat.RemoveFromCombat(id)
}

// PushStack pushes a stack object onto the stack.
func (g *Game) PushStack(obj *StackObject) {
	if obj.IsAbility && obj.sourceIncarnation == uuid.Nil {
		if p := g.FindPermanent(obj.SourceID); p != nil {
			obj.sourceIncarnation = p.incarnationID
		}
	}
	g.pushStack(obj)
}

// AddContinuousEffect registers a continuous effect with the effect manager.
func (g *Game) AddContinuousEffect(e ContinuousEffect) {
	if _, ok := e.(*countCharacteristicPT); ok {
		if source := g.FindPermanent(e.SourceID()); source != nil {
			e = g.bindCharacteristicEffect(e, source)
		}
	}
	g.effects.Add(e)
	g.effects.Apply(g)
}

// ApplyContinuousEffects re-applies all continuous effects to current permanents.
func (g *Game) ApplyContinuousEffects() {
	g.effects.Apply(g)
}

// SetPreventCombatDamage flags that all combat damage is prevented this turn.
func (g *Game) SetPreventCombatDamage() {
	g.effects.AddReplacement(&fogReplacement{replacementBase: replacementBase{sourceID: uuid.Nil, duration: EndOfTurn}})
}

// PreventCombatDamageToAndBy adds a replacement effect preventing all combat damage dealt to and dealt by creatureID this turn.
func (g *Game) PreventCombatDamageToAndBy(creatureID uuid.UUID) {
	permanent := g.FindPermanent(creatureID)
	if permanent == nil {
		return
	}
	g.effects.AddReplacement(&creatureCombatDamagePreventionReplacement{
		replacementBase: replacementBase{sourceID: uuid.Nil, duration: EndOfTurn},
		creatureID:      creatureID,
		incarnation:     permanent.incarnationID,
	})
}

// PreventDamageToPlayerByCreaturesWithFlying adds a replacement effect preventing all damage dealt to playerID by creatures with flying this turn.
func (g *Game) PreventDamageToPlayerByCreaturesWithFlying(playerID uuid.UUID) {
	g.effects.AddReplacement(&playerDamageFromFlyingCreaturesPreventionReplacement{
		replacementBase: replacementBase{sourceID: uuid.Nil, duration: EndOfTurn},
		playerID:        playerID,
	})
}

// AddRegenerationShield adds a regeneration shield to the specified permanent.
func (g *Game) AddRegenerationShield(id uuid.UUID) {
	p := g.FindPermanent(id)
	if p == nil || p.PhasedOut {
		return
	}
	for _, r := range g.effects.replacements {
		if regen, ok := r.(*regenerationReplacement); ok && regen.permanentID == id && regen.incarnation == p.incarnationID {
			regen.shields++
			return
		}
	}
	g.effects.AddReplacement(&regenerationReplacement{replacementBase: replacementBase{duration: EndOfTurn}, permanentID: id, incarnation: p.incarnationID, shields: 1})
}

// HasRegenerationShield reports whether the specified permanent currently has at
// least one active regeneration shield.
func (g *Game) HasRegenerationShield(id uuid.UUID) bool {
	p := g.FindPermanent(id)
	if p == nil || p.PhasedOut {
		return false
	}
	for _, r := range g.effects.replacements {
		if regen, ok := r.(*regenerationReplacement); ok && regen.permanentID == id && regen.incarnation == p.incarnationID && regen.shields > 0 {
			return true
		}
	}
	return false
}

// AddPreventionShield adds a damage prevention shield to the specified permanent or player.
func (g *Game) AddPreventionShield(id uuid.UUID, amount int) {
	if amount <= 0 {
		return
	}
	var incarnation uuid.UUID
	if p := g.FindPermanent(id); p != nil {
		incarnation = p.incarnationID
	}
	// Find existing prevention shield for this target and add to it
	for _, r := range g.effects.replacements {
		if ps, ok := r.(*preventionShieldReplacement); ok && ps.targetID == id && ps.targetIncarnation == incarnation {
			ps.remaining += amount
			return
		}
	}
	g.effects.AddReplacement(&preventionShieldReplacement{replacementBase: replacementBase{duration: EndOfTurn}, targetID: id, targetIncarnation: incarnation, remaining: amount})
}

// AddForcefieldShield adds a Forcefield shield for the specified player against
// the chosen attacker. The shield reduces the next combat damage from attackerID
// to playerID down to 1, then deactivates (CR 614 — "the next time").
func (g *Game) AddForcefieldShield(playerID, attackerID uuid.UUID) {
	g.effects.AddReplacement(&forcefieldReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		attackerID:      attackerID,
	})
}

// IsLichActive reports whether the Lich enchantment is active for the player.
func (g *Game) IsLichActive(playerID uuid.UUID) bool {
	return g.effects.Rules.IsLichActive(g, playerID)
}

// SetLichActive marks the Lich enchantment as active for the player.
func (g *Game) SetLichActive(playerID, sourceID uuid.UUID) {
	g.effects.Rules.SetLichActive(playerID, sourceID)
	// Register the life-gain replacement (Lich: draw cards instead of gaining life)
	g.effects.AddReplacement(&lichLifeGainReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
		playerID:        playerID,
	})
}

// ClearLich removes the Lich enchantment state for the player.
func (g *Game) ClearLich(playerID uuid.UUID) {
	// Find and remove the Lich's source ID before clearing
	if sourceID, ok := g.effects.Rules.lichActive[playerID]; ok {
		g.effects.RemoveReplacements(sourceID)
	}
	g.effects.Rules.ClearLich(playerID)
}

// AddColorPrevention adds a color-based damage prevention rule for the player.
func (g *Game) AddColorPrevention(playerID uuid.UUID, color Color) {
	g.effects.AddReplacement(&colorPreventionReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, color: color})
}

// ChooseDamageSource asks playerID to choose a permanent or spell on the stack
// as a damage source without targeting it.
func (g *Game) ChooseDamageSource(playerID uuid.UUID) uuid.UUID {
	player := g.GetPlayer(playerID)
	if player == nil {
		return uuid.Nil
	}
	seen := make(map[uuid.UUID]bool)
	var possible []uuid.UUID
	add := func(id uuid.UUID) {
		if id == uuid.Nil || seen[id] || g.FindCardAnywhere(id) == nil {
			return
		}
		seen[id] = true
		possible = append(possible, id)
	}
	for _, permanent := range g.AllBattlefield() {
		if !permanent.PhasedOut {
			add(permanent.ID())
		}
	}
	for _, object := range g.stack.Objects() {
		if !object.IsAbility {
			add(object.SourceID)
		}
	}
	chosen := player.ChooseTargets(possible, 1, 1, g)
	if len(chosen) == 0 || !seen[chosen[0]] {
		return uuid.Nil
	}
	return chosen[0]
}

// AddReverseDamageShield adds a source-bound reverse-damage shield for the player.
// A live permanent keeps its incarnation, so a returned object cannot use it.
// Prepending preserves the existing default replacement order.
func (g *Game) AddReverseDamageShield(playerID, sourceID uuid.UUID) {
	if sourceID == uuid.Nil {
		return
	}
	g.effects.PrependReplacement(&reverseDamageReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
		incarnation:     g.legacyPreventionSourceIncarnation(sourceID),
	})
}

// SetChannelActive marks the Channel ability as active for the player.
func (g *Game) SetChannelActive(playerID uuid.UUID) {
	g.effects.Rules.SetChannelActive(playerID)
}

// AddCantCastSpells registers a continuous "this player can't cast spells"
// rule for the current Apply() cycle. Used by Angelic Arbiter and similar
// effects. The flag is cleared at the start of each Apply() cycle, so
// continuous effects must re-register it every cycle while the source
// permanent is on the battlefield.
func (g *Game) AddCantCastSpells(playerID uuid.UUID) {
	g.effects.Rules.AddCantCastSpells(playerID)
}

// PlayerCantCastSpells reports whether a continuous effect currently
// forbids the given player from casting spells.
func (g *Game) PlayerCantCastSpells(playerID uuid.UUID) bool {
	return g.effects.Rules.PlayerCantCastSpells(playerID)
}

// SetCreatureDamageRedirect saves a live creature for one damage redirect.
// An absent object cannot supply a later permanent.
func (g *Game) SetCreatureDamageRedirect(creatureID, playerID uuid.UUID) {
	subject := g.FindPermanent(creatureID)
	if subject == nil || g.GetPlayer(playerID) == nil {
		return
	}
	g.effects.AddReplacement(&creatureDamageRedirectReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		creatureID:      creatureID,
		incarnation:     subject.incarnationID,
		targetPlayerID:  playerID,
	})
}

// SetAttackerDamageRedirect saves live source and destination identities.
// The resolved rule keeps its references until cleanup.
func (g *Game) SetAttackerDamageRedirect(attackerID, absorberID uuid.UUID) {
	source, destination := g.FindPermanent(attackerID), g.FindPermanent(absorberID)
	if source == nil || destination == nil {
		return
	}
	g.effects.AddReplacement(&attackerDamageRedirectReplacement{
		replacementBase:     replacementBase{duration: EndOfTurn},
		attackerID:          attackerID,
		attackerIncarnation: source.incarnationID,
		absorberPermID:      absorberID,
		absorberIncarnation: destination.incarnationID,
	})
}

// SetSkipNextDraw sets a flag to skip the next draw step for the player.
func (g *Game) SetSkipNextDraw(playerID uuid.UUID) {
	g.effects.AddReplacement(&skipDrawReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID})
}

// SetSanctuaryActive marks the Ivory Tower sanctuary effect as active.
func (g *Game) SetSanctuaryActive(playerID uuid.UUID) {
	g.effects.Rules.SetSanctuaryActive(playerID)
}

// AddIslandSanctuaryReplacement registers the Island Sanctuary draw
// replacement: while sourceID is on the battlefield, the controller may skip
// their normal draw during their draw step to activate sanctuary protection.
func (g *Game) AddIslandSanctuaryReplacement(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&islandSanctuaryReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
		playerID:        playerID,
	})
}

// SetDeepWaterActive activates Deep Water for playerID until end of turn.
func (g *Game) SetDeepWaterActive(playerID uuid.UUID) {
	g.effects.Rules.SetDeepWaterActive(playerID)
}

// SetMinimumLife marks a player as having minimum-life protection (Ali from Cairo).
func (g *Game) SetMinimumLife(playerID uuid.UUID) {
	g.effects.AddCycleReplacement(&minimumLifeReplacement{playerID: playerID})
}

// AddSourcePrevention adds a one-shot damage prevention for the next damage
// from a specific source to the specified player (Circle of Protection: Artifacts).
func (g *Game) AddSourcePrevention(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&sourcePreventionReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, dmgSource: sourceID})
}

// AddSourcePreventionShield prevents up to amount damage from sourceID to the
// specified player this turn. A live source keeps its original incarnation.
func (g *Game) AddSourcePreventionShield(playerID, sourceID uuid.UUID, amount int) {
	if amount <= 0 {
		return
	}
	g.effects.AddReplacement(&sourcePreventionShieldReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
		remaining:       amount,
		incarnation:     g.legacyPreventionSourceIncarnation(sourceID),
	})
}

// AddHalfDamageFromSourcePreventionShield prevents half the damage (rounded down)
// from sourceID to playerID the next time sourceID would deal damage to playerID this turn.
// A live source keeps its original incarnation.
func (g *Game) AddHalfDamageFromSourcePreventionShield(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&halfDamagePreventionShieldReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
		incarnation:     g.legacyPreventionSourceIncarnation(sourceID),
	})
}

// AddTypePrevention adds a card-type damage prevention rule for the player.
func (g *Game) AddTypePrevention(playerID uuid.UUID, ct CardType) {
	g.effects.AddReplacement(&typePreventionReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, cardType: ct})
}

// PreventAllDamageFrom prevents all damage from the specified source until end of turn.
func (g *Game) PreventAllDamageFrom(sourceID uuid.UUID) {
	g.effects.AddReplacement(&damagePreventionRuleReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		from: NewPermanentFilter("specific source", func(p *Permanent, _ *Game) bool {
			return p.ID() == sourceID
		}),
	})
}

// SetArtifactDamageRedirect sets a creature that absorbs artifact damage dealt to a player.
func (g *Game) SetArtifactDamageRedirect(controllerID, permID uuid.UUID) {
	g.effects.AddCycleReplacement(&artifactDamageRedirectReplacement{
		controllerID:   controllerID,
		redirectPermID: permID,
		incarnation:    g.legacyPreventionSourceIncarnation(permID),
	})
}

// SetDamageReflection sets a one-shot damage reflection for a player (Eye for an Eye).
// This stays as inline logic (post-damage effect, not a replacement).
func (g *Game) SetDamageReflection(playerID, eyeSourceID, chosenSourceID uuid.UUID) {
	g.damage.SetDamageReflection(playerID, eyeSourceID, chosenSourceID)
}

// SetDrawReplacement stores a pending draw replacement for a player (Aladdin's Lamp).
func (g *Game) SetDrawReplacement(playerID uuid.UUID, count int) {
	g.effects.AddReplacement(&drawReplacementEffect{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, count: count})
}

// AddEmptyLibraryDrawReplacement registers a replacement effect that fires
// when playerID would draw a card while their library is empty. The
// replacement stays active while sourceID is on the battlefield (CR 614 +
// 614.6: replacement effects on a permanent function only while it's on
// the battlefield). The callback runs in place of the draw, receiving
// the game and source permanent ID. Used by Ormos, Archive Keeper —
// "If you would draw a card while your library has no cards in it,
// instead put five +1/+1 counters on Ormos."
func (g *Game) AddEmptyLibraryDrawReplacement(sourceID, playerID uuid.UUID, callback func(g *Game, sourceID uuid.UUID)) {
	g.effects.AddReplacement(&emptyLibraryDrawReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
		playerID:        playerID,
		callback:        callback,
	})
}

// AddReplacementEffect adds a replacement effect to the effect manager.
func (g *Game) AddReplacementEffect(r ReplacementEffect) {
	g.effects.AddReplacement(r)
}

// AddCountersWithReplacement places counters on a permanent after running the
// counter-placement replacement pipeline (CR 614). This is the API engine
// effects should use for counter placement so doubling effects (Branching
// Evolution) and "enters with an additional counter" effects (Oona's
// Blackguard) can intercept the placement. Direct calls to
// Permanent.AddCounter bypass the pipeline and are reserved for replacement
// implementations themselves and very low-level mechanics (Vanishing's Time
// counters, etc.).
//
// sourceID identifies the spell/ability/permanent causing the placement.
// onEntry must be true exactly when the placement happens during the
// permanent's enter-the-battlefield resolution, before EvtEntersBattlefield
// fires (used by EntersWithXCounters and EntersWithNCounters).
func (g *Game) AddCountersWithReplacement(perm *Permanent, ct CounterType, n int, sourceID uuid.UUID, onEntry bool) {
	g.addCountersWithReplacementLimit(perm, ct, n, sourceID, onEntry, nil)
}

func (g *Game) addCountersWithReplacementLimit(perm *Permanent, ct CounterType, n int, sourceID uuid.UUID, onEntry bool, limit *int) {
	if perm == nil || n <= 0 {
		return
	}
	action := NewAddCountersAction(sourceID, perm.ID(), ct, n, onEntry)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return
	}
	aca, ok := result.(*AddCountersAction)
	if !ok {
		return
	}
	target := g.MutablePermanent(aca.PermanentID())
	if target == nil {
		// During PutOnBattlefield the permanent isn't yet on the
		// battlefield slice; fall back to the caller-supplied pointer when
		// the IDs match (ETB additional/doubling for the same permanent).
		if perm.ID() == aca.PermanentID() {
			target = perm
		}
	}
	if target == nil {
		return
	}
	amount := aca.Amount()
	if limit != nil && aca.PermanentID() == perm.ID() {
		live := g.FindPermanent(perm.ID())
		if live == nil || live.incarnationID != perm.incarnationID {
			return
		}
		if aca.CounterType() == ct {
			amount = min(amount, max(*limit-int(target.Counters[ct]), 0))
		}
	}
	if amount > 0 {
		target.AddCounter(aca.CounterType(), amount)
	}
}

// AddCountersTogether prepares each placement before it changes any counters.
func (g *Game) AddCountersTogether(perms []*Permanent, ct CounterType, n int, sourceID uuid.UUID) {
	if n <= 0 {
		return
	}
	type placement struct {
		id          uuid.UUID
		incarnation uuid.UUID
		amount      int
		counter     CounterType
	}
	prepared := make([]placement, 0, len(perms))
	seen := make(map[uuid.UUID]bool, len(perms))
	for _, perm := range perms {
		if perm == nil || seen[perm.ID()] {
			continue
		}
		live := g.FindPermanent(perm.ID())
		if live == nil || live.incarnationID != perm.incarnationID {
			continue
		}
		seen[perm.ID()] = true
		result, ok := g.effects.ApplyReplacements(NewAddCountersAction(sourceID, live.ID(), ct, n, false), g).(*AddCountersAction)
		if !ok || result.Amount() <= 0 {
			continue
		}
		target := g.FindPermanent(result.PermanentID())
		if target != nil {
			prepared = append(prepared, placement{target.ID(), target.incarnationID, result.Amount(), result.CounterType()})
		}
	}
	for _, p := range prepared {
		if target := g.FindPermanent(p.id); target != nil && target.incarnationID == p.incarnation {
			g.MutablePermanent(p.id).AddCounter(p.counter, p.amount)
		}
	}
}

// AddCounterDoubler registers a counter-doubling replacement effect (CR
// 614.1c) for the given counter type. The filter restricts which permanents
// the doubler applies to (e.g. ControlledBy(playerID) for Branching
// Evolution). Pass an empty PermanentFilter to apply globally (Doubling
// Season-style for a specific counter type).
func (g *Game) AddCounterDoubler(sourceID uuid.UUID, ct CounterType, filter PermanentFilter) {
	g.effects.AddReplacement(&counterDoublerReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
		counterType:     ct,
		filter:          filter,
	})
}

// AddETBAdditionalCounters registers a replacement that puts N additional
// counters of the given type on each permanent matching filter as it enters
// the battlefield. excludeSelf=true keeps the source out of its own filter
// (e.g. "Each other Rogue creature you control enters with an additional
// +1/+1 counter on it").
func (g *Game) AddETBAdditionalCounters(sourceID uuid.UUID, ct CounterType, extra int, filter PermanentFilter, excludeSelf bool) {
	var incarnation uuid.UUID
	if source := g.FindPermanent(sourceID); source != nil {
		incarnation = source.incarnationID
	}
	g.effects.AddReplacement(&etbAdditionalCountersReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
		incarnation:     incarnation,
		counterType:     ct,
		extra:           extra,
		filter:          filter,
		excludeSelf:     excludeSelf,
	})
}

// GetArtifactDamageTaken returns the artifact damage the player has taken this turn.
func (g *Game) GetArtifactDamageTaken(playerID uuid.UUID) int {
	return g.trackers.Turn.ArtifactDamageTaken(playerID)
}

// CopyEffectCurrentName returns the name of the creature currently being copied
// by the Doppelganger copy effect for the specified permanent.
func (g *Game) CopyEffectCurrentName(permID uuid.UUID) string {
	return g.effects.CopyEffectCurrentName(permID)
}

// UpdateCopyEffect updates the Doppelganger copy effect to copy a new target.
func (g *Game) UpdateCopyEffect(permID uuid.UUID, target *Permanent) {
	g.effects.UpdateCopyEffect(permID, target)
}

// --- New Phase 1 proxy methods ---

// AllBattlefield returns all permanents on the battlefield.
func (g *Game) AllBattlefield() []*Permanent {
	if len(g.zones.battlefield) == 0 {
		return nil
	}
	return g.zones.battlefield[:len(g.zones.battlefield):len(g.zones.battlefield)]
}

// GetResolvingTargets returns the targets of the spell currently being resolved.
func (g *Game) GetResolvingTargets() []uuid.UUID { return g.resolution.ResolvingTargets() }

// GetArtifactUntapMax returns the maximum number of artifacts that may untap per turn.
func (g *Game) GetArtifactUntapMax() int { return g.effects.Rules.ArtifactUntapMax }

// ActivePlayerIndex returns the index of the active player in the Players slice.
func (g *Game) ActivePlayerIndex() int { return g.turns.ActivePlayerIndex() }

// SetXValue sets the X value for the currently resolving spell.
func (g *Game) SetXValue(x int) { g.resolution.SetX(x) }

// GrantAttr grants an attribute to a permanent via the effect manager.
func (g *Game) GrantAttr(permID uuid.UUID, a Attr) { g.effects.GrantAttr(permID, a) }

// RevokeAttr revokes an attribute from a permanent via the effect manager.
func (g *Game) RevokeAttr(permID uuid.UUID, a Attr) { g.effects.RevokeAttr(permID, a) }

// PreventBlockPair prevents a specific blocker from blocking a specific attacker.
func (g *Game) PreventBlockPair(blockerID, attackerID uuid.UUID) {
	g.effects.PreventBlockPair(blockerID, attackerID)
}

// AddDamagePreventionRule adds a damage prevention rule to the effect manager.
func (g *Game) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	dpr := &damagePreventionRule{}
	for _, opt := range opts {
		opt(dpr)
	}
	g.effects.AddCycleReplacement(&damagePreventionRuleReplacement{
		from:       dpr.from,
		to:         dpr.to,
		toPlayerID: dpr.toPlayerID,
		oneShot:    dpr.oneShot,
		combatOnly: dpr.combatOnly,
		playerOnly: dpr.playerOnly,
	})
}

// AddCycleReplacement adds a replacement effect that lasts for the current effect cycle.
func (g *Game) AddCycleReplacement(r ReplacementEffect) {
	g.effects.AddCycleReplacement(r)
}

// SetArtifactUntapMax sets the maximum number of artifacts that may untap per turn.
func (g *Game) SetArtifactUntapMax(maximum int) { g.effects.Rules.ArtifactUntapMax = maximum }

// AddSpellTypeCostReduction adds a generic cost reduction for spells of the given type.
func (g *Game) AddSpellTypeCostReduction(ct CardType, amount int) {
	g.effects.Rules.SpellTypeCostReductions[ct] += amount
}

// AddActivationCostReduction sets a generic mana reduction for a permanent's activated abilities.
func (g *Game) AddActivationCostReduction(permID uuid.UUID, amount int) {
	g.effects.Rules.ActivationCostReductions[permID] = amount
}

// SetMaxHandSize sets the maximum hand size for a player.
func (g *Game) SetMaxHandSize(playerID uuid.UUID, size int) {
	g.effects.Rules.SetMaxHandSize(playerID, size)
}

// SetNoMaximumHandSize removes the player's maximum hand size for the current
// continuous-effect cycle.
func (g *Game) SetNoMaximumHandSize(playerID uuid.UUID) {
	g.effects.Rules.SetNoMaximumHandSize(playerID)
}

// MaximumHandSize returns the player's current maximum hand size. A negative
// value means the player has no maximum hand size.
func (g *Game) MaximumHandSize(playerID uuid.UUID) int {
	return g.maximumHandSizeWithStaticRules(playerID)
}

// AddExpansionCastBlock blocks spells from the given set code from being cast.
func (g *Game) AddExpansionCastBlock(setCode string) {
	g.effects.Rules.AddExpansionCastBlock(setCode)
}

// AddEntersTappedRule registers a rule that causes matching permanents to enter tapped.
func (g *Game) AddEntersTappedRule(f func(*Permanent) bool) {
	g.effects.Rules.AddEntersTappedRule(f)
}

// SetCoinFlipResults sets deterministic coin flip results for testing.
func (g *Game) SetCoinFlipResults(results []bool) {
	g.random.SetCoinFlipResults(results)
}

// SetRandomResults sets raw deterministic RandIntn results for testing. Values
// are consumed in order and normalized to each requested range. The input is
// copied so later caller mutations do not affect the game.
func (g *Game) SetRandomResults(results []int) {
	g.random.SetRandomResults(results)
}

// SetOnPriority sets the priority handler callback.
func (g *Game) SetOnPriority(h PriorityHandler) { g.onPriority = h }

// SetAfterPriorityAction sets the after-priority-action callback.
func (g *Game) SetAfterPriorityAction(f func(*Game, int, PriorityAction)) {
	g.afterPriorityAction = f
}

// SetBeforeStackResolve sets the before-stack-resolve callback.
func (g *Game) SetBeforeStackResolve(f func(*Game)) { g.beforeStackResolve = f }

// SetOnDamageDealt sets the callback invoked after damage is dealt.
func (g *Game) SetOnDamageDealt(f func(sourceName, targetName string, amount int, isCombat bool)) {
	g.damage.SetOnDamageDealt(f)
}

// SetStep sets the current phase step.
func (g *Game) SetStep(s PhaseStep) { g.turns.SetStep(s) }

// GetStep returns the current phase step.
func (g *Game) GetStep() PhaseStep { return g.turns.Step() }

// SetTurn sets the current turn number.
func (g *Game) SetTurn(n int) {
	g.recordActiveTurnStart()
	g.turns.SetTurn(n)
}

// SetActivePlayerIndex sets which player is the active player by index.
func (g *Game) SetActivePlayerIndex(idx int) {
	g.recordActiveTurnStart()
	g.turns.SetActivePlayerIndex(idx)
}

// MostRecentTurnStart reads the player's last turn that began.
// Extra turns belong to their active player; skipped turns do not count.
func (g *Game) MostRecentTurnStart(player uuid.UUID) int {
	if g.GetPlayer(player) == nil {
		return 0
	}
	if g.ActivePlayerObj().PlayerID() == player {
		return g.CurrentTurn()
	}
	for index, candidate := range g.players {
		if candidate.PlayerID() == player {
			return g.turns.turnStarts[index]
		}
	}
	return 0
}

func (g *Game) recordActiveTurnStart() {
	g.turns.turnStarts[g.turns.ActivePlayerIndex()] = g.CurrentTurn()
}

func (g *Game) clearTurnStartSickness() {
	// The turn-start check must run even if the untap step is skipped.
	for _, permanent := range g.zones.battlefield {
		if permanent.ControllerID() == g.ActivePlayerObj().PlayerID() && permanent.turnControlGained < g.CurrentTurn() && permanent.baseAttrs[AttrSummonSick] > 0 {
			if current := g.MutablePermanentIncludingPhased(permanent.ID()); current != nil {
				current.RevokeBaseAttr(AttrSummonSick)
			}
		}
	}
}

// PopExtraTurn removes and returns the next extra turn player ID, if any.
func (g *Game) PopExtraTurn() (uuid.UUID, bool) {
	return g.turns.PopExtraTurn()
}

// HasExtraTurns reports whether there are pending extra turns.
func (g *Game) HasExtraTurns() bool { return g.turns.HasExtraTurns() }

// GetLandsPlayedThisTurn returns the number of lands played this turn.
func (g *Game) GetLandsPlayedThisTurn() int { return g.trackers.Turn.LandsPlayed() }

// GetCleanupPriorityRounds returns the total cleanup priority rounds granted.
func (g *Game) GetCleanupPriorityRounds() int { return g.trackers.Turn.CleanupPriorityRounds() }

// ApplyEffects applies all continuous effects to current permanents.
func (g *Game) ApplyEffects() { g.effects.Apply(g) }

// StackPeek returns the top stack object without removing it, or nil.
func (g *Game) StackPeek() *StackObject { return g.stack.Peek() }

// StackSize returns the number of objects on the stack.
func (g *Game) StackSize() int { return g.stack.Size() }

// StackObjects returns all objects on the stack.
func (g *Game) StackObjects() []*StackObject { return g.stack.Objects() }

// IsBandedWith reports whether two permanents are in the same attacking band.
func (g *Game) IsBandedWith(a, b uuid.UUID) bool {
	if g.combat == nil {
		return false
	}
	return g.combat.IsBandedWith(a, b)
}

// GetExile returns all exiled cards.
func (g *Game) GetExile() []ExiledCard { return g.zones.Exile() }

// PlayerCount returns the number of players in the game.
func (g *Game) PlayerCount() int { return len(g.players) }

// PlayerAt returns the player at the given index.
func (g *Game) PlayerAt(idx int) Player { return g.players[idx] }

// GetCombat returns the current combat state, or nil.
func (g *Game) GetCombat() *Combat { return g.combat }

// GetEffects returns the effect manager.
func (g *Game) GetEffects() *EffectManager { return g.effects }

// GetStack returns the stack.
func (g *Game) GetStack() *Stack { return g.stack }

// GetSchedule returns the current turn schedule, initializing it if needed.
func (g *Game) GetSchedule() *TurnSchedule {
	return g.turns.Schedule()
}

// AddToBattlefield appends permanents directly to the battlefield without ETB processing.
// Used by tests that construct permanents manually.
func (g *Game) AddToBattlefield(perms ...*Permanent) {
	g.zones.AddToBattlefield(perms...)
}

// SetLandsPlayedThisTurn sets the number of lands played this turn.
func (g *Game) SetLandsPlayedThisTurn(n int) { g.trackers.Turn.SetLandsPlayed(n) }

// TruncateBattlefield truncates the battlefield to the given length (for undo snapshots).
func (g *Game) TruncateBattlefield(n int) {
	g.zones.TruncateBattlefield(n)
}

// SetPlayerAt replaces the player at the given index.
func (g *Game) SetPlayerAt(idx int, p Player) { g.players[idx] = p }

// ExecuteAttackers declares the given creatures as attackers for AI search clones.
// It mirrors doDeclareAttackers but accepts explicit attacker IDs instead of
// querying the player.
func (g *Game) ExecuteAttackers(playerID uuid.UUID, attackerIDs []uuid.UUID) {
	var defender Player
	for _, p := range g.players {
		if p.PlayerID() != playerID {
			defender = p
			break
		}
	}
	if defender == nil {
		return
	}
	// Fresh search clones haven't run an Apply() cycle yet, so the per-cycle
	// attack-cost registry may be empty; rebuild it before consulting it.
	g.effects.Apply(g)
	attackerIDs = g.completeRequiredAttackers(playerID, attackerIDs, false)
	declaredCount := 0
	for _, id := range attackerIDs {
		if g.combat.IsAttacking(id) {
			continue
		}
		atk := g.FindPermanent(id)
		if atk == nil || !atk.CanDeclareAsAttacker(g) {
			continue
		}
		if !g.PayAttackCosts(atk, playerID, false) {
			continue
		}
		atk = g.attackerAfterPayment(atk, playerID)
		if atk == nil {
			continue
		}
		if !atk.HasKeyword(Vigilance) {
			g.TapPermanent(atk)
		}
		g.combat.AddAttackerIncarnation(id, atk.incarnationID, defender.PlayerID())
		declaredCount++
		g.trackers.Turn.RecordAttacked(id)
		g.trackers.Turn.RecordAttackedIncarnation(id, atk.incarnationID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: playerID,
			TargetID: defender.PlayerID(),
		})
	}
	g.combat.SnapshotAttackedAlone()
	if declaredCount > 0 {
		g.FireEvent(GameEvent{Type: EvtAttackersDeclared, PlayerID: playerID, Amount: declaredCount})
	}
}

// ExecuteBlockers declares blockers from explicit assignments for AI search clones.
// It mirrors doDeclareBlockers but accepts explicit block assignments.
func (g *Game) ExecuteBlockers(assignments []BlockAssignment) {
	blockedBefore := g.blockedAttackersBeforeDeclaration()
	pairsBefore := g.blockerPairsBeforeDeclaration()
	g.commitBlockAssignments(g.combatDefender(), assignments)
	g.combat.SnapshotBlockedAlone()
	g.fireDeclaredBlockerPairs(pairsBefore)
	g.fireNewBlockerPairs(pairsBefore)
	g.fireNewBlockerGroups(pairsBefore)
	g.fireNewlyBlockedAttackers(blockedBefore)
	g.fireUnblockedAttackers()

	// Mirror doDeclareBlockers: fire EvtBlockersDecl once after all blockers are
	// assigned, then resolve the resulting triggers. Without this, combat
	// abilities that key off the declare-blockers step are invisible to AI
	// search clones — e.g. Murk Dwellers' "attacks and isn't blocked, it gets
	// +2/+0" never fires, so the solver evaluates an unblocked Murk Dwellers as
	// a 2/2 dealing 2 instead of a 4/2 dealing 4. The event is fired even when
	// no blockers were declared, since "isn't blocked" triggers fire then.
	g.FireEvent(GameEvent{Type: EvtBlockersDecl})
	g.PutTriggersOnStack()
	g.ResolveStack()
}

// ExecuteCombatDamage resolves first-strike and normal combat damage for AI search clones.
func (g *Game) ExecuteCombatDamage() {
	g.damage.SetResolvingCombatDamage(true)
	if g.combat.HasFirstStrikers(g) {
		g.combat.ResolveDamage(g, true)
		g.CheckStateBasedActions()
		g.flushCombatDamageAggregator()
	}
	g.combat.ResolveDamage(g, false)
	g.damage.SetResolvingCombatDamage(false)
	g.flushCombatDamageAggregator()
}

// PutOnBattlefieldForSetup adds a starting permanent without state-trigger checks.
// State triggers observe the complete fixture when game actions begin.
func (g *Game) PutOnBattlefieldForSetup(card Card, controller uuid.UUID) *Permanent {
	paused := g.triggers.stateChecksPaused
	g.triggers.stateChecksPaused = true
	defer func() { g.triggers.stateChecksPaused = paused }()
	return g.PutOnBattlefield(card, controller)
}

// UpkeepHasEnded reports whether an upkeep has ended in this turn.
func (g *Game) UpkeepHasEnded() bool { return g.turns.firstUpkeepEnded }
