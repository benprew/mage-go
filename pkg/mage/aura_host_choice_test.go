package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAuraHostChoice(t *testing.T) {
	for _, state := range []string{"legal", "shroud", "hexproof", "protection", "forbidden", "phased", "wrong-type", "stale-card", "missing-card", "non-aura", "phase-cycle"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), b.PlayerID())
			switch state {
			case "shroud":
				host.GrantBaseAttr(core.Shroud)
			case "hexproof":
				host.GrantBaseAttr(core.Hexproof)
			case "protection":
				g.RemoveFromBattlefield(host)
				host = g.PutOnBattlefield(NewCreature("Protected", "", 2, 2, WithAbility(ProtectionFromColor(core.Black))), b.PlayerID())
			case "forbidden":
				host.GrantBaseAttr(core.AttrCantBeEnchanted)
			case "phased":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "wrong-type":
				g.RemoveFromBattlefield(host)
				host = g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
			}
			card := NewAura("Guest", "{B}", WithCastTarget(TargetCreature()))
			if state == "non-aura" {
				card = NewEnchantment("Plain", "")
			}
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			saved := BindCard("guest")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{saved: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
			if state == "stale-card" || state == "missing-card" {
				c, _ := a.RemoveFromGraveyard(card.ID())
				if state == "stale-card" {
					a.AddToGraveyard(c)
				}
			}
			result := BindPermanentChoice("host")
			effect := ChoosePermanentForPlayerOrElse(You, LegalHostsForAuraCard(FromCard(saved), You, MatchingPermanents(AnyPermanent)), result, Sequence(TapSubject(PermanentChosenIn(result)), GainLifeFor(You, Fixed(2))), GainLifeFor(You, Fixed(3)))
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			legal := state == "legal" || state == "shroud" || state == "hexproof" || state == "phase-cycle"
			want := 23
			if legal {
				want = 22
			}
			if a.Life() != want || host.Tapped != legal {
				t.Fatalf("life=%d tapped=%v", a.Life(), host.Tapped)
			}
			if legal {
				g.RemoveFromBattlefield(host)
				if err := effect.Apply(ctx); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 25 {
					t.Fatal("old choice supplied success")
				}
			}
		})
	}
}

func TestAuraHostChoiceInvalid(t *testing.T) {
	result := BindPermanentChoice("host")
	for _, effect := range []ResolvingEffect{
		ChoosePermanentForPlayerOrElse(nil, MatchingPermanents(AnyPermanent), result, GainLifeFor(You, Fixed(1)), nil),
		ChoosePermanentForPlayerOrElse(You, nil, result, GainLifeFor(You, Fixed(1)), nil),
		ChoosePermanentForPlayerOrElse(You, MatchingPermanents(AnyPermanent), nil, GainLifeFor(You, Fixed(1)), nil),
		ChoosePermanentForPlayerOrElse(You, MatchingPermanents(AnyPermanent), result, nil, nil),
		ChoosePermanentForPlayerOrElse(You, LegalHostsForAuraCard(FromCard(BindCard("unbound")), You, MatchingPermanents(AnyPermanent)), result, GainLifeFor(You, Fixed(1)), nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestAuraHostChoiceController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	own := g.PutOnBattlefield(NewCreature("Own", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	card := NewAura("Guest", "{B}", WithCastTarget(DeclarePermanentTarget("own creature", IsCreature, TargetYouControl())))
	card.SetOwner(b.PlayerID())
	b.AddToGraveyard(card)
	saved := BindCard("card")
	g.resolution.cardBindings = map[*CardBinding]capturedCard{saved: {id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard}}
	result := BindPermanentChoice("host")
	body := ChoosePermanentForPlayerOrElse(ControllerOf(This), LegalHostsForAuraCard(FromCard(saved), You, MatchingPermanents(IsCreature)), result, TapSubject(PermanentChosenIn(result)), GainLifeFor(You, Fixed(3)))
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: other.ID()}); err != nil {
		t.Fatal(err)
	}
	if !own.Tapped || other.Tapped || len(b.Graveyard()) != 1 || g.zones.EnteringPermanent() != nil {
		t.Fatal("owner or chooser supplied entry controller, or query moved card")
	}
}

func TestAuraHostChoiceWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindCard("guest")
	result := BindPermanentChoice("host")
	body := ChoosePermanentForPlayerOrElse(You, LegalHostsForAuraCard(FromCard(saved), You, MatchingPermanents(IsCreature)), result, Sequence(ReturnCardToBattlefield(FromCard(saved), You, EnterAttachedTo(PermanentChosenIn(result))), TapSubject(This)), GainLifeFor(You, Fixed(3)))
	card := NewAura("Guest", "{B}", WithCastTarget(TargetCreature()), WithAbility(WhenEvent(AttachedPermanentDies().WithDepartedAuraCard(saved), AtNext(EachEndStep, body))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(b.PlayerID())
	hostCard := NewCreature("Host", "", 2, 2)
	hostCard.SetOwner(b.PlayerID())
	host := g.PutOnBattlefield(hostCard, b.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.DestroyPermanent(host)
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	next := g.PutOnBattlefield(NewCreature("Next", "", 2, 2), b.PlayerID())
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.WaitingDelayedGraveyardSources()) != 1 || g.FindPermanent(card.ID()) != nil || next.Tapped {
			t.Fatal("waiting query lost source or executed choice")
		}
	}
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	got := g.FindPermanent(card.ID())
	if got == nil || got.AttachedTo != next.ID() || a.Life() != 23 || clone.FindPermanent(card.ID()) != nil {
		t.Fatalf("copy result: aura=%v life=%d cloneAura=%v", got, a.Life(), clone.FindPermanent(card.ID()))
	}
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.FindPermanent(card.ID()) == nil || clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("clone lost captured binding")
	}
}
