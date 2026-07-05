package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

var cloneCardSliceSink []Card

// setupTestGame creates a mid-game state with permanents, hand cards, and various state.
func setupTestGame() *Game {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Give players some life variation.
	pA.SetLife(15)
	pB.SetLife(18)

	// Add cards to hands.
	cardA1 := NewCreature("Grizzly Bears", "1G", 2, 2)
	cardA1.SetOwner(pA.PlayerID())
	pA.AddToHand(cardA1)

	cardA2 := NewCreature("Fireball Proxy", "XR", 0, 0) // proxy for test
	cardA2.SetOwner(pA.PlayerID())
	pA.AddToHand(cardA2)

	cardB1 := NewCreature("Hill Giant", "3R", 3, 3)
	cardB1.SetOwner(pB.PlayerID())
	pB.AddToHand(cardB1)

	// Add cards to libraries.
	for range 5 {
		lc := NewLand("Forest")
		lc.SetOwner(pA.PlayerID())
		pA.AddToLibrary(lc)
	}
	for range 5 {
		lc := NewLand("Mountain")
		lc.SetOwner(pB.PlayerID())
		pB.AddToLibrary(lc)
	}

	// Add cards to graveyards.
	gCard := NewCreature("Llanowar Elves", "G", 1, 1)
	gCard.SetOwner(pA.PlayerID())
	pA.AddToGraveyard(gCard)

	// Put permanents on the battlefield.
	land1 := NewLand("Forest")
	land1.SetOwner(pA.PlayerID())
	g.battlefield = append(g.battlefield, NewPermanent(land1, pA.PlayerID()))

	creature1Card := NewCreature("Savannah Lions", "W", 2, 1)
	creature1Card.SetOwner(pA.PlayerID())
	creature1 := NewPermanent(creature1Card, pA.PlayerID())
	creature1.Tapped = true
	creature1.Damage = 1
	creature1.Counters[P1P1] = 2
	g.battlefield = append(g.battlefield, creature1)

	creature2Card := NewCreature("Serra Angel", "3WW", 4, 4, WithKeyword(Flying))
	creature2Card.SetOwner(pB.PlayerID())
	creature2 := NewPermanent(creature2Card, pB.PlayerID())
	g.battlefield = append(g.battlefield, creature2)

	// Set game state.
	g.turn = 5
	g.step = PrecombatMain
	g.activePlayer = 0
	g.landsPlayedThisTurn = 1
	g.attackedThisTurn[creature1.ID()] = true
	g.damageTakenThisTurn[pB.PlayerID()] = 3

	// Add mana to pools.
	pA.ManaPool().Add(Green, 2)
	pA.ManaPool().Add(White, 1)

	return g
}

func TestCloneBasicFields(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// Scalar fields must match.
	if c.turn != g.turn {
		t.Errorf("Turn: got %d, want %d", c.turn, g.turn)
	}
	if c.step != g.step {
		t.Errorf("Step: got %v, want %v", c.step, g.step)
	}
	if c.activePlayer != g.activePlayer {
		t.Errorf("ActivePlayer: got %d, want %d", c.activePlayer, g.activePlayer)
	}
	if c.landsPlayedThisTurn != g.landsPlayedThisTurn {
		t.Errorf("LandsPlayedThisTurn: got %d, want %d", c.landsPlayedThisTurn, g.landsPlayedThisTurn)
	}

	// Players must preserve UUIDs.
	for i, p := range g.players {
		if c.players[i].PlayerID() != p.PlayerID() {
			t.Errorf("Player %d UUID mismatch: got %s, want %s", i, c.players[i].PlayerID(), p.PlayerID())
		}
		if c.players[i].Life() != p.Life() {
			t.Errorf("Player %d life: got %d, want %d", i, c.players[i].Life(), p.Life())
		}
	}

	// Players should be SearchPlayers.
	for i, p := range c.players {
		if _, ok := p.(*SearchPlayer); !ok {
			t.Errorf("Player %d should be *SearchPlayer, got %T", i, p)
		}
	}

	// Battlefield count.
	if len(c.battlefield) != len(g.battlefield) {
		t.Fatalf("Battlefield length: got %d, want %d", len(c.battlefield), len(g.battlefield))
	}

	// Permanents match.
	for i, p := range g.battlefield {
		cp := c.battlefield[i]
		if cp.ID() != p.ID() {
			t.Errorf("Perm %d ID mismatch", i)
		}
		if cp.Tapped != p.Tapped {
			t.Errorf("Perm %d Tapped mismatch", i)
		}
		if cp.Damage != p.Damage {
			t.Errorf("Perm %d Damage: got %d want %d", i, cp.Damage, p.Damage)
		}
		if cp.Controller != p.Controller {
			t.Errorf("Perm %d Controller mismatch", i)
		}
	}

	// Counters on cloned permanent.
	origP := g.battlefield[1]
	cloneP := c.battlefield[1]
	if cloneP.Counters[P1P1] != origP.Counters[P1P1] {
		t.Errorf("Counters: got %d, want %d", cloneP.Counters[P1P1], origP.Counters[P1P1])
	}

	// Hand cards.
	if len(c.players[0].Hand()) != len(g.players[0].Hand()) {
		t.Errorf("Hand length mismatch: got %d, want %d", len(c.players[0].Hand()), len(g.players[0].Hand()))
	}

	// Mana pool.
	if c.players[0].ManaPool().Count(Green) != 2 {
		t.Errorf("Green mana: got %d, want 2", c.players[0].ManaPool().Count(Green))
	}
	if c.players[0].ManaPool().Count(White) != 1 {
		t.Errorf("White mana: got %d, want 1", c.players[0].ManaPool().Count(White))
	}

	// Callbacks should be nil.
	if c.onPriority != nil {
		t.Error("OnPriority should be nil in clone")
	}
	if c.afterPriorityAction != nil {
		t.Error("AfterPriorityAction should be nil in clone")
	}
	if c.beforeStackResolve != nil {
		t.Error("BeforeStackResolve should be nil in clone")
	}

	// UUID-keyed maps.
	pBID := g.players[1].PlayerID()
	if c.damageTakenThisTurn[pBID] != 3 {
		t.Errorf("DamageTakenThisTurn: got %d, want 3", c.damageTakenThisTurn[pBID])
	}
	if !c.attackedThisTurn[g.battlefield[1].ID()] {
		t.Error("AttackedThisTurn should be copied")
	}
}

