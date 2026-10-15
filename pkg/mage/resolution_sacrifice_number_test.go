package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRequiredSacrificeNumberPayment(t *testing.T) {
	for _, state := range []string{"paid", "short", "phased", "opponent", "invalid", "replaced", "cancelled", "zero", "negative", "stale-phase", "stale-return", "stale-control"} {
		t.Run(state, func(t *testing.T) {
			a := &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			victims := make([]*Permanent, 0, 2)
			for range 2 {
				card := NewCreature("Payment", "", 1, 1, WithKeyword(core.Indestructible))
				card.SetOwner(b.PlayerID())
				victims = append(victims, g.PutOnBattlefield(card, a.PlayerID()))
			}
			amount := 2
			switch state {
			case "short":
				amount = 3
			case "zero":
				amount = 0
			case "negative":
				amount = -2
			case "phased":
				g.PhaseOut(victims[1].ID())
			case "opponent":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: victims[1].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "invalid":
				a.invalidAt = 2
				a.invalid = victims[0]
			case "replaced":
				g.AddExileIfWouldGoToGraveyardThisTurn(victims[0].ID(), uuid.New())
			case "cancelled":
				g.effects.AddReplacement(&collectionSacrificeReplacement{blocked: victims[1].ID()})
			}
			if state == "stale-phase" || state == "stale-return" || state == "stale-control" {
				a.check = func(_ []*Permanent, _ GameReader) {
					if a.calls != 2 {
						return
					}
					switch state {
					case "stale-phase":
						g.PhaseOut(victims[1].ID())
					case "stale-return":
						g.RemoveFromBattlefield(victims[1])
						g.PutOnBattlefield(victims[1].Card, a.PlayerID())
					case "stale-control":
						g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: victims[1].ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				}
			}
			g.AddRegenerationShield(victims[0].ID())
			e := RequiredSacrificeNumberOrElse(You, Fixed(amount), IsCreature, GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(1)))
			clone := g.Clone()
			if err := ApplyEffect(g, e, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := state == "paid" || state == "replaced" || state == "cancelled" || state == "zero" || state == "negative"
			want := 21
			if paid {
				want = 23
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("life %d/%d want %d/20", a.Life(), b.Life(), want)
			}
			moved := paid && amount > 0
			if (g.FindPermanentIncludingPhased(victims[0].ID()) == nil) != moved || (g.FindPermanentIncludingPhased(victims[1].ID()) == nil) != (moved && state != "cancelled") {
				t.Fatal("payment did not commit atomically")
			}
			if clone.FindPermanent(victims[0].ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("payment changed clone")
			}
			if state == "replaced" && (len(g.GetExile()) != 1 || len(b.Graveyard()) != 1) {
				t.Fatal("replacement or owner destination")
			}
			if (state == "short" || state == "zero" || state == "negative" || state == "phased" || state == "opponent") && a.calls != 0 {
				t.Fatal("unavailable or empty payment made choices")
			}
		})
	}
}

func TestRequiredSacrificeNumberBatchAndAmount(t *testing.T) {
	a := &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	g.SetStep(core.PrecombatMain)
	var ids []uuid.UUID
	observed := 0
	watch := func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.FromZone != core.ZoneBattlefield {
			return false
		}
		if g.FindPermanent(ids[0]) != nil && g.FindPermanent(ids[1]) != nil {
			return false
		}
		observed++
		for _, id := range ids {
			if reader.FindPermanent(id) == nil || g.FindPermanent(id) != nil {
				t.Fatal("partial sacrifice observation")
			}
		}
		return true
	}
	source := g.PutOnBattlefield(NewCreature("Source", "", 2, 2, WithAbility(NewTriggered(core.EvtZoneChange, false, GainLifeFor(You, Fixed(1))).SetCondition(watch))), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	ids = []uuid.UUID{source.ID(), other.ID()}
	observed = 0
	a.check = func(_ []*Permanent, _ GameReader) { source.Counters[core.Charge] = 9 }
	amount := CountersOn(This, core.Charge)
	source.Counters[core.Charge] = 2
	e := RequiredSacrificeNumberOrElse(You, amount, IsCreature, GainLifeFor(You, Fixed(3)), GainLifeFor(You, Fixed(100)))
	if err := ApplyEffect(g, e, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 2 || a.calls != 2 || a.Life() != 23 {
		t.Fatalf("observed=%d calls=%d life=%d", observed, a.calls, a.Life())
	}
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatal("source simultaneous triggers lost")
	}
}

func TestRequiredSacrificeNumberDefinition(t *testing.T) {
	for i, e := range []ResolvingEffect{
		RequiredSacrificeNumberOrElse(nil, Fixed(2), IsCreature, GainLifeFor(You, Fixed(1)), nil),
		RequiredSacrificeNumberOrElse(You, nil, IsCreature, GainLifeFor(You, Fixed(1)), nil),
		RequiredSacrificeNumberOrElse(You, Fixed(2), PermanentFilter{}, GainLifeFor(You, Fixed(1)), nil),
		RequiredSacrificeNumberOrElse(You, Fixed(2), IsCreature, nil, nil),
		RequiredSacrificeNumberOrElse(You, FromNumber(BindNumber("unbound")), IsCreature, GainLifeFor(You, Fixed(1)), nil),
	} {
		card := NewArtifact(fmt.Sprint(i), "", WithAbility(Activate(Costs(), e)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted invalid declaration %d", i)
		}
	}
	e := RequiredSacrificeNumberOrElse(You, Fixed(2), IsCreature, nil, GainLifeFor(You, Fixed(1)))
	card := NewArtifact("Valid", "", WithAbility(Activate(Costs(), e)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}
