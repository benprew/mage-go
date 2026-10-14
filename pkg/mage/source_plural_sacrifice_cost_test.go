package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSourcePluralSacrificeCostBatch(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: count}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		power, cards, sourceCard := BindNumber("power"), BindCards("cards"), BindCard("source")
		source := g.PutOnBattlefield(NewArtifact("Source", "", WithStaticAbility(BoostControlledCreatures(2, 2, IsCreature))), a.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 3, 3), a.PlayerID())
		otherCard := NewCreature("Other owner", "", -1, 3)
		otherCard.SetOwner(b.PlayerID())
		second := g.PutOnBattlefield(otherCard, a.PlayerID())
		cost := SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, sourceCard)
		if !cost.CanPay(source.ID(), a.PlayerID(), g) || a.numbers != 0 {
			t.Fatal("query changed choices")
		}
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
		if err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(source.ID()) == nil || len(a.Graveyard()) != 0 {
			t.Fatal("preparation moved source")
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		if tx.paidNumbers[power] != []int{0, 5, 6}[count] || len(tx.paidCardSets[cards]) != count || a.numbers != 1 {
			t.Fatal("wrong grouped results", tx.paidNumbers[power], count)
		}
		saved := tx.paidCards[sourceCard]
		if saved.id != source.ID() || saved.visit != g.graveyardVisit(source.ID()) || saved.zone != core.ZoneGraveyard {
			t.Fatal("wrong source visit")
		}
		if count == 0 && (g.FindPermanent(first.ID()) == nil || g.FindPermanent(second.ID()) == nil) {
			t.Fatal("zero moved optional subjects")
		}
	}
}

func TestSourcePluralSacrificeCostObservations(t *testing.T) {
	a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 1}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewCreature("Source", "", 9, 9), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), a.PlayerID())
	cost := SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("power"), BindCards("cards"), BindCard("source")).(*pluralSacrificeCost)
	events := 0
	watcher := NewLand("Watcher", WithAbility(NewTriggered(core.EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		events++
		if g.FindPermanent(source.ID()) != nil || g.FindPermanent(other.ID()) != nil {
			t.Fatal("partial live batch")
		}
		if reader.FindPermanent(source.ID()) == nil || reader.FindPermanent(other.ID()) == nil {
			t.Fatal("missing simultaneous LKI")
		}
		if cost.total != 3 || len(cost.results) != 1 {
			t.Fatal("optional result includes source or facts were late")
		}
		return true
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	if err := cost.Pay(source.ID(), a.PlayerID(), g); err != nil {
		t.Fatal(err)
	}
	if events != 2 {
		t.Fatal("source was selected twice or events missing", events)
	}
}

func TestSourcePluralSacrificeCostRollback(t *testing.T) {
	for _, change := range []string{"source-phase", "source-return", "source-control", "other-phase", "other-return", "other-control", "invalid-number", "source-choice", "cancel-source", "cancel-other"} {
		t.Run(change, func(t *testing.T) {
			a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 1}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			sourceCard := NewCreature("Source", "", 9, 9)
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			otherCard := NewCreature("Other", "", 3, 3)
			other := g.PutOnBattlefield(otherCard, a.PlayerID())
			a.ManaPool().Add(core.Red, 1)
			if change == "invalid-number" {
				a.number = 2
			}
			if change == "source-choice" {
				a.invalidAt = 1
				a.invalid = source
			}
			if change == "cancel-source" {
				g.effects.AddReplacement(&collectionSacrificeReplacement{blocked: source.ID()})
			}
			if change == "cancel-other" {
				g.effects.AddReplacement(&collectionSacrificeReplacement{blocked: other.ID()})
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{ManaCostOf("{R}"), SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("power"), BindCards("cards"), BindCard("source"))}})
			switch change {
			case "invalid-number", "source-choice", "cancel-source", "cancel-other":
				if err == nil {
					t.Fatal("accepted invalid payment")
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				p, card := source, sourceCard
				if change == "other-phase" || change == "other-return" || change == "other-control" {
					p, card = other, otherCard
				}
				switch change {
				case "source-phase", "other-phase":
					g.MutablePermanent(p.ID()).PhasedOut = true
				case "source-return", "other-return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-control", "other-control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || len(tx.paidCards) != 0 || len(tx.paidCardSets) != 0 {
					t.Fatal("stale payment committed")
				}
			}
			if a.ManaPool().Count(core.Red) != 1 || g.FindPermanentIncludingPhased(source.ID()) == nil || g.FindPermanentIncludingPhased(other.ID()) == nil {
				t.Fatal("failed payment spent live resources")
			}
		})
	}
}

