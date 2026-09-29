package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// --- from composable_attack_batch.go ---
type controlledAttackBatchPattern struct {
	controlled     bool
	count          *NumberBinding
	captureCount   bool
	duplicateCount bool
}

// OneOrMoreCreaturesYouControlAttack matches one legal declaration group.
func OneOrMoreCreaturesYouControlAttack() controlledAttackBatchPattern {
	return controlledAttackBatchPattern{controlled: true}
}

// OneOrMoreCreaturesAttack matches one attack declaration by either player.
func OneOrMoreCreaturesAttack() controlledAttackBatchPattern {
	return controlledAttackBatchPattern{}
}

// WithCount saves the number of creatures declared in this attack group.
func (p controlledAttackBatchPattern) WithCount(count *NumberBinding) controlledAttackBatchPattern {
	if p.captureCount {
		p.duplicateCount = true
	}
	p.count = count
	p.captureCount = true
	return p
}

func (controlledAttackBatchPattern) eventPattern() {}

func (p controlledAttackBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.controlled {
		return NewTriggered(EvtAttackersDeclared, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
		}})
	}
	return NewTriggered(EvtAttackersDeclared, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
}

func (p controlledAttackBatchPattern) inspectDefinition() definitionNode {
	text := "one or more creatures attack"
	if p.controlled {
		text = "one or more creatures you control attack"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true}
	if p.captureCount {
		if p.count == nil || p.duplicateCount {
			n.problem = "attack group: supply one count binding"
		} else {
			n.bindingWrite = p.count
			n.children = append(n.children, definitionEdge{"count", &eventAmountCapture{binding: p.count}})
		}
	}
	return n
}

func (p controlledAttackBatchPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureCount {
		return effects
	}
	return []Effect{Sequence(&eventAmountCapture{binding: p.count, amount: event.Amount}, Sequence(effects...))}
}

// --- from composable_blocked_attackers_set.go ---
type blockedAttackersOfSet struct{ blocker PermanentRef }

// BlockedAttackersOf selects attackers that the referenced creature is blocking now.
func BlockedAttackersOf(blocker PermanentRef) PermanentSet {
	return blockedAttackersOfSet{blocker: blocker}
}

func (blockedAttackersOfSet) permanentSet() {}

func (s blockedAttackersOfSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "attackers blocked by creature"}
	if nilDefinition(s.blocker) {
		n.problem = "blocked attackers require a blocker"
	} else {
		n.children = []definitionEdge{{"blocker", s.blocker}}
	}
	return n
}

func (s blockedAttackersOfSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.blocker) || ctx == nil || ctx.Game == nil || ctx.Game.combat == nil {
		return nil
	}
	blocker, ok := s.blocker.resolvePermanent(ctx)
	if !ok || !blocker.HasType(TypeCreature) || !ctx.Game.combat.IsBlockingIncarnation(blocker.ID(), blocker.incarnationID) {
		return nil
	}
	var result []*Permanent
	seen := make(map[uuid.UUID]bool)
	for _, group := range ctx.Game.CombatGroups() {
		if seen[group.AttackerID] {
			continue
		}
		linked := false
		for i, id := range group.BlockerIDs {
			if id == blocker.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == blocker.incarnationID {
				linked = true
				break
			}
		}
		if !linked {
			continue
		}
		attacker := ctx.Game.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !attacker.HasType(TypeCreature) || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
			continue
		}
		seen[attacker.ID()] = true
		result = append(result, attacker)
	}
	return result
}

// --- from composable_blocked_by.go ---
type blockedByPattern struct {
	subtype         string
	filterSubtype   bool
	excludeSubtype  bool
	duplicateFilter bool
	blocker         *PermanentBinding
	captureBlocker  bool
}

// BecomesBlockedBy matches each creature that blocks the source.
func BecomesBlockedBy(sourceRef) blockedByPattern { return blockedByPattern{} }

// OfSubtype limits the blocking creature to one creature subtype.
func (p blockedByPattern) OfSubtype(subtype string) blockedByPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes a blocker with this creature subtype at the block event.
func (p blockedByPattern) WithoutSubtype(subtype string) blockedByPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithBlocker saves the blocking creature's battlefield incarnation.
func (p blockedByPattern) WithBlocker(blocker *PermanentBinding) blockedByPattern {
	p.blocker = blocker
	p.captureBlocker = true
	return p
}

func (blockedByPattern) eventPattern() {}

