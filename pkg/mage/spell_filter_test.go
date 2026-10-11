package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestContextualSpellFilterTargets(t *testing.T) {
	for _, change := range []string{"none", "control", "color", "phased", "phase-cycle", "departed", "returned"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			donorCard := NewCreature("Donor", "{G}{U}", 2, 2)
			donorCard.SetOwner(a.PlayerID())
			donor := g.PutOnBattlefield(donorCard, a.PlayerID())
			victim := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Card: NewInstant("Victim", "{G}"), Controller: a.PlayerID()}
			g.pushStack(victim)
			ability := &StackObject{ID: uuid.New(), Card: victim.Card, Controller: a.PlayerID(), IsAbility: true}
			g.pushStack(ability)
			filter := AllSpellFilters(SpellControlledByYou(), SpellSharesColorWithPermanents(IsCreature, true))
			target := DeclareSpellTarget("spell").WithFilter(filter)
			counter := NewInstant("Counter", "", Spell(Counter(target), WithTargets(target)))
			counter.SetOwner(a.PlayerID())
			a.AddToHand(counter)
			if got := target.Possible(a.PlayerID(), counter, g); !slices.Equal(got, []uuid.UUID{victim.ID}) {
				t.Fatalf("possible = %v", got)
			}
			if len(target.Possible(b.PlayerID(), counter, g)) != 0 {
				t.Fatal("used card owner or spell controller as action controller")
			}
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{victim.ID}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "control":
				victim.Controller = b.PlayerID()
			case "color":
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Red)), donor.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phased":
				g.PhaseOut(donor.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(donor.ID()))
			case "departed":
				g.BouncePermanentToHand(donor)
			case "returned":
				g.BouncePermanentToHand(donor)
				a.RemoveFromHand(donorCard.ID())
				g.PutOnBattlefield(donorCard, a.PlayerID())
			}
			want := change == "none" || change == "phase-cycle" || change == "returned"
			g.ResolveTopOfStack()
			if got := g.Stack().FindByID(victim.ID) == nil; got != want {
				t.Fatalf("countered = %v, want %v", got, want)
			}
			clone.ResolveTopOfStack()
			if clone.Stack().FindByID(victim.ID) != nil {
				t.Fatal("clone lost independent battlefield or target")
			}
		})
	}
}

func TestContextualSpellFilterDeclarations(t *testing.T) {
	valid := SpellCardMatches(NewCardFilter("any", func(Card) bool { return true }))
	for _, filter := range []SpellFilter{{}, AllSpellFilters(), AnySpellFilter(), NotSpellFilter(SpellFilter{}), AllSpellFilters(valid, SpellFilter{})} {
		target := DeclareSpellTarget("spell").WithFilter(filter)
		card := NewInstant("Invalid", "", Spell(Counter(target), WithTargets(target)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid target filter accepted")
		}
		observer := NewArtifact("Invalid", "", WithAbility(WhenEvent(SpellCast().Matching(filter), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(observer.Name(), observer) == nil {
			t.Fatal("invalid event filter accepted")
		}
	}
	target := DeclareSpellTarget("spell").WithFilter(valid).WithFilter(valid)
	card := NewInstant("Duplicate", "", Spell(Counter(target), WithTargets(target)))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("duplicate filter accepted")
	}
	observer := NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().Matching(valid).Matching(valid), GainLifeFor(You, Fixed(1)))))
	if ValidateCardDefinition(observer.Name(), observer) == nil {
		t.Fatal("duplicate event filter accepted")
	}
}

func TestContextualSpellFilterRejectsUnionDeclaration(t *testing.T) {
	target := DeclareSpellOrPermanentTarget("object")
	spell := SpellFromObjectTarget(target).WithFilter(SpellControlledByYou())
	card := NewInstant("Union", "", Spell(Counter(spell), WithTargets(target)))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("unsupported union filter was ignored")
	}
}