func TestSourcePluralSacrificeCostDefinition(t *testing.T) {
	power, cards, source := BindNumber("power"), BindCards("cards"), BindCard("source")
	for _, cost := range []Cost{
		SacrificeSourceAndAnyMatchingCostInto(PermanentFilter{}, "Sacrifice", power, cards, source),
		SacrificeSourceAndAnyMatchingCostInto(IsCreature, "", power, cards, source),
		SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", nil, cards, source),
		SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, nil, source),
		SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, nil),
		SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, BindCard("")),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	cost := SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, source)
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(cost), GainLifeFor(You, FromNumber(power)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewArtifact("Resolving", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(cost), GainLifeFor(You, FromNumber(power))))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("resolution cost accepted")
	}
}

func TestSourcePluralSacrificeCostReplacementAndTokens(t *testing.T) {
	for _, tokenSource := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 2}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetActivePlayerIndex(1)
			power, cards, paid := BindNumber("power"), BindCards("cards"), BindCard("source")
			sourceCard := NewArtifact("Source", "", WithKeyword(core.Indestructible))
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			g.MutablePermanent(source.ID()).IsToken = tokenSource
			first := g.PutOnBattlefield(NewCreature("First", "", 3, 3, WithKeyword(core.Indestructible)), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Token", "", 4, 4), a.PlayerID())
			g.MutablePermanent(second.ID()).IsToken = true
			g.AddRegenerationShield(first.ID())
			if replace {
				g.AddReplacementEffect(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: source.ID(), duration: core.Indefinite}})
			}
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, paid)}})
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if tx.paidNumbers[power] != 7 || len(tx.paidCardSets[cards]) != 1 {
				t.Fatal("token, regeneration or indestructible changed results")
			}
			saved := tx.paidCards[paid]
			if tokenSource {
				if saved.id != uuid.Nil {
					t.Fatal("token source saved a card")
				}
			} else if replace {
				if saved.zone != core.ZoneExile || saved.visit != g.FindExiledCard(source.ID()).Visit {
					t.Fatal("wrong replacement source visit")
				}
			} else if saved.zone != core.ZoneGraveyard || len(b.Graveyard()) != 1 {
				t.Fatal("wrong source owner destination")
			}
			if replace && tx.paidCardSets[cards][0].zone != core.ZoneExile {
				t.Fatal("wrong optional replacement destination")
			}
		}
	}
}

func TestSourcePluralSacrificeCostFrames(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		a := &pluralSacrificeChooser{countSacrificeChooser: &countSacrificeChooser{BasePlayer: NewBasePlayer("A")}, number: 1}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		power, cards, paid := BindNumber("power"), BindCards("cards"), BindCard("source")
		body := Sequence(GainLifeFor(You, FromNumber(power)), ExileGraveyardCard(FromCard(paid)))
		if delayed {
			body = AtNext(EachEndStep, body)
		}
		source := NewArtifact("Source", "", WithAbility(Activate(Costs(SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", power, cards, paid)), body)))
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		g.PutOnBattlefield(source, a.PlayerID())
		victim := NewCreature("Victim", "", 5, 5)
		g.PutOnBattlefield(victim, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		original := g.StackPeek()
		if len(original.capturedCardSets[cards]) != 1 || original.capturedCards[paid].id != source.ID() {
			t.Fatal("missing distinct stack results")
		}
		if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		clone := g.Clone()
		a.RemoveFromGraveyard(source.ID())
		g.PutOnBattlefield(source, a.PlayerID())
		a.number = 0
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackPeek().capturedNumbers[power] != 0 {
			t.Fatal("new visit inherited old result")
		}
		a.RemoveFromGraveyard(victim.ID())
		g.PutOnBattlefield(victim, a.PlayerID())
		g.MutablePermanent(victim.ID()).AddCounter(core.P1P1, 9)
		for _, game := range []*Game{g, clone} {
			game.ResolveStack()
			if delayed {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
			}
			if game.GetPlayer(a.PlayerID()).Life() != 25 || game.GetPlayer(b.PlayerID()).Life() != 25 {
				t.Fatal("saved power changed after return, new payment or copy")
			}
			if game.FindExiledCard(source.ID()) == nil {
				t.Fatal("new source result failed or clone lost old visit")
			}
		}
	}
}

func TestSourcePluralSacrificeCostDuplicateSourceResult(t *testing.T) {
	saved := BindCard("source")
	for _, first := range []Cost{SacrificeSourceCostWithCard(saved), SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("first power"), BindCards("first cards"), saved)} {
		second := SacrificeSourceAndAnyMatchingCostInto(IsCreature, "Sacrifice", BindNumber("second power"), BindCards("second cards"), saved)
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(first, second), GainLifeFor(You, Fixed(1)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("duplicate source result accepted")
		}
	}
}
