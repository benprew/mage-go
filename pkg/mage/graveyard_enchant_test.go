package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestGraveyardEnchantEntry(t *testing.T) {
	for _, cast := range []bool{false, true} {
		for _, ownerB := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)

			owner := a
			if ownerB {
				owner = b
			}
			host := NewCreature("Host", "", 2, 2, WithKeyword(core.Shroud), WithAbility(ProtectionFromColor(core.Black)))
			host.SetOwner(owner.PlayerID())
			owner.AddToGraveyard(host)
			aura := NewAura("Guest", "{B}", WithCastTarget(GraveyardCreatureEnchantTarget()))
			aura.SetOwner(a.PlayerID())
			if cast {
				a.AddToHand(aura)
				a.ManaPool().Add(core.Black, 1)
				if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
			} else {
				g.PutOnBattlefield(aura, a.PlayerID())
			}
			p := g.FindPermanent(aura.ID())
			if p == nil {
				t.Fatal("graveyard Aura did not enter")
			}
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
			got, ok := EnchantedGraveyardCard(This).resolveCard(ctx)
			if !ok || got.ID() != host.ID() {
				t.Fatal("missing exact graveyard host")
			}
			g.CheckStateBasedActions()
			if g.FindPermanent(aura.ID()) == nil {
				t.Fatal("legal graveyard Aura failed state check")
			}
			clone := g.Clone()
			removed, _ := owner.RemoveFromGraveyard(host.ID())
			owner.AddToGraveyard(removed)
			if _, ok := EnchantedGraveyardCard(This).resolveCard(ctx); ok {
				t.Fatal("returned card replaced saved visit")
			}
			g.CheckStateBasedActions()
			clone.CheckStateBasedActions()
			if g.FindPermanent(aura.ID()) != nil || clone.FindPermanent(aura.ID()) == nil {
				t.Fatal("host identity or clone isolation")
			}
		}
	}
}

func TestGraveyardEnchantLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	host := NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	b.AddToGraveyard(land)
	aura := NewAura("Guest", "{B}", WithCastTarget(GraveyardCreatureEnchantTarget()))
	aura.SetOwner(a.PlayerID())
	a.AddToHand(aura)
	if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err == nil || len(a.Hand()) != 1 {
		t.Fatal("unpaid cast changed hand")
	}
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{land.ID()}, 0); err == nil || !g.CanAfford(a.PlayerID(), core.ParseManaCost("{B}"), nil) {
		t.Fatal("illegal target spent mana")
	}
	if err := g.CastSpellByID(a.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	moved, _ := b.RemoveFromGraveyard(host.ID())
	b.AddToGraveyard(moved)
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(aura.ID()) != nil || clone.FindPermanent(aura.ID()) == nil || len(a.Graveyard()) != 1 {
		t.Fatal("stale spell target or pending clone")
	}
}

func TestGraveyardEnchantPhaseAndLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	aura := NewAura("Guest", "", WithCastTarget(GraveyardCreatureEnchantTarget()))
	aura.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(aura, a.PlayerID())
	visit := p.incarnationID
	g.PhaseIn(g.PhaseOut(p.ID()))
	if p.incarnationID != visit {
		t.Fatal("phase cycle changed identity")
	}
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: visit}
	g.BouncePermanentToHand(p)
	if card, ok := EnchantedGraveyardCard(This).resolveCard(ctx); !ok || card.ID() != host.ID() {
		t.Fatal("departure lost enchanted host")
	}
	other := NewCreature("Other", "", 1, 1)
	other.SetOwner(a.PlayerID())
	a.AddToGraveyard(other)
	b.RemoveFromGraveyard(host.ID())
	a.RemoveFromHand(aura.ID())
	newer := g.PutOnBattlefield(aura, a.PlayerID())
	b.AddToGraveyard(host)
	if newer.incarnationID == visit {
		t.Fatal("return kept old identity")
	}
	card, ok := EnchantedGraveyardCard(This).resolveCard(ctx)
	if ok || card != nil {
		t.Fatal("old source read a returned host or new attachment")
	}
	phase := g.PhaseOut(newer.ID())
	a.RemoveFromGraveyard(other.ID())
	g.CheckStateBasedActions()
	if g.FindPermanentIncludingPhased(newer.ID()) == nil {
		t.Fatal("phased Aura failed state check")
	}
	g.PhaseIn(phase)
	g.CheckStateBasedActions()
	if g.FindPermanent(newer.ID()) != nil {
		t.Fatal("phase-in did not check absent host")
	}
}

func TestGraveyardEnchantEntryObservations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	observations := 0
	watcher := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if event.ToZone == core.ZoneBattlefield {
			p := g.FindPermanent(event.SourceID)
			if p != nil && p.enchantedCard.id != uuid.Nil {
				observations++
				if card, ok := g.graveyardEnchantHost(p); !ok || card.ID() != host.ID() {
					t.Error("entry observed no attachment")
				}
			}
		}
		return false
	})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(watcher)), a.PlayerID())
	for range 2 {
		g.PutOnBattlefield(NewAura("Guest", "", WithCastTarget(GraveyardCreatureEnchantTarget())), a.PlayerID())
	}
	if observations != 2 {
		t.Fatal("entry observations or independent Aura visits")
	}
}