func TestCloneIsolation(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// Mutate clone player life.
	c.players[0].SetLife(1)
	if g.players[0].Life() != 15 {
		t.Errorf("Original player life modified: got %d, want 15", g.players[0].Life())
	}

	// Mutate clone permanent through the engine write API.
	clonePerm := c.MutablePermanent(c.battlefield[1].ID())
	clonePerm.Tapped = false
	clonePerm.Damage = 0
	clonePerm.Counters[P1P1] = 99
	if !g.battlefield[1].Tapped {
		t.Error("Original perm Tapped should still be true")
	}
	if g.battlefield[1].Damage != 1 {
		t.Errorf("Original perm Damage should be 1, got %d", g.battlefield[1].Damage)
	}
	if g.battlefield[1].Counters[P1P1] != 2 {
		t.Errorf("Original perm counters should be 2, got %d", g.battlefield[1].Counters[P1P1])
	}

	// Mutate clone hand — add a card.
	extraCard := NewCreature("Goblin", "R", 1, 1)
	c.players[0].AddToHand(extraCard)
	if len(g.players[0].Hand()) != 2 {
		t.Errorf("Original hand should still be 2 cards, got %d", len(g.players[0].Hand()))
	}

	// Mutate clone mana pool.
	c.players[0].ManaPool().Add(Red, 5)
	if g.players[0].ManaPool().Count(Red) != 0 {
		t.Errorf("Original mana pool should have 0 Red, got %d", g.players[0].ManaPool().Count(Red))
	}

	// Mutate clone battlefield slice.
	newCard := NewCreature("Goblin Token", "0", 1, 1)
	newCard.SetOwner(c.players[0].PlayerID())
	c.battlefield = append(c.battlefield, NewPermanent(newCard, c.players[0].PlayerID()))
	if len(g.battlefield) != 3 {
		t.Errorf("Original battlefield should still have 3 perms, got %d", len(g.battlefield))
	}

	// Mutate clone map.
	c.damageTakenThisTurn[g.players[0].PlayerID()] = 99
	if g.damageTakenThisTurn[g.players[0].PlayerID()] != 0 {
		t.Errorf("Original DamageTakenThisTurn should be 0, got %d", g.damageTakenThisTurn[g.players[0].PlayerID()])
	}

	// Mutate clone library.
	c.players[0].DrawCard()
	if len(g.players[0].Library()) != 5 {
		t.Errorf("Original library should still have 5 cards, got %d", len(g.players[0].Library()))
	}
}

func TestCloneSharesPermanentsUntilMutation(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	if !g.battlefieldShared || !c.battlefieldShared {
		t.Fatal("clone should mark both branches as sharing battlefield permanents")
	}
	if g.battlefield[1] != c.battlefield[1] {
		t.Fatal("clone should initially share permanent pointers")
	}

	clonePerm := c.MutablePermanent(g.battlefield[1].ID())
	if clonePerm == nil {
		t.Fatal("expected mutable clone permanent")
	}
	if g.battlefield[1] == c.battlefield[1] {
		t.Fatal("first clone mutation should detach that permanent pointer")
	}
	clonePerm.Tapped = false
	clonePerm.Damage = 0
	clonePerm.Counters[P1P1] = 7

	if !g.battlefield[1].Tapped || g.battlefield[1].Damage != 1 || g.battlefield[1].Counters[P1P1] != 2 {
		t.Fatalf("clone mutation leaked to original: tapped=%v damage=%d counters=%d",
			g.battlefield[1].Tapped, g.battlefield[1].Damage, g.battlefield[1].Counters[P1P1])
	}
}

