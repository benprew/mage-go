package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type pluralSacrificeChooser struct {
	*countSacrificeChooser
	number  int
	numbers int
}

func (p *pluralSacrificeChooser) ChooseNumber(_, _ int, _ string) int {
	p.numbers++
	return p.number
}

func TestPluralSacrificeCostResults(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: count}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		total, cards := BindNumber("total"), BindCards("cards")
		source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 4, 5), a.PlayerID())
		secondCard := NewCreature("Second", "", -1, 5)
		secondCard.SetOwner(b.PlayerID())
		second := g.PutOnBattlefield(secondCard, a.PlayerID())
		a.ManaPool().Add(core.Red, 1)
		cost := SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice creatures", total, cards)
		if !cost.CanPay(source.ID(), a.PlayerID(), g) || a.numbers != 0 {
			t.Fatal("query chose")
		}
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{ManaCostOf("{R}"), cost}})
		if err != nil {
			t.Fatal(err)
		}
		if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
			t.Fatal("preparation moved cards")
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		want := []int{0, 4, 3}[count]
		if tx.paidNumbers[total] != want || len(tx.paidCardSets[cards]) != count || a.numbers != 1 {
			t.Fatalf("total=%d cards=%d choices=%d", tx.paidNumbers[total], len(tx.paidCardSets[cards]), a.numbers)
		}
		for _, saved := range tx.paidCardSets[cards] {
			if saved.zone != core.ZoneGraveyard || saved.visit != g.graveyardVisit(saved.id) {
				t.Fatal("wrong saved visit")
			}
		}
		if count == 0 && (g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil) {
			t.Fatal("zero moved subjects")
		}
	}
}

func TestPluralSacrificeCostRollback(t *testing.T) {
	for _, change := range []string{"phase", "return", "control", "invalid-number", "invalid-subject"} {
		t.Run(change, func(t *testing.T) {
			a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 1}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Victim", "", 3, 3)
			victim := g.PutOnBattlefield(card, a.PlayerID())
			total, cards := BindNumber("total"), BindCards("cards")
			a.ManaPool().Add(core.Red, 1)
			if change == "invalid-number" {
				a.number = 2
			}
			if change == "invalid-subject" {
				a.invalidAt = 1
				a.invalid = source
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{ManaCostOf("{R}"), SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)}})
			if change == "invalid-number" || change == "invalid-subject" {
				if err == nil {
					t.Fatal("accepted invalid choice")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				switch change {
				case "phase":
					g.MutablePermanent(victim.ID()).PhasedOut = true
				case "return":
					g.RemoveFromBattlefield(victim)
					g.PutOnBattlefield(card, a.PlayerID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || len(tx.paidCardSets) != 0 {
					t.Fatal("stale choice paid")
				}
			}
			if a.ManaPool().Count(core.Red) != 1 || g.FindPermanentIncludingPhased(victim.ID()) == nil {
				t.Fatal("failed payment spent resources")
			}
		})
	}
}

func TestPluralSacrificeCostDefinition(t *testing.T) {
	total, cards := BindNumber("total"), BindCards("cards")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)), GainLifeFor(You, FromNumber(total)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, cost := range []Cost{
		SacrificeAnyMatchingCostInto(PermanentFilter{}, "Sacrifice", total, cards),
		SacrificeAnyMatchingCostInto(IsCreature, "", total, cards),
		SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", nil, cards),
		SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, nil),
		SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber(""), cards),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid cost accepted")
		}
	}
	for _, card := range []Card{
		NewInstant("Additional", "", WithAbility(Spell(GainLifeFor(You, FromNumber(total)))), WithAdditionalCost(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards))),
		NewArtifact("Resolving", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)), GainLifeFor(You, FromNumber(total)))))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards), SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)), GainLifeFor(You, FromNumber(total)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("duplicate producers accepted")
	}
}

func TestPluralSacrificeCostStackFrames(t *testing.T) {
	a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 2}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	total, cards := BindNumber("total"), BindCards("cards")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)), GainLifeFor(You, FromNumber(total)))))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 2, 3)
	second := NewCreature("Second", "", 3, 4)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.MutablePermanent(second.ID()).AddCounter(core.P1P1, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackObjects()[0]
	if original.capturedNumbers[total] != 6 || len(original.capturedCardSets[cards]) != 2 {
		t.Fatal("missing stack facts")
	}
	g.CopyStackObjectDirect(original, b.PlayerID(), false)
	a.number = 0
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	zero := g.StackObjects()[2]
	if zero.capturedNumbers[total] != 0 || len(zero.capturedCardSets[cards]) != 0 {
		t.Fatal("zero inherited old results")
	}
	clone := g.Clone()
	a.RemoveFromGraveyard(first.ID())
	g.PutOnBattlefield(first, a.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardsBindings = original.capturedCardSets
	if len(FromCards(cards).resolveCards(ctx)) != 1 {
		t.Fatal("saved cards followed return")
	}
	g.resolution.cardsBindings = nil
	clone.resolution.cardsBindings = clone.StackObjects()[0].capturedCardSets
	if len(FromCards(cards).resolveCards(&EffectContext{Game: clone, Controller: a.PlayerID()})) != 2 {
		t.Fatal("clone lost card visits")
	}
	clone.resolution.cardsBindings = nil
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed live players")
	}
	g.ResolveStack()
	if a.Life() != 26 || b.Life() != 26 || clone.GetPlayer(a.PlayerID()).Life() != 26 || a.numbers != 2 {
		t.Fatal("copy or later activation changed historical power")
	}
}

