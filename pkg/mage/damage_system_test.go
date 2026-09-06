package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageSystem_PlayerDamageBasic(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	sourceID := uuid.New()
	g.DealDamageToPlayer(pB, 3, sourceID)

	if pB.Life() != 17 {
		t.Fatalf("expected Bob's life to be 17, got %d", pB.Life())
	}
	if !g.HasDealtDamageToPlayer(sourceID, pB.PlayerID()) {
		t.Fatalf("expected sourceID to be recorded as dealing damage to Bob")
	}
	if g.HasDealtDamageToPlayer(sourceID, pA.PlayerID()) {
		t.Fatalf("sourceID should not have dealt damage to Alice")
	}
	if g.trackers.Turn.DamageTaken(pB.PlayerID()) != 3 {
		t.Fatalf("expected turn damage taken to be 3, got %d", g.trackers.Turn.DamageTaken(pB.PlayerID()))
	}
}

func TestDamageSystem_PlayerDamageWithLifelinkAndFaceDown(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	vampire := NewCreature("Vampire Nighthawk", "{1}{B}{B}", 2, 3, WithKeyword(Lifelink))
	vampPerm := g.PutOnBattlefield(vampire, pA.PlayerID())

	g.DealDamageToPlayer(pB, 2, vampPerm.ID())

	if pB.Life() != 18 {
		t.Fatalf("expected Bob's life to be 18, got %d", pB.Life())
	}
	if pA.Life() != 22 {
		t.Fatalf("expected Alice's life to be 22 (lifelink), got %d", pA.Life())
	}

	facedownCreature := NewCreature("Hidden Assassin", "{1}{B}", 2, 2)
	facedownPerm := g.PutOnBattlefield(facedownCreature, pA.PlayerID())
	facedownPerm.FaceDown = true

	g.DealDamageToPlayer(pB, 2, facedownPerm.ID())

	if facedownPerm.FaceDown {
		t.Fatalf("expected damage dealing permanent to be turned face-up")
	}
}

func TestDamageSystem_CreatureDamageAndKeywords(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	target := NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	targetPerm := g.PutOnBattlefield(target, pB.PlayerID())

	source := NewCreature("Poisonous Snake", "{G}", 1, 1, WithKeyword(Deathtouch))
	sourcePerm := g.PutOnBattlefield(source, pA.PlayerID())

	g.DealDamageToPermanent(targetPerm, 1, sourcePerm.ID())

	if targetPerm.Damage != 2 {
		t.Fatalf("expected deathtouch to mark lethal damage (2), got %d", targetPerm.Damage)
	}
	if !g.HasDealtDamageToPermanent(sourcePerm.ID(), targetPerm.ID()) {
		t.Fatalf("expected duel damage to be tracked for permanent")
	}
	sources := g.GetDamageSources(targetPerm.ID())
	if !sources[sourcePerm.ID()] {
		t.Fatalf("expected turn damage source to include snake")
	}
}

func TestDamageSystem_CombatDamageAggregation(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	c1 := NewCreature("Attacker 1", "{R}", 2, 1)
	perm1 := g.PutOnBattlefield(c1, pA.PlayerID())
	c2 := NewCreature("Attacker 2", "{R}", 3, 1)
	perm2 := g.PutOnBattlefield(c2, pA.PlayerID())

	g.damage.SetResolvingCombatDamage(true)
	g.damage.DealDamageToPlayer(g, pB, 2, perm1.ID())
	g.damage.DealDamageToPlayer(g, pB, 3, perm2.ID())
	g.damage.SetResolvingCombatDamage(false)

	sources := g.CombatDamageSourcesThisStep(pA.PlayerID(), pB.PlayerID())
	if sources == nil {
		t.Fatalf("expected combat damage sources to be populated")
	}
	if sources[perm1.ID()] != 2 {
		t.Errorf("expected perm1 damage to be 2, got %d", sources[perm1.ID()])
	}
	if sources[perm2.ID()] != 3 {
		t.Errorf("expected perm2 damage to be 3, got %d", sources[perm2.ID()])
	}

	triggerCard := NewCreature("Triggerer", "{G}", 1, 1,
		WithAbility(WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger(
			DrawCards(Fixed(1)), false,
		)),
	)
	g.PutOnBattlefield(triggerCard, pA.PlayerID())

	g.damage.FlushCombatDamageAggregator(g)

	if len(g.triggers.Pending()) != 1 {
		t.Fatalf("expected 1 pending trigger from EvtCombatDamageDealt, got %d", len(g.triggers.Pending()))
	}
	if g.triggers.Pending()[0].event.Amount != 5 {
		t.Fatalf("expected aggregated amount 5, got %d", g.triggers.Pending()[0].event.Amount)
	}

	// Step aggregators should now be cleared
	if g.CombatDamageSourcesThisStep(pA.PlayerID(), pB.PlayerID()) != nil {
		t.Fatalf("expected combat damage sources to be cleared after flush")
	}
}

