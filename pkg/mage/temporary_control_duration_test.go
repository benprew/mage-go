package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTemporaryControlDuration(t *testing.T) {
	for _, change := range []string{"untap-retap", "source-phase", "source-return", "source-control", "ability-removal", "stun", "target-phase", "target-return", "phased-expiry", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("subject", IsCreature)
			effect := UntilSourceUntaps(This, ChangeController(target, You))
			card := NewArtifact("Source", "", WithAbility(Activate(Costs(Tap()), effect, WithTargets(target))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			victimCard := NewCreature("Victim", "", 2, 3)
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("control did not begin")
			}
			clone := g.Clone()
			active := false
			switch change {
			case "untap-retap":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "source-phase":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				active = true
			case "ability-removal":
				g.GetEffects().Add(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				active = true
			case "stun":
				g.MutablePermanent(source.ID()).AddCounter(core.Stun, 1)
				g.UntapPermanent(g.FindPermanent(source.ID()))
				active = true
			case "target-phase":
				g.PhaseIn(g.PhaseOut(victim.ID()))
				active = true
			case "target-return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(victimCard, b.PlayerID())
			case "phased-expiry":
				ids := g.PhaseOut(victim.ID())
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.PhaseIn(ids)
			case "cleanup":
				g.GetEffects().RemoveEndOfTurn()
				active = true
			}
			g.ApplyContinuousEffects()
			want := b.PlayerID()
			if active {
				want = a.PlayerID()
			}
			if g.FindPermanent(victim.ID()).ControllerID() != want {
				t.Fatal("wrong duration endpoint")
			}
			if clone.FindPermanent(victim.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("clone changed with original")
			}
			clone.UntapPermanent(clone.FindPermanent(source.ID()))
			if clone.FindPermanent(victim.ID()).ControllerID() != b.PlayerID() {
				t.Fatal("clone duration did not end")
			}
		})
	}
}

func TestTemporaryControlDurationStart(t *testing.T) {
	for _, change := range []string{"untapped", "phased", "absent", "returned", "retapped"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Source", "")
			source := g.PutOnBattlefield(card, a.PlayerID())
			original := source.incarnationID
			target := g.PutOnBattlefield(NewCreature("Victim", "", 2, 3), b.PlayerID())
			binding := BindPermanent("subject")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{binding: {target.ID(), target.incarnationID}}
			g.TapPermanent(source)
			switch change {
			case "untapped", "retapped":
				g.UntapPermanent(g.FindPermanent(source.ID()))
				if change == "retapped" {
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "absent", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returned" {
					g.TapPermanent(g.PutOnBattlefield(card, a.PlayerID()))
				}
			}
			effect := UntilSourceUntaps(This, ChangeController(From(binding), You))
			if err := effect.Apply(&EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: original}); err != nil {
				t.Fatal(err)
			}
			want := b.PlayerID()
			if change == "retapped" {
				want = a.PlayerID()
			}
			if g.FindPermanent(target.ID()).ControllerID() != want {
				t.Fatal("invalid source started duration")
			}
		})
	}
}

func TestTemporaryControlDurationSimultaneousUntap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	sources := make([]*Permanent, 0, 2)
	victims := make([]uuid.UUID, 0, 2)
	for range 2 {
		target := DeclarePermanentTarget("subject", IsCreature)
		card := NewArtifact("Source", "", WithAbility(Activate(Costs(Tap()), UntilSourceUntaps(This, ChangeController(target, You)), WithTargets(target))))
		source := g.PutOnBattlefield(card, a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 3), b.PlayerID())
		sources = append(sources, source)
		victims = append(victims, victim.ID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	events := 0
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtBecameUntapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
		events++
		for _, id := range victims {
			if game.FindPermanent(id).ControllerID() != b.PlayerID() {
				t.Fatal("untap event saw stale control")
			}
		}
		for _, p := range sources {
			if game.FindPermanent(p.ID()).Tapped {
				t.Fatal("untap event saw partial batch")
			}
		}
		return false
	})))
	g.PutOnBattlefield(observer, a.PlayerID())
	g.untapPermanentsTogether(sources)
	if events != 2 {
		t.Fatal("missing untap events")
	}
}
