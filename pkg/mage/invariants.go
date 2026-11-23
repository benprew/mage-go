package mage

import (
	"fmt"
	"slices"
	"strings"
	"uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// InvariantViolation describes one broken engine or rules invariant.
type InvariantViolation struct {
	Category string
	Message  string
}

func (v InvariantViolation) String() string {
	return fmt.Sprintf("[%s] %s", v.Category, v.Message)
}

// InvariantOptions configures which invariant checks run.
type InvariantOptions struct {
	// PostSBA enables checks that hold only after state-based actions settle,
	// such as when a player receives priority.
	PostSBA bool

	// AtPriority checks that the game can grant priority and enables PostSBA.
	AtPriority bool

	// InitialCards enables card conservation and token-leak checks.
	// Use SnapshotInitialCards right after libraries are populated.
	InitialCards map[uuid.UUID]string

	// CheckContinuousIdempotency checks current characteristics against a
	// recomputation and checks that a second recomputation gives the same result.
	CheckContinuousIdempotency bool
}

// SnapshotInitialCards records all non-token card IDs currently in any zone.
// Pass the map via InvariantOptions.InitialCards to detect lost, duplicated, or leaked cards.
func (g *Game) SnapshotInitialCards() map[uuid.UUID]string {
	cards := make(map[uuid.UUID]string)
	record := func(c *Card) {
		if c != nil && c.ID() != uuid.Nil() {
			cards[c.ID()] = c.Name()
		}
	}
	for _, p := range g.players {
		if p == nil {
			continue
		}
		for _, c := range p.Library() {
			record(c)
		}
		for _, c := range p.Hand() {
			record(c)
		}
		for _, c := range p.Graveyard() {
			record(c)
		}
		for _, c := range p.Ante() {
			record(c)
		}
	}
	for _, perm := range g.zones.battlefield {
		if perm != nil && !perm.IsToken {
			record(perm.Card)
		}
	}
	for _, ec := range g.zones.exile {
		record(ec.Card)
	}
	if g.stack != nil {
		for _, obj := range g.stack.Objects() {
			if obj != nil && !obj.IsAbility && !obj.IsCopy {
				record(obj.Card)
			}
		}
	}
	return cards
}

// CheckInvariants validates structural and rules invariants on g.
// The engine never calls this method automatically, so normal play pays no cost.
func (g *Game) CheckInvariants(opts InvariantOptions) []InvariantViolation {
	var violations []InvariantViolation
	add := func(category, format string, args ...any) {
		violations = append(violations, InvariantViolation{
			Category: category,
			Message:  fmt.Sprintf(format, args...),
		})
	}

	validPlayers := make(map[uuid.UUID]Player, len(g.players))
	for i, p := range g.players {
		if p == nil {
			add("player", "player at index %d is nil", i)
			continue
		}
		if p.PlayerID() == uuid.Nil() {
			add("player", "player %q has nil ID", p.Name())
			continue
		}
		if _, dup := validPlayers[p.PlayerID()]; dup {
			add("player", "duplicate player ID %s", p.PlayerID())
		}
		validPlayers[p.PlayerID()] = p
	}

	g.checkZonesAndConservation(validPlayers, opts.InitialCards, add)
	g.checkStack(validPlayers, add)
	if len(violations) > 0 {
		return violations
	}
	g.checkCombat(validPlayers, add)
	gameOver := g.IsGameOver()
	postSBA := (opts.PostSBA || opts.AtPriority) && !gameOver
	if opts.AtPriority {
		if gameOver {
			add("priority", "player received priority after the game ended")
		}
		if g.turns.Step() == Untap {
			add("priority", "player received priority during the untap step")
		}
		if g.resolution.resolvingSpell != nil || g.damage.resolvingCombatDamage || g.triggers.zoneBatch != nil {
			add("priority", "player received priority during an unresolved action")
		}
	}
	g.checkAttachments(postSBA, add)
	if postSBA {
		g.checkPostSBA(add)
	}
	if opts.CheckContinuousIdempotency && len(violations) == 0 {
		g.checkContinuousIdempotency(add)
	}

	return violations
}

func (g *Game) checkZonesAndConservation(
	validPlayers map[uuid.UUID]Player,
	initialCards map[uuid.UUID]string,
	add func(category, format string, args ...any),
) {
	if g.zones.enteringPermanent != nil {
		add("zone", "transient enteringPermanent is still set")
	}

	seenCards := make(map[uuid.UUID]string)
	recordCard := func(c *Card, zoneDesc string, expectedOwner uuid.UUID, checkInitial bool) {
		if c == nil {
			add("zone", "nil card in %s", zoneDesc)
			return
		}
		if c.ID() == uuid.Nil() {
			add("zone", "card %q in %s has nil ID", c.Name(), zoneDesc)
			return
		}
		owner := c.Owner()
		if _, ok := validPlayers[owner]; !ok {
			add("zone", "card %q (%s) in %s has unknown owner %s", c.Name(), c.ID(), zoneDesc, owner)
		} else if expectedOwner != uuid.Nil() && owner != expectedOwner {
			add("zone", "card %q (%s) in %s belongs to %s, not %s", c.Name(), c.ID(), zoneDesc, owner, expectedOwner)
		}
		if prevZone, dup := seenCards[c.ID()]; dup {
			add("conservation", "card %q (%s) appears in multiple zones: %s and %s", c.Name(), c.ID(), prevZone, zoneDesc)
		} else {
			seenCards[c.ID()] = zoneDesc
		}
		if checkInitial && initialCards != nil {
			if _, ok := initialCards[c.ID()]; !ok {
				add("conservation", "card %q (%s) in %s is not in initial card set", c.Name(), c.ID(), zoneDesc)
			}
		}
	}

	for _, p := range g.players {
		if p == nil {
			continue
		}
		pid := p.PlayerID()
		for _, c := range p.Library() {
			recordCard(c, p.Name()+"'s library", pid, true)
		}
		for _, c := range p.Hand() {
			recordCard(c, p.Name()+"'s hand", pid, true)
		}
		for _, c := range p.Graveyard() {
			recordCard(c, p.Name()+"'s graveyard", pid, true)
		}
		for _, c := range p.Ante() {
			recordCard(c, "ante", uuid.Nil(), true)
		}
	}

	for _, ec := range g.zones.exile {
		recordCard(ec.Card, "exile", uuid.Nil(), true)
	}

	seenPerms := make(map[uuid.UUID]bool, len(g.zones.battlefield))
	seenIncarnations := make(map[uuid.UUID]bool, len(g.zones.battlefield))
	for i, perm := range g.zones.battlefield {
		if perm == nil {
			add("zone", "nil permanent at battlefield index %d", i)
			continue
		}
		if perm.Card == nil {
			add("zone", "permanent at battlefield index %d has nil Card", i)
			continue
		}
		if _, ok := validPlayers[perm.ControllerID()]; !ok {
			add("zone", "permanent %q (%s) has unknown controller %s", perm.Name(), perm.ID(), perm.ControllerID())
		}
		if perm.incarnationID == uuid.Nil() {
			add("zone", "permanent %q (%s) has nil incarnationID", perm.Name(), perm.ID())
		} else if seenIncarnations[perm.incarnationID] {
			add("zone", "duplicate permanent incarnationID %s on %q", perm.incarnationID, perm.Name())
		} else {
			seenIncarnations[perm.incarnationID] = true
		}
		if seenPerms[perm.ID()] {
			add("zone", "duplicate permanent ID %s (%q) on battlefield", perm.ID(), perm.Name())
		}
		seenPerms[perm.ID()] = true
		if perm.Damage < 0 {
			add("zone", "permanent %q (%s) has negative damage %d", perm.Name(), perm.ID(), perm.Damage)
		}
		if !perm.FaceDown && (perm.HasType(TypeInstant) || perm.HasType(TypeSorcery)) {
			add("zone", "permanent %q (%s) has non-permanent card type on battlefield", perm.Name(), perm.ID())
		}
		recordCard(perm.Card, "battlefield", uuid.Nil(), !perm.IsToken)
	}

	if g.stack != nil {
		for _, obj := range g.stack.Objects() {
			if obj != nil && !obj.IsAbility && !obj.IsCopy && obj.Card != nil {
				recordCard(obj.Card, "stack", uuid.Nil(), true)
			}
		}
	}

	for id, name := range initialCards {
		if _, ok := seenCards[id]; !ok {
			add("conservation", "initial card %q (%s) is missing from all zones", name, id)
		}
	}
}

func (g *Game) checkAttachments(postSBA bool, add func(category, format string, args ...any)) {
	byID := make(map[uuid.UUID]*Permanent, len(g.zones.battlefield))
	for _, perm := range g.zones.battlefield {
		if perm != nil && perm.Card != nil {
			byID[perm.ID()] = perm
		}
	}

	for _, perm := range g.zones.battlefield {
		if perm == nil || perm.Card == nil {
			continue
		}
		if perm.AttachedTo != uuid.Nil() {
			if perm.AttachedTo == perm.ID() {
				add("attachment", "permanent %q (%s) is attached to itself", perm.Name(), perm.ID())
			} else {
				host, ok := byID[perm.AttachedTo]
				if !ok && !perm.PhasedOut && (postSBA || !perm.HasSubType("Aura")) {
					add("attachment", "permanent %q (%s) is attached to missing host %s", perm.Name(), perm.ID(), perm.AttachedTo)
				} else if ok && !slices.Contains(host.Attachments, perm.ID()) {
					add("attachment", "permanent %q (%s) has AttachedTo=%q but is not listed in host.Attachments", perm.Name(), perm.ID(), host.Name())
				}
			}
		}

		seenAtt := make(map[uuid.UUID]bool, len(perm.Attachments))
		for _, attID := range perm.Attachments {
			if seenAtt[attID] {
				add("attachment", "permanent %q (%s) lists duplicate attachment %s", perm.Name(), perm.ID(), attID)
				continue
			}
			seenAtt[attID] = true
			att, ok := byID[attID]
			if !ok {
				add("attachment", "permanent %q (%s) lists missing attachment %s", perm.Name(), perm.ID(), attID)
			} else if att.AttachedTo != perm.ID() {
				add("attachment", "permanent %q (%s) lists attachment %q (%s) whose AttachedTo is %s", perm.Name(), perm.ID(), att.Name(), att.ID(), att.AttachedTo)
			}
		}

		if !postSBA || perm.PhasedOut {
			continue
		}
		if !perm.IsAttached() && perm.HasSubType("Aura") {
			add("attachment", "unattached Aura %q (%s) on battlefield after SBA", perm.Name(), perm.ID())
		} else if perm.IsAttached() && !g.attachmentIsLegal(perm, byID[perm.AttachedTo]) {
			add("attachment", "permanent %q (%s) has an illegal attachment to %s after SBA", perm.Name(), perm.ID(), perm.AttachedTo)
		}
	}
}

func (g *Game) checkPostSBA(add func(category, format string, args ...any)) {
	for _, p := range g.players {
		if p == nil || !g.playerIsAlive(p) {
			continue
		}
		if p.PoisonCounters() >= 10 {
			add("sba", "player %q is still alive with %d poison counters after SBA", p.Name(), p.PoisonCounters())
		}
		if p.DrewFromEmpty() {
			add("sba", "player %q is still alive after drawing from empty library", p.Name())
		}
	}

	if len(g.triggers.pending) > 0 {
		add("sba", "%d pending trigger(s) still queued when player received priority", len(g.triggers.pending))
	}

	legendCounts := make(map[uuid.UUID]map[string]int)
	worldCount := 0
	for _, perm := range g.zones.battlefield {
		if perm == nil || perm.Card == nil || perm.PhasedOut {
			continue
		}
		if perm.HasType(TypeCreature) {
			if tough := perm.CurrentToughness(g); tough <= 0 {
				add("sba", "creature %q (%s) has toughness %d after SBA", perm.Name(), perm.ID(), tough)
			} else if perm.LethalDamage(g) && !perm.HasKeyword(Indestructible) {
				add("sba", "creature %q (%s) has lethal damage (%d/%d) without Indestructible after SBA", perm.Name(), perm.ID(), perm.Damage, tough)
			}
		}
		if perm.deathtouchDamage {
			add("sba", "permanent %q (%s) has unprocessed deathtouch damage after SBA", perm.Name(), perm.ID())
		}
		if perm.HasType(TypePlaneswalker) && int(perm.Counters[Loyalty]) <= 0 {
			add("sba", "planeswalker %q (%s) has %d loyalty after SBA", perm.Name(), perm.ID(), perm.Counters[Loyalty])
		}
		if perm.Counters[P1P1] > 0 && perm.Counters[M1M1] > 0 {
			add("sba", "permanent %q (%s) has both +1/+1 (%d) and -1/-1 (%d) counters after SBA", perm.Name(), perm.ID(), perm.Counters[P1P1], perm.Counters[M1M1])
		}
		if perm.HasSuperType(SuperLegendary) {
			ctrl := perm.ControllerID()
			if legendCounts[ctrl] == nil {
				legendCounts[ctrl] = make(map[string]int)
			}
			legendCounts[ctrl][perm.Name()]++
		}
		if perm.HasSuperType(SuperWorld) {
			worldCount++
		}
	}

	for ctrl, byName := range legendCounts {
		for name, count := range byName {
			if count > 1 {
				add("sba", "controller %s controls %d legendary permanents named %q after SBA", ctrl, count, name)
			}
		}
	}
	if worldCount > 1 {
		add("sba", "%d World permanents on battlefield after SBA", worldCount)
	}
}

func isCombatStep(step PhaseStep) bool {
	switch step {
	case BeginCombat, DeclareAttackers, DeclareBlockers, FirstStrikeDamage, CombatDamage, EndCombat:
		return true
	default:
		return false
	}
}

func (g *Game) checkCombat(validPlayers map[uuid.UUID]Player, add func(category, format string, args ...any)) {
	if g.combat == nil {
		add("combat", "combat state is nil")
		return
	}
	if !isCombatStep(g.turns.Step()) {
		if len(g.combat.Groups) > 0 || len(g.combat.Attackers) > 0 {
			add("combat", "combat has %d groups and %d attackers outside combat step (%s)", len(g.combat.Groups), len(g.combat.Attackers), g.turns.Step())
		}
		return
	}

	seenAttackers := make(map[uuid.UUID]bool, len(g.combat.Groups))
	for i, grp := range g.combat.Groups {
		if grp == nil {
			add("combat", "nil combat group at index %d", i)
			continue
		}
		if grp.AttackerID != uuid.Nil() {
			if seenAttackers[grp.AttackerID] {
				add("combat", "duplicate attacker %s in combat groups", grp.AttackerID)
			}
			seenAttackers[grp.AttackerID] = true
			if !g.combat.Attackers[grp.AttackerID] {
				add("combat", "combat group attacker %s is missing from combat.Attackers map", grp.AttackerID)
			}
			if attacker := g.FindPermanent(grp.AttackerID); attacker == nil || !attacker.HasType(TypeCreature) {
				add("combat", "combat group %d attacker %s is not a creature on the battlefield", i, grp.AttackerID)
			}
		}
		if grp.DefenderID != uuid.Nil() && validPlayers[grp.DefenderID] == nil {
			if defender := g.FindPermanent(grp.DefenderID); defender == nil || !defender.HasType(TypePlaneswalker) {
				add("combat", "combat group %d has invalid defender %s", i, grp.DefenderID)
			}
		}
		if len(grp.BlockerIDs) > 0 && !grp.Blocked {
			add("combat", "combat group %d has %d blockers but Blocked=false", i, len(grp.BlockerIDs))
		}
		for j, id := range grp.BlockerIDs {
			if id == uuid.Nil() || slices.Contains(grp.BlockerIDs[:j], id) {
				add("combat", "combat group %d has invalid blocker entry %s", i, id)
			} else if blocker := g.FindPermanent(id); blocker == nil || !blocker.HasType(TypeCreature) {
				add("combat", "combat group %d blocker %s is not a creature on the battlefield", i, id)
			}
		}
	}
	for attackerID, active := range g.combat.Attackers {
		if active && !seenAttackers[attackerID] {
			add("combat", "attacker %s is in combat.Attackers map without a CombatGroup", attackerID)
		}
	}
}

func (g *Game) checkStack(validPlayers map[uuid.UUID]Player, add func(category, format string, args ...any)) {
	if g.stack == nil {
		add("stack", "stack is nil")
		return
	}
	seenIDs := make(map[uuid.UUID]bool, g.stack.Size())
	for i, obj := range g.stack.Objects() {
		if obj == nil {
			add("stack", "nil stack object at index %d", i)
			continue
		}
		if obj.ID == uuid.Nil() {
			add("stack", "stack object at index %d has nil ID", i)
		} else if seenIDs[obj.ID] {
			add("stack", "duplicate stack object ID %s", obj.ID)
		} else {
			seenIDs[obj.ID] = true
		}
		if _, ok := validPlayers[obj.Controller]; !ok {
			add("stack", "stack object %s has unknown controller %s", obj.ID, obj.Controller)
		}
		if !obj.IsAbility && obj.Card == nil {
			add("stack", "non-ability stack object %s has nil Card", obj.ID)
		}
		if obj.XValue < 0 {
			add("stack", "stack object %s has negative XValue %d", obj.ID, obj.XValue)
		}
		if obj.TargetGroups != nil && !slices.Equal(flattenTargetGroups(obj.TargetGroups), obj.Targets) {
			add("stack", "stack object %s has target groups that do not match its target slots", obj.ID)
		}
		if obj.declaredTargets && len(obj.TargetSpecs) != len(obj.Targets) {
			add("stack", "stack object %s has %d declared target slots and %d target specifications", obj.ID, len(obj.Targets), len(obj.TargetSpecs))
		}
		hasDamageDivision := obj.damageTotal != 0 || obj.DamageDistribution != nil
		hasCounterDivision := obj.counterTotal != 0 || obj.CounterDistribution != nil
		if len(obj.Targets) > 0 {
			for _, effect := range obj.Effects {
				switch effect.(type) {
				case *dealDividedDamageEffect:
					hasDamageDivision = true
				case *randomCounterDistributionEffect:
					hasCounterDivision = true
				}
			}
		}
		if hasDamageDivision && !validTargetDivision(obj.DamageDistribution, obj.Targets, obj.damageTotal) {
			add("stack", "stack object %s has a damage distribution that does not assign the announced total %d to its targets", obj.ID, obj.damageTotal)
		}
		if hasCounterDivision && !validTargetDivision(obj.CounterDistribution, obj.Targets, obj.counterTotal) {
			add("stack", "stack object %s has a counter distribution that does not assign the announced total %d to its targets", obj.ID, obj.counterTotal)
		}
	}
}

type permanentCharacteristics struct {
	name       string
	controller uuid.UUID
	power      int
	toughness  int
	types      [TypePlaneswalker + 1]bool
	supertypes [SuperWorld + 1]bool
	subtypes   []string
	colors     []Color
	attrs      [NumAttrs]bool
	counters   [NumCounters]uint8
	abilities  []runtimeAbilitySnapshot
}

type runtimeAbilitySnapshot struct {
	definition uuid.UUID
	grant      uuid.UUID
	text       string
}

func snapshotRuntimeAbility(ability Ability) runtimeAbilitySnapshot {
	inner := UnwrapAbility(ability)
	if inner == nil {
		return runtimeAbilitySnapshot{}
	}
	state := runtimeAbilitySnapshot{definition: inner.AbilityID()}
	if granted, ok := ability.(*grantedByEffect); ok {
		state.grant = granted.grantID
	}
	switch a := inner.(type) {
	case *ProtectionAbility:
		state.definition = uuid.Nil()
		state.text = fmt.Sprintf("protection from %s with exemption %s", a.Filter.Text(), a.ExemptCardID)
	case *ActionDefinition:
		if _, granted := ability.(*grantedByEffect); granted && a.Type() == AbilityMana {
			var text strings.Builder
			for _, cost := range a.costs {
				fmt.Fprintln(&text, cost.Text())
			}
			for _, effect := range a.effects {
				fmt.Fprintln(&text, effect.Text())
			}
			state.definition = uuid.Nil()
			state.text = text.String()
		}
	}
	return state
}

func snapshotCharacteristics(g *Game) map[uuid.UUID]permanentCharacteristics {
	out := make(map[uuid.UUID]permanentCharacteristics, len(g.zones.battlefield))
	for _, perm := range g.zones.battlefield {
		if perm == nil || perm.Card == nil {
			continue
		}
		snap := permanentCharacteristics{
			controller: perm.ControllerID(),
			power:      perm.CurrentPower(g),
			toughness:  perm.CurrentToughness(g),
			subtypes:   append([]string(nil), perm.SubTypes()...),
			colors:     append([]Color(nil), perm.Colors()...),
			counters:   perm.Counters,
		}
		if !perm.FaceDown {
			snap.name = perm.Name()
		}
		for i := range snap.types {
			snap.types[i] = perm.HasType(CardType(i))
		}
		for i := range snap.supertypes {
			snap.supertypes[i] = perm.HasSuperType(SuperType(i))
		}
		slices.Sort(snap.subtypes)
		slices.Sort(snap.colors)
		for a := range int(NumAttrs) {
			snap.attrs[a] = perm.HasAttr(Attr(a))
		}
		for _, ability := range perm.RuntimeAbilities {
			snap.abilities = append(snap.abilities, snapshotRuntimeAbility(ability))
		}
		out[perm.ID()] = snap
	}
	return out
}

func (g *Game) checkContinuousIdempotency(add func(category, format string, args ...any)) {
	clone := g.Clone()
	current := snapshotCharacteristics(g)
	clone.ApplyContinuousEffects()
	first := snapshotCharacteristics(clone)
	compareCharacteristics(current, first, "continuous effects were stale", add)
	clone.ApplyContinuousEffects()
	compareCharacteristics(first, snapshotCharacteristics(clone), "ApplyContinuousEffects is not idempotent", add)
}

func compareCharacteristics(before, after map[uuid.UUID]permanentCharacteristics, message string, add func(category, format string, args ...any)) {
	for id, b := range before {
		a, ok := after[id]
		if !ok {
			add("layers", "permanent %s disappeared during ApplyContinuousEffects", id)
			continue
		}
		if a.name != b.name ||
			a.controller != b.controller ||
			a.power != b.power ||
			a.toughness != b.toughness ||
			a.types != b.types ||
			a.supertypes != b.supertypes ||
			!slices.Equal(a.subtypes, b.subtypes) ||
			!slices.Equal(a.colors, b.colors) ||
			a.attrs != b.attrs ||
			a.counters != b.counters ||
			!slices.Equal(a.abilities, b.abilities) {
			add("layers", "%s for %q (%s), P/T %d/%d to %d/%d and %d to %d runtime abilities",
				message, b.name, id, b.power, b.toughness, a.power, a.toughness, len(b.abilities), len(a.abilities))
		}
	}
	for id := range after {
		if _, ok := before[id]; !ok {
			add("layers", "permanent %s appeared during ApplyContinuousEffects", id)
		}
	}
}
