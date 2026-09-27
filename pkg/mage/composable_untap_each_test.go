package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntapEachBatchAndStun(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	stunned := g.PutOnBattlefield(NewCreature("Stunned", "", 1, 1), a.PlayerID())
	already := g.PutOnBattlefield(NewCreature("Already", "", 1, 1), b.PlayerID())
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	for _, p := range []*Permanent{first, second, stunned, phased} {
		g.TapPermanent(p)
	}
	stunned.AddCounter(core.Stun, 1)
	g.PhaseOut(phased.ID())
	var events, intermediate int
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtBecameUntapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == first.ID() || event.SourceID == second.ID() {
			events++
			if game.FindPermanent(first.ID()).Tapped || game.FindPermanent(second.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := UntapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if events != 2 || intermediate != 0 || first.Tapped || second.Tapped || !stunned.Tapped || stunned.Counters[core.Stun] != 0 || already.Tapped || !phased.Tapped {
		t.Fatalf("events %d, intermediate %d, state %v/%v/%v/%d/%v/%v", events, intermediate, first.Tapped, second.Tapped, stunned.Tapped, stunned.Counters[core.Stun], already.Tapped, phased.Tapped)
	}
}

func TestUntapEachIdentityCloneAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.TapPermanent(old)
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	g.TapPermanent(current)
	g.TapPermanent(other)
	clone := g.Clone()
	if err := UntapEach(fixedTapSet{[]*Permanent{old, current, current, other}}).Apply(&EffectContext{Game: clone}); err != nil {
		t.Fatal(err)
	}
	if clone.FindPermanent(current.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped || !current.Tapped || !other.Tapped {
		t.Fatal("stale or duplicate member changed the original or clone")
	}
	valid := NewInstant("Valid", "", Spell(UntapEach(ControlledPermanents(IsCreature))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]); len(got.Children) == 0 {
		t.Fatal("untap collection missing from inspection")
	}
	bad := NewInstant("Bad", "", Spell(UntapEach(nil)))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("nil permanent set accepted")
	}
}

func TestUntapEachTargetGroupKeepsAnnouncementIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	g.TapPermanent(old)
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	g.TapPermanent(current)
	if err := UntapEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !current.Tapped {
		t.Fatal("returned target was untapped through an old target group")
	}
}
