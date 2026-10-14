package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalReturnPlayer struct {
	*BasePlayer
	accept  bool
	offers  int
	onOffer func()
}

func (p *optionalReturnPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.onOffer != nil {
		p.onOffer()
	}
	return p.accept
}

func TestOptionalGraveyardReturn(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "absent", "returned", "choice-return", "choice-depart", "replacement", "nonpermanent", "aura", "wrong-zone", "missing-player"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalReturnPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindCard("guest")
			card := Card(NewCreature("Guest", "", 1, 1, WithAbility(EntersWithNCounters(core.P1P1, 2))))
			if scenario == "nonpermanent" {
				card = NewInstant("Guest", "")
			}
			if scenario == "aura" {
				card = NewEnchantment("Guest", "", WithSubTypes("Aura"))
			}
			card.SetOwner(b.PlayerID())
			b.AddToGraveyard(card)
			zone := core.ZoneGraveyard
			visit := g.graveyardVisit(card.ID())
			if scenario == "wrong-zone" {
				b.RemoveFromGraveyard(card.ID())
				g.ExileCard(card, card.ID())
				zone = core.ZoneExile
				visit = g.FindExiledCard(card.ID()).Visit
			}
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: visit, zone: zone}}
			if scenario == "absent" || scenario == "returned" {
				b.RemoveFromGraveyard(card.ID())
				if scenario == "returned" {
					b.AddToGraveyard(card)
				}
			}
			if scenario == "choice-return" || scenario == "choice-depart" {
				a.onOffer = func() {
					b.RemoveFromGraveyard(card.ID())
					if scenario == "choice-return" {
						b.AddToGraveyard(card)
					}
				}
			}
			if scenario == "replacement" {
				g.AddEntersTappedRule(func(p *Permanent) bool { return p.HasType(core.TypeCreature) })
			}
			player := PlayerRef(You)
			if scenario == "missing-player" {
				player = FromPlayer(BindPlayer("missing"))
			}
			if err := MayReturnNonAuraCardToBattlefield(FromCard(binding), player).Apply(&EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			wantOffers := 1
			switch scenario {
			case "absent", "returned", "nonpermanent", "aura", "wrong-zone", "missing-player":
				wantOffers = 0
			}
			if a.offers != wantOffers {
				t.Fatalf("offers = %d, want %d", a.offers, wantOffers)
			}
			p := g.FindPermanent(card.ID())
			wantEntry := scenario == "accept" || scenario == "replacement"
			if (p != nil) != wantEntry {
				t.Fatalf("entry = %v, want %v", p != nil, wantEntry)
			}
			if p != nil && (p.ControllerID() != a.PlayerID() || p.Card.Owner() != b.PlayerID() || p.Counters[core.P1P1] != 2 || p.Tapped != (scenario == "replacement")) {
				t.Fatal("entry lost owner, controller, counters or replacement")
			}
		})
	}
}

func TestOptionalGraveyardReturnDeclarations(t *testing.T) {
	source := BindCard("source")
	for _, body := range []ResolvingEffect{
		MayReturnNonAuraCardToBattlefield(nil, You),
		MayReturnNonAuraCardToBattlefield(FromCard(nil), You),
		MayReturnNonAuraCardToBattlefield(FromCard(BindCard("unbound")), You),
		MayReturnNonAuraCardToBattlefield(DeclareGraveyardCardTarget("undeclared"), You),
		MayReturnNonAuraCardToBattlefield(FromCard(source), nil),
	} {
		card := NewArtifact("Invalid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), body, WithSourceCardInZone(core.ZoneGraveyard, source))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid optional return accepted")
		}
	}
	binding := BindCard("source")
	card := NewArtifact("Valid", "", WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), MayReturnNonAuraCardToBattlefield(FromCard(binding), You), WithSourceCardInZone(core.ZoneGraveyard, binding))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalGraveyardReturnWaiting(t *testing.T) {
	a := &optionalReturnPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	binding := BindCard("unavailable")
	body := MayReturnNonAuraCardToBattlefield(FromCard(binding), ControllerOfWithLKI(This))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.offers != 0 {
			t.Fatal("query lost identity or made a choice")
		}
	}
	check()
	group := g.PhaseOut(source.ID())
	check()
	g.PhaseIn(group)
	g.BouncePermanentToHand(source)
	check()
	a.RemoveFromHand(source.ID())
	g.PutOnBattlefield(source.Card, a.PlayerID())
	check()
	clone := g.Clone()
	clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("clone shares duration")
	}
	reads := 0
	opaque := MayReturnNonAuraCardToBattlefield(FromCard(binding), waitingReturnPlayer{ControllerOfWithLKI(This), &reads})
	if err := ApplyEffect(g, AtNext(EachEndStep, opaque), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check()
	if reads != 0 {
		t.Fatal("query read the opaque player")
	}
}
