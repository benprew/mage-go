package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestFilterPermanentSetDeclaration(t *testing.T) {
	for _, subjects := range []PermanentSet{nil, DeclarePermanentTargets("unbound", 1, 1, IsCreature)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(TapEach(FilterPermanentSet(subjects, AnyPermanent)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted missing collection declaration")
		}
	}
	targets := DeclarePermanentTargets("valid", 1, 2, IsCreature)
	card := NewInstant("Valid", "", WithAbility(Spell(TapEach(FilterPermanentSet(targets, HasKeywordFilter(Flying))), WithTargets(targets))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestFilterPermanentSetTargets(t *testing.T) {
	for _, change := range []string{"none", "gain", "lose", "phase", "phase-in", "return", "shroud", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			targets := DeclarePermanentTargets("targets", 2, 2, IsCreature)
			card := NewInstant("Filter", "", WithAbility(Spell(DealDamageEach(This, PermanentDamageRecipients(FilterPermanentSet(targets, HasKeywordFilter(Flying))), Fixed(2)), WithTargets(targets))))
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := NewCreature("Flying", "", 2, 8, WithKeyword(Flying))
			victim.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(victim, b.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Ground", "", 2, 8), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			current := first.ID()
			switch change {
			case "gain":
				g.MutablePermanent(second.ID()).GrantBaseAttr(Flying)
			case "lose":
				g.MutablePermanent(first.ID()).RevokeBaseAttr(Flying)
			case "phase":
				g.PhaseOut(first.ID())
			case "phase-in":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(first.ID()))
				b.RemoveFromHand(victim.ID())
				current = g.PutOnBattlefield(victim, b.PlayerID()).ID()
			case "shroud":
				g.MutablePermanent(first.ID()).GrantBaseAttr(Shroud)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: current, TargetID: current, ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{current})
			want := 2
			if change == "lose" || change == "phase" || change == "return" || change == "shroud" {
				want = 0
			}
			if got := g.FindPermanent(current).Damage; got != want {
				t.Fatalf("first damage=%d want=%d", got, want)
			}
			want = 0
			if change == "gain" {
				want = 2
			}
			if g.FindPermanent(second.ID()).Damage != want {
				t.Fatal("filter did not use current characteristics")
			}
			if clone.FindPermanent(first.ID()).Damage != 0 {
				t.Fatal("clone changed before resolution")
			}
			clone.ResolveStack()
			if clone.FindPermanent(first.ID()).Damage != 2 || clone.FindPermanent(second.ID()).Damage != 0 {
				t.Fatal("clone lost its target/filter context")
			}
		})
	}
}
