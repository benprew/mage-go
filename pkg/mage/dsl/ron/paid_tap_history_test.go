package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPaidTapHistoryWarMachine(t *testing.T) {
	g, a, _ := ronGame()
	sourceCard := repositoryCard(t, "FEM", "Vodalian War Machine")
	sourceCard.SetOwner(a.PlayerID())
	a.AddToHand(sourceCard)
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), sourceCard.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 {
		t.Fatal("cast rollback")
	}
	a.ManaPool().Add(core.Blue, 1)
	if err := g.CastSpellByID(a.PlayerID(), sourceCard.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(sourceCard.ID())
	payer := g.PutOnBattlefield(mage.NewCreature("Payer", "{U}", 1, 3, mage.WithSubTypes("Merfolk")), a.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other", "{U}", 1, 3, mage.WithSubTypes("Merfolk")), a.PlayerID())
	payer.Card.SetOwner(a.PlayerID())
	other.Card.SetOwner(a.PlayerID())
	g.TapPermanent(other)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source = g.MutablePermanent(source.ID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if !source.CanDeclareAsAttacker(g) || !source.HasKeyword(core.Defender) {
		t.Fatal("defender permission")
	}
	g.UntapPermanent(g.FindPermanent(payer.ID()))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if source.CurrentPower(g) != 2 || source.CurrentToughness(g) != 5 {
		t.Fatal("pump missing")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		if game.FindPermanent(payer.ID()) != nil || game.FindPermanent(other.ID()) == nil {
			t.Fatal("death history membership")
		}
	}
}

func TestRONPaidTapHistoryLifetime(t *testing.T) {
	for _, change := range []string{"untap", "payer-control", "payer-phase", "payer-phase-cycle", "payer-return", "source-return", "source-control", "source-phase-cycle", "cleanup", "lose-type", "indestructible", "regenerate", "copy", "another-source", "new-merfolk", "source-ability-removal", "source-return-after-death", "payer-return-after-death", "type-after-death", "activation-copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Vodalian War Machine")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			payerCard := mage.NewCreature("Payer", "{U}", 1, 3, mage.WithSubTypes("Merfolk"))
			payerCard.SetOwner(a.PlayerID())
			payer := g.PutOnBattlefield(payerCard, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if change == "activation-copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("activation copy")
				}
			}
			g.ResolveStack()
			switch change {
			case "untap":
				g.UntapPermanent(g.FindPermanent(payer.ID()))
			case "payer-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: payer.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "payer-phase":
				g.PhaseOut(payer.ID())
			case "payer-phase-cycle":
				g.PhaseIn(g.PhaseOut(payer.ID()))
			case "payer-return":
				g.BouncePermanentToHand(g.FindPermanent(payer.ID()))
				a.RemoveFromHand(payerCard.ID())
				g.PutOnBattlefield(payerCard, a.PlayerID())
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "cleanup":
				g.RunStepWithPriority(core.Cleanup)
			case "lose-type":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Goblin")), payer.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "indestructible":
				g.MutablePermanent(payer.ID()).GrantBaseAttr(core.Indestructible)
			case "regenerate":
				g.AddRegenerationShield(payer.ID())
			case "another-source":
				secondCard := repositoryCard(t, "FEM", "Vodalian War Machine")
				secondCard.SetOwner(a.PlayerID())
				second := g.PutOnBattlefield(secondCard, a.PlayerID())
				g.DestroyPermanent(g.FindPermanent(second.ID()))
				g.ResolveStack()
				if g.FindPermanent(payer.ID()) == nil {
					t.Fatal("another source inherited history")
				}
			case "source-ability-removal":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "new-merfolk":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Goblin")), payer.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
			}
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			g.PutTriggersOnStack()
			switch change {
			case "source-return-after-death":
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "payer-return-after-death":
				g.BouncePermanentToHand(g.FindPermanent(payer.ID()))
				a.RemoveFromHand(payerCard.ID())
				g.PutOnBattlefield(payerCard, a.PlayerID())
			case "type-after-death":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetSubtypeFamily(mage.This, mage.SubtypeCreature, "Goblin")), payer.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("trigger copy")
				}
			}
			if change == "source-return" {
				if g.StackSize() == 0 {
					t.Fatal("new source death missing")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				survives := change == "payer-phase" || change == "payer-return" || change == "source-return" || change == "cleanup" || change == "lose-type" || change == "indestructible" || change == "regenerate" || change == "source-ability-removal" || change == "payer-return-after-death" || change == "type-after-death"
				if (game.FindPermanentIncludingPhased(payer.ID()) != nil) != survives {
					t.Fatal("wrong historical membership", change)
				}
			}
		})
	}
}

func TestRONPaidTapHistoryInvalid(t *testing.T) {
	for i, ref := range []string{`WasTappedToPayForAbilitiesOfPermanent`, `WasTappedToPayForAbilitiesOfPermanent()`, `WasTappedToPayForAbilitiesOfPermanent(ThisPermanent,ThisPermanent)`, `WasTappedToPayForAbilitiesOfPermanent(You)`, `WasTappedToPayForAbilitiesOfPermanent(Trigger_ThatPermanent)`, `WasTappedToPayForAbilitiesOfPermanent(ThisPermanent())`} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([DestroyEachPermanent(` + ref + `)]))])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid history declaration accepted", ref)
			}
		})
	}
	for _, rule := range []string{
		`EachPermanentLayerEffect(WasTappedToPayForAbilitiesOfPermanent(ThisPermanent),[AdjustPT(1,1)])`,
		`Activated(TapAPermanent(WasTappedToPayForAbilitiesOfPermanent(ThisPermanent)),ActionList([GainLife(Integer(1))]))`,
		`AsPermanentEnters(ThisPermanent,[DestroyEachPermanent(WasTappedToPayForAbilitiesOfPermanent(ThisPermanent))])`,
		`Activated(PayMana([]),Targeted([TargetPermanent(WasTappedToPayForAbilitiesOfPermanent(ThisPermanent))],ActionList([TapPermanent(Ref_TargetPermanent)])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported context accepted", rule)
		}
	}
}

func TestRONPaidTapHistoryInventory(t *testing.T) {
	node := chosenDiscardRecord(t, "FEM", "Vodalian War Machine")
	entries := map[string]*InventorySignature{}
	walkInventory(node, "Vodalian War Machine", "Card", "card", "collection", false, entries)
	found := false
	for _, entry := range entries {
		if entry.Name == "Trigger_ThatPermanent" && strings.Contains(entry.Context, "WasTappedToPayForAbilitiesOfPermanent") {
			found = true
			if entry.Role != "permanent_reference" {
				t.Fatal(entry)
			}
		}
	}
	if !found {
		t.Fatal("missing typed history source")
	}
}

func TestRONPaidTapHistoryMultiplePayers(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "FEM", "Vodalian War Machine")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	firstCard := mage.NewCreature("First", "{U}", 1, 3, mage.WithSubTypes("Merfolk"))
	firstCard.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(firstCard, a.PlayerID())
	secondCard := mage.NewCreature("Second", "{U}", 1, 3, mage.WithSubTypes("Merfolk"))
	secondCard.SetOwner(a.PlayerID())
	second := g.PutOnBattlefield(secondCard, a.PlayerID())
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("distinct payers not selected")
	}
	if g.FindPermanent(source.ID()).CurrentPower(g) != 4 {
		t.Fatal("two activations did not resolve")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("death did not destroy all payers")
	}
}
