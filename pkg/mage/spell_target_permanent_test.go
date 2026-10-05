package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellTargetTargetingControlledPermanent(t *testing.T) {
	for _, change := range []string{"none", "shroud", "control", "departed", "returned", "phased", "phase-cycle", "retargeted", "other-target-remains"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := NewCreature("Original", "", 2, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewArtifact("Other", ""), a.PlayerID())
			victim := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: NewInstant("Victim", ""), Controller: b.PlayerID(), Targets: []uuid.UUID{p.ID()}}
			if change == "other-target-remains" {
				victim.Targets = append(victim.Targets, other.ID())
			}
			g.pushStack(victim)
			target := DeclareSpellTargetTargetingControlledPermanent("spell")
			counter := NewInstant("Counter", "", Spell(Counter(target), WithTargets(target)))
			if err := ValidateCardDefinition(counter.Name(), counter); err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(target.Possible(a.PlayerID(), counter, g), victim.ID) {
				t.Fatal("matching spell absent")
			}
			if slices.Contains(target.Possible(b.PlayerID(), counter, g), victim.ID) {
				t.Fatal("used victim controller")
			}
			counter.SetOwner(a.PlayerID())
			a.AddToHand(counter)
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "shroud":
				p.GrantBaseAttr(Shroud)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "departed", "other-target-remains":
				g.BouncePermanentToHand(p)
			case "returned":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "retargeted":
				victim.Targets = []uuid.UUID{b.PlayerID()}
			}
			want := change == "none" || change == "shroud" || change == "phase-cycle" || change == "other-target-remains"
			if got := slices.Contains(target.Possible(a.PlayerID(), counter, g), victim.ID); got != want {
				t.Fatalf("current target match = %v, want %v", got, want)
			}
			g.ResolveTopOfStack()
			if got := g.Stack().FindByID(victim.ID) == nil; got != want {
				t.Fatalf("countered = %v, want %v", got, want)
			}
			if clone.Stack().FindByID(victim.ID) == nil {
				t.Fatal("live resolution changed clone")
			}
			clone.ResolveTopOfStack()
			if clone.Stack().FindByID(victim.ID) != nil {
				t.Fatal("clone lost saved target")
			}
		})
	}
}

func TestSpellTargetTargetingControlledPermanentIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	p := g.PutOnBattlefield(NewArtifact("Mine", ""), a.PlayerID())
	card := NewInstant("Same card", "")
	good := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: b.PlayerID(), Targets: []uuid.UUID{p.ID()}}
	bad := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: b.PlayerID(), Targets: []uuid.UUID{b.PlayerID()}}
	ability := &StackObject{ID: uuid.New(), SourceID: p.ID(), Card: card, IsAbility: true, Targets: []uuid.UUID{p.ID()}}
	g.pushStack(good)
	g.pushStack(bad)
	g.pushStack(ability)
	copied := g.CopyStackObjectDirect(good, b.PlayerID(), false)
	target := DeclareSpellTargetTargetingControlledPermanent("spell", NewCardFilter("instant", func(c Card) bool { return c.HasType(TypeInstant) }))
	possible := target.Possible(a.PlayerID(), card, g)
	if len(possible) != 2 || !slices.Contains(possible, good.ID) || !slices.Contains(possible, copied.ID) {
		t.Fatal("stack identity or ability exclusion")
	}
	g.RemoveFromBattlefield(p)
	if len(target.Possible(a.PlayerID(), card, g)) != 0 {
		t.Fatal("used departure information")
	}
}

func TestSpellTargetTargetingControlledPermanentDeclarations(t *testing.T) {
	target := DeclareSpellTargetTargetingControlledPermanent("spell")
	for _, tc := range []struct {
		name    string
		card    Card
		invalid bool
	}{
		{"declared", NewInstant("Declared", "", Spell(Counter(target), WithTargets(target))), false},
		{"undeclared", NewInstant("Undeclared", "", Spell(Counter(target))), true},
		{"unnamed", NewInstant("Unnamed", "", Spell(GainLifeFor(You, Fixed(1)), WithTargets(DeclareSpellTargetTargetingControlledPermanent("")))), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateCardDefinition(tc.card.Name(), tc.card) != nil; got != tc.invalid {
				t.Fatalf("invalid = %v", got)
			}
		})
	}
}
