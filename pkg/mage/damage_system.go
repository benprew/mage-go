package mage

import (
	"maps"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// DamageSystem manages damage execution, replacement and prevention coordination,
// combat damage aggregation, per-turn/per-duel damage history, and damage reflection.
type DamageSystem struct {
	damageReflection                   map[uuid.UUID]damageReflectionEntry
	damageDealtBy                      map[uuid.UUID]map[uuid.UUID]bool
	damageDealtToPlayersByPermanent    map[uuid.UUID]map[uuid.UUID]bool
	damageDealtToPermanentsByPermanent map[uuid.UUID]map[uuid.UUID]bool
	combatDamageThisStep               map[uuid.UUID]map[uuid.UUID]int
	combatDamageSourcesThisStep        map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int
	resolvingCombatDamage              bool
	onDamageDealt                      func(source, target string, amount int, isCombat bool)
}

type damageRequest struct {
	permanent   *Permanent
	incarnation uuid.UUID
	playerID    uuid.UUID
	amount      int
	sourceID    uuid.UUID
	source      DamageSource
}

// DamageAssignment identifies one recipient, amount, and captured source.
type DamageAssignment struct{ request damageRequest }

// PlayerDamageAssignment prepares damage to one player.
func PlayerDamageAssignment(player Player, amount int, source DamageSource) DamageAssignment {
	request := damageRequest{amount: amount, source: source, sourceID: source.ID()}
	if player != nil {
		request.playerID = player.PlayerID()
	}
	return DamageAssignment{request}
}

// PermanentDamageAssignment prepares damage to one permanent.
func PermanentDamageAssignment(permanent *Permanent, amount int, source DamageSource) DamageAssignment {
	request := damageRequest{permanent: permanent, amount: amount, sourceID: source.ID(), source: source}
	if permanent != nil {
		request.incarnation = permanent.incarnationID
	}
	return DamageAssignment{request}
}

// DealSimultaneousDamage applies all damage before it sends damage events.
func (g *Game) DealSimultaneousDamage(assignments ...DamageAssignment) {
	g.dealSimultaneousDamageTotal(assignments...)
}

func (g *Game) dealSimultaneousDamageTotal(assignments ...DamageAssignment) int {
	total := 0
	for _, outcome := range g.dealSimultaneousDamageOutcomes(assignments...) {
		total += outcome.amount
	}
	return total
}

func (g *Game) dealSimultaneousDamageOutcomes(assignments ...DamageAssignment) []damageOutcome {
	requests := make([]damageRequest, 0, len(assignments))
	for _, assignment := range assignments {
		requests = append(requests, assignment.request)
	}
	return g.damage.dealSimultaneousDamageOutcomes(g, requests)
}

type damageNotification struct {
	source, target string
	amount         int
	combat         bool
}

type damageBatch struct {
	events        []GameEvent
	notifications []damageNotification
}

// damageReflectionEntry tracks the Eye for an Eye reflection state for a player.
type damageReflectionEntry struct {
	eyeSourceID  uuid.UUID
	chosenSource uuid.UUID
}

// NewDamageSystem creates a DamageSystem with all maps initialized.
func NewDamageSystem() DamageSystem {
	return DamageSystem{
		damageReflection:                   make(map[uuid.UUID]damageReflectionEntry),
		damageDealtBy:                      make(map[uuid.UUID]map[uuid.UUID]bool),
		damageDealtToPlayersByPermanent:    make(map[uuid.UUID]map[uuid.UUID]bool),
		damageDealtToPermanentsByPermanent: make(map[uuid.UUID]map[uuid.UUID]bool),
		combatDamageThisStep:               make(map[uuid.UUID]map[uuid.UUID]int),
		combatDamageSourcesThisStep:        make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int),
	}
}