func TestDamageSystem_TurnResetVsDuelPersistence(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	c := NewCreature("The Fallen", "{1}{B}{B}{B}", 2, 3)
	perm := g.PutOnBattlefield(c, pA.PlayerID())

	g.DealDamageToPlayer(pB, 1, perm.ID())
	if !g.HasDealtDamageToPlayer(perm.ID(), pB.PlayerID()) {
		t.Fatalf("expected duel damage to be tracked")
	}

	g.damage.ClearEndOfTurn()

	// Duel tracking persists across end of turn
	if !g.HasDealtDamageToPlayer(perm.ID(), pB.PlayerID()) {
		t.Fatalf("duel damage tracking must persist across ClearEndOfTurn")
	}
	// Turn-scoped damage source tracking is cleared
	if g.damage.DamageDealtBy(perm.ID()) != nil {
		t.Fatalf("turn damage source tracking must be cleared at end of turn")
	}
}

func TestDamageSystem_DamageReflection(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	eyeID := uuid.New()
	attackerID := uuid.New()

	g.SetDamageReflection(pB.PlayerID(), eyeID, attackerID)

	entry, ok := g.damage.GetDamageReflection(pB.PlayerID())
	if !ok || entry.eyeSourceID != eyeID || entry.chosenSource != attackerID {
		t.Fatalf("failed to retrieve reflection entry")
	}

	g.damage.ClearDamageReflection(pB.PlayerID())
	if _, ok := g.damage.GetDamageReflection(pB.PlayerID()); ok {
		t.Fatalf("reflection entry should be cleared")
	}
}

func TestDamageSystem_PreventionShields(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	target := NewCreature("Grizzly Bears", "{1}{G}", 2, 2)
	targetPerm := g.PutOnBattlefield(target, pB.PlayerID())

	// Prevention shield on creature
	g.AddPreventionShield(targetPerm.ID(), 2)
	sourceID := uuid.New()
	g.DealDamageToPermanent(targetPerm, 3, sourceID)

	// 2 prevented, 1 marked
	if targetPerm.Damage != 1 {
		t.Fatalf("expected targetPerm damage to be 1 after 2 prevented, got %d", targetPerm.Damage)
	}

	// Prevention shield on player
	g.AddPreventionShield(pB.PlayerID(), 4)
	g.DealDamageToPlayer(pB, 5, sourceID)

	// Bob took 5 - 4 = 1 damage, life 20 - 1 = 19
	if pB.Life() != 19 {
		t.Fatalf("expected Bob's life to be 19 after 4 prevented, got %d", pB.Life())
	}
}

func TestDamageSystem_MinimumLifeAndLich(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	// Minimum life rule (Ali from Cairo)
	g.SetMinimumLife(pB.PlayerID())
	sourceID := uuid.New()
	g.DealDamageToPlayer(pB, 25, sourceID)

	// Life cannot go below 1
	if pB.Life() != 1 {
		t.Fatalf("expected Bob's life to be 1 with MinimumLife active, got %d", pB.Life())
	}

	// Lich rule: player sacrifices permanents instead of losing life
	pB.SetLife(10)
	g.effects.Rules.ClearMinimumLife()

	lichCard := NewEnchantment("Lich", "{B}{B}{B}{B}")
	lichPerm := g.PutOnBattlefield(lichCard, pB.PlayerID())
	g.effects.Rules.SetLichActive(pB.PlayerID(), lichPerm.ID())

	fodder1 := NewCreature("Fodder 1", "{G}", 1, 1)
	fodder2 := NewCreature("Fodder 2", "{G}", 1, 1)
	g.PutOnBattlefield(fodder1, pB.PlayerID())
	g.PutOnBattlefield(fodder2, pB.PlayerID())

	g.DealDamageToPlayer(pB, 2, sourceID)

	// Life remains 10, but 2 permanents sacrificed
	if pB.Life() != 10 {
		t.Fatalf("expected Bob's life to remain 10 under Lich, got %d", pB.Life())
	}
	if len(pB.Graveyard()) != 2 {
		t.Fatalf("expected 2 sacrificed permanents in graveyard, got %d", len(pB.Graveyard()))
	}
}

func TestDamageSystem_BasiliskTouch(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	target := NewCreature("Colossal Dreadmaw", "{4}{G}{G}", 6, 6)
	targetPerm := g.PutOnBattlefield(target, pB.PlayerID())

	basilisk := NewCreature("Thicket Basilisk", "{3}{G}{G}", 2, 4, WithKeyword(BasiliskTouch))
	basiliskPerm := g.PutOnBattlefield(basilisk, pA.PlayerID())

	g.DealDamageToPermanent(targetPerm, 1, basiliskPerm.ID())

	if targetPerm.Damage != 6 {
		t.Fatalf("expected BasiliskTouch to mark lethal damage (6), got %d", targetPerm.Damage)
	}

	// BasiliskTouch does not destroy walls lethal damage style
	wall := NewCreature("Wall of Wood", "{G}", 0, 3, WithSubTypes("Wall"))
	wallPerm := g.PutOnBattlefield(wall, pB.PlayerID())

	g.DealDamageToPermanent(wallPerm, 1, basiliskPerm.ID())
	if wallPerm.Damage != 1 {
		t.Fatalf("expected Wall of Wood to only take 1 damage from BasiliskTouch, got %d", wallPerm.Damage)
	}
}