func TestCloneMutablePermanentFieldIsolation(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	hostCard := NewCreature("Clone Host", "{1}{W}", 2, 2)
	hostCard.SetOwner(pA.PlayerID())
	host := NewPermanent(hostCard, pA.PlayerID())

	auraCard := NewAura("Clone Aura", "{W}")
	auraCard.SetOwner(pA.PlayerID())
	aura := NewPermanent(auraCard, pA.PlayerID())

	g.AddToBattlefield(host, aura)
	c := g.Clone()

	cloneHost := c.MutablePermanent(host.ID())
	cloneAura := c.MutablePermanent(aura.ID())
	if cloneHost == nil || cloneAura == nil {
		t.Fatal("expected clone permanents")
	}

	cloneHost.Tapped = true
	cloneHost.Damage = 3
	cloneHost.Counters[P1P1] = 2
	cloneHost.Attachments = append(cloneHost.Attachments, aura.ID())
	cloneHost.Controller = pB.PlayerID()
	cloneHost.RuntimeAbilities = append(cloneHost.RuntimeAbilities, NewManaAbility(Blue))
	cloneHost.SubTypeOverride = []string{"Rogue"}
	cloneHost.SubTypeAdditions = []string{"Wizard"}
	cloneHost.BasePTOverride = &[2]int{5, 6}
	colors := []Color{Black}
	cloneHost.ColorOverride = &colors
	cloneAura.AttachedTo = host.ID()

	if host.Tapped || host.Damage != 0 || host.Counters[P1P1] != 0 {
		t.Fatalf("basic clone field mutation leaked to original: tapped=%v damage=%d counters=%d", host.Tapped, host.Damage, host.Counters[P1P1])
	}
	if len(host.Attachments) != 0 || aura.AttachedTo != uuid.Nil {
		t.Fatalf("attachment mutation leaked to original: host attachments=%d aura attachedTo=%s", len(host.Attachments), aura.AttachedTo)
	}
	if host.Controller != pA.PlayerID() {
		t.Fatalf("controller mutation leaked to original: got %s", host.Controller)
	}
	if len(host.RuntimeAbilities) != len(hostCard.Abilities()) {
		t.Fatalf("runtime ability mutation leaked to original: got %d abilities", len(host.RuntimeAbilities))
	}
	if len(host.SubTypeOverride) != 0 || len(host.SubTypeAdditions) != 0 || host.BasePTOverride != nil || host.ColorOverride != nil {
		t.Fatal("continuous projection field mutation leaked to original")
	}
}

func TestCloneOriginalMutationDoesNotAffectClone(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	origPerm := g.MutablePermanent(g.battlefield[1].ID())
	origPerm.Tapped = false
	origPerm.Damage = 0

	if !c.battlefield[1].Tapped {
		t.Fatal("original mutation leaked tapped state to clone")
	}
	if c.battlefield[1].Damage != 1 {
		t.Fatalf("original mutation leaked damage to clone: got %d", c.battlefield[1].Damage)
	}
}

func TestCloneOfCloneIsolatesBranches(t *testing.T) {
	g := setupTestGame()
	c1 := g.Clone()
	c2 := c1.Clone()

	p1 := c1.MutablePermanent(c1.battlefield[1].ID())
	p2 := c2.MutablePermanent(c2.battlefield[2].ID())
	p1.Damage = 4
	p2.Tapped = true

	if g.battlefield[1].Damage != 1 || c2.battlefield[1].Damage != 1 {
		t.Fatal("c1 mutation leaked to original or sibling clone")
	}
	if g.battlefield[2].Tapped || c1.battlefield[2].Tapped {
		t.Fatal("c2 mutation leaked to original or sibling clone")
	}
}