func (p blockedByPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtCreatureBlockedBy, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	} else if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p blockedByPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes blocked by a creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.excludeSubtype {
			n.Text = "source becomes blocked by a creature without " + p.subtype
		} else {
			n.Text = "source becomes blocked by a " + p.subtype
		}
		if p.subtype == "" {
			n.problem = "blocked-by event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocked-by event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocked-by event: supply one subtype filter"
	}
	if p.blocker != nil {
		n.bindingWrite = p.blocker
		n.children = append(n.children, definitionEdge{"blocker", &eventPermanentCapture{binding: p.blocker}})
	} else if p.captureBlocker {
		n.problem = "blocked-by event: supply a blocker binding"
	}
	return n
}

type blockerLacksSubtype struct{ subtype string }

func (c blockerLacksSubtype) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	blocker := reader.FindPermanent(event.TargetID)
	return blocker != nil && !blocker.PhasedOut && blocker.HasType(TypeCreature) && !blocker.HasSubType(c.subtype)
}

func (p blockedByPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureBlocker || p.blocker == nil {
		return nil
	}
	blocker := observer.FindPermanent(event.TargetID)
	if blocker == nil || blocker.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.blocker, value: capturedPermanent{blocker.ID(), blocker.incarnationID}}}
}

// --- from composable_blocker_group.go ---
type blockerGroupPattern struct {
	subtype         string
	filterSubtype   bool
	excludeSubtype  bool
	duplicateFilter bool
}

// BlocksOrBecomesBlockedBy matches one group of new blocker pairs for the source.
func BlocksOrBecomesBlockedBy(sourceRef) blockerGroupPattern { return blockerGroupPattern{} }

// OfSubtype requires at least one other creature with this creature subtype.
func (p blockerGroupPattern) OfSubtype(subtype string) blockerGroupPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes a creature subtype from the new opposing group.
func (p blockerGroupPattern) WithoutSubtype(subtype string) blockerGroupPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

func (blockerGroupPattern) eventPattern() {}

func (p blockerGroupPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtBlockerGroup, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventSourceIsSelf{}, blockerGroupMatches{subtype: p.subtype, filterSubtype: p.filterSubtype, excludeSubtype: p.excludeSubtype},
	}})
}

func (p blockerGroupPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks or becomes blocked by one or more creatures", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.excludeSubtype {
			n.Text = "source blocks or becomes blocked by one or more creatures without " + p.subtype
		} else {
			n.Text = "source blocks or becomes blocked by one or more " + p.subtype + " creatures"
		}
		if p.subtype == "" {
			n.problem = "blocker group event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocker group event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocker group event: supply one subtype filter"
	}
	return n
}

type blockerGroupMatches struct {
	subtype        string
	filterSubtype  bool
	excludeSubtype bool
}

func (c blockerGroupMatches) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	for _, id := range event.RelatedIDs {
		other := reader.FindPermanent(id)
		if other != nil && !other.PhasedOut && other.HasType(TypeCreature) &&
			(!c.filterSubtype || other.HasSubType(c.subtype)) && (!c.excludeSubtype || !other.HasSubType(c.subtype)) {
			return true
		}
	}
	return false
}

// --- from composable_blockers_set.go ---
type blockersOfSet struct{ attacker PermanentRef }

// BlockersOf selects creatures that currently block the chosen attacker.
func BlockersOf(attacker PermanentRef) PermanentSet {
	return blockersOfSet{attacker: attacker}
}

func (blockersOfSet) permanentSet() {}

func (s blockersOfSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "creatures blocking attacker"}
	if nilDefinition(s.attacker) {
		n.problem = "blockers require an attacker"
	} else {
		n.children = []definitionEdge{{"attacker", s.attacker}}
	}
	return n
}

func (s blockersOfSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.attacker) || ctx == nil || ctx.Game == nil || ctx.Game.combat == nil {
		return nil
	}
	attacker, ok := s.attacker.resolvePermanent(ctx)
	if !ok || !attacker.HasType(TypeCreature) || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
		return nil
	}
	var result []*Permanent
	seen := make(map[uuid.UUID]bool)
	for _, group := range ctx.Game.CombatGroups() {
		if group.AttackerID != attacker.ID() {
			continue
		}
		for i, id := range group.BlockerIDs {
			if i >= len(group.BlockerIncarnations) || seen[id] {
				continue
			}
			blocker := ctx.Game.FindPermanent(id)
			if blocker == nil || blocker.PhasedOut || blocker.incarnationID != group.BlockerIncarnations[i] || !blocker.HasType(TypeCreature) {
				continue
			}
			seen[id] = true
			result = append(result, blocker)
		}
	}
	return result
}