// Clone creates a deep copy of DamageSystem for search clones.
func (ds *DamageSystem) Clone() DamageSystem {
	clone := DamageSystem{
		resolvingCombatDamage: ds.resolvingCombatDamage,
	}
	if len(ds.damageReflection) > 0 {
		clone.damageReflection = make(map[uuid.UUID]damageReflectionEntry, len(ds.damageReflection))
		maps.Copy(clone.damageReflection, ds.damageReflection)
	} else {
		clone.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
	}
	if len(ds.damageDealtBy) > 0 {
		clone.damageDealtBy = cloneNestedUUIDMap(ds.damageDealtBy)
	} else {
		clone.damageDealtBy = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	if len(ds.damageDealtToPlayersByPermanent) > 0 {
		clone.damageDealtToPlayersByPermanent = cloneNestedUUIDMap(ds.damageDealtToPlayersByPermanent)
	} else {
		clone.damageDealtToPlayersByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	if len(ds.damageDealtToPermanentsByPermanent) > 0 {
		clone.damageDealtToPermanentsByPermanent = cloneNestedUUIDMap(ds.damageDealtToPermanentsByPermanent)
	} else {
		clone.damageDealtToPermanentsByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
	}
	if len(ds.combatDamageThisStep) > 0 {
		clone.combatDamageThisStep = make(map[uuid.UUID]map[uuid.UUID]int, len(ds.combatDamageThisStep))
		for k, inner := range ds.combatDamageThisStep {
			clone.combatDamageThisStep[k] = cloneUUIDIntMap(inner)
		}
	} else {
		clone.combatDamageThisStep = make(map[uuid.UUID]map[uuid.UUID]int)
	}
	if len(ds.combatDamageSourcesThisStep) > 0 {
		clone.combatDamageSourcesThisStep = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int, len(ds.combatDamageSourcesThisStep))
		for k, byRecip := range ds.combatDamageSourcesThisStep {
			dst := make(map[uuid.UUID]map[uuid.UUID]int, len(byRecip))
			for r, bySrc := range byRecip {
				dst[r] = cloneUUIDIntMap(bySrc)
			}
			clone.combatDamageSourcesThisStep[k] = dst
		}
	} else {
		clone.combatDamageSourcesThisStep = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int)
	}
	return clone
}

// ResetPerCycle resets all state that is recomputed each Apply() cycle.
func (ds *DamageSystem) ResetPerCycle() {
}

// ClearEndOfTurn resets all turn-scoped damage state.
func (ds *DamageSystem) ClearEndOfTurn() {
	ds.expireEndOfTurnEffects()
	ds.damageDealtBy = make(map[uuid.UUID]map[uuid.UUID]bool)
}

func (ds *DamageSystem) expireEndOfTurnEffects() {
	ds.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
}

// SetDamageReflection sets a one-shot damage reflection for a player (Eye for an Eye).
func (ds *DamageSystem) SetDamageReflection(playerID, eyeSourceID, chosenSourceID uuid.UUID) {
	if ds.damageReflection == nil {
		ds.damageReflection = make(map[uuid.UUID]damageReflectionEntry)
	}
	ds.damageReflection[playerID] = damageReflectionEntry{
		eyeSourceID:  eyeSourceID,
		chosenSource: chosenSourceID,
	}
}

// GetDamageReflection returns the reflection entry if reflection is active for the player.
func (ds *DamageSystem) GetDamageReflection(playerID uuid.UUID) (damageReflectionEntry, bool) {
	if ds.damageReflection == nil {
		return damageReflectionEntry{}, false
	}
	entry, ok := ds.damageReflection[playerID]
	return entry, ok
}

// ClearDamageReflection clears the damage reflection for a player.
func (ds *DamageSystem) ClearDamageReflection(playerID uuid.UUID) {
	if ds.damageReflection != nil {
		delete(ds.damageReflection, playerID)
	}
}

// SetResolvingCombatDamage sets whether combat damage is currently resolving.
func (ds *DamageSystem) SetResolvingCombatDamage(v bool) {
	ds.resolvingCombatDamage = v
}

// IsResolvingCombatDamage reports whether combat damage is currently resolving.
func (ds *DamageSystem) IsResolvingCombatDamage() bool {
	return ds.resolvingCombatDamage
}

// SetOnDamageDealt registers a callback invoked when damage is dealt.
func (ds *DamageSystem) SetOnDamageDealt(cb func(source, target string, amount int, isCombat bool)) {
	ds.onDamageDealt = cb
}

// HasDealtDamageToPlayer reports whether the given source permanent has dealt damage to playerID this game.
func (ds *DamageSystem) HasDealtDamageToPlayer(sourceID, playerID uuid.UUID) bool {
	if ds.damageDealtToPlayersByPermanent == nil {
		return false
	}
	if m, ok := ds.damageDealtToPlayersByPermanent[sourceID]; ok {
		return m[playerID]
	}
	return false
}

