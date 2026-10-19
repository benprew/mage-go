package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSetSnapshotVisitsAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("First", "", 1, 4)
	p := g.PutOnBattlefield(card, a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 4), a.PlayerID())
	saved := BindPermanentSet("saved")
	subjects := UnionPermanents(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature))
	snapshot := SnapshotPermanentSet(subjects, saved)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := snapshot.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if got := len(FromPermanentSet(saved).resolvePermanents(ctx)); got != 2 {
		t.Fatal("duplicate visits", got)
	}
	clone := g.Clone()
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 4), a.PlayerID())
	g.PhaseOut(p.ID())
	if got := len(FromPermanentSet(saved).resolvePermanents(ctx)); got != 1 {
		t.Fatal("phased visit", got)
	}
	g.PhaseIn(g.PhaseOut(second.ID()))
	g.PhaseIn([]uuid.UUID{p.ID()})
	if got := len(FromPermanentSet(saved).resolvePermanents(ctx)); got != 2 {
		t.Fatal("phase cycle lost visit", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	control := Until(core.Indefinite, ChangeEachController(FromPermanentSet(saved), You))
	if err := control.Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() {
		t.Fatal("saved survivor was not transferred")
	}
	if g.FindPermanent(returned.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(late.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("new visit joined saved set")
	}
	if err := SnapshotPermanentSet(MatchingPermanents(IsLand), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(FromPermanentSet(saved).resolvePermanents(ctx)) != 0 {
		t.Fatal("empty snapshot kept old visits")
	}
	if len(FromPermanentSet(saved).resolvePermanents(&EffectContext{Game: clone})) != 2 {
		t.Fatal("clone shared snapshot")
	}
}

func TestPermanentSetSnapshotControlLifetime(t *testing.T) {
	for _, change := range []string{"source-left", "source-phase", "ability", "subject-phase", "subject-return", "cleanup", "later-control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			saved := BindPermanentSet("set")
			body := Sequence(SnapshotPermanentSet(MatchingPermanents(IsCreature), saved), TapEach(FromPermanentSet(saved)), Until(core.Indefinite, ChangeEachController(FromPermanentSet(saved), You)))
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 4)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 2, 4), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			pending := g.Clone()
			g.ResolveStack()
			for _, p := range []*Permanent{victim, second} {
				if live := g.FindPermanent(p.ID()); live.ControllerID() != a.PlayerID() || !live.Tapped {
					t.Fatal("batch did not apply")
				}
			}
			clone := g.Clone()
			want := a.PlayerID()
			switch change {
			case "source-left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "source-phase":
				g.PhaseOut(source.ID())
			case "ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
			case "subject-phase":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "subject-return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				want = b.PlayerID()
			case "cleanup":
				g.GetEffects().RemoveEndOfTurn()
			case "later-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				want = b.PlayerID()
			}
			g.ApplyContinuousEffects()
			if g.FindPermanent(victim.ID()).ControllerID() != want || g.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("wrong control lifetime")
			}
			pending.ResolveStack()
			if pending.FindPermanent(second.ID()).ControllerID() != a.PlayerID() || clone.FindPermanent(second.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone lost control")
			}
		})
	}
}

func TestPermanentSetSnapshotDefinition(t *testing.T) {
	saved := BindPermanentSet("saved")
	for _, body := range []Effect{
		SnapshotPermanentSet(nil, saved), SnapshotPermanentSet(MatchingPermanents(IsCreature), nil),
		Until(core.Indefinite, ChangeEachController(nil, You)), Until(core.Indefinite, ChangeEachController(MatchingPermanents(IsCreature), nil)),
		Until(core.Indefinite, ChangeEachController(FromPermanentSet(saved), You)),
		Sequence(TapEach(FromPermanentSet(saved)), SnapshotPermanentSet(MatchingPermanents(IsCreature), saved)),
	} {
		if ValidateCardDefinition("Bad", NewSorcery("Bad", "", WithAbility(Spell(body)))) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestPermanentSetSnapshotDelayedFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 4), a.PlayerID())
	saved := BindPermanentSet("saved")
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	body := Sequence(SnapshotPermanentSet(MatchingPermanents(IsCreature), saved), AtNext(EachEndStep, TapEach(FromPermanentSet(saved))))
	if err := ValidateCardDefinition("Delayed", NewSorcery("Delayed", "", WithAbility(Spell(body)))); err != nil {
		t.Fatal(err)
	}
	if err := body.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 4), a.PlayerID())
	if err := SnapshotPermanentSet(MatchingPermanents(IsLand), saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.PhaseOut(first.ID())
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
	}
	if !clone.FindPermanent(first.ID()).Tapped || clone.FindPermanent(late.ID()).Tapped {
		t.Fatal("delay lost saved membership")
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	if g.FindPermanent(first.ID()).Tapped {
		t.Fatal("phased visit acted or clone changed original")
	}
}