func TestGraveyardEnchantDeclarations(t *testing.T) {
	for _, card := range []Card{
		NewArtifact("Bad", "", WithCastTarget(GraveyardCreatureEnchantTarget())),
		NewAura("Bad", "", WithAbility(At(EachUpkeep, ReturnCardToHand(EnchantedGraveyardCard(nil))))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid graveyard enchant declaration accepted")
		}
	}
}

func TestGraveyardEnchantUntargetedZones(t *testing.T) {
	for _, zone := range []core.Zone{core.ZoneHand, core.ZoneGraveyard, core.ZoneExile} {
		for _, hostExists := range []bool{false, true} {
			a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			if hostExists {
				host := NewCreature("Host", "", 2, 2)
				host.SetOwner(b.PlayerID())
				b.AddToGraveyard(host)
			}
			aura := NewAura("Guest", "", WithCastTarget(GraveyardCreatureEnchantTarget()))
			aura.SetOwner(a.PlayerID())
			switch zone {
			case core.ZoneHand:
				a.AddToHand(aura)
				if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case core.ZoneGraveyard:
				a.AddToGraveyard(aura)
				binding := BindCard("guest")
				g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: aura.ID(), visit: g.graveyardVisit(aura.ID()), zone: zone}}
				if err := ReturnCardToBattlefield(FromCard(binding), You).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			case core.ZoneExile:
				g.ExileCard(aura, uuid.Nil)
				exiled := g.FindExiledCard(aura.ID())
				effect := PutExiledCardsOntoBattlefieldUnderOwnersControl(savedTransportVisits{{id: aura.ID(), visit: exiled.Visit, zone: zone}})
				if err := effect.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
			}
			if (g.FindPermanent(aura.ID()) != nil) != hostExists {
				t.Fatalf("zone=%v host=%v", zone, hostExists)
			}
			if !hostExists && g.zoneOfTarget(aura.ID()) != zone {
				t.Fatal("blocked Aura left its zone")
			}
		}
	}
}

func TestGraveyardEnchantWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := NewCreature("Host", "", 2, 2)
	host.SetOwner(b.PlayerID())
	b.AddToGraveyard(host)
	aura := NewAura("Guest", "", WithCastTarget(GraveyardCreatureEnchantTarget()))
	aura.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(aura, a.PlayerID())
	effect := AtNext(EachEndStep, ReturnCardToHand(EnchantedGraveyardCard(This)))
	if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || len(b.Graveyard()) != 1 {
			t.Fatal("waiting query changed state or lost source")
		}
	}
}

type graveyardEnchantChooser struct {
	*BasePlayer
	choose func([]Card, GameReader) Card
}

func (p *graveyardEnchantChooser) ChooseCardFromGraveyard(cards []Card, _ string, g GameReader) Card {
	return p.choose(cards, g)
}

func TestGraveyardEnchantBatchChoiceRecheck(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &graveyardEnchantChooser{BasePlayer: base}
	g := NewGame(a, b)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	for _, card := range []Card{first, second} {
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	choices := 0
	a.choose = func(_ []Card, _ GameReader) Card {
		choices++
		if choices == 1 {
			return first
		}
		card, _ := b.RemoveFromGraveyard(first.ID())
		b.AddToGraveyard(card)
		return second
	}
	saved := make(savedTransportVisits, 0, 2)
	cards := make([]Card, 0, 2)
	for range 2 {
		card := NewAura("Guest", "", WithCastTarget(GraveyardCreatureEnchantTarget()))
		card.SetOwner(a.PlayerID())
		cards = append(cards, card)
		g.ExileCard(card, uuid.Nil)
		saved = append(saved, capturedCard{id: card.ID(), visit: g.FindExiledCard(card.ID()).Visit, zone: core.ZoneExile})
	}
	if err := PutExiledCardsOntoBattlefieldUnderOwnersControl(saved).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if choices != 2 {
		t.Fatal("entry did not choose both hosts")
	}
	entered, retained := 0, 0
	for _, card := range cards {
		if p := g.FindPermanent(card.ID()); p != nil {
			entered++
			if p.enchantedCard.id != second.ID() {
				t.Fatal("stale host survived later entry choices")
			}
		} else if g.FindExiledCard(card.ID()) != nil {
			retained++
		}
	}
	if entered != 1 || retained != 1 {
		t.Fatal("batch did not keep blocked Aura in exile")
	}
}

func TestGraveyardEnchantCopyEntry(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2)
	first.SetOwner(b.PlayerID())
	b.AddToGraveyard(first)
	plan, err := NewCardDefinitionPlan(func() Card { return NewAura("Model", "", WithCastTarget(GraveyardCreatureEnchantTarget())) })
	if err != nil {
		t.Fatal(err)
	}
	donor, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	model := g.PutOnBattlefield(donor, b.PlayerID())
	second := NewCreature("Second", "", 2, 2)
	second.SetOwner(a.PlayerID())
	a.AddToGraveyard(second)
	copiedPlan, err := NewCardDefinitionPlan(func() Card {
		return NewAura("Copy", "", WithCastTarget(TargetCreature()), WithAbility(MayCopyAsEnters(Named("Model"))))
	})
	if err != nil {
		t.Fatal(err)
	}
	card, err := copiedPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(card, a.PlayerID())
	if copied == nil || copied.Name() != "Model" || copied.enchantedCard.id != second.ID() || model.enchantedCard.id != first.ID() {
		t.Fatal("entry did not use copied enchant declaration with fresh attachment")
	}
}