// HasDealtDamageToPermanent reports whether the given source permanent has dealt damage to permID this game.
func (ds *DamageSystem) HasDealtDamageToPermanent(sourceID, permID uuid.UUID) bool {
	if ds.damageDealtToPermanentsByPermanent == nil {
		return false
	}
	if m, ok := ds.damageDealtToPermanentsByPermanent[sourceID]; ok {
		return m[permID]
	}
	return false
}

// DamageDealtBy returns the set of sources that dealt damage to permID this turn.
func (ds *DamageSystem) DamageDealtBy(permID uuid.UUID) map[uuid.UUID]bool {
	if ds.damageDealtBy == nil {
		return nil
	}
	return ds.damageDealtBy[permID]
}

// CombatDamageSourcesThisStep returns the per-source combat damage breakdown for the given pair this step.
func (ds *DamageSystem) CombatDamageSourcesThisStep(controllerID, recipientID uuid.UUID) map[uuid.UUID]int {
	if ds.combatDamageSourcesThisStep == nil {
		return nil
	}
	byCtrl, ok := ds.combatDamageSourcesThisStep[controllerID]
	if !ok {
		return nil
	}
	return byCtrl[recipientID]
}

// FlushCombatDamageAggregator fires EvtCombatDamageDealt once per (controller, recipient-player)
// pair that took combat damage this step and resets step aggregators.
func (ds *DamageSystem) FlushCombatDamageAggregator(g *Game) {
	if len(ds.combatDamageThisStep) == 0 {
		return
	}
	for ctrlID, byRecipient := range ds.combatDamageThisStep {
		for recipID, amount := range byRecipient {
			g.FireEvent(GameEvent{
				Type:     EvtCombatDamageDealt,
				PlayerID: ctrlID,
				TargetID: recipID,
				Amount:   amount,
			})
		}
	}
	ds.combatDamageThisStep = make(map[uuid.UUID]map[uuid.UUID]int)
	ds.combatDamageSourcesThisStep = make(map[uuid.UUID]map[uuid.UUID]map[uuid.UUID]int)
}

// DealDamageToPlayer deals damage to a player, running it through the replacement pipeline.
func (ds *DamageSystem) DealDamageToPlayer(g *Game, p Player, amount int, sourceID uuid.UUID) {
	source := g.CaptureDamageSource(sourceID)
	if !source.Valid() {
		source.id = sourceID
	}
	ds.DealDamageToPlayerFromSource(g, p, amount, source)
}

func (ds *DamageSystem) DealDamageToPlayerFromSource(g *Game, p Player, amount int, source DamageSource) {
	ds.dealDamageToPlayerFromSource(g, p, amount, source)
}

func (ds *DamageSystem) dealDamageToPlayerFromSource(g *Game, p Player, amount int, source DamageSource) int {
	return ds.dealDamageToPlayerFromSourceOutcome(g, p, amount, source).amount
}

type damageOutcome struct {
	amount    int
	playerID  uuid.UUID
	permanent capturedPermanent
	source    DamageSource
}

func executeDamageOutcome(g *Game, action Action) damageOutcome {
	result := damageOutcome{}
	if permanentAction, ok := action.(*DamageToCreatureAction); ok {
		if permanent := g.FindPermanent(permanentAction.PermanentID()); permanent != nil && !permanent.PhasedOut {
			result.permanent = capturedPermanent{permanent.ID(), permanent.incarnationID}
		}
	}
	dealt := g.executeDamageAction(action)
	if dealt <= 0 {
		return damageOutcome{}
	}
	result.amount = dealt
	switch a := action.(type) {
	case *DamageToPlayerAction:
		result.source = a.DamageSource()
	case *DamageToCreatureAction:
		result.source = a.DamageSource()
	}
	if player, ok := action.(*DamageToPlayerAction); ok {
		result.playerID = player.PlayerID()
	}
	return result
}

func (ds *DamageSystem) dealDamageToPlayerFromSourceOutcome(g *Game, p Player, amount int, source DamageSource) damageOutcome {
	if amount <= 0 {
		return damageOutcome{}
	}
	action := NewDamageToPlayerAction(source.ID(), p.PlayerID(), amount, ds.resolvingCombatDamage).withDamageSource(source)
	result := g.effects.ApplyReplacements(action, g)
	if result == nil {
		return damageOutcome{}
	}
	return executeDamageOutcome(g, result)
}

