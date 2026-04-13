package mage

import (
	"testing"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

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
	for i := 0; i < 5; i++ {
		lc := NewLand("Forest")
		lc.SetOwner(pA.PlayerID())
		pA.AddToLibrary(lc)
	}
	for i := 0; i < 5; i++ {
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
	g.Battlefield = append(g.Battlefield, NewPermanent(land1, pA.PlayerID()))

	creature1Card := NewCreature("Savannah Lions", "W", 2, 1)
	creature1Card.SetOwner(pA.PlayerID())
	creature1 := NewPermanent(creature1Card, pA.PlayerID())
	creature1.Tapped = true
	creature1.Damage = 1
	creature1.Counters[P1P1] = 2
	g.Battlefield = append(g.Battlefield, creature1)

	creature2Card := NewCreature("Serra Angel", "3WW", 4, 4, WithKeyword(Flying))
	creature2Card.SetOwner(pB.PlayerID())
	creature2 := NewPermanent(creature2Card, pB.PlayerID())
	g.Battlefield = append(g.Battlefield, creature2)

	// Set game state.
	g.Turn = 5
	g.Step = PrecombatMain
	g.ActivePlayer = 0
	g.LandsPlayedThisTurn = 1
	g.AttackedThisTurn[creature1.ID()] = true
	g.DamageTakenThisTurn[pB.PlayerID()] = 3

	// Add mana to pools.
	pA.ManaPool().Add(Green, 2)
	pA.ManaPool().Add(White, 1)

	return g
}

func TestCloneBasicFields(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// Scalar fields must match.
	if c.Turn != g.Turn {
		t.Errorf("Turn: got %d, want %d", c.Turn, g.Turn)
	}
	if c.Step != g.Step {
		t.Errorf("Step: got %v, want %v", c.Step, g.Step)
	}
	if c.ActivePlayer != g.ActivePlayer {
		t.Errorf("ActivePlayer: got %d, want %d", c.ActivePlayer, g.ActivePlayer)
	}
	if c.LandsPlayedThisTurn != g.LandsPlayedThisTurn {
		t.Errorf("LandsPlayedThisTurn: got %d, want %d", c.LandsPlayedThisTurn, g.LandsPlayedThisTurn)
	}

	// Players must preserve UUIDs.
	for i, p := range g.Players {
		if c.Players[i].PlayerID() != p.PlayerID() {
			t.Errorf("Player %d UUID mismatch: got %s, want %s", i, c.Players[i].PlayerID(), p.PlayerID())
		}
		if c.Players[i].Life() != p.Life() {
			t.Errorf("Player %d life: got %d, want %d", i, c.Players[i].Life(), p.Life())
		}
	}

	// Players should be SearchPlayers.
	for i, p := range c.Players {
		if _, ok := p.(*SearchPlayer); !ok {
			t.Errorf("Player %d should be *SearchPlayer, got %T", i, p)
		}
	}

	// Battlefield count.
	if len(c.Battlefield) != len(g.Battlefield) {
		t.Fatalf("Battlefield length: got %d, want %d", len(c.Battlefield), len(g.Battlefield))
	}

	// Permanents match.
	for i, p := range g.Battlefield {
		cp := c.Battlefield[i]
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
	origP := g.Battlefield[1]
	cloneP := c.Battlefield[1]
	if cloneP.Counters[P1P1] != origP.Counters[P1P1] {
		t.Errorf("Counters: got %d, want %d", cloneP.Counters[P1P1], origP.Counters[P1P1])
	}

	// Hand cards.
	if len(c.Players[0].Hand()) != len(g.Players[0].Hand()) {
		t.Errorf("Hand length mismatch: got %d, want %d", len(c.Players[0].Hand()), len(g.Players[0].Hand()))
	}

	// Mana pool.
	if c.Players[0].ManaPool().Count(Green) != 2 {
		t.Errorf("Green mana: got %d, want 2", c.Players[0].ManaPool().Count(Green))
	}
	if c.Players[0].ManaPool().Count(White) != 1 {
		t.Errorf("White mana: got %d, want 1", c.Players[0].ManaPool().Count(White))
	}

	// Callbacks should be nil.
	if c.OnPriority != nil {
		t.Error("OnPriority should be nil in clone")
	}
	if c.AfterPriorityAction != nil {
		t.Error("AfterPriorityAction should be nil in clone")
	}
	if c.BeforeStackResolve != nil {
		t.Error("BeforeStackResolve should be nil in clone")
	}

	// UUID-keyed maps.
	pBID := g.Players[1].PlayerID()
	if c.DamageTakenThisTurn[pBID] != 3 {
		t.Errorf("DamageTakenThisTurn: got %d, want 3", c.DamageTakenThisTurn[pBID])
	}
	if !c.AttackedThisTurn[g.Battlefield[1].ID()] {
		t.Error("AttackedThisTurn should be copied")
	}
}

func TestCloneIsolation(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// Mutate clone player life.
	c.Players[0].SetLife(1)
	if g.Players[0].Life() != 15 {
		t.Errorf("Original player life modified: got %d, want 15", g.Players[0].Life())
	}

	// Mutate clone permanent.
	c.Battlefield[1].Tapped = false
	c.Battlefield[1].Damage = 0
	c.Battlefield[1].Counters[P1P1] = 99
	if !g.Battlefield[1].Tapped {
		t.Error("Original perm Tapped should still be true")
	}
	if g.Battlefield[1].Damage != 1 {
		t.Errorf("Original perm Damage should be 1, got %d", g.Battlefield[1].Damage)
	}
	if g.Battlefield[1].Counters[P1P1] != 2 {
		t.Errorf("Original perm counters should be 2, got %d", g.Battlefield[1].Counters[P1P1])
	}

	// Mutate clone hand — add a card.
	extraCard := NewCreature("Goblin", "R", 1, 1)
	c.Players[0].AddToHand(extraCard)
	if len(g.Players[0].Hand()) != 2 {
		t.Errorf("Original hand should still be 2 cards, got %d", len(g.Players[0].Hand()))
	}

	// Mutate clone mana pool.
	c.Players[0].ManaPool().Add(Red, 5)
	if g.Players[0].ManaPool().Count(Red) != 0 {
		t.Errorf("Original mana pool should have 0 Red, got %d", g.Players[0].ManaPool().Count(Red))
	}

	// Mutate clone battlefield slice.
	newCard := NewCreature("Goblin Token", "0", 1, 1)
	newCard.SetOwner(c.Players[0].PlayerID())
	c.Battlefield = append(c.Battlefield, NewPermanent(newCard, c.Players[0].PlayerID()))
	if len(g.Battlefield) != 3 {
		t.Errorf("Original battlefield should still have 3 perms, got %d", len(g.Battlefield))
	}

	// Mutate clone map.
	c.DamageTakenThisTurn[g.Players[0].PlayerID()] = 99
	if g.DamageTakenThisTurn[g.Players[0].PlayerID()] != 0 {
		t.Errorf("Original DamageTakenThisTurn should be 0, got %d", g.DamageTakenThisTurn[g.Players[0].PlayerID()])
	}

	// Mutate clone library.
	c.Players[0].DrawCard()
	if len(g.Players[0].Library()) != 5 {
		t.Errorf("Original library should still have 5 cards, got %d", len(g.Players[0].Library()))
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
	g.Battlefield = append(g.Battlefield, perm)

	// Add a continuous effect that grants +1/+1.
	eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(game *Game, srcID uuid.UUID) error {
		for _, p := range game.Battlefield {
			if p.ID() == srcID {
				p.powerBonus += 1
				p.toughBonus += 1
			}
		}
		return nil
	})
	eff.SetSourceID(perm.ID())
	g.Effects.Add(eff)
	g.Effects.Apply(g)

	// Verify effect applied.
	if perm.CurrentPower(g) != 3 || perm.CurrentToughness(g) != 3 {
		t.Fatalf("Before clone: expected 3/3, got %d/%d", perm.CurrentPower(g), perm.CurrentToughness(g))
	}

	// Clone and verify effects still work.
	c := g.Clone()
	c.Effects.Apply(c)
	clonePerm := c.Battlefield[0]
	if clonePerm.CurrentPower(c) != 3 || clonePerm.CurrentToughness(c) != 3 {
		t.Errorf("After clone: expected 3/3, got %d/%d", clonePerm.CurrentPower(c), clonePerm.CurrentToughness(c))
	}

	// Verify original still works after clone Apply.
	g.Effects.Apply(g)
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
	g.Battlefield = append(g.Battlefield, perm)

	// Add a regeneration shield.
	g.AddRegenerationShield(perm.ID())

	// Add a prevention shield on player B.
	g.AddPreventionShield(pB.PlayerID(), 3)

	// Clone the game.
	c := g.Clone()

	// Verify replacement effects exist in clone.
	if len(c.Effects.replacements) != len(g.Effects.replacements) {
		t.Fatalf("Replacement count: got %d, want %d", len(c.Effects.replacements), len(g.Effects.replacements))
	}

	// Mutate a clone replacement (consume the prevention shield).
	for _, r := range c.Effects.replacements {
		if ps, ok := r.(*preventionShieldReplacement); ok {
			ps.remaining = 0
		}
	}

	// Original should be unchanged.
	for _, r := range g.Effects.replacements {
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
	g.Battlefield = append(g.Battlefield, attacker)

	blockerCard := NewCreature("Hill Giant", "3R", 3, 3)
	blockerCard.SetOwner(pB.PlayerID())
	blocker := NewPermanent(blockerCard, pB.PlayerID())
	g.Battlefield = append(g.Battlefield, blocker)

	// Set up combat.
	g.Combat.AddAttacker(attacker.ID(), pB.PlayerID())
	g.Combat.AddBlocker(blocker.ID(), attacker.ID())

	// Clone.
	c := g.Clone()

	// Verify combat state.
	if !c.Combat.IsAttacking(attacker.ID()) {
		t.Error("Attacker should be attacking in clone")
	}
	if !c.Combat.IsBlocking(blocker.ID()) {
		t.Error("Blocker should be blocking in clone")
	}
	if len(c.Combat.Groups) != 1 {
		t.Fatalf("Combat groups: got %d, want 1", len(c.Combat.Groups))
	}
	if c.Combat.Groups[0].AttackerID != attacker.ID() {
		t.Error("Combat group attacker ID mismatch")
	}
	if len(c.Combat.Groups[0].BlockerIDs) != 1 || c.Combat.Groups[0].BlockerIDs[0] != blocker.ID() {
		t.Error("Combat group blocker IDs mismatch")
	}

	// Mutate clone combat — verify isolation.
	c.Combat.Reset()
	if !g.Combat.IsAttacking(attacker.ID()) {
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
	g.Stack.Push(obj)

	// Clone.
	c := g.Clone()

	// Verify stack.
	if c.Stack.Size() != 1 {
		t.Fatalf("Stack size: got %d, want 1", c.Stack.Size())
	}
	cloneObj := c.Stack.Peek()
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
	c.Stack.Pop()
	if g.Stack.Size() != 1 {
		t.Error("Original stack should still have 1 object")
	}
}

func TestClonePreservesUUIDs(t *testing.T) {
	g := setupTestGame()
	c := g.Clone()

	// All player UUIDs must match.
	for i, p := range g.Players {
		if c.Players[i].PlayerID() != p.PlayerID() {
			t.Errorf("Player %d UUID: got %s, want %s", i, c.Players[i].PlayerID(), p.PlayerID())
		}
	}

	// All permanent UUIDs must match.
	for i, p := range g.Battlefield {
		if c.Battlefield[i].ID() != p.ID() {
			t.Errorf("Permanent %d UUID: got %s, want %s", i, c.Battlefield[i].ID(), p.ID())
		}
	}

	// Hand card UUIDs must match.
	for i, card := range g.Players[0].Hand() {
		if c.Players[0].Hand()[i].ID() != card.ID() {
			t.Errorf("Hand card %d UUID mismatch", i)
		}
	}
}

func TestCloneGameRules(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	g.Effects.Rules.SetChannelActive(pA.PlayerID())
	g.Effects.Rules.SetMinimumLife(pA.PlayerID())
	g.Effects.Rules.LandUntapMax = 2
	g.Effects.Rules.UnlimitedLandPlays = true
	g.Effects.Rules.SpellCostIncreases[Red] = 1

	c := g.Clone()

	// Verify rules copied.
	if !c.Effects.Rules.IsChannelActive(pA.PlayerID()) {
		t.Error("ChannelActive should be copied")
	}
	if !c.Effects.Rules.IsMinimumLifeActive(pA.PlayerID()) {
		t.Error("MinimumLife should be copied")
	}
	if c.Effects.Rules.LandUntapMax != 2 {
		t.Errorf("LandUntapMax: got %d, want 2", c.Effects.Rules.LandUntapMax)
	}
	if !c.Effects.Rules.UnlimitedLandPlays {
		t.Error("UnlimitedLandPlays should be true")
	}
	if c.Effects.Rules.SpellCostIncreases[Red] != 1 {
		t.Errorf("SpellCostIncreases[Red]: got %d, want 1", c.Effects.Rules.SpellCostIncreases[Red])
	}

	// Mutate clone rules.
	c.Effects.Rules.LandUntapMax = 99
	c.Effects.Rules.SpellCostIncreases[Blue] = 5
	if g.Effects.Rules.LandUntapMax != 2 {
		t.Errorf("Original LandUntapMax should still be 2, got %d", g.Effects.Rules.LandUntapMax)
	}
	if g.Effects.Rules.SpellCostIncreases[Blue] != 0 {
		t.Errorf("Original SpellCostIncreases[Blue] should be 0, got %d", g.Effects.Rules.SpellCostIncreases[Blue])
	}
}

func TestCloneDamageSystem(t *testing.T) {
	pA := NewBasePlayer("Alice")
	pB := NewBasePlayer("Bob")
	g := NewGame(pA, pB)

	eyeID := uuid.New()
	sourceID := uuid.New()
	g.Effects.Damage.SetDamageReflection(pA.PlayerID(), eyeID, sourceID)

	c := g.Clone()

	// Verify reflection copied.
	entry, ok := c.Effects.Damage.GetDamageReflection(pA.PlayerID())
	if !ok {
		t.Fatal("Damage reflection should exist in clone")
	}
	if entry.eyeSourceID != eyeID || entry.chosenSource != sourceID {
		t.Error("Damage reflection entry mismatch")
	}

	// Mutate clone.
	c.Effects.Damage.ClearDamageReflection(pA.PlayerID())
	_, ok = g.Effects.Damage.GetDamageReflection(pA.PlayerID())
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
	g.Exile = append(g.Exile, ExiledCard{
		Card:     exiledCard,
		ExiledBy: exiledBy,
	})

	c := g.Clone()
	if len(c.Exile) != 1 {
		t.Fatalf("Exile count: got %d, want 1", len(c.Exile))
	}
	if c.Exile[0].Card.ID() != exiledCard.ID() {
		t.Error("Exile card ID mismatch")
	}
	if c.Exile[0].ExiledBy != exiledBy {
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

	if c2.Players[0].PlayerID() != pA.PlayerID() {
		t.Errorf("Double-clone player UUID: got %s, want %s", c2.Players[0].PlayerID(), pA.PlayerID())
	}
	if _, ok := c2.Players[0].(*SearchPlayer); !ok {
		t.Errorf("Double-clone player should be *SearchPlayer, got %T", c2.Players[0])
	}
}

func TestCloneNestedUUIDMapIsolation(t *testing.T) {
	g := setupTestGame()

	// Set up DamageDealtBy with nested map.
	permID := g.Battlefield[1].ID()
	sourceID := uuid.New()
	g.DamageDealtBy[permID] = map[uuid.UUID]bool{sourceID: true}

	c := g.Clone()

	// Verify nested map copied.
	if !c.DamageDealtBy[permID][sourceID] {
		t.Error("DamageDealtBy should be copied")
	}

	// Mutate clone inner map.
	newSource := uuid.New()
	c.DamageDealtBy[permID][newSource] = true
	if g.DamageDealtBy[permID][newSource] {
		t.Error("Original inner map should not be affected by clone mutation")
	}
}

func BenchmarkClone(b *testing.B) {
	g := setupTestGame()

	// Add more permanents for a realistic mid-game board.
	for i := 0; i < 10; i++ {
		card := NewCreature("Soldier Token", "0", 1, 1)
		card.SetOwner(g.Players[i%2].PlayerID())
		perm := NewPermanent(card, g.Players[i%2].PlayerID())
		if i%3 == 0 {
			perm.Tapped = true
		}
		perm.Counters[P1P1] = uint8(i % 4)
		g.Battlefield = append(g.Battlefield, perm)
	}

	// Add a few replacement effects.
	g.AddPreventionShield(g.Players[0].PlayerID(), 3)
	g.AddRegenerationShield(g.Battlefield[0].ID())

	// Add continuous effects.
	eff := FuncContinuousEffect(LayerPT, WhileOnBattlefield, func(game *Game, srcID uuid.UUID) error {
		return nil
	})
	eff.SetSourceID(g.Battlefield[0].ID())
	g.Effects.Add(eff)

	// Stack object.
	obj := &StackObject{
		ID:         uuid.New(),
		Controller: g.Players[0].PlayerID(),
		SourceID:   uuid.New(),
		Targets:    []uuid.UUID{g.Players[1].PlayerID()},
	}
	g.Stack.Push(obj)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = g.Clone()
	}
}
