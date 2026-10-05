package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUntilSourceUntapsLifetime(t *testing.T) {
	for _, change := range []string{"untap-retap", "phase-cycle", "leave-return", "control", "stun", "target-phase", "target-return", "target-phase-source-untap", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewArtifact("Source", "")
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			targetCard := NewCreature("Target", "", 2, 3)
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			g.TapPermanent(source)
			effect := UntilSourceUntaps(This,
				ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(2)),
				GrantEachKeyword(MatchingPermanents(IsCreature), Flying))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "untap-retap":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "leave-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.TapPermanent(g.PutOnBattlefield(sourceCard, a.PlayerID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "stun":
				g.MutablePermanent(source.ID()).AddCounter(Stun, 1)
				if g.UntapPermanent(g.FindPermanent(source.ID())) {
					t.Fatal("stun did not replace untap")
				}
			case "target-phase":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				target = g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase-source-untap":
				ids := g.PhaseOut(target.ID())
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.PhaseIn(ids)
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "cleanup":
				g.effects.RemoveEndOfTurn()
			}
			g.ApplyContinuousEffects()
			active := change == "control" || change == "stun" || change == "target-phase" || change == "cleanup"
			p, toughness := 2, 3
			if active {
				p, toughness = 3, 5
			}
			live := g.FindPermanent(target.ID())
			if live.CurrentPower(g) != p || live.CurrentToughness(g) != toughness || live.HasKeyword(Flying) != active {
				t.Fatal("wrong lifetime or captured member")
			}
			late := g.PutOnBattlefield(NewCreature("Late", "", 2, 3), a.PlayerID())
			if late.CurrentPower(g) != 2 || late.HasKeyword(Flying) {
				t.Fatal("duration added a later member")
			}
			old := clone.FindPermanent(targetCard.ID())
			if old.CurrentPower(clone) != 3 || !old.HasKeyword(Flying) {
				t.Fatal("live expiration changed clone")
			}
			clone.UntapPermanent(clone.FindPermanent(source.ID()))
			clone.TapPermanent(clone.FindPermanent(source.ID()))
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(targetCard.ID()).CurrentPower(clone) != 2 {
				t.Fatal("clone revived an expired duration")
			}
		})
	}
}

func TestUntilSourceUntapsStartAndValidation(t *testing.T) {
	for _, change := range []string{"untapped", "absent", "phased", "returned", "retapped"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 3)
			source := g.PutOnBattlefield(card, a.PlayerID())
			original := source.incarnationID
			g.TapPermanent(source)
			switch change {
			case "untapped", "retapped":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				if change == "retapped" {
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "absent", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returned" {
					g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				}
			case "phased":
				g.PhaseOut(source.ID())
			}
			effect := UntilSourceUntaps(This, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(2)))
			if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: original}); err != nil {
				t.Fatal(err)
			}
			g.PhaseIn([]uuid.UUID{source.ID()})
			if live := g.FindPermanent(source.ID()); live != nil {
				g.TapPermanent(live)
			}
			g.ApplyContinuousEffects()
			if len(g.effects.effects) != 0 && change != "retapped" {
				t.Fatal("duration began after absent or untapped source")
			}
			if change == "retapped" && g.FindPermanent(source.ID()).CurrentPower(g) != 3 {
				t.Fatal("duration did not start at resolution")
			}
		})
	}
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(), UntilSourceUntaps(This, ModifyPT(This, Fixed(1), Fixed(1))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, card := range []Card{
		NewInstant("No source", "", WithAbility(Spell(UntilSourceUntaps(This, ModifyPT(This, Fixed(1), Fixed(1)))))),
		NewArtifact("Empty", "", WithAbility(Activate(Costs(), UntilSourceUntaps(This)))),
		NewArtifact("Control", "", WithAbility(Activate(Costs(), UntilSourceUntaps(This, ChangeController(This, You))))),
	} {
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatalf("accepted %s", card.Name())
		}
	}
}

func TestUntilSourceUntapsTransitionsBeforeNotifications(t *testing.T) {
	for _, together := range []bool{false, true} {
		t.Run(fmt.Sprint(together), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewArtifact("First", ""), a.PlayerID())
			second := g.PutOnBattlefield(NewArtifact("Second", ""), b.PlayerID())
			target := g.PutOnBattlefield(NewCreature("Target", "", 2, 3), a.PlayerID())
			for _, source := range []*Permanent{first, second} {
				g.TapPermanent(source)
				if err := ApplyEffect(g, UntilSourceUntaps(This, ModifyEachPT(MatchingPermanents(IsCreature), Fixed(1), Fixed(0)), GrantEachKeyword(MatchingPermanents(IsCreature), Flying)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			events := 0
			observer := NewArtifact("Observer", "", WithAbility(NewTriggered(EvtBecameUntapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *GameEvent, game GameReader, _, _ uuid.UUID) bool {
				events++
				want := 3
				if together || event.SourceID == second.ID() {
					want = 2
				}
				live := game.FindPermanent(target.ID())
				if live.CurrentPower(g) != want || live.HasKeyword(Flying) != (want == 3) {
					t.Fatal("untap event saw a stale layer")
				}
				if together && (game.FindPermanent(first.ID()).Tapped || game.FindPermanent(second.ID()).Tapped) {
					t.Fatal("batch notification saw intermediate untaps")
				}
				return false
			})))
			g.PutOnBattlefield(observer, a.PlayerID())
			if together {
				g.untapPermanentsTogether([]*Permanent{first, second})
			} else {
				g.UntapPermanent(first)
				g.UntapPermanent(second)
			}
			if events != 2 {
				t.Fatal("missing untap notifications")
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 3), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Source", ""), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.TapPermanent(aura)
	if err := ApplyEffect(g, UntilSourceUntaps(This, ModifyPT(AttachedPermanent, Fixed(1), Fixed(1))), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseIn(g.PhaseOut(host.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(host.ID()).CurrentPower(g) != 2 {
		t.Fatal("indirect source phasing revived duration")
	}
}
