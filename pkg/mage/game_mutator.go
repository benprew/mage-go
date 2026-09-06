package mage

import (
	"maps"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// GameReader is the read-only view of Game used by ValueSource and PlayerSelector.
// It exposes query methods but no mutation. *Game satisfies this interface.
type GameReader interface {
	GetPlayer(uuid.UUID) Player
	GetOpponent(uuid.UUID) Player
	ActivePlayerObj() Player
	NonActivePlayerObj() Player
	FindPermanent(uuid.UUID) *Permanent
	FindPermanentByName(string, uuid.UUID) *Permanent
	FindCardAnywhere(uuid.UUID) Card
	AnyBattlefield(PermanentFilter) bool
	FilterBattlefield(PermanentFilter) []*Permanent
	CountBattlefield(PermanentFilter) int
	AllPlayers() []Player
	XValue() int
	ModeValue() int
	EventAmount() int
	EventSourceID() uuid.UUID
	GetResolvingCard() Card
	ResolvingCastZone() Zone
	ResolvingCastContext() *CastContext
	FindStackObject(uuid.UUID) *StackObject
	CombatGroups() []*CombatGroup
	CombatGroupFor(uuid.UUID) *CombatGroup
	IsAttackingInCombat(uuid.UUID) bool
	IsBlockingInCombat(uuid.UUID) bool
	DamageTakenByPlayer(uuid.UUID) int
	HasAttackedThisTurn(uuid.UUID) bool
	CreatureDeaths() int
	CurrentTurn() int
	GetDamageSources(uuid.UUID) map[uuid.UUID]bool
	GetBlockedThisTurn(uuid.UUID) []uuid.UUID
	GetInstantsCastThisTurn(uuid.UUID) int
	UntappedLandsAtTurnStart(uuid.UUID) int
	GetSorceriesCastThisTurn(uuid.UUID) int
	GetInstantOrSorceryCastThisTurn(uuid.UUID) int
	TimesTargetedThisTurn(uuid.UUID) int
	AllBattlefield() []*Permanent
	GetResolvingTargets() []uuid.UUID
	FindPermanentIncludingPhased(uuid.UUID) *Permanent
	GetArtifactUntapMax() int
	ActivePlayerIndex() int
	CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int
	HypotheticalMana(uuid.UUID) int // returns the amount of hypthetical mana a player has available
	PlayerCardsDrawnThisTurn(uuid.UUID) int
	PlayerCardsLeftGraveyardThisTurn(uuid.UUID) int
	PlayerHadCardLeaveGraveyardThisTurn(uuid.UUID) bool
	CardsPutIntoExileThisTurn() int
	// FlipCoin is NOT a pure query: each call is a fresh flip (and consumes
	// scripted test results). It exists here so FlipCoinCond can run at effect
	// resolution; never call it from trigger or state-trigger conditions,
	// which may be evaluated repeatedly.
	FlipCoin(uuid.UUID) bool
	LastExiledCard() Card
}

// Compile-time check that *Game satisfies GameReader.
var _ GameReader = (*Game)(nil)

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
		ControllerSubtypesAtCast: make(map[string]bool),
	}
	for _, perm := range g.AllBattlefield() {
		if perm == nil || perm.ControllerID() != playerID || perm.FaceDown {
			continue
		}
		for _, st := range perm.computedSubtypes() {
			ctx.ControllerSubtypesAtCast[st] = true
		}
	}
	if lastReveal := g.resolution.LastCostReveal(); lastReveal != nil {
		ctx.RevealedAtCast = append(ctx.RevealedAtCast, lastReveal)
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

// IsAttackingInCombat reports whether the permanent is a declared attacker.
func (g *Game) IsAttackingInCombat(permID uuid.UUID) bool {
	if g.combat == nil {
		return false
	}
	return g.combat.IsAttacking(permID)
}

// IsBlockingInCombat reports whether the permanent is a declared blocker.
func (g *Game) IsBlockingInCombat(permID uuid.UUID) bool {
	if g.combat == nil {
		return false
	}
	return g.combat.IsBlocking(permID)
}

// DamageTakenByPlayer returns the total damage the given player has taken this turn.
func (g *Game) DamageTakenByPlayer(playerID uuid.UUID) int {
	return g.trackers.Turn.DamageTaken(playerID)
}

// HasAttackedThisTurn reports whether the permanent with the given ID attacked this turn.
func (g *Game) HasAttackedThisTurn(permID uuid.UUID) bool {
	return g.trackers.Turn.Attacked(permID)
}

// CreatureDeaths returns the number of creatures that died this turn.
func (g *Game) CreatureDeaths() int {
	return g.trackers.Turn.CreatureDeaths()
}

func (g *Game) CurrentTurn() int {
	return g.turn
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
	g.extraTurns = append(g.extraTurns, playerID)
}

// RemoveFromCombat removes a permanent from combat by ID.
func (g *Game) RemoveFromCombat(id uuid.UUID) {
	g.combat.RemoveFromCombat(id)
}

// PushStack pushes a stack object onto the stack.
func (g *Game) PushStack(obj *StackObject) {
	g.pushStack(obj)
}

// AddContinuousEffect registers a continuous effect with the effect manager.
func (g *Game) AddContinuousEffect(e ContinuousEffect) {
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
	g.effects.AddReplacement(&creatureCombatDamagePreventionReplacement{
		replacementBase: replacementBase{sourceID: uuid.Nil, duration: EndOfTurn},
		creatureID:      creatureID,
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
	// Find existing regeneration replacement for this permanent and increment
	for _, r := range g.effects.replacements {
		if regen, ok := r.(*regenerationReplacement); ok && regen.permanentID == id {
			regen.shields++
			return
		}
	}
	g.effects.AddReplacement(&regenerationReplacement{replacementBase: replacementBase{duration: EndOfTurn}, permanentID: id, shields: 1})
}

// HasRegenerationShield reports whether the specified permanent currently has at
// least one active regeneration shield.
func (g *Game) HasRegenerationShield(id uuid.UUID) bool {
	for _, r := range g.effects.replacements {
		if regen, ok := r.(*regenerationReplacement); ok && regen.permanentID == id && regen.shields > 0 {
			return true
		}
	}
	return false
}

// AddPreventionShield adds a damage prevention shield to the specified permanent or player.
func (g *Game) AddPreventionShield(id uuid.UUID, amount int) {
	// Find existing prevention shield for this target and add to it
	for _, r := range g.effects.replacements {
		if ps, ok := r.(*preventionShieldReplacement); ok && ps.targetID == id {
			ps.remaining += amount
			return
		}
	}
	g.effects.AddReplacement(&preventionShieldReplacement{replacementBase: replacementBase{duration: EndOfTurn}, targetID: id, remaining: amount})
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
// Prepended so it is checked before any prevention shields (which would
// otherwise absorb the damage before the reverse replacement sees it).
func (g *Game) AddReverseDamageShield(playerID, sourceID uuid.UUID) {
	if sourceID == uuid.Nil {
		return
	}
	g.effects.PrependReplacement(&reverseDamageReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
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

// SetCreatureDamageRedirect redirects damage dealt to a creature to a player.
func (g *Game) SetCreatureDamageRedirect(creatureID, playerID uuid.UUID) {
	g.effects.AddReplacement(&creatureDamageRedirectReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		creatureID:      creatureID,
		targetPlayerID:  playerID,
	})
}

// SetAttackerDamageRedirect sets a redirect: damage from a specific attacking creature
// to a player is dealt to the absorber permanent instead (Shimian Night Stalker).
func (g *Game) SetAttackerDamageRedirect(attackerID, absorberID uuid.UUID) {
	g.effects.AddReplacement(&attackerDamageRedirectReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		attackerID:      attackerID,
		absorberPermID:  absorberID,
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

// AddFastingReplacement registers the Fasting draw replacement: while sourceID is
// on the battlefield, the controller may skip their draw step to gain 2 life.
func (g *Game) AddFastingReplacement(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&fastingReplacement{
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
// specified player during the next matching damage event.
func (g *Game) AddSourcePreventionShield(playerID, sourceID uuid.UUID, amount int) {
	if amount <= 0 {
		return
	}
	g.effects.AddReplacement(&sourcePreventionShieldReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
		remaining:       amount,
	})
}

// AddHalfDamageFromSourcePreventionShield prevents half the damage (rounded down)
// from sourceID to playerID the next time sourceID would deal damage to playerID this turn.
func (g *Game) AddHalfDamageFromSourcePreventionShield(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&halfDamagePreventionShieldReplacement{
		replacementBase: replacementBase{duration: EndOfTurn},
		playerID:        playerID,
		dmgSource:       sourceID,
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
	if aca.Amount() > 0 {
		target.AddCounter(aca.CounterType(), aca.Amount())
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
	g.effects.AddReplacement(&etbAdditionalCountersReplacement{
		replacementBase: replacementBase{sourceID: sourceID, duration: WhileOnBattlefield},
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
	if len(g.battlefield) == 0 {
		return nil
	}
	return g.battlefield[:len(g.battlefield):len(g.battlefield)]
}

// GetResolvingTargets returns the targets of the spell currently being resolved.
func (g *Game) GetResolvingTargets() []uuid.UUID { return g.resolution.ResolvingTargets() }

// GetArtifactUntapMax returns the maximum number of artifacts that may untap per turn.
func (g *Game) GetArtifactUntapMax() int { return g.effects.Rules.ArtifactUntapMax }

// ActivePlayerIndex returns the index of the active player in the Players slice.
func (g *Game) ActivePlayerIndex() int { return g.activePlayer }

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
	return g.effects.Rules.MaxHandSize(playerID)
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
func (g *Game) SetStep(s PhaseStep) { g.step = s }

// GetStep returns the current phase step.
func (g *Game) GetStep() PhaseStep { return g.step }

// SetTurn sets the current turn number.
func (g *Game) SetTurn(n int) { g.turn = n }

// SetActivePlayerIndex sets which player is the active player by index.
func (g *Game) SetActivePlayerIndex(idx int) { g.activePlayer = idx }

// PopExtraTurn removes and returns the next extra turn player ID, if any.
func (g *Game) PopExtraTurn() (uuid.UUID, bool) {
	if len(g.extraTurns) == 0 {
		return uuid.UUID{}, false
	}
	id := g.extraTurns[0]
	g.extraTurns = g.extraTurns[1:]
	return id, true
}

// HasExtraTurns reports whether there are pending extra turns.
func (g *Game) HasExtraTurns() bool { return len(g.extraTurns) > 0 }

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
func (g *Game) GetExile() []ExiledCard { return g.exile }

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
	if g.schedule == nil {
		g.schedule = newTurnSchedule()
	}
	return g.schedule
}

// AddToBattlefield appends permanents directly to the battlefield without ETB processing.
// Used by tests that construct permanents manually.
func (g *Game) AddToBattlefield(perms ...*Permanent) {
	g.ensureBattlefieldSliceOwned()
	for _, p := range perms {
		g.addOwnedPermanent(p)
	}
	g.battlefield = append(g.battlefield, perms...)
}

// SetLandsPlayedThisTurn sets the number of lands played this turn.
func (g *Game) SetLandsPlayedThisTurn(n int) { g.trackers.Turn.SetLandsPlayed(n) }

// TruncateBattlefield truncates the battlefield to the given length (for undo snapshots).
func (g *Game) TruncateBattlefield(n int) {
	g.ensureBattlefieldSliceOwned()
	if g.ownedPermanents != nil {
		for _, p := range g.battlefield[n:] {
			delete(g.ownedPermanents, p.ID())
		}
	}
	g.battlefield = g.battlefield[:n]
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
	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil || !atk.CanDeclareAsAttacker(g) {
			continue
		}
		if !g.PayAttackCosts(atk, playerID, false) {
			continue
		}
		if !atk.HasKeyword(Vigilance) {
			g.TapPermanent(atk)
		}
		g.combat.AddAttacker(id, defender.PlayerID())
		g.trackers.Turn.RecordAttacked(id)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredAttacker,
			SourceID: id,
			PlayerID: playerID,
		})
	}
	g.combat.SnapshotAttackedAlone()
}

// ExecuteBlockers declares blockers from explicit assignments for AI search clones.
// It mirrors doDeclareBlockers but accepts explicit block assignments.
func (g *Game) ExecuteBlockers(assignments []BlockAssignment) {
	blockerCount := make(map[uuid.UUID]int)
	for _, ba := range assignments {
		blocker := g.FindPermanent(ba.BlockerID)
		attacker := g.FindPermanent(ba.AttackerID)
		if blocker == nil || attacker == nil {
			continue
		}
		if !blocker.CanDeclareAsBlocker(g) || !CanBlock(blocker, attacker, g) {
			continue
		}
		maxBlocks := 1
		if blocker.HasKeyword(CanBlockAny) {
			maxBlocks = 999
		} else if blocker.HasKeyword(CanBlockAdditional) {
			maxBlocks = 2
		}
		if blockerCount[ba.BlockerID] >= maxBlocks {
			continue
		}
		firstForBlocker := blockerCount[ba.BlockerID] == 0
		blockerCount[ba.BlockerID]++
		g.combat.AddBlocker(ba.BlockerID, ba.AttackerID)
		g.trackers.Turn.RecordBlocked(ba.BlockerID, ba.AttackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: ba.AttackerID,
			Flag:     firstForBlocker,
		})
	}
	g.combat.SnapshotBlockedAlone()

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
