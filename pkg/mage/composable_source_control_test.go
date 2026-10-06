package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestGainControlWhileSourceControlled(t *testing.T) {
	for _, tapped := range []bool{false, true} {
		for _, change := range []string{"none", "source-leave", "source-return", "source-phase", "source-cycle", "source-control", "untap", "untap-retap", "target-return", "target-phase", "remove-ability"} {
			t.Run(change+map[bool]string{false: "/controlled", true: "/tapped"}[tapped], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				target := DeclarePermanentTarget("subject", IsCreature)
				card := NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), GainControlWhileSourceControlled(target, tapped), WithTargets(target))))
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				victimCard := NewCreature("Victim", "", 2, 2)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("control did not start")
				}
				clone := g.Clone()
				switch change {
				case "source-leave", "source-return":
					g.RemoveFromBattlefield(source)
					if change == "source-return" {
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "source-phase", "source-cycle":
					phased := g.PhaseOut(source.ID())
					if change == "source-cycle" {
						g.PhaseIn(phased)
					}
				case "source-control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
				case "untap", "untap-retap":
					g.UntapPermanent(source)
					if change == "untap-retap" {
						g.TapPermanent(g.FindPermanent(source.ID()))
					}
				case "target-return":
					g.RemoveFromBattlefield(victim)
					victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				case "target-phase":
					phased := g.PhaseOut(victim.ID())
					g.PhaseIn(phased)
				case "remove-ability":
					g.AddContinuousEffect(RemoveAllAbilities(source.ID(), Indefinite))
				}
				g.ApplyContinuousEffects()
				ended := change == "source-leave" || change == "source-return" || change == "source-phase" || change == "source-cycle" || change == "source-control" || change == "target-return" || ((change == "untap" || change == "untap-retap") && tapped)
				want := a.PlayerID()
				if ended {
					want = b.PlayerID()
				}
				if g.FindPermanent(victim.ID()).ControllerID() != want {
					t.Fatalf("controller=%s want %s", g.FindPermanent(victim.ID()).ControllerID(), want)
				}
				g.effects.RemoveEndOfTurn()
				if s := g.MutablePermanent(source.ID()); s != nil {
					s.Tapped = true
				}
				g.ApplyContinuousEffects()
				if g.FindPermanent(victim.ID()).ControllerID() != want {
					t.Fatal("expired duration restarted")
				}
				clone.ApplyContinuousEffects()
				if clone.FindPermanent(victimCard.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("clone changed")
				}
			})
		}
	}
}

func TestGainControlWhileSourceControlledStartAndValidation(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase", "control", "untap", "copy"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("subject", IsCreature)
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), GainControlWhileSourceControlled(target, true), WithTargets(target))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "untap":
				g.UntapPermanent(source)
			case "copy":
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				before := len(g.effects.effects)
				g.ResolveTopOfStack()
				if len(g.effects.effects) != before {
					t.Fatal("foreign controller copy installed a duration")
				}
			}
			g.ResolveStack()
			want := b.PlayerID()
			if change == "copy" {
				want = a.PlayerID()
			}
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("invalid duration started")
			}
		})
	}
	for _, subject := range []PermanentRef{nil, DeclarePermanentTarget("unbound", IsCreature)} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), GainControlWhileSourceControlled(subject, false))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewInstant("Bad source", "", WithAbility(Spell(GainControlWhileSourceControlled(This, false))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("spell source accepted")
	}
}

func TestGainControlWhileSourceControlledUntapNotifications(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sources := make([]*Permanent, 0, 2)
	victims := make([]uuid.UUID, 0, 2)
	for range 2 {
		target := DeclarePermanentTarget("subject", IsCreature)
		source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(TapCost(This)), GainControlWhileSourceControlled(target, true), WithTargets(target)))), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 2), b.PlayerID())
		sources = append(sources, source)
		victims = append(victims, victim.ID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	calls := 0
	observer := NewTriggered(EvtBecameUntapped, false).SetCondition(func(_ *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		calls++
		for _, id := range victims {
			if g.FindPermanent(id).ControllerID() != b.PlayerID() {
				t.Fatal("untap notification saw old control")
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
	g.untapPermanentsTogether(sources)
	if calls != 2 {
		t.Fatal("wrong untap event count")
	}
}
