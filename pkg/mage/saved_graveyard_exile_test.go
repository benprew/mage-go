package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedGraveyardExileBatch(t *testing.T) {
	g, a, b := newObjectColorGame()
	source, group := BindCard("source"), BindCards("group")
	first, second, returned := NewArtifact("Source", ""), NewCreature("Other owner", "", 3, 3), NewLand("Returned")
	first.SetOwner(a.PlayerID())
	a.AddToGraveyard(first)
	for _, c := range []Card{second, returned} {
		c.SetOwner(b.PlayerID())
		b.AddToGraveyard(c)
	}
	g.resolution.cardBindings = map[*CardBinding]capturedCard{source: {id: first.ID(), visit: g.graveyardVisit(first.ID()), zone: core.ZoneGraveyard}}
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{group: {
		{id: second.ID(), visit: g.graveyardVisit(second.ID()), zone: core.ZoneGraveyard},
		{id: second.ID(), visit: g.graveyardVisit(second.ID()), zone: core.ZoneGraveyard},
		{id: returned.ID(), visit: g.graveyardVisit(returned.ID()), zone: core.ZoneGraveyard},
	}}
	b.RemoveFromGraveyard(returned.ID())
	b.AddToGraveyard(returned)
	zones, batches := 0, 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneExile {
			zones++
			if g.FindExiledCard(first.ID()) == nil || g.FindExiledCard(second.ID()) == nil {
				t.Fatal("partial batch observed")
			}
		}
		return false
	})
	batch := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		batches++
		if e.Amount != 1 {
			t.Fatal("wrong owner count", e.Amount)
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher), WithAbility(batch)), a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: uuid.New()}
	effect := ExileGraveyardCards(SingletonCard(FromCard(source)), FromCards(group))
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if zones != 2 || batches != 2 || len(b.Graveyard()) != 1 || b.Graveyard()[0].ID() != returned.ID() {
		t.Fatal("wrong saved selection", zones, batches)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if zones != 2 {
		t.Fatal("repeated move")
	}
}

func TestSavedGraveyardExileDefinition(t *testing.T) {
	for _, effect := range []Effect{
		ExileGraveyardCards(), ExileGraveyardCards(nil),
		ExileGraveyardCards(SingletonCard(nil)),
		ExileGraveyardCards(FromCards(BindCards("unbound"))),
		ExileGraveyardCards(SingletonCard(FromCard(BindCard("unbound")))),
	} {
		c := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSavedGraveyardExilePaymentReplacement(t *testing.T) {
	for _, replace := range []bool{false, true} {
		a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 2}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		power, cards, paid := BindNumber("power"), BindCards("cards"), BindCard("source")
		card := NewArtifact("Source", "", WithStaticAbility(BoostControlledCreatures(2, 2, IsCreature)), WithAbility(Activate(Costs(SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, paid)), Sequence(GainLifeFor(You, FromNumber(power)), ExileGraveyardCards(SingletonCard(FromCard(paid)), FromCards(cards))))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		source := g.PutOnBattlefield(card, a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Paid", "", 3, 3), a.PlayerID())
		token := g.PutOnBattlefield(NewCreature("Token", "", 4, 4), a.PlayerID())
		g.MutablePermanent(token.ID()).IsToken = true
		if replace {
			g.AddReplacementEffect(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: source.ID(), duration: core.Indefinite}})
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
			t.Fatal(err)
		}
		var sourceVisit, victimVisit uuid.UUID
		if replace {
			sourceVisit = g.FindExiledCard(source.ID()).Visit
			victimVisit = g.FindExiledCard(victim.ID()).Visit
		}
		g.ResolveStack()
		if a.Life() != 31 || len(g.GetExile()) != 2 {
			t.Fatal("lost payment-time bonus or token power", a.Life(), len(g.GetExile()))
		}
		if replace && (g.FindExiledCard(source.ID()).Visit != sourceVisit || g.FindExiledCard(victim.ID()).Visit != victimVisit) {
			t.Fatal("exile replacement visits moved again")
		}
	}
}