// --- from composable_blocks_creature.go ---
type blocksCreaturePattern struct {
	subtype          string
	filterSubtype    bool
	excludeSubtype   bool
	duplicateFilter  bool
	opponent         *PermanentBinding
	captureOpponent  bool
	duplicateCapture bool
}

// BlocksCreature matches each attacking creature that the source blocks.
func BlocksCreature(sourceRef) blocksCreaturePattern { return blocksCreaturePattern{} }

// OfSubtype requires the attacking creature to have this creature subtype.
func (p blocksCreaturePattern) OfSubtype(subtype string) blocksCreaturePattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes attacking creatures with this creature subtype.
func (p blocksCreaturePattern) WithoutSubtype(subtype string) blocksCreaturePattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithOpponent saves the blocked attacker's battlefield incarnation.
func (p blocksCreaturePattern) WithOpponent(binding *PermanentBinding) blocksCreaturePattern {
	if p.captureOpponent {
		p.duplicateCapture = true
	}
	p.opponent = binding
	p.captureOpponent = true
	return p
}

func (blocksCreaturePattern) eventPattern() {}

func (p blocksCreaturePattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtDeclaredBlocker, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	}
	if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p blocksCreaturePattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks an attacking creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.subtype == "" {
			n.problem = "blocks-creature event: supply a creature subtype"
		}
		if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "blocks-creature event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "blocks-creature event: supply one subtype filter"
	}
	if p.opponent != nil {
		n.bindingWrite = p.opponent
		n.children = append(n.children, definitionEdge{"opponent", &eventPermanentCapture{binding: p.opponent}})
	} else if p.captureOpponent {
		n.problem = "blocks-creature event: supply an opponent binding"
	}
	if p.duplicateCapture {
		n.problem = "blocks-creature event: save one opponent"
	}
	return n
}

func (p blocksCreaturePattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureOpponent || p.opponent == nil {
		return nil
	}
	opponent := observer.FindPermanent(event.TargetID)
	if opponent == nil || opponent.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}}}
}

// --- from composable_combat_damage_batch.go ---
type combatDamageBatchPattern struct {
	player           *PlayerBinding
	capturePlayer    bool
	duplicatePlayer  bool
	excludedSubtype  string
	filterSubtype    bool
	duplicateSubtype bool
}

// OneOrMoreCreaturesYouControlDealCombatDamageToPlayer matches one damage step group.
func OneOrMoreCreaturesYouControlDealCombatDamageToPlayer() combatDamageBatchPattern {
	return combatDamageBatchPattern{}
}

// WithPlayer saves the player who received combat damage.
func (p combatDamageBatchPattern) WithPlayer(player *PlayerBinding) combatDamageBatchPattern {
	if p.capturePlayer {
		p.duplicatePlayer = true
	}
	p.player, p.capturePlayer = player, true
	return p
}

// WithoutSubtype checks the damaging creatures when they deal damage.
func (p combatDamageBatchPattern) WithoutSubtype(subtype string) combatDamageBatchPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.excludedSubtype, p.filterSubtype = subtype, true
	return p
}

func (combatDamageBatchPattern) eventPattern() {}

func (p combatDamageBatchPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	conditions := []TriggerConditionData{
		EventPlayerIsController{}, EventAmountGreaterThan{N: 0},
	}
	if p.filterSubtype {
		conditions = append(conditions, combatDamageSourceWithoutSubtype{subtype: p.excludedSubtype})
	}
	return NewTriggered(EvtCombatDamageDealt, false, effect).SetConditionData(AndTriggerCond{Conditions: conditions})
}

type combatDamageSourceWithoutSubtype struct{ subtype string }

func (c combatDamageSourceWithoutSubtype) CheckTriggerCond(event *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
	game, ok := reader.(*Game)
	if !ok {
		return false
	}
	for _, source := range game.damage.combatDamageSourceStatesThisStep[event.PlayerID][event.TargetID] {
		if source.HasType(TypeCreature) && !source.HasSubType(c.subtype) {
			return true
		}
	}
	return false
}

