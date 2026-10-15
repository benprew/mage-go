package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRetainedPlayerGrant(t *testing.T) {
	for _, state := range []string{"normal", "control", "phase", "cycle", "return", "remove", "queued", "twice"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewArtifact("Recipient", "")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			player := BindPlayer("upkeep")
			prototype := WhenEvent(TurnStepBegins(EachUpkeep, player), GainLifeFor(FromPlayer(player), Fixed(1))).(*GenericTriggered)
			grant := Until(core.Indefinite, TriggeredAbilityForPlayerModification(This, You, prototype))
			if err := ApplyEffect(g, grant, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			want := 21
			switch state {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
				want = 20
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
				want = 20
			case "remove":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
				want = 20
			case "twice":
				if err := ApplyEffect(g, grant, p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if state == "queued" {
				g.BouncePermanentToHand(p)
			}
			g.ResolveStack()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			wantB := 20
			if state == "twice" {
				wantB = 21
			}
			if a.Life() != want || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d; want %d %d", a.Life(), b.Life(), want, wantB)
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 21 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone shared grant changes")
			}
		})
	}
}

func TestRetainedPlayerGrantDeclarations(t *testing.T) {
	for _, player := range []PlayerRef{nil, FromPlayer(BindPlayer("missing"))} {
		trigger := At(EachUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Until(core.Indefinite, TriggeredAbilityForPlayerModification(This, player, trigger)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid player accepted")
		}
	}
}

func TestRetainedPlayerGrantWaitingLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := g.PutOnBattlefield(NewCreature("Recipient", "", 2, 2), b.PlayerID())
	card := NewAura("Donor", "")
	card.SetOwner(a.PlayerID())
	donor := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	bound := BindPermanent("recipient")
	prototype := At(EachUpkeep, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
	body := Sequence(Snapshot(AttachedPermanent, bound), AtNext(EachEndStep,
		Until(core.Indefinite, TriggeredAbilityForPlayerModification(From(bound), ControllerOfWithLKI(This), prototype))))
	if err := ApplyEffect(g, body, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(donor)
	for range 2 {
		found := false
		for _, source := range g.WaitingDelayedPermanentSources() {
			found = found || source.ID() == donor.ID()
		}
		if !found || len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
			t.Fatal("waiting query lost source or installed grant")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	for _, player := range []Player{a, b} {
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
	}
	if a.Life() != 20 || b.Life() != 21 {
		t.Fatal("grant lost source LKI or recipient controller")
	}
}