func (ds *DamageSystem) prepareDamageToPlayer(g *Game, request damageRequest) Action {
	if request.amount <= 0 || request.playerID == uuid.Nil || g.GetPlayer(request.playerID) == nil {
		return nil
	}
	action := NewDamageToPlayerAction(request.sourceID, request.playerID, request.amount, ds.resolvingCombatDamage).withDamageSource(request.source)
	return g.effects.ApplyReplacements(action, g)
}

// ExecuteDamageToPlayer applies damage to a player after all replacements have been applied.
func (ds *DamageSystem) ExecuteDamageToPlayer(g *Game, a *DamageToPlayerAction) int {
	if a.Amount() <= 0 {
		return 0
	}
	p := g.GetPlayer(a.PlayerID())
	if p == nil {
		return 0
	}
	amount := a.Amount()
	sourceID := a.ActionSource()

	if g.effects.Rules.IsMinimumLifeActive(p.PlayerID()) {
		maxDamage := max(p.Life()-1, 0)
		if amount > maxDamage {
			amount = maxDamage
		}
		if amount <= 0 {
			return 0
		}
	}

	if g.effects.Rules.IsLichActive(g, p.PlayerID()) {
		g.sacrificePermanents(p.PlayerID(), amount)
	} else {
		p.LoseLife(amount)
		g.FireEvent(GameEvent{
			Type:     EvtLifeLost,
			PlayerID: p.PlayerID(),
			Amount:   amount,
		})
	}
	source := a.DamageSource()
	isArtifact := source.HasType(TypeArtifact)
	g.trackers.Turn.RecordDamageTaken(p.PlayerID(), amount, isArtifact)
	if sourceID != uuid.Nil {
		if ds.damageDealtToPlayersByPermanent == nil {
			ds.damageDealtToPlayersByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
		}
		if ds.damageDealtToPlayersByPermanent[sourceID] == nil {
			ds.damageDealtToPlayersByPermanent[sourceID] = make(map[uuid.UUID]bool)
		}
		ds.damageDealtToPlayersByPermanent[sourceID][p.PlayerID()] = true
	}
	if a.IsCombatDamage() {
		if srcPerm := g.FindPermanent(sourceID); srcPerm != nil {
			byCtrl, ok := ds.combatDamageThisStep[srcPerm.ControllerID()]
			if !ok {
				byCtrl = make(map[uuid.UUID]int)
				ds.combatDamageThisStep[srcPerm.ControllerID()] = byCtrl
			}
			byCtrl[p.PlayerID()] += amount
			byCtrlSrcs, ok := ds.combatDamageSourcesThisStep[srcPerm.ControllerID()]
			if !ok {
				byCtrlSrcs = make(map[uuid.UUID]map[uuid.UUID]int)
				ds.combatDamageSourcesThisStep[srcPerm.ControllerID()] = byCtrlSrcs
			}
			bySrc, ok := byCtrlSrcs[p.PlayerID()]
			if !ok {
				bySrc = make(map[uuid.UUID]int)
				byCtrlSrcs[p.PlayerID()] = bySrc
			}
			bySrc[sourceID] += amount
		}
	}
	g.FireEvent(GameEvent{
		Type:               EvtDamageDealt,
		SourceID:           sourceID,
		SourceControllerID: source.Controller(),
		TargetID:           p.PlayerID(),
		Amount:             amount,
		Flag:               a.IsCombatDamage(),
	})
	if ds.onDamageDealt != nil {
		sourceName := "unknown"
		if source.Valid() {
			sourceName = source.Name()
		}
		if g.damageBatch != nil {
			g.damageBatch.notifications = append(g.damageBatch.notifications, damageNotification{sourceName, p.Name(), amount, a.IsCombatDamage()})
		} else {
			ds.onDamageDealt(sourceName, p.Name(), amount, a.IsCombatDamage())
		}
	}
	if source.HasKeyword(Lifelink) {
		controller := source.Controller()
		if controller == uuid.Nil {
			controller = source.Owner()
		}
		srcPlayer := g.GetPlayer(controller)
		if srcPlayer != nil {
			g.gainLifeWithEvent(srcPlayer, amount)
		}
	}
	if src := g.FindPermanent(sourceID); src != nil && src.incarnationID == source.Incarnation() && src.FaceDown {
		g.turnFaceUp(src)
	}
	if reflectEntry, ok := ds.GetDamageReflection(p.PlayerID()); ok {
		if reflectEntry.chosenSource == uuid.Nil || reflectEntry.chosenSource == sourceID {
			ds.ClearDamageReflection(p.PlayerID())
			reflectSourceCard := g.findCardForDamageSource(sourceID)
			if reflectSourceCard != nil {
				sourceOwner := reflectSourceCard.Owner()
				if sourceOwner != uuid.Nil {
					ownerPlayer := g.GetPlayer(sourceOwner)
					if ownerPlayer != nil {
						ds.DealDamageToPlayer(g, ownerPlayer, amount, reflectEntry.eyeSourceID)
					}
				}
			}
		}
	}
	return amount
}

