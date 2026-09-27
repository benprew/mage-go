package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTapEachTransitionsTogether(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), b.PlayerID())
	already := g.PutOnBattlefield(NewCreature("Already", "", 1, 1), b.PlayerID())
	g.TapPermanent(already)
	phased := g.PutOnBattlefield(NewCreature("Phased", "", 1, 1), a.PlayerID())
	g.PhaseOut(phased.ID())
	var events, intermediate int
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtTapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		if event.SourceID == first.ID() || event.SourceID == second.ID() {
			events++
			if !game.FindPermanent(first.ID()).Tapped || !game.FindPermanent(second.ID()).Tapped {
				intermediate++
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	if err := TapEach(MatchingPermanents(IsCreature)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if events != 2 || intermediate != 0 || !first.Tapped || !second.Tapped || !already.Tapped || phased.Tapped {
		t.Fatalf("events %d, intermediate %d, tapped %v/%v/%v/%v", events, intermediate, first.Tapped, second.Tapped, already.Tapped, phased.Tapped)
	}
}

func TestTapEachCloneAndDeclaration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Blue", "{U}", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	if err := TapEach(MatchingPermanents(HasColorFilter(core.Blue))).Apply(&EffectContext{Game: clone, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(card.ID()).Tapped || g.FindPermanent(card.ID()).Tapped {
		t.Fatal("tap changed original game or missed clone")
	}
	valid := NewInstant("Valid", "", Spell(TapEach(MatchingPermanents(HasColorFilter(core.Blue)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]); len(got.Children) == 0 {
		t.Fatal("tap collection missing from inspection")
	}
	bad := NewInstant("Bad", "", Spell(TapEach(nil)))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("nil permanent set accepted")
	}
}

type fixedTapSet struct{ permanents []*Permanent }

func (fixedTapSet) permanentSet()                                   {}
func (s fixedTapSet) resolvePermanents(*EffectContext) []*Permanent { return s.permanents }

func TestTapEachSkipsStaleAndDuplicateObjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Returned", "", 1, 1)
	old := g.PutOnBattlefield(card, a.PlayerID())
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	if err := TapEach(fixedTapSet{[]*Permanent{old, current, current, other}}).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	if !current.Tapped || !other.Tapped || old.Tapped {
		t.Fatal("stale or duplicate member changed tap outcome")
	}
}

func TestTapEachTargetGroupKeepsAnnouncementIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Target", "", 1, 1)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{
		targets.targetDeclaration: {{id: old.ID(), incarnation: old.incarnationID}},
	}}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	if err := TapEach(targets).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if current.Tapped {
		t.Fatal("returned target was tapped through an old target group")
	}
}