func (p combatDamageBatchPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "one or more creatures you control deal combat damage to a player", battlefieldSource: true}
	if p.filterSubtype {
		if p.excludedSubtype == "" || p.duplicateSubtype {
			n.problem = "combat damage group: supply one creature subtype"
		} else if family, known := knownSubtypeFamily(p.excludedSubtype); known && family != SubtypeCreature {
			n.problem = "combat damage group: subtype is not a creature subtype"
		}
		n.Text = "one or more creatures you control without " + p.excludedSubtype + " deal combat damage to a player"
	}
	if p.capturePlayer {
		if p.player == nil || p.duplicatePlayer {
			n.problem = "combat damage group: supply one player binding"
		} else {
			n.bindingWrite = p.player
			n.children = append(n.children, definitionEdge{"player", &eventPlayerCapture{binding: p.player}})
		}
	}
	return n
}

func (p combatDamageBatchPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.capturePlayer {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.player, player: event.TargetID}, Sequence(effects...))}
}

// --- from composable_combat_opponents.go ---
// BlockingOrBlockedBy selects creatures currently in combat with one permanent.
func BlockingOrBlockedBy(subject PermanentRef) PermanentSet {
	return combatOpponentsSet{subject: subject}
}

type combatOpponentsSet struct{ subject PermanentRef }

func (combatOpponentsSet) permanentSet() {}

func (s combatOpponentsSet) inspectDefinition() definitionNode {
	n := definitionNode{Role: "permanent-set", Text: "creatures blocking or blocked by permanent"}
	if nilDefinition(s.subject) {
		n.problem = "combat opponents require a permanent source"
	} else {
		n.children = []definitionEdge{{"subject", s.subject}}
	}
	return n
}

func (s combatOpponentsSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if nilDefinition(s.subject) {
		return nil
	}
	source, ok := s.subject.resolvePermanent(ctx)
	if !ok || !source.HasType(TypeCreature) {
		return nil
	}
	seen := make(map[*Permanent]bool)
	var result []*Permanent
	add := func(id, incarnation uuid.UUID) {
		p := ctx.Game.FindPermanent(id)
		if p == nil || p.PhasedOut || p.incarnationID != incarnation || !p.HasType(TypeCreature) || seen[p] {
			return
		}
		seen[p] = true
		result = append(result, p)
	}
	for _, group := range ctx.Game.CombatGroups() {
		if group.AttackerID == source.ID() && ctx.Game.combat.IsAttackingIncarnation(source.ID(), source.incarnationID) {
			for i, id := range group.BlockerIDs {
				if i < len(group.BlockerIncarnations) {
					add(id, group.BlockerIncarnations[i])
				}
			}
			continue
		}
		attacker := ctx.Game.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !ctx.Game.combat.IsAttackingIncarnation(attacker.ID(), attacker.incarnationID) {
			continue
		}
		for i, id := range group.BlockerIDs {
			if id == source.ID() && i < len(group.BlockerIncarnations) && group.BlockerIncarnations[i] == source.incarnationID {
				add(attacker.ID(), attacker.incarnationID)
			}
		}
	}
	return result
}

// --- from composable_combat_pair.go ---
type combatPairPattern struct {
	subtype          string
	filterSubtype    bool
	excludeSubtype   bool
	duplicateFilter  bool
	opponent         *PermanentBinding
	captureOpponent  bool
	duplicateCapture bool
}

// BlocksOrBecomesBlockedByCreature matches each new opposing creature in combat.
func BlocksOrBecomesBlockedByCreature(sourceRef) combatPairPattern { return combatPairPattern{} }

// OfSubtype requires the opposing creature to have this creature subtype.
func (p combatPairPattern) OfSubtype(subtype string) combatPairPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithoutSubtype excludes an opposing creature with this creature subtype.
func (p combatPairPattern) WithoutSubtype(subtype string) combatPairPattern {
	if p.filterSubtype || p.excludeSubtype {
		p.duplicateFilter = true
	}
	p.subtype = subtype
	p.excludeSubtype = true
	return p
}

// WithOpponent saves the opposing creature's battlefield incarnation.
func (p combatPairPattern) WithOpponent(binding *PermanentBinding) combatPairPattern {
	if p.captureOpponent {
		p.duplicateCapture = true
	}
	p.opponent = binding
	p.captureOpponent = true
	return p
}

func (combatPairPattern) eventPattern() {}

func (p combatPairPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtCombatPair, false, effect).SetConditionData(EventSourceIsSelf{})
	if p.filterSubtype {
		t.AndConditionData(EventTargetHasSubType{SubType: p.subtype})
	} else if p.excludeSubtype {
		t.AndConditionData(blockerLacksSubtype{subtype: p.subtype})
	}
	return t
}