// DealDamageToPermanent deals damage to a permanent, running it through the replacement pipeline.
func (ds *DamageSystem) DealDamageToPermanent(g *Game, perm *Permanent, amount int, sourceID uuid.UUID) {
	source := g.CaptureDamageSource(sourceID)
	if !source.Valid() {
		source.id = sourceID
	}
	ds.DealDamageToPermanentFromSource(g, perm, amount, source)
}

func (ds *DamageSystem) DealDamageToPermanentFromSource(g *Game, perm *Permanent, amount int, source DamageSource) {
	ds.dealDamageToPermanentFromSource(g, perm, amount, source)
}

func (ds *DamageSystem) dealDamageToPermanentFromSource(g *Game, perm *Permanent, amount int, source DamageSource) int {
	return ds.dealDamageToPermanentFromSourceOutcome(g, perm, amount, source).amount
}

func (ds *DamageSystem) dealDamageToPermanentFromSourceOutcome(g *Game, perm *Permanent, amount int, source DamageSource) damageOutcome {
	if action := ds.prepareDamageToPermanent(g, damageRequest{permanent: perm, amount: amount, sourceID: source.ID(), source: source}); action != nil {
		return executeDamageOutcome(g, action)
	}
	return damageOutcome{}
}

func (ds *DamageSystem) prepareDamageToPermanent(g *Game, request damageRequest) Action {
	perm, amount, sourceID := request.permanent, request.amount, request.sourceID
	if amount <= 0 || perm == nil || perm.PhasedOut {
		return nil
	}
	if request.incarnation != uuid.Nil {
		perm = g.FindPermanent(perm.ID())
		if perm == nil || perm.incarnationID != request.incarnation || perm.PhasedOut {
			return nil
		}
	}
	source := request.source
	if !source.Valid() {
		source = g.CaptureDamageSource(sourceID)
	}
	if protectedFromDamageSource(perm, source) {
		return nil
	}
	action := NewDamageToCreatureAction(sourceID, perm.ID(), amount, ds.resolvingCombatDamage).withDamageSource(source)
	return g.effects.ApplyReplacements(action, g)
}

// dealSimultaneousDamageToPermanents applies one group of damage before it sends events.
func (ds *DamageSystem) dealSimultaneousDamageToPermanents(g *Game, requests []damageRequest) {
	ds.dealSimultaneousDamage(g, requests)
}

func (ds *DamageSystem) dealSimultaneousDamage(g *Game, requests []damageRequest) int {
	total := 0
	for _, outcome := range ds.dealSimultaneousDamageOutcomes(g, requests) {
		total += outcome.amount
	}
	return total
}

func (ds *DamageSystem) dealSimultaneousDamageOutcomes(g *Game, requests []damageRequest) []damageOutcome {
	actions := make([]Action, 0, len(requests))
	for _, request := range requests {
		var action Action
		if request.playerID != uuid.Nil {
			action = ds.prepareDamageToPlayer(g, request)
		} else {
			action = ds.prepareDamageToPermanent(g, request)
		}
		if action != nil {
			actions = append(actions, action)
		}
	}
	batch := &damageBatch{}
	g.damageBatch = batch
	outcomes := make([]damageOutcome, 0, len(actions))
	for _, action := range actions {
		if outcome := executeDamageOutcome(g, action); outcome.amount > 0 {
			outcomes = append(outcomes, outcome)
		}
	}
	g.damageBatch = nil
	for _, event := range batch.events {
		g.FireEvent(event)
	}
	for _, notification := range batch.notifications {
		if ds.onDamageDealt != nil {
			ds.onDamageDealt(notification.source, notification.target, notification.amount, notification.combat)
		}
	}
	return outcomes
}