func TestCloneContinuousEffectsDoNotLeakAcrossBranches(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	sourceCard := NewArtifact("Effect Source", "{2}")
	sourceCard.SetOwner(pA.PlayerID())
	source := NewPermanent(sourceCard, pA.PlayerID())

	targetCard := NewCreature("Effect Target", "{1}{G}", 2, 2)
	targetCard.SetOwner(pA.PlayerID())
	target := NewPermanent(targetCard, pA.PlayerID())

	landCard := NewLand("Forest")
	landCard.SetOwner(pA.PlayerID())
	land := NewPermanent(landCard, pA.PlayerID())

	auraCard := NewAura("Effect Aura", "{W}")
	auraCard.SetOwner(pA.PlayerID())
	aura := NewPermanent(auraCard, pA.PlayerID())
	aura.AttachedTo = target.ID()
	target.Attachments = append(target.Attachments, aura.ID())

	copyTargetCard := NewCreature("Copy Target", "{3}{W}", 4, 4, WithKeyword(Flying))
	copyTargetCard.SetOwner(pA.PlayerID())
	copyTarget := NewPermanent(copyTargetCard, pA.PlayerID())

	doppelCard := NewCreature("Doppelganger", "{3}{U}", 0, 0)
	doppelCard.SetOwner(pA.PlayerID())
	doppel := NewPermanent(doppelCard, pA.PlayerID())

	g.AddToBattlefield(source, target, land, aura, copyTarget, doppel)

	landOnly := NewPermanentFilter("test land", func(p *Permanent, _ *Game) bool {
		return p.ID() == land.ID()
	})
	for _, eff := range []ContinuousEffect{
		TemporaryBoost(target.ID(), 3, 3),
		TargetEffect(LayerControl, Indefinite, target.ID(), func(_ *Game, target *Permanent) error {
			target.Controller = pB.PlayerID()
			return nil
		}),
		GrantSubTypeToTarget(target.ID(), "Rogue", EndOfTurn),
		BecomesColor(target.ID(), Blue, EndOfTurn),
	} {
		g.effects.Add(eff)
	}
	landEffect := AnimateLands(landOnly, 1, 1)
	landEffect.SetSourceID(source.ID())
	g.effects.Add(landEffect)
	for _, eff := range []ContinuousEffect{
		BoostAttached(1, 1, AttachAura),
		GrantProtectionToAttached(Red, AttachAura),
	} {
		eff.SetSourceID(aura.ID())
		g.effects.Add(eff)
	}
	g.effects.AddCopyEffect(doppel.ID(), copyTarget)

	c := g.Clone()
	c.effects.Apply(c)

	cloneTarget := c.FindPermanent(target.ID())
	cloneLand := c.FindPermanent(land.ID())
	cloneDoppel := c.FindPermanent(doppel.ID())
	if cloneTarget == nil || cloneLand == nil || cloneDoppel == nil {
		t.Fatal("expected clone permanents after continuous effect apply")
	}
	if cloneTarget.Controller != pB.PlayerID() {
		t.Fatalf("clone control effect not applied: got controller %s", cloneTarget.Controller)
	}
	if cloneTarget.powerBonus != 4 || cloneTarget.toughBonus != 4 {
		t.Fatalf("clone P/T bonuses not applied: got +%d/+%d", cloneTarget.powerBonus, cloneTarget.toughBonus)
	}
	if !cloneTarget.HasSubType("Rogue") {
		t.Fatal("clone subtype effect not applied")
	}
	if cloneTarget.ColorOverride == nil || len(*cloneTarget.ColorOverride) != 1 || (*cloneTarget.ColorOverride)[0] != Blue {
		t.Fatalf("clone color override not applied: %#v", cloneTarget.ColorOverride)
	}
	if len(cloneTarget.RuntimeAbilities) <= len(target.RuntimeAbilities) {
		t.Fatal("clone granted runtime ability not applied")
	}
	if cloneLand.BasePTOverride == nil || cloneLand.BasePTOverride[0] != 1 || cloneLand.BasePTOverride[1] != 1 || !cloneLand.HasType(TypeCreature) {
		t.Fatal("clone land animation not applied")
	}
	if cloneDoppel.BasePTOverride == nil || cloneDoppel.BasePTOverride[0] != 4 || cloneDoppel.BasePTOverride[1] != 4 {
		t.Fatal("clone copy effect not applied")
	}

	if target.Controller != pA.PlayerID() || target.powerBonus != 0 || target.toughBonus != 0 {
		t.Fatalf("continuous effect leaked to original target: controller=%s +%d/+%d", target.Controller, target.powerBonus, target.toughBonus)
	}
	if len(target.SubTypeOverride) != 0 || len(target.SubTypeAdditions) != 0 || target.ColorOverride != nil {
		t.Fatal("type/color continuous effect leaked to original target")
	}
	if len(target.RuntimeAbilities) != len(targetCard.Abilities()) {
		t.Fatal("granted runtime ability leaked to original target")
	}
	if land.BasePTOverride != nil || land.HasType(TypeCreature) {
		t.Fatal("land animation leaked to original")
	}
	if doppel.BasePTOverride != nil {
		t.Fatal("copy effect leaked to original")
	}
}

// A while-conditioned target effect (e.g. Tawnos's Weaponry: "+1/+1 for as long
// as ~ remains tapped") carries a mutable expired latch. An AI search clone that
// simulates the source untapping must not corrupt the real effect through a
// shared pointer.
func TestCloneWhileConditionedEffectDoesNotLeakExpiry(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	sourceCard := NewArtifact("Weapon", "{2}")
	sourceCard.SetOwner(pA.PlayerID())
	source := NewPermanent(sourceCard, pA.PlayerID())

	targetCard := NewCreature("Shade", "{1}{B}", 2, 2)
	targetCard.SetOwner(pA.PlayerID())
	target := NewPermanent(targetCard, pA.PlayerID())

	g.AddToBattlefield(source, target)
	source.Tapped = true
	targetID := target.ID()

	eff := TargetEffectWhen(LayerPT, WhileOnBattlefield, targetID, func(_ *Game, p *Permanent) error {
		p.powerBonus++
		p.toughBonus++
		return nil
	}, SourceTapped)
	eff.SetSourceID(source.ID())
	g.effects.Add(eff)

	g.effects.Apply(g)
	if p := g.FindPermanent(targetID); p.powerBonus != 1 {
		t.Fatalf("expected boost active on original, got +%d", p.powerBonus)
	}

	// Simulate the source untapping in a throwaway clone; IsActive latches
	// expired on the clone's copy only.
	c := g.Clone()
	c.MutablePermanent(source.ID()).Tapped = false
	c.effects.Apply(c)

	// The real game's source is still tapped; the boost must persist.
	g.effects.Apply(g)
	if p := g.FindPermanent(targetID); p.powerBonus != 1 {
		t.Fatalf("boost leaked expiry from clone: got +%d, want +1", p.powerBonus)
	}
}