func TestPluralSacrificeCostReplacementBatch(t *testing.T) {
	for _, replace := range []bool{false, true} {
		a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 3}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		total, cards := BindNumber("total"), BindCards("cards")
		source := g.PutOnBattlefield(NewCreature("Source", "", 2, 3), a.PlayerID())
		owned := NewCreature("Other owner", "", 3, 3, WithKeyword(core.Indestructible))
		owned.SetOwner(b.PlayerID())
		other := g.PutOnBattlefield(owned, a.PlayerID())
		token := g.PutOnBattlefield(NewCreature("Token", "", 4, 4), a.PlayerID())
		g.MutablePermanent(token.ID()).IsToken = true
		g.AddRegenerationShield(other.ID())
		if replace {
			g.effects.AddReplacement(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: source.ID()}})
		}
		cost := SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards).(*pluralSacrificeCost)
		observed := 0
		watcher := NewLand("Watcher", WithAbility(NewTriggered(core.EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			observed++
			if cost.total != 9 || len(cost.results) != 2 {
				t.Fatal("event ran before saved facts")
			}
			for _, p := range []*Permanent{source, other, token} {
				if g.FindPermanent(p.ID()) != nil || reader.FindPermanent(p.ID()) == nil {
					t.Fatal("event saw partial batch or lost departure facts")
				}
			}
			return true
		})))
		g.PutOnBattlefield(watcher, a.PlayerID())
		if err := cost.Pay(source.ID(), a.PlayerID(), g); err != nil {
			t.Fatal(err)
		}
		if cost.total != 9 || len(cost.results) != 2 || observed != 3 {
			t.Fatalf("power=%d cards=%d events=%d", cost.total, len(cost.results), observed)
		}
		for _, saved := range cost.results {
			if replace && (saved.zone != core.ZoneExile || g.FindExiledCard(saved.id).Visit != saved.visit) {
				t.Fatal("wrong replaced visit")
			}
			if !replace && saved.zone != core.ZoneGraveyard {
				t.Fatal("wrong graveyard visit")
			}
		}
		if !replace && (len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1) {
			t.Fatal("owner destinations")
		}
	}
}

func TestPluralSacrificeCostCancellation(t *testing.T) {
	a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 2}
	g := NewGame(a, NewBasePlayer("B"))
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 3, 3), a.PlayerID())
	g.effects.AddReplacement(&collectionSacrificeReplacement{blocked: second.ID()})
	a.ManaPool().Add(core.Red, 1)
	_, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{ManaCostOf("{R}"), SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("total"), BindCards("cards"))}})
	if err == nil || g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil || a.ManaPool().Count(core.Red) != 1 {
		t.Fatal("canceled payment changed live resources")
	}
}

func TestPluralSacrificeCostDelayedAndFailureIsolation(t *testing.T) {
	a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 1}
	g := NewGame(a, NewBasePlayer("B"))
	total, cards := BindNumber("total"), BindCards("cards")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", total, cards)), AtNext(EachEndStep, GainLifeFor(You, FromNumber(total))))))
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 5, 5)
	g.PutOnBattlefield(victim, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || g.StackSize() != 1 {
		t.Fatal("invalid later payment changed pending body")
	}
	g.ResolveStack()
	a.RemoveFromGraveyard(victim.ID())
	g.PutOnBattlefield(victim, a.PlayerID())
	g.MutablePermanent(victim.ID()).AddCounter(core.P1P1, 9)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if a.Life() != 20 {
		t.Fatal("delay resolved early")
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 25 {
			t.Fatal("delay lost historical payment")
		}
	}
}

func TestPluralSacrificeCostPowerAfterReplacementChoices(t *testing.T) {
	a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 2}
	g := NewGame(a, NewBasePlayer("B"))
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 3), a.PlayerID())
	g.effects.AddReplacement(&collectionSacrificeReplacement{inspect: func(Action) {
		for _, p := range []*Permanent{first, second} {
			g.MutablePermanent(p.ID()).AddCounter(core.P1P1, 1)
		}
	}})
	cost := SacrificeAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("power"), BindCards("cards")).(*pluralSacrificeCost)
	if err := cost.Pay(first.ID(), a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if cost.total != 7 {
		t.Fatal("power did not use the complete pre-movement battlefield", cost.total)
	}
}
