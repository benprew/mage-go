package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

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
	TimesTargetedThisTurn(uuid.UUID) int
	AllBattlefield() []*Permanent
	GetResolvingTargets() []uuid.UUID
	FindPermanentIncludingPhased(uuid.UUID) *Permanent
	GetArtifactUntapMax() int
	ActivePlayerIndex() int
	CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int
}

// Compile-time check that *Game satisfies GameReader.
var _ GameReader = (*Game)(nil)

// --- GameReader proxy methods on *Game ---

// AllPlayers returns all players in the game.
func (g *Game) AllPlayers() []Player { return g.players }

// XValue returns the current X value for the resolving spell/ability.
func (g *Game) XValue() int { return g.currentX }

// ModeValue returns the current chosen mode for the resolving modal spell.
func (g *Game) ModeValue() int { return g.currentMode }

// EventAmount returns the amount from the triggering event (e.g. damage dealt).
func (g *Game) EventAmount() int { return g.currentEventAmount }

// EventSourceID returns the SourceID of the event that triggered the
// currently-resolving triggered ability. For an EvtDamageDealt trigger,
// this is the damager's permanent ID. Returns uuid.Nil when there is no
// trigger context (e.g. spell resolution).
func (g *Game) EventSourceID() uuid.UUID { return g.currentEventSourceID }

// GetResolvingCard returns the card currently being resolved from the stack.
func (g *Game) GetResolvingCard() Card { return g.resolvingCard }

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
	return g.damageTakenThisTurn[playerID]
}

// HasAttackedThisTurn reports whether the permanent with the given ID attacked this turn.
func (g *Game) HasAttackedThisTurn(permID uuid.UUID) bool {
	return g.attackedThisTurn[permID]
}

// CreatureDeaths returns the number of creatures that died this turn.
func (g *Game) CreatureDeaths() int {
	return g.creatureDeathsThisTurn
}

func (g *Game) CurrentTurn() int {
	return g.turn
}

// GetDamageSources returns the set of permanent IDs that dealt damage to the
// given permanent this turn. Returns nil if nothing dealt damage.
func (g *Game) GetDamageSources(permID uuid.UUID) map[uuid.UUID]bool {
	return g.damageDealtBy[permID]
}

// GetBlockedThisTurn returns the list of attacker IDs that the given blocker
// blocked this turn. Returns nil if it didn't block anything.
func (g *Game) GetBlockedThisTurn(blockerID uuid.UUID) []uuid.UUID {
	return g.blockedThisTurn[blockerID]
}

// GetInstantsCastThisTurn returns the number of instants the given player has cast this turn.
func (g *Game) GetInstantsCastThisTurn(playerID uuid.UUID) int {
	return g.instantsCastThisTurn[playerID]
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
	g.stack.Push(obj)
}

// AddContinuousEffect registers a continuous effect with the effect manager.
func (g *Game) AddContinuousEffect(e ContinuousEffect) {
	g.effects.Add(e)
}

// ApplyContinuousEffects re-applies all continuous effects to current permanents.
func (g *Game) ApplyContinuousEffects() {
	g.effects.Apply(g)
}

// SetPreventCombatDamage flags that all combat damage is prevented this turn.
func (g *Game) SetPreventCombatDamage() {
	g.effects.AddReplacement(&fogReplacement{replacementBase: replacementBase{sourceID: uuid.Nil, duration: EndOfTurn}})
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

// AddForcefieldShield adds a Forcefield shield for the specified player.
func (g *Game) AddForcefieldShield(id uuid.UUID) {
	g.effects.AddReplacement(&forcefieldReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: id})
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

// AddReverseDamageShield adds a reverse-damage shield for the player.
// Prepended so it is checked before any prevention shields (which would
// otherwise absorb the damage before the reverse replacement sees it).
func (g *Game) AddReverseDamageShield(playerID uuid.UUID) {
	g.effects.PrependReplacement(&reverseDamageReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID})
}