func TestDamageSystem_ArtifactDamageTracking(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	rod := NewArtifact("Black Vise", "{1}")
	rodPerm := g.PutOnBattlefield(rod, pA.PlayerID())

	g.DealDamageToPlayer(pB, 3, rodPerm.ID())

	if g.trackers.Turn.ArtifactDamageTaken(pB.PlayerID()) != 3 {
		t.Fatalf("expected artifact damage taken this turn to be 3, got %d", g.trackers.Turn.ArtifactDamageTaken(pB.PlayerID()))
	}
}

func TestDamageSystem_DamageReflectionFiltering(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	attacker1 := NewCreature("Attacker 1", "{R}", 2, 2)
	attacker1.SetOwner(pA.PlayerID())
	attacker2 := NewCreature("Attacker 2", "{R}", 2, 2)
	attacker2.SetOwner(pA.PlayerID())
	perm1 := g.PutOnBattlefield(attacker1, pA.PlayerID())
	perm2 := g.PutOnBattlefield(attacker2, pA.PlayerID())

	eyeSourceID := uuid.New()
	// Reflect only damage from perm1
	g.SetDamageReflection(pB.PlayerID(), eyeSourceID, perm1.ID())

	// Damage from perm2 should NOT trigger reflection
	g.DealDamageToPlayer(pB, 2, perm2.ID())
	if pA.Life() != 20 {
		t.Fatalf("Alice should not have taken reflected damage from perm2, got %d", pA.Life())
	}
	if _, ok := g.damage.GetDamageReflection(pB.PlayerID()); !ok {
		t.Fatalf("reflection should still be active for perm1")
	}

	// Damage from perm1 DOES trigger reflection
	g.DealDamageToPlayer(pB, 2, perm1.ID())
	if pA.Life() != 18 {
		t.Fatalf("Alice should have taken 2 reflected damage, got %d", pA.Life())
	}
	if _, ok := g.damage.GetDamageReflection(pB.PlayerID()); ok {
		t.Fatalf("reflection should be consumed")
	}
}

func TestDamageSystem_CloneIsolation(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	sourceID := uuid.New()
	targetID := uuid.New()

	g.damage.damageDealtToPlayersByPermanent[sourceID] = map[uuid.UUID]bool{pB.PlayerID(): true}
	g.damage.damageDealtToPermanentsByPermanent[sourceID] = map[uuid.UUID]bool{targetID: true}
	g.damage.damageDealtBy[targetID] = map[uuid.UUID]bool{sourceID: true}
	g.damage.combatDamageThisStep[pA.PlayerID()] = map[uuid.UUID]int{pB.PlayerID(): 5}
	g.damage.combatDamageSourcesThisStep[pA.PlayerID()] = map[uuid.UUID]map[uuid.UUID]int{
		pB.PlayerID(): {sourceID: 5},
	}
	g.damage.SetResolvingCombatDamage(true)

	clone := g.damage.Clone()

	if !clone.HasDealtDamageToPlayer(sourceID, pB.PlayerID()) {
		t.Errorf("clone should have duel damage to player")
	}
	if !clone.HasDealtDamageToPermanent(sourceID, targetID) {
		t.Errorf("clone should have duel damage to permanent")
	}
	if !clone.IsResolvingCombatDamage() {
		t.Errorf("clone should have resolvingCombatDamage true")
	}
	if clone.CombatDamageSourcesThisStep(pA.PlayerID(), pB.PlayerID())[sourceID] != 5 {
		t.Errorf("clone should have combat damage sources")
	}

	// Mutate clone
	newPermID := uuid.New()
	clone.damageDealtBy[targetID][newPermID] = true
	clone.combatDamageThisStep[pA.PlayerID()][pB.PlayerID()] = 10
	clone.combatDamageSourcesThisStep[pA.PlayerID()][pB.PlayerID()][sourceID] = 10
	clone.SetResolvingCombatDamage(false)

	if g.damage.damageDealtBy[targetID][newPermID] {
		t.Errorf("original damageDealtBy should not be affected by clone mutation")
	}
	if g.damage.combatDamageThisStep[pA.PlayerID()][pB.PlayerID()] != 5 {
		t.Errorf("original combatDamageThisStep should not be affected by clone mutation")
	}
	if g.damage.combatDamageSourcesThisStep[pA.PlayerID()][pB.PlayerID()][sourceID] != 5 {
		t.Errorf("original combatDamageSourcesThisStep should not be affected by clone mutation")
	}
	if !g.damage.IsResolvingCombatDamage() {
		t.Errorf("original resolvingCombatDamage should still be true")
	}
}
