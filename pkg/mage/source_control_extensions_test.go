package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourceControlExtensionsPower(t *testing.T) {
	for _, change := range []string{"none", "target-power", "source-power", "untap", "control", "ability", "source-phase", "target-phase", "target-return", "layer-source", "layer-target", "switch"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("subject", IsCreature, TargetPowerLessThanOrEqualToSourcePower())
			card := NewCreature("Source", "", 2, 8, WithAbility(Activate(Costs(TapCost(This)), GainControlWhileSourceTappedAndPowerLE(target), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(AttrSummonSick)
			victimCard := NewCreature("Victim", "", 2, 8)
			victimCard.SetOwner(b.PlayerID())
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("control did not start at equal power")
			}
			clone := g.Clone()
			source = g.MutablePermanent(source.ID())
			victim = g.MutablePermanent(victim.ID())
			ended := false
			switch change {
			case "target-power":
				victim.AddCounter(P1P1, 1)
				ended = true
			case "source-power":
				source.AddCounter(M1M1, 1)
				ended = true
			case "untap":
				g.UntapPermanent(source)
				g.TapPermanent(source)
				ended = true
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
			case "ability":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), Indefinite))
			case "source-phase":
				g.PhaseIn(g.PhaseOut(source.ID()))
				ended = true
			case "target-phase":
				g.PhaseIn(g.PhaseOut(victim.ID()))
				ended = true
			case "target-return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
				ended = true
			case "layer-source":
				if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(3), Fixed(0))), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				victim.AddCounter(P1P1, 2)
			case "layer-target":
				if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0))), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				ended = true
			case "switch":
				if err := ApplyEffect(g, Until(EndOfTurn, SwitchPowerToughness(This)), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				ended = true
			}
			g.ApplyContinuousEffects()
			want := a.PlayerID()
			if ended {
				want = b.PlayerID()
			}
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("wrong duration endpoint", change)
			}
			g.effects.RemoveEndOfTurn()
			g.MutablePermanent(source.ID()).Counters[M1M1] = 0
			g.MutablePermanent(victim.ID()).Counters[P1P1] = 0
			g.ApplyContinuousEffects()
			if ended && g.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("ended duration restarted")
			}
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone shared expiration")
			}
		})
	}
}

func TestSourceControlExtensionsPlural(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 8), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 8), b.PlayerID())
	if err := ApplyEffect(g, GainControlEachWhileSourceControlled(UnionPermanents(MatchingPermanents(IsCreature), MatchingPermanents(IsCreature))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	late := g.PutOnBattlefield(NewCreature("Late", "", 1, 8), b.PlayerID())
	for _, p := range []*Permanent{first, second} {
		if g.FindPermanent(p.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("member not captured")
		}
	}
	if late.ControllerID() != b.PlayerID() {
		t.Fatal("late entrant joined")
	}
	clone := g.Clone()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	for _, p := range []*Permanent{first, second} {
		if g.FindPermanent(p.ID()).ControllerID() != b.PlayerID() || clone.FindPermanent(p.ID()).ControllerID() != a.PlayerID() {
			t.Fatal("plural duration restarted or shared clone")
		}
	}
}

func TestSourceControlExtensionsInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{GainControlWhileSourceTappedAndPowerLE(nil), GainControlEachWhileSourceControlled(nil)} {
		card := NewCreature("Invalid", "", 2, 3, WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("nil subject accepted")
		}
	}
	card := NewInstant("Invalid", "", WithAbility(Spell(GainControlEachWhileSourceControlled(MatchingPermanents(IsCreature)))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("spell source accepted")
	}
}

func TestSourceControlExtensionsTargetLegality(t *testing.T) {
	for _, change := range []string{"live", "power", "negative", "departure", "return", "phase", "phase-cycle", "layer", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			target := DeclarePermanentTarget("subject", IsCreature, TargetPowerLessThanOrEqualToSourcePower())
			card := NewCreature("Source", "", 2, 8, WithAbility(Activate(Costs(TapCost(This)), TapSubject(target), WithTargets(target))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(AttrSummonSick)
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 8), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "power":
				source.AddCounter(M1M1, 1)
			case "negative":
				source.AddCounter(M1M1, 4)
			case "departure", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID()).AddCounter(M1M1, 1)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "layer":
				if err := ApplyEffect(g, Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0))), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			if change == "clone" {
				clone.MutablePermanent(victim.ID()).AddCounter(P1P1, 1)
			}
			g.ResolveStack()
			clone.ResolveStack()
			want := change != "power" && change != "negative" && change != "layer"
			if g.FindPermanent(victim.ID()).Tapped != want || clone.FindPermanent(victim.ID()).Tapped != (want && change != "clone") {
				t.Fatal("source power or visit legality", change)
			}
		})
	}
}

func TestSourceControlExtensionsStart(t *testing.T) {
	for _, plural := range []bool{false, true} {
		for _, change := range []string{"source-leave", "source-return", "source-phase", "target-phase", "untap", "power", "control"} {
			t.Run(change+map[bool]string{false: "/power", true: "/plural"}[plural], func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				target := DeclarePermanentTarget("subject", IsCreature)
				effect := GainControlWhileSourceTappedAndPowerLE(target)
				if plural {
					effect = GainControlEachWhileSourceControlled(SingletonPermanent(target))
				}
				card := NewCreature("Source", "", 2, 8, WithAbility(Activate(Costs(TapCost(This)), effect, WithTargets(target))))
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(AttrSummonSick)
				victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 8), b.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "source-leave", "source-return":
					g.RemoveFromBattlefield(source)
					if change == "source-return" {
						g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "target-phase":
					g.PhaseOut(victim.ID())
				case "untap":
					g.UntapPermanent(source)
				case "power":
					victim.AddCounter(P1P1, 1)
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
				}
				g.ResolveStack()
				changed := g.FindPermanentIncludingPhased(victim.ID()).ControllerID() == a.PlayerID()
				want := (plural && (change == "untap" || change == "power")) || (!plural && change == "control")
				if changed != want {
					t.Fatal("invalid duration started", change, plural)
				}
			})
		}
	}
}