// SetChannelActive marks the Channel ability as active for the player.
func (g *Game) SetChannelActive(playerID uuid.UUID) {
	g.effects.Rules.SetChannelActive(playerID)
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

// SetMinimumLife marks a player as having minimum-life protection (Ali from Cairo).
func (g *Game) SetMinimumLife(playerID uuid.UUID) {
	g.effects.AddCycleReplacement(&minimumLifeReplacement{playerID: playerID})
}

// AddSourcePrevention adds a one-shot damage prevention for the next damage
// from a specific source to the specified player (Circle of Protection: Artifacts).
func (g *Game) AddSourcePrevention(playerID, sourceID uuid.UUID) {
	g.effects.AddReplacement(&sourcePreventionReplacement{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, dmgSource: sourceID})
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
	g.effects.Damage.SetDamageReflection(playerID, eyeSourceID, chosenSourceID)
}

// SetDrawReplacement stores a pending draw replacement for a player (Aladdin's Lamp).
func (g *Game) SetDrawReplacement(playerID uuid.UUID, count int) {
	g.effects.AddReplacement(&drawReplacementEffect{replacementBase: replacementBase{duration: EndOfTurn}, playerID: playerID, count: count})
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
	target := g.FindPermanent(aca.PermanentID())
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

// SetArtifactManaOnly marks a player as having artifact-only mana restriction active.
func (g *Game) SetArtifactManaOnly(playerID uuid.UUID) {
	if g.artifactManaOnly == nil {
		g.artifactManaOnly = make(map[uuid.UUID]bool)
	}
	g.artifactManaOnly[playerID] = true
}

// SetCreatureManaOnly marks a player as having creature-only mana restriction active.
func (g *Game) SetCreatureManaOnly(playerID uuid.UUID) {
	if g.creatureManaOnly == nil {
		g.creatureManaOnly = make(map[uuid.UUID]bool)
	}
	g.creatureManaOnly[playerID] = true
}

// GetArtifactDamageTaken returns the artifact damage the player has taken this turn.
func (g *Game) GetArtifactDamageTaken(playerID uuid.UUID) int {
	return g.artifactDamageTakenThisTurn[playerID]
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
func (g *Game) AllBattlefield() []*Permanent { return g.battlefield }

// GetResolvingTargets returns the targets of the spell currently being resolved.
func (g *Game) GetResolvingTargets() []uuid.UUID { return g.resolvingTargets }

// GetArtifactUntapMax returns the maximum number of artifacts that may untap per turn.
func (g *Game) GetArtifactUntapMax() int { return g.effects.Rules.ArtifactUntapMax }

// ActivePlayerIndex returns the index of the active player in the Players slice.
func (g *Game) ActivePlayerIndex() int { return g.activePlayer }

// SetXValue sets the X value for the currently resolving spell.
func (g *Game) SetXValue(x int) { g.currentX = x }

// GrantAttr grants an attribute to a permanent via the effect manager.
func (g *Game) GrantAttr(permID uuid.UUID, a Attr) { g.effects.GrantAttr(permID, a) }

// RevokeAttr revokes an attribute from a permanent via the effect manager.
func (g *Game) RevokeAttr(permID uuid.UUID, a Attr) { g.effects.RevokeAttr(permID, a) }

// PreventBlockPair prevents a specific blocker from blocking a specific attacker.
func (g *Game) PreventBlockPair(blockerID, attackerID uuid.UUID) {
	g.effects.PreventBlockPair(blockerID, attackerID)
}

// AddDamagePreventionRule adds a damage prevention rule to the damage system.
func (g *Game) AddDamagePreventionRule(opts ...damagePreventionRuleOption) {
	g.effects.Damage.AddDamagePreventionRule(opts...)
}

// AddCycleReplacement adds a replacement effect that lasts for the current effect cycle.
func (g *Game) AddCycleReplacement(r ReplacementEffect) {
	g.effects.AddCycleReplacement(r)
}

// SetArtifactUntapMax sets the maximum number of artifacts that may untap per turn.
func (g *Game) SetArtifactUntapMax(max int) { g.effects.Rules.ArtifactUntapMax = max }

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

// AddExpansionCastBlock blocks spells from the given set code from being cast.
func (g *Game) AddExpansionCastBlock(setCode string) {
	g.effects.Rules.AddExpansionCastBlock(setCode)
}

// AddEntersTappedRule registers a rule that causes matching permanents to enter tapped.
func (g *Game) AddEntersTappedRule(f func(*Permanent) bool) {
	g.effects.Rules.AddEntersTappedRule(f)
}

// SetCoinFlipResults sets deterministic coin flip results for testing.
func (g *Game) SetCoinFlipResults(results []bool) { g.coinFlipResults = results }

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
	g.onDamageDealt = f
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
func (g *Game) GetLandsPlayedThisTurn() int { return g.landsPlayedThisTurn }

// GetCleanupPriorityRounds returns the total cleanup priority rounds granted.
func (g *Game) GetCleanupPriorityRounds() int { return g.cleanupPriorityRounds }

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
	g.battlefield = append(g.battlefield, perms...)
}

// SetLandsPlayedThisTurn sets the number of lands played this turn.
func (g *Game) SetLandsPlayedThisTurn(n int) { g.landsPlayedThisTurn = n }

// TruncateBattlefield truncates the battlefield to the given length (for undo snapshots).
func (g *Game) TruncateBattlefield(n int) { g.battlefield = g.battlefield[:n] }

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
	for _, id := range attackerIDs {
		atk := g.FindPermanent(id)
		if atk == nil || !atk.CanDeclareAsAttacker(g) {
			continue
		}
		if !atk.HasKeyword(Vigilance) {
			g.TapPermanent(atk)
		}
		g.combat.AddAttacker(id, defender.PlayerID())
		g.attackedThisTurn[id] = true
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
	if len(assignments) == 0 {
		return
	}
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
		g.blockedThisTurn[ba.BlockerID] = append(g.blockedThisTurn[ba.BlockerID], ba.AttackerID)
		g.FireEvent(GameEvent{
			Type:     EvtDeclaredBlocker,
			SourceID: ba.BlockerID,
			TargetID: ba.AttackerID,
			Flag:     firstForBlocker,
		})
	}
	g.combat.SnapshotBlockedAlone()
}

// ExecuteCombatDamage resolves first-strike and normal combat damage for AI search clones.
func (g *Game) ExecuteCombatDamage() {
	g.resolvingCombatDamage = true
	if g.combat.HasFirstStrikers(g) {
		g.combat.ResolveDamage(g, true)
		g.CheckStateBasedActions()
		g.flushCombatDamageAggregator()
	}
	g.combat.ResolveDamage(g, false)
	g.resolvingCombatDamage = false
	g.flushCombatDamageAggregator()
}
