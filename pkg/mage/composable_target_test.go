package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetIncarnationAfterReturn(t *testing.T) {
	for _, copySpell := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		c := NewCreature("Target", "", 2, 2)
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
		obj := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, TargetSource: c, Targets: []uuid.UUID{c.ID()}, TargetSpecs: []Target{TargetCreature()}, Effects: []Effect{DestroyTarget()}}
		g.pushStack(obj)
		g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
		g.PutOnBattlefield(c, b.PlayerID())
		if copySpell {
			obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
		}
		g.ResolveStackObject(obj)
		if g.FindPermanent(c.ID()) == nil {
			t.Fatalf("returned object destroyed (copy=%v)", copySpell)
		}
	}
}

func TestTypedTargetGroups(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Target", "", 2, 2)
	c.SetOwner(b.PlayerID())
	g.PutOnBattlefield(c, b.PlayerID())
	creature := DeclarePermanentTarget("creature", IsCreature)
	player := DeclarePlayerTarget("player")
	ability := Spell(Sequence(Destroy(creature), DrawFor(player, Fixed(1))), WithTargets(creature, player))
	card := NewSorcery("Test", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Test", card); err != nil {
		t.Fatal(err)
	}
	drawn := NewCreature("Drawn", "", 1, 1)
	b.AddToLibrary(drawn)
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID(), b.PlayerID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
	g.ResolveStackObject(obj)
	if len(b.Hand()) != 1 {
		t.Fatal("legal player binding lost after creature became illegal")
	}
	if len(a.Hand()) != 0 {
		t.Fatal("player binding fell back to controller")
	}
}

func TestTypedTargetValidation(t *testing.T) {
	for _, kind := range []string{"undeclared", "duplicate name", "duplicate declaration", "nil amount", "valid"} {
		t.Run(kind, func(t *testing.T) {
			target := DeclarePermanentTarget("target", IsCreature)
			var ability *ActionDefinition
			switch kind {
			case "undeclared":
				ability = Spell(Destroy(target))
			case "duplicate name":
				ability = Spell(Destroy(target), WithTargets(target, DeclarePlayerTarget("target")))
			case "duplicate declaration":
				ability = Spell(Destroy(target), WithTargets(target, target))
			case "nil amount":
				ability = Spell(DrawFor(You, nil))
			default:
				ability = Spell(Destroy(target), WithTargets(target))
			}
			err := ValidateCardDefinition("Validation", NewSorcery("Validation", "", WithAbility(ability)))
			if (err == nil) != (kind == "valid") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestTypedTargetCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Destroy(target), WithTargets(target))
	card := NewSorcery("Spell", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	g.PutOnBattlefield(victim, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{victim.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), a.PlayerID(), false)
	clone.ResolveStackObject(copied)
	if clone.FindPermanent(victim.ID()) != nil {
		t.Fatal("copy lost target binding")
	}
	if g.FindPermanent(victim.ID()) == nil {
		t.Fatal("clone changed original")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(victim.ID()) != nil {
		t.Fatal("original lost binding after clone resolution")
	}
}

func TestTypedTargetCannotEscapeDelayedScope(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	ability := Spell(RegisterDelayedTriggerStep(EvtTapped, "", Destroy(target)), WithTargets(target))
	if err := ValidateCardDefinition("Delayed", NewSorcery("Delayed", "", WithAbility(ability))); err == nil {
		t.Fatal("delayed trigger read an outer target binding")
	}
}

func TestComposedRemovalMetadata(t *testing.T) {
	effect := Sequence(Destroy(This), DrawFor(You, Fixed(2)))
	m := InspectDefinition(effect)
	if len(m.Children) != 2 || !m.Children[0].Properties.Destroys || m.Children[1].Properties.DrawCount != 2 || !m.Properties.Destroys {
		t.Fatalf("metadata lost removal: %+v", m)
	}
}

func TestTypedTargetCollectionPartialLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, c := range []Card{first, second} {
		c.SetOwner(b.PlayerID())
		g.PutOnBattlefield(c, b.PlayerID())
	}
	creatures := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	player := DeclarePlayerTarget("player")
	ability := Spell(Sequence(DestroyEach(creatures), DrawFor(player, Fixed(1))), WithTargets(creatures, player))
	card := NewSorcery("Collection", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	b.AddToLibrary(NewLand("Land"))
	targets := []uuid.UUID{first.ID(), second.ID(), b.PlayerID()}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: targets, TargetSpecs: expandTargetSpecs(ability.Targets(), targets, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, b.PlayerID())
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) != nil || len(b.Hand()) != 1 {
		t.Fatal("collection failed to retain its legal members and adjacent player group")
	}
}

func TestCopyCannotChangeTargetCount(t *testing.T) {
	g, a, b := newSpellCopyGame()
	targets := DeclarePermanentTargets("creatures", 0, 2, IsCreature)
	ability := Spell(DestroyEach(targets), WithTargets(targets))
	spell := NewSorcery("Copy test", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	c := NewCreature("Creature", "", 2, 2)
	c.SetOwner(b.PlayerID())
	g.PutOnBattlefield(c, b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{c.ID()}, TargetSpecs: expandTargetSpecs(ability.Targets(), []uuid.UUID{c.ID()}, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{}}
	copied := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if len(copied.Targets) != 1 || copied.Targets[0] != c.ID() {
		t.Fatal("copy changed the number of targets")
	}
}

func TestCopyRetargetsDividedValues(t *testing.T) {
	g, a, b := newSpellCopyGame()
	spell := NewInstant("Divided", "", NewTargetedSpell(TargetPlayer(), DealDamage(Fixed(3))))
	spell.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}, DamageDistribution: map[uuid.UUID]int{b.PlayerID(): 3}, CounterDistribution: map[uuid.UUID]int{b.PlayerID(): 2}}
	g.pushStack(obj)
	a.chooseQueue = [][]uuid.UUID{{a.PlayerID()}}
	cp := g.CopySpellOnStack(spell.ID(), a.PlayerID(), true)
	if cp.DamageDistribution[a.PlayerID()] != 3 || cp.CounterDistribution[a.PlayerID()] != 2 {
		t.Fatal("retargeting lost divided amounts")
	}
	if obj.DamageDistribution[b.PlayerID()] != 3 {
		t.Fatal("copy mutated original distribution")
	}
}

func TestTypedVariableGroupsRequireBoundaries(t *testing.T) {
	first := DeclarePermanentTargets("first", 0, 2, IsCreature)
	second := DeclarePermanentTargets("second", 0, 2, IsCreature)
	card := NewSorcery("Ambiguous", "", WithAbility(Spell(DestroyEach(first), WithTargets(first, second))))
	if err := ValidateCardDefinition("Ambiguous", card); err == nil {
		t.Fatal("accepted ambiguous variable target groups")
	}
}