func (p combatPairPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source blocks or becomes blocked by an opposing creature", battlefieldSource: true}
	if p.filterSubtype || p.excludeSubtype {
		if p.subtype == "" {
			n.problem = "combat-pair event: supply a creature subtype"
		} else if family, known := knownSubtypeFamily(p.subtype); known && family != SubtypeCreature {
			n.problem = "combat-pair event: subtype is not a creature subtype"
		}
	}
	if p.duplicateFilter {
		n.problem = "combat-pair event: supply one subtype filter"
	}
	if p.opponent != nil {
		n.bindingWrite = p.opponent
		n.children = append(n.children, definitionEdge{"opponent", &eventPermanentCapture{binding: p.opponent}})
	} else if p.captureOpponent {
		n.problem = "combat-pair event: supply an opponent binding"
	}
	if p.duplicateCapture {
		n.problem = "combat-pair event: save one opponent"
	}
	return n
}

func (p combatPairPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureOpponent || p.opponent == nil {
		return nil
	}
	opponent := observer.FindPermanent(event.TargetID)
	if opponent == nil || opponent.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}}}
}

// --- from composable_combat_state_branch.go ---
type combatStateBranch struct {
	subject   PermanentRef
	attacking bool
	body      ResolvingEffect
}

// IfPermanentAttacking checks combat membership when the effect resolves.
func IfPermanentAttacking(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &combatStateBranch{subject: subject, attacking: true, body: body}
}

// IfPermanentBlocking checks combat membership when the effect resolves.
func IfPermanentBlocking(subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &combatStateBranch{subject: subject, body: body}
}

func (e *combatStateBranch) Text() string {
	if e.attacking {
		return "if permanent is attacking"
	}
	return "if permanent is blocking"
}

func (e *combatStateBranch) Properties() EffectProperties {
	if e.body == nil {
		return EffectProperties{}
	}
	return e.body.Properties()
}

func (e *combatStateBranch) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok || p.PhasedOut {
		return nil
	}
	match := ctx.Game.IsBlockingIncarnation(p.ID(), p.incarnationID)
	if e.attacking {
		match = ctx.Game.IsAttackingIncarnation(p.ID(), p.incarnationID)
	}
	if match {
		return e.body.Apply(ctx)
	}
	return nil
}

func (e *combatStateBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"then", e.body}}}
}

// --- from composable_defending_control_attack.go ---
type defendingPlayerControlsAttackRule struct {
	filter PermanentFilter
}

// CanAttackOnlyIfDefendingPlayerControls checks the defender at attacker declaration.
func CanAttackOnlyIfDefendingPlayerControls(filter PermanentFilter) StaticRule {
	return defendingPlayerControlsAttackRule{filter: filter}
}

func (defendingPlayerControlsAttackRule) staticRule() {}

func (r defendingPlayerControlsAttackRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{PreventFromAttackingIfDefendingPlayerControls(r.filter)}
}

func (r defendingPlayerControlsAttackRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-attack-restriction", Text: "can attack only if defending player controls matching permanent"}
}

// --- from composable_each_attacker.go ---
type eachControlledAttackerPattern struct {
	attacker          *PermanentBinding
	defender          *PlayerBinding
	captureAttacker   bool
	captureDefender   bool
	duplicateAttacker bool
	duplicateDefender bool
}

// EachCreatureYouControlAttacks matches each declared attacker you control.
func EachCreatureYouControlAttacks() eachControlledAttackerPattern {
	return eachControlledAttackerPattern{}
}

// WithAttacker saves the attacking permanent's battlefield incarnation.
func (p eachControlledAttackerPattern) WithAttacker(binding *PermanentBinding) eachControlledAttackerPattern {
	if p.captureAttacker {
		p.duplicateAttacker = true
	}
	p.attacker, p.captureAttacker = binding, true
	return p
}

// WithDefendingPlayer saves the attacked player in this combat model.
func (p eachControlledAttackerPattern) WithDefendingPlayer(binding *PlayerBinding) eachControlledAttackerPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender, p.captureDefender = binding, true
	return p
}

func (eachControlledAttackerPattern) eventPattern() {}

func (eachControlledAttackerPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtDeclaredAttacker, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventPlayerIsController{}, declaredAttackerPresent{},
	}})
}

type declaredAttackerPresent struct{}

func (declaredAttackerPresent) CheckTriggerCond(event *GameEvent, reader GameReader, _, controller uuid.UUID) bool {
	permanent := reader.FindPermanent(event.SourceID)
	return permanent != nil && !permanent.PhasedOut && permanent.ControllerID() == controller && permanent.HasType(TypeCreature)
}