func TestCloneWithContinuousEffects(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Put a creature on the battlefield.
	creatureCard := NewCreature("Grizzly Bears", "1G", 2, 2)
	creatureCard.SetOwner(pA.PlayerID())
	perm := NewPermanent(creatureCard, pA.PlayerID())
	g.battlefield = append(g.battlefield, perm)

	// Add a continuous effect that grants +1/+1.
	eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(game *Game, srcID uuid.UUID) error {
		for _, p := range game.battlefield {
			if p.ID() == srcID {
				p = game.MutablePermanent(p.ID())
				if p == nil {
					continue
				}
				p.powerBonus++
				p.toughBonus++
			}
		}
		return nil
	})
	eff.SetSourceID(perm.ID())
	g.effects.Add(eff)
	g.effects.Apply(g)

	// Verify effect applied.
	if perm.CurrentPower(g) != 3 || perm.CurrentToughness(g) != 3 {
		t.Fatalf("Before clone: expected 3/3, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// Clone and verify effects still work.
	c := g.Clone()
	c.effects.Apply(c)
	clonePerm := c.battlefield[0]
	if clonePerm.CurrentPower(c) != 3 || clonePerm.CurrentToughness(c) != 3 {
		t.Errorf("After clone: expected 3/3, got %d/%d", clonePerm.CurrentPower(c), clonePerm.CurrentToughness(c))
	}

	// Verify original still works after clone Apply.
	g.effects.Apply(g)
	if perm.CurrentPower(g) != 3 || perm.CurrentToughness(g) != 3 {
		t.Errorf("Original after clone Apply: expected 3/3, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}
}

func TestCloneWithReplacementEffects(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Put a creature on the battlefield.
	creatureCard := NewCreature("Wall of Stone", "1RR", 0, 8)
	creatureCard.SetOwner(pA.PlayerID())
	perm := NewPermanent(creatureCard, pA.PlayerID())
	g.battlefield = append(g.battlefield, perm)

	// Add a regeneration shield.
	g.AddRegenerationShield(perm.ID())

	// Add a prevention shield on player B.
	g.AddPreventionShield(pB.PlayerID(), 3)

	// Clone the game.
	c := g.Clone()

	// Verify replacement effects exist in clone.
	if len(c.effects.replacements) != len(g.effects.replacements) {
		t.Fatalf("Replacement count: got %d, want %d", len(c.effects.replacements), len(g.effects.replacements))
	}

	// Mutate a clone replacement (consume the prevention shield).
	for _, r := range c.effects.replacements {
		if ps, ok := r.(*preventionShieldReplacement); ok {
			ps.remaining = 0
		}
	}

	// Original should be unchanged.
	for _, r := range g.effects.replacements {
		if ps, ok := r.(*preventionShieldReplacement); ok {
			if ps.remaining != 3 {
				t.Errorf("Original prevention shield remaining: got %d, want 3", ps.remaining)
			}
		}
	}
}

func TestCloneWithCombat(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	attackerCard := NewCreature("Grizzly Bears", "1G", 2, 2)
	attackerCard.SetOwner(pA.PlayerID())
	attacker := NewPermanent(attackerCard, pA.PlayerID())
	g.battlefield = append(g.battlefield, attacker)

	blockerCard := NewCreature("Hill Giant", "3R", 3, 3)
	blockerCard.SetOwner(pB.PlayerID())
	blocker := NewPermanent(blockerCard, pB.PlayerID())
	g.battlefield = append(g.battlefield, blocker)

	// Set up combat.
	g.combat.AddAttacker(attacker.ID(), pB.PlayerID())
	g.combat.AddBlocker(blocker.ID(), attacker.ID())

	// Clone.
	c := g.Clone()

	// Verify combat state.
	if !c.combat.IsAttacking(attacker.ID()) {
		t.Error("Attacker should be attacking in clone")
	}
	if !c.combat.IsBlocking(blocker.ID()) {
		t.Error("Blocker should be blocking in clone")
	}
	if len(c.combat.Groups) != 1 {
		t.Fatalf("Combat groups: got %d, want 1", len(c.combat.Groups))
	}
	if c.combat.Groups[0].AttackerID != attacker.ID() {
		t.Error("Combat group attacker ID mismatch")
	}
	if len(c.combat.Groups[0].BlockerIDs) != 1 || c.combat.Groups[0].BlockerIDs[0] != blocker.ID() {
		t.Error("Combat group blocker IDs mismatch")
	}

	// Mutate clone combat — verify isolation.
	c.combat.Reset()
	if !g.combat.IsAttacking(attacker.ID()) {
		t.Error("Original combat should still have attacker")
	}
}

func TestCloneWithStack(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Push a stack object.
	spellCard := NewCreature("Lightning Bolt Proxy", "R", 0, 0) // proxy for stack test
	spellCard.SetOwner(pA.PlayerID())
	targetID := pB.PlayerID()
	obj := &StackObject{
		ID:         uuid.New(),
		Card:       spellCard,
		Controller: pA.PlayerID(),
		SourceID:   spellCard.ID(),
		Targets:    []uuid.UUID{targetID},
		XValue:     3,
	}
	g.pushStack(obj)

	// Clone.
	c := g.Clone()

	// Verify stack.
	if c.stack.Size() != 1 {
		t.Fatalf("Stack size: got %d, want 1", c.stack.Size())
	}
	cloneObj := c.stack.Peek()
	if cloneObj.ID != obj.ID {
		t.Error("Stack object ID mismatch")
	}
	if cloneObj.XValue != 3 {
		t.Errorf("Stack object XValue: got %d, want 3", cloneObj.XValue)
	}
	if len(cloneObj.Targets) != 1 || cloneObj.Targets[0] != targetID {
		t.Error("Stack object targets mismatch")
	}

	// Mutate clone stack.
	c.stack.Pop()
	if g.stack.Size() != 1 {
		t.Error("Original stack should still have 1 object")
	}
}

func TestClonePreservesUUIDs(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// All player UUIDs must match.
	for i, p := range g.players {
		if c.players[i].PlayerID() != p.PlayerID() {
			t.Errorf("Player %d UUID: got %s, want %s", i, c.players[i].PlayerID(), p.PlayerID())
		}
	}

	// All permanent UUIDs must match.
	for i, p := range g.battlefield {
		if c.battlefield[i].ID() != p.ID() {
			t.Errorf("Permanent %d UUID: got %s, want %s", i, c.battlefield[i].ID(), p.ID())
		}
	}

	// Hand card UUIDs must match.
	for i, card := range g.players[0].Hand() {
		if c.players[0].Hand()[i].ID() != card.ID() {
			t.Errorf("Hand card %d UUID mismatch", i)
		}
	}
}

func TestCloneGameRules(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	g.effects.Rules.SetChannelActive(pA.PlayerID())
	g.effects.Rules.SetMinimumLife(pA.PlayerID())
	g.effects.Rules.LandUntapMax = 2
	g.effects.Rules.UnlimitedLandPlays = true
	g.effects.Rules.SpellCostIncreases[Red] = 1

	c := g.Clone()

	// Verify rules copied.
	if !c.effects.Rules.IsChannelActive(pA.PlayerID()) {
		t.Error("ChannelActive should be copied")
	}
	if !c.effects.Rules.IsMinimumLifeActive(pA.PlayerID()) {
		t.Error("MinimumLife should be copied")
	}
	if c.effects.Rules.LandUntapMax != 2 {
		t.Errorf("LandUntapMax: got %d, want 2", c.effects.Rules.LandUntapMax)
	}
	if !c.effects.Rules.UnlimitedLandPlays {
		t.Error("UnlimitedLandPlays should be true")
	}
	if c.effects.Rules.SpellCostIncreases[Red] != 1 {
		t.Errorf("SpellCostIncreases[Red]: got %d, want 1", c.effects.Rules.SpellCostIncreases[Red])
	}

	// Mutate clone rules.
	c.effects.Rules.LandUntapMax = 99
	c.effects.Rules.SpellCostIncreases[Blue] = 5
	if g.effects.Rules.LandUntapMax != 2 {
		t.Errorf("Original LandUntapMax should still be 2, got %d", g.effects.Rules.LandUntapMax)
	}
	if g.effects.Rules.SpellCostIncreases[Blue] != 0 {
		t.Errorf("Original SpellCostIncreases[Blue] should be 0, got %d", g.effects.Rules.SpellCostIncreases[Blue])
	}
}

func TestCloneDamageSystem(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	eyeID := uuid.New()
	sourceID := uuid.New()
	g.effects.Damage.SetDamageReflection(pA.PlayerID(), eyeID, sourceID)

	c := g.Clone()

	// Verify reflection copied.
	entry, ok := c.effects.Damage.GetDamageReflection(pA.PlayerID())
	if !ok {
		t.Fatal("Damage reflection should exist in clone")
	}
	if entry.eyeSourceID != eyeID || entry.chosenSource != sourceID {
		t.Error("Damage reflection entry mismatch")
	}

	// Mutate clone.
	c.effects.Damage.ClearDamageReflection(pA.PlayerID())
	_, ok = g.effects.Damage.GetDamageReflection(pA.PlayerID())
	if !ok {
		t.Error("Original damage reflection should still exist")
	}
}

func TestCloneExile(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	exiledCard := NewCreature("Goblin", "R", 1, 1)
	exiledCard.SetOwner(pA.PlayerID())
	exiledBy := uuid.New()
	g.exile = append(g.exile, ExiledCard{
		Card:     exiledCard,
		ExiledBy: exiledBy,
	})

	c := g.Clone()
	if len(c.exile) != 1 {
		t.Fatalf("Exile count: got %d, want 1", len(c.exile))
	}
	if c.exile[0].Card.ID() != exiledCard.ID() {
		t.Error("Exile card ID mismatch")
	}
	if c.exile[0].ExiledBy != exiledBy {
		t.Error("Exile ExiledBy mismatch")
	}
}

func TestCloneDelayedTriggers(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	g.delayedTriggers = append(g.delayedTriggers, &DelayedTrigger{
		EventType:  EvtEndStep,
		TargetID:   uuid.New(),
		SourceID:   uuid.New(),
		Controller: pA.PlayerID(),
		Persistent: true,
	})

	c := g.Clone()
	if len(c.delayedTriggers) != 1 {
		t.Fatalf("Delayed triggers count: got %d, want 1", len(c.delayedTriggers))
	}
	if c.delayedTriggers[0].Persistent != true {
		t.Error("Delayed trigger Persistent should be true")
	}

	// Mutate clone.
	c.delayedTriggers[0].Persistent = false
	if !g.delayedTriggers[0].Persistent {
		t.Error("Original delayed trigger should still be persistent")
	}
}

func TestSearchPlayerDefaults(t *testing.T) {
	bp := NewBasePlayer("SearchTest")
	sp := NewSearchPlayer(bp)

	// ChooseTargets returns first min.
	targets := sp.ChooseTargets([]uuid.UUID{uuid.New(), uuid.New()}, 1, 1, nil)
	if len(targets) != 1 {
		t.Errorf("ChooseTargets: got %d targets, want 1", len(targets))
	}

	// DeclareAttackers returns empty.
	attackers := sp.DeclareAttackers(nil)
	if len(attackers) != 0 {
		t.Errorf("DeclareAttackers: got %d, want 0", len(attackers))
	}

	// DeclareBlockers returns empty.
	blockers := sp.DeclareBlockers(nil)
	if len(blockers) != 0 {
		t.Errorf("DeclareBlockers: got %d, want 0", len(blockers))
	}

	// ChooseMayAbility declines.
	if sp.ChooseMayAbility("test") {
		t.Error("ChooseMayAbility should return false")
	}

	// ChooseMode returns 0.
	if sp.ChooseMode([]string{"a", "b"}, "") != 0 {
		t.Error("ChooseMode should return 0")
	}

	// ChooseNumber returns max.
	if sp.ChooseNumber(1, 10, "") != 10 {
		t.Error("ChooseNumber should return max")
	}

	// ChooseCardsFromHand returns cheapest first.
	expensive := NewCreature("Big Guy", "{5}{G}{G}", 7, 7)
	cheap := NewCreature("Small Guy", "{G}", 1, 1)
	sp.AddToHand(expensive)
	sp.AddToHand(cheap)
	discards := sp.ChooseCardsFromHand(1, "discard", nil)
	if len(discards) != 1 {
		t.Fatalf("ChooseCardsFromHand: got %d, want 1", len(discards))
	}
	if discards[0].Name() != "Small Guy" {
		t.Errorf("ChooseCardsFromHand should pick cheapest: got %s, want Small Guy", discards[0].Name())
	}

	// ChooseManaColor returns first available.
	sp.ManaPool().Add(Blue, 1)
	if sp.ChooseManaColor("") != Blue {
		t.Errorf("ChooseManaColor: got %v, want Blue", sp.ChooseManaColor(""))
	}
}

func TestCloneFromSearchPlayer(t *testing.T) {
	// Clone a game that already has SearchPlayers (e.g., double clone).
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	c1 := g.Clone()
	c2 := c1.Clone()

	if c2.players[0].PlayerID() != pA.PlayerID() {
		t.Errorf("Double-clone player UUID: got %s, want %s", c2.players[0].PlayerID(), pA.PlayerID())
	}
	if _, ok := c2.players[0].(*SearchPlayer); !ok {
		t.Errorf("Double-clone player should be *SearchPlayer, got %T", c2.players[0])
	}
}

func TestCloneNestedUUIDMapIsolation(t *testing.T) {
	g := setupTestGame()

	// Set up DamageDealtBy with nested map.
	permID := g.battlefield[1].ID()
	sourceID := uuid.New()
	g.damageDealtBy[permID] = map[uuid.UUID]bool{sourceID: true}

	c := g.Clone()

	// Verify nested map copied.
	if !c.damageDealtBy[permID][sourceID] {
		t.Error("DamageDealtBy should be copied")
	}

	// Mutate clone inner map.
	newSource := uuid.New()
	c.damageDealtBy[permID][newSource] = true
	if g.damageDealtBy[permID][newSource] {
		t.Error("Original inner map should not be affected by clone mutation")
	}
}

func TestCloneCardSliceAllocFree(t *testing.T) {
	card := NewCreature("Alloc Bear", "{1}{G}", 2, 2)
	cards := []Card{card, card, card}
	allocs := testing.AllocsPerRun(1000, func() {
		cloneCardSliceSink = cloneCardSlice(cards)
	})
	if allocs != 0 {
		t.Fatalf("cloneCardSlice allocated: got %.2f allocs/run, want 0", allocs)
	}
}

func BenchmarkClone(b *testing.B) {
	g := setupTestGame()

	// Add more permanents for a realistic mid-game board.
	for i := range 10 {
		card := NewCreature("Soldier Token", "0", 1, 1)
		card.SetOwner(g.players[i%2].PlayerID())
		perm := NewPermanent(card, g.players[i%2].PlayerID())
		if i%3 == 0 {
			perm.Tapped = true
		}
		perm.Counters[P1P1] = uint8(i % 4)
		g.battlefield = append(g.battlefield, perm)
	}

	// Add a few replacement effects.
	g.AddPreventionShield(g.players[0].PlayerID(), 3)
	g.AddRegenerationShield(g.battlefield[0].ID())

	// Add continuous effects.
	eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(game *Game, srcID uuid.UUID) error {
		return nil
	})
	eff.SetSourceID(g.battlefield[0].ID())
	g.effects.Add(eff)

	// Stack object.
	obj := &StackObject{
		ID:         uuid.New(),
		Controller: g.players[0].PlayerID(),
		SourceID:   uuid.New(),
		Targets:    []uuid.UUID{g.players[1].PlayerID()},
	}
	g.pushStack(obj)

	b.ReportAllocs()
	for b.Loop() {
		_ = g.Clone()
	}
}

func setupRogueBoardCloneBenchmarkGame() *Game {
	g := setupTestGame()
	owners := []uuid.UUID{g.players[0].PlayerID(), g.players[1].PlayerID()}

	var lastEquipment *Permanent
	for i := range 96 {
		owner := owners[i%len(owners)]
		var card Card
		switch i % 4 {
		case 0:
			card = NewCreature("Rogue Board Creature", "{2}{B}", 2+i%4, 2+i%5, WithSubTypes("Human", "Rogue"))
		case 1:
			card = NewLand("Swamp")
		case 2:
			card = NewArtifact("Rogue Board Artifact", "{2}")
		default:
			card = NewEquipment("Rogue Board Equipment", "{1}")
		}
		card.SetOwner(owner)
		perm := NewPermanent(card, owner)
		if i%3 == 0 {
			perm.Tapped = true
		}
		if i%5 == 0 {
			perm.Damage = i % 4
		}
		perm.Counters[P1P1] = uint8(i % 3)
		perm.Counters[Charge] = uint8(i % 4)
		if i%7 == 0 {
			perm.RuntimeAbilities = append(perm.RuntimeAbilities, NewManaAbility(Black))
		}
		if i%11 == 0 {
			perm.SubTypeAdditions = []string{"Wizard"}
		}
		if i%13 == 0 {
			perm.BasePTOverride = &[2]int{3, 3}
		}
		if i%17 == 0 {
			colors := []Color{Blue}
			perm.ColorOverride = &colors
		}
		g.battlefield = append(g.battlefield, perm)
		if lastEquipment != nil && perm.HasType(TypeCreature) {
			lastEquipment.AttachedTo = perm.ID()
			perm.Attachments = append(perm.Attachments, lastEquipment.ID())
			lastEquipment = nil
		}
		if perm.HasSubType("Equipment") {
			lastEquipment = perm
		}
	}

	for i := range 12 {
		g.AddPreventionShield(owners[i%2], 1+i%3)
		g.AddRegenerationShield(g.battlefield[i+1].ID())
	}
	for i := range 8 {
		target := g.battlefield[3+i].ID()
		eff := TemporaryBoost(target, i%3, i%2)
		eff.SetSourceID(g.battlefield[0].ID())
		g.effects.Add(eff)
	}
	for i := range 6 {
		obj := &StackObject{
			ID:         uuid.New(),
			Controller: owners[i%2],
			SourceID:   uuid.New(),
			Targets:    []uuid.UUID{owners[(i+1)%2], g.battlefield[i+2].ID()},
			XValue:     i,
		}
		g.pushStack(obj)
	}
	return g
}

func BenchmarkCloneRogueBoard(b *testing.B) {
	g := setupRogueBoardCloneBenchmarkGame()
	b.ReportAllocs()
	for b.Loop() {
		_ = g.Clone()
	}
}