// ExecuteDamageToCreature applies damage to a creature after all replacements have been applied.
func (ds *DamageSystem) ExecuteDamageToCreature(g *Game, a *DamageToCreatureAction) int {
	if a.Amount() <= 0 {
		return 0
	}
	perm := g.MutablePermanent(a.PermanentID())
	if perm == nil {
		return 0
	}
	amount := a.Amount()
	sourceID := a.ActionSource()

	if perm.HasType(TypePlaneswalker) {
		perm.Counters[Loyalty] -= uint8(min(amount, int(perm.Counters[Loyalty])))
	}
	if perm.HasType(TypeCreature) {
		perm.Damage += amount
	}
	if ds.damageDealtBy[perm.ID()] == nil {
		ds.damageDealtBy[perm.ID()] = make(map[uuid.UUID]bool)
	}
	ds.damageDealtBy[perm.ID()][sourceID] = true
	if sourceID != uuid.Nil {
		if ds.damageDealtToPermanentsByPermanent == nil {
			ds.damageDealtToPermanentsByPermanent = make(map[uuid.UUID]map[uuid.UUID]bool)
		}
		if ds.damageDealtToPermanentsByPermanent[sourceID] == nil {
			ds.damageDealtToPermanentsByPermanent[sourceID] = make(map[uuid.UUID]bool)
		}
		ds.damageDealtToPermanentsByPermanent[sourceID][perm.ID()] = true
	}
	g.FireEvent(GameEvent{
		Type:               EvtDamageDealt,
		SourceID:           sourceID,
		SourceControllerID: a.DamageSource().Controller(),
		TargetID:           perm.ID(),
		Amount:             amount,
	})
	if ds.onDamageDealt != nil {
		sourceName := "unknown"
		if source := a.DamageSource(); source.Valid() {
			sourceName = source.Name()
		}
		if g.damageBatch != nil {
			g.damageBatch.notifications = append(g.damageBatch.notifications, damageNotification{sourceName, perm.Name(), amount, ds.resolvingCombatDamage})
		} else {
			ds.onDamageDealt(sourceName, perm.Name(), amount, ds.resolvingCombatDamage)
		}
	}
	source := a.DamageSource()
	if source.HasKeyword(Deathtouch) && amount > 0 && perm.HasType(TypeCreature) {
		perm.Damage = perm.CurrentToughness(g)
	} else if source.HasKeyword(BasiliskTouch) && amount > 0 && perm.HasType(TypeCreature) && !perm.HasSubType("Wall") {
		perm.Damage = perm.CurrentToughness(g)
	}
	if source.HasKeyword(Lifelink) {
		controller := source.Controller()
		if controller == uuid.Nil {
			controller = source.Owner()
		}
		srcPlayer := g.GetPlayer(controller)
		if srcPlayer != nil {
			g.gainLifeWithEvent(srcPlayer, amount)
		}
	}
	if perm.FaceDown {
		g.turnFaceUp(perm)
	}
	if src := g.FindPermanent(sourceID); src != nil && src.incarnationID == source.Incarnation() && src.FaceDown {
		g.turnFaceUp(src)
	}
	return amount
}

// ---------------------------------------------------------------------------
// Legacy API: AddDamagePreventionRule — delegates to the replacement system
// ---------------------------------------------------------------------------

type damagePreventionRule struct {
	from       PermanentFilter
	to         PermanentFilter
	toPlayerID uuid.UUID
	oneShot    bool
	combatOnly bool
	playerOnly bool
}

type damagePreventionRuleOption func(*damagePreventionRule)

// WithFrom sets the "from" filter for a damage prevention rule.
func WithFrom(from PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.from = from
	}
}

// WithTo sets the "to" filter for a damage prevention rule.
func WithTo(to PermanentFilter) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.to = to
	}
}

// WithToPlayer restricts prevention to damage dealt to one player.
func WithToPlayer(playerID uuid.UUID) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.toPlayerID = playerID
	}
}

// WithOneShot marks a rule as consumed after first match.
func WithOneShot(oneshot bool) damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.oneShot = oneshot
	}
}

// WithCombatOnly restricts a damage prevention rule to combat damage only.
func WithCombatOnly() damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.combatOnly = true
	}
}

// WithPlayerOnly restricts a damage prevention rule to damage dealt to players only.
func WithPlayerOnly() damagePreventionRuleOption {
	return func(dpr *damagePreventionRule) {
		dpr.playerOnly = true
	}
}