func (p eachControlledAttackerPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "each creature you control attacks", battlefieldSource: true}
	if p.captureAttacker {
		if p.attacker == nil || p.duplicateAttacker {
			n.problem = "attack event: supply one attacker binding"
		} else {
			n.children = append(n.children, definitionEdge{"attacker", &eventPermanentCapture{binding: p.attacker}})
			n.bindingWrites = append(n.bindingWrites, p.attacker)
		}
	}
	if p.captureDefender {
		if p.defender == nil || p.duplicateDefender {
			n.problem = "attack event: supply one defending player binding"
		} else {
			n.children = append(n.children, definitionEdge{"defender", &eventPlayerCapture{binding: p.defender}})
			n.bindingWrites = append(n.bindingWrites, p.defender)
		}
	}
	return n
}

func (p eachControlledAttackerPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureAttacker {
		if permanent := observer.FindPermanent(event.SourceID); permanent != nil && !permanent.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.attacker, value: capturedPermanent{permanent.ID(), permanent.incarnationID}})
		}
	}
	if p.captureDefender {
		captures = append(captures, &eventPlayerCapture{binding: p.defender, player: event.TargetID})
	}
	return captures
}

// --- from composable_source_combat_event.go ---
type sourceCombatEventPattern struct {
	blocking          bool
	unblocked         bool
	defender          *PlayerBinding
	captureDefender   bool
	duplicateDefender bool
}
type sourceBecomesBlockedPattern struct {
	defender          *PlayerBinding
	captureDefender   bool
	duplicateDefender bool
}

// Attacks matches the source when it is declared as an attacker.
func Attacks(sourceRef) sourceCombatEventPattern { return sourceCombatEventPattern{} }

// AttacksUnblocked matches an attacker after legal blockers are declared.
func AttacksUnblocked(sourceRef) sourceCombatEventPattern {
	return sourceCombatEventPattern{unblocked: true}
}

// WithDefendingPlayer saves the player defending against the source attack.
func (p sourceCombatEventPattern) WithDefendingPlayer(binding *PlayerBinding) sourceCombatEventPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender = binding
	p.captureDefender = true
	return p
}

// Blocks matches the source once when it is declared as a blocker.
func Blocks(sourceRef) EventPattern { return sourceCombatEventPattern{blocking: true} }

// BecomesBlocked matches the source's first transition to blocked in combat.
func BecomesBlocked(sourceRef) sourceBecomesBlockedPattern { return sourceBecomesBlockedPattern{} }

// WithDefendingPlayer saves the defending player when the source becomes blocked.
func (p sourceBecomesBlockedPattern) WithDefendingPlayer(binding *PlayerBinding) sourceBecomesBlockedPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender = binding
	p.captureDefender = true
	return p
}

func (sourceCombatEventPattern) eventPattern()    {}
func (sourceBecomesBlockedPattern) eventPattern() {}

func (sourceBecomesBlockedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtCreatureBecomesBlocked, false, effect).SetConditionData(EventSourceIsSelf{})
}

func (p sourceBecomesBlockedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "source becomes blocked", battlefieldSource: true}
	if p.captureDefender {
		if p.defender == nil || p.duplicateDefender {
			n.problem = "blocked event: supply one defending player binding"
		} else {
			n.bindingWrite = p.defender
			n.children = append(n.children, definitionEdge{"defender", &eventPlayerCapture{binding: p.defender}})
		}
	}
	return n
}

func (p sourceBecomesBlockedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureDefender {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.defender, player: event.TargetID}, Sequence(effects...))}
}

func (p sourceCombatEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.blocking {
		return BlocksTrigger(effect, false)
	}
	if p.unblocked {
		return NewTriggered(EvtAttacksUnblocked, false, effect).SetConditionData(EventSourceIsSelf{})
	}
	return AttacksTrigger(effect, false)
}

func (p sourceCombatEventPattern) inspectDefinition() definitionNode {
	label := "source attacks"
	if p.blocking {
		label = "source blocks"
	} else if p.unblocked {
		label = "source attacks unblocked"
	}
	n := definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
	if p.captureDefender {
		if p.blocking || p.defender == nil || p.duplicateDefender {
			n.problem = "attack event: supply one defending player binding"
		} else {
			n.bindingWrite = p.defender
			n.children = append(n.children, definitionEdge{"defender", &eventPlayerCapture{binding: p.defender}})
		}
	}
	return n
}

func (p sourceCombatEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureDefender {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.defender, player: event.TargetID}, Sequence(effects...))}
}
