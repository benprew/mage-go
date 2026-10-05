package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentMatchesFilterIdentity(t *testing.T) {
	for _, attached := range []bool{false, true} {
		t.Run(map[bool]string{false: "source", true: "host"}[attached], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 2, 3, WithSubTypes("Goblin"))
			card.SetOwner(a.PlayerID())
			subject := g.PutOnBattlefield(card, a.PlayerID())
			source := subject
			var ref PermanentRef = This
			if attached {
				source = g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
				g.Attach(source.ID(), subject.ID())
				ref = AttachedPermanent
			}
			condition := PermanentMatchesFilter(ref, And(IsCreature, HasSubType("Goblin")))
			ctx := StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: source.ControllerID()}
			check := func(game *Game, want bool) {
				t.Helper()
				if condition.checkState(game, ctx) != want {
					t.Fatalf("condition want %v", want)
				}
			}
			check(g, true)
			if !PermanentMatchesFilter(ref, PermanentFilter{}).checkState(g, ctx) {
				t.Fatal("zero filter did not match the live subject")
			}
			clone := g.Clone()
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: subject.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			check(g, true)
			phased := g.PhaseOut(subject.ID())
			check(g, false)
			check(clone, true)
			g.PhaseIn(phased)
			check(g, true)
			if attached {
				phasedSource := g.PhaseOut(source.ID())
				check(g, false)
				g.PhaseIn(phasedSource)
				check(g, true)
				other := g.PutOnBattlefield(NewCreature("Other", "", 2, 3), a.PlayerID())
				g.Attach(source.ID(), other.ID())
				check(g, false)
				g.Attach(source.ID(), subject.ID())
				check(g, true)
				g.MutablePermanent(source.ID()).attachedIncarnation = uuid.New()
				check(g, false)
				g.Attach(source.ID(), subject.ID())
			}
			g.BouncePermanentToHand(g.FindPermanent(subject.ID()))
			check(g, false)
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, a.PlayerID())
			check(g, false)
			check(clone, true)
			ctx.SourceIncarnation = uuid.New()
			check(clone, false)
			ctx.SourceID = uuid.New()
			check(g, false)
		})
	}
}

func TestPermanentMatchesFilterDeclarationsAndLayers(t *testing.T) {
	for _, ref := range []PermanentRef{nil, DeclarePermanentTarget("unbound", IsCreature), From(BindPermanent("saved"))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Static(While(PermanentMatchesFilter(ref, IsCreature), ModifyPT(This, Fixed(1), Fixed(0))))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid static reference accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	condition := PermanentMatchesFilter(This, HasColorFilter(core.Red))
	card := NewCreature("Rule", "{R}", 2, 2, WithAbility(Static(While(condition, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(2))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	check := func(power int) {
		t.Helper()
		if g.FindPermanent(other.ID()).CurrentPower(g) != power {
			t.Fatalf("power want %d", power)
		}
	}
	check(3)
	if err := ApplyEffect(g, Until(core.EndOfTurn, SetColors(This, core.Blue)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check(2)
	g.RunStepWithPriority(core.Cleanup)
	check(3)
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check(2)
	g.RunStepWithPriority(core.Cleanup)
	check(3)
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	check(2)
}
