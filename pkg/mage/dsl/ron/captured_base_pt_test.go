package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCapturedBasePTBrineHag(t *testing.T) {
	for _, change := range []string{"none", "prevented", "phase-before", "return-before", "hag-return", "control", "counter", "clone", "phase-after", "return-after", "type-before", "type-after"} {
		t.Run(change, func(t *testing.T) {
			factory, err := Compile(hostUntapRecord(t, "LEG", "Brine Hag"))
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition("Brine Hag", factory()); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hagCard := factory()
			hagCard.SetOwner(a.PlayerID())
			hag := g.PutOnBattlefield(hagCard, a.PlayerID())
			dealerCard := mage.NewCreature("Dealer", "", 4, 5)
			dealerCard.SetOwner(b.PlayerID())
			dealer := g.PutOnBattlefield(dealerCard, b.PlayerID())
			spared := g.PutOnBattlefield(mage.NewCreature("Spared", "", 3, 5), b.PlayerID())
			if change == "prevented" {
				g.AddPreventionShield(hag.ID(), 1)
			}
			g.DealDamageToPermanent(hag, 1, dealer.ID())
			g.DestroyPermanent(hag)
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || dealer.CurrentPower(g) != 4 {
				t.Fatal("missing death trigger or early layer effect")
			}
			switch change {
			case "phase-before":
				g.PhaseOut(dealer.ID())
			case "return-before":
				g.BouncePermanentToHand(dealer)
				b.RemoveFromHand(dealerCard.ID())
				dealer = g.PutOnBattlefield(dealerCard, b.PlayerID())
			case "hag-return":
				a.RemoveFromGraveyard(hagCard.ID())
				g.PutOnBattlefield(hagCard, a.PlayerID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: spared.ID(), TargetID: dealer.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "counter":
				g.GetStack().RemoveByID(g.StackPeek().ID)
			case "type-before":
				dealer.RevokeBaseAttr(core.AttrIsCreature)
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			dealer = g.FindPermanentIncludingPhased(dealer.ID())
			changed := change != "prevented" && change != "phase-before" && change != "return-before" && change != "counter" && change != "type-before"
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{dealer.ID()})
			}
			wantPower, wantToughness := 4, 5
			if changed {
				wantPower, wantToughness = 0, 2
			}
			if dealer.CurrentPower(g) != wantPower || dealer.CurrentToughness(g) != wantToughness || spared.CurrentPower(g) != 3 {
				t.Fatalf("P/T = %d/%d, want %d/%d; unrelated %d", dealer.CurrentPower(g), dealer.CurrentToughness(g), wantPower, wantToughness, spared.CurrentPower(g))
			}
			clonedDealer := clone.FindPermanentIncludingPhased(dealer.ID())
			if clonedDealer.CurrentPower(clone) != wantPower || clonedDealer.CurrentToughness(clone) != wantToughness {
				t.Fatal("clone lost captured event facts or layer subjects")
			}
			if change == "clone" {
				clone.BouncePermanentToHand(clone.FindPermanent(dealer.ID()))
				if g.FindPermanent(dealer.ID()) == nil || g.FindPermanent(dealer.ID()).CurrentPower(g) != 0 {
					t.Fatal("clone changed original capture")
				}
			}
			if change == "phase-after" {
				g.PhaseIn(g.PhaseOut(dealer.ID()))
			}
			if change == "return-after" {
				g.BouncePermanentToHand(dealer)
				b.RemoveFromHand(dealerCard.ID())
				dealer = g.PutOnBattlefield(dealerCard, b.PlayerID())
				wantPower, wantToughness = 4, 5
			}
			if change == "type-after" {
				dealer.RevokeBaseAttr(core.AttrIsCreature)
				g.ApplyContinuousEffects()
				dealer.GrantBaseAttr(core.AttrIsCreature)
			}
			g.RunStepWithPriority(core.Cleanup)
			g.SetTurn(2)
			g.ApplyContinuousEffects()
			if dealer.CurrentPower(g) != wantPower || dealer.CurrentToughness(g) != wantToughness {
				t.Fatal("captured set or indefinite lifetime changed")
			}
		})
	}
}

func TestRONCapturedBasePTInvalid(t *testing.T) {
	for _, action := range []string{
		`CreateEachPermanentLayerEffect`, `CreateEachPermanentLayerEffect()`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[])`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[SetPT(PT(0,2)),SetPT(PT(1,3))])`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[SetPT(PT(Integer(0),2))])`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[SetPT(PT(0))])`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[AdjustPT(0,2)])`,
		`CreateEachPermanentLayerEffect(IsCardtype(Creature),[SetPT(PT(0,2))],UntilEndOfTurn)`,
		`CreateEachPermanentLayerEffect(DealtDamageToPermanentThisTurn(Trigger_ThatPermanent),[SetPT(PT(0,2))])`,
		`CreateEachPermanentLayerEffect(DealtDamageToPermanentThisTurn(ThisPermanent()),[SetPT(PT(0,2))])`,
		`CreateEachPermanentLayerEffect(DealtDamageToPermanentThisTurn(),[SetPT(PT(0,2))])`,
		`CreateEachPermanentLayerEffect(DealtDamageToPermanentThisTurn(ThisPermanent,ThisPermanent),[SetPT(PT(0,2))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+action+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid declaration", action)
		}
	}
}

func TestRONCapturedBasePTNative(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Brine Hag"`) {
			record = line
		}
	}
	files := fstest.MapFS{"LEG.ron": &fstest.MapFile{Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Brine Hag")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Brine Hag")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factories share declaration state")
	}
	g, a, _ := ronGame()
	first.SetOwner(a.PlayerID())
	a.AddToHand(first)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed casting payment changed state")
	}
	a.ManaPool().Add(core.Colorless, 2)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if hag := g.FindPermanent(first.ID()); hag == nil || hag.CurrentPower(g) != 2 || hag.CurrentToughness(g) != 2 {
		t.Fatal("strict RON factory did not enter with printed characteristics")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONCapturedBasePTDeathPaths(t *testing.T) {
	for _, path := range []string{"destroy", "sacrifice", "lethal", "batch", "exile", "regenerate", "removed-ability"} {
		t.Run(path, func(t *testing.T) {
			g, a, b := ronGame()
			hag := g.PutOnBattlefield(repositoryCard(t, "LEG", "Brine Hag"), a.PlayerID())
			hag.Card.SetOwner(a.PlayerID())
			dealer := g.PutOnBattlefield(mage.NewCreature("Dealer", "", 4, 5), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 5), a.PlayerID())
			g.DealDamageToPermanent(hag, 1, dealer.ID())
			g.DealDamageToPermanent(hag, 1, second.ID())
			switch path {
			case "sacrifice":
				if err := mage.ApplyEffect(g, mage.Sacrifice(mage.This), hag.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "lethal":
				g.CheckStateBasedActions()
			case "batch":
				if err := mage.ApplyEffect(g, mage.DestroyEach(mage.MatchingPermanents(mage.HasSubType("Hag"))), a.PlayerID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "exile":
				if err := mage.ApplyEffect(g, mage.ExileIfWouldDieThisTurn(mage.This), hag.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.DestroyPermanent(hag)
			case "regenerate":
				g.AddRegenerationShield(hag.ID())
				g.DestroyPermanent(hag)
			case "removed-ability":
				g.AddContinuousEffect(mage.RemoveAllAbilities(hag.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
				g.DestroyPermanent(hag)
			default:
				g.DestroyPermanent(hag)
			}
			g.PutTriggersOnStack()
			wantTrigger := path != "exile" && path != "regenerate" && path != "removed-ability"
			if (g.StackSize() == 1) != wantTrigger {
				t.Fatal("wrong death trigger outcome", g.StackSize())
			}
			if wantTrigger && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("trigger copy failed")
			}
			g.ResolveStack()
			dealer = g.FindPermanent(dealer.ID())
			second = g.FindPermanent(second.ID())
			if (dealer.CurrentPower(g) == 0) != wantTrigger || (second.CurrentPower(g) == 0) != wantTrigger {
				t.Fatal("death capture lost simultaneous subjects or copied facts")
			}
		})
	}
}

func TestRONCapturedBasePTSpellTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([OneOrMoreTargetPermanents(IsCardtype(Creature))],ActionList([CreateEachPermanentLayerEffect(Ref_TargetPermanents,[SetPT(PT(0,2))])])) )`)
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 4, 5), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 5), b.PlayerID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
		t.Fatal("spell copy failed")
	}
	g.PhaseOut(second.ID())
	g.ResolveStack()
	g.PhaseIn([]uuid.UUID{second.ID()})
	if first.CurrentPower(g) != 0 || first.CurrentToughness(g) != 2 || second.CurrentPower(g) != 3 {
		t.Fatal("partial target legality changed the captured membership")
	}
}

func TestRONCapturedBasePTInventory(t *testing.T) {
	n := hostUntapRecord(t, "LEG", "Brine Hag")
	entries := map[string]*InventorySignature{}
	walkInventory(n, "Brine Hag", "card", "card", "card", true, entries)
	for name, role := range map[string]string{"DealtDamageToPermanentThisTurn": "permanent_collection", "Trigger_ThatPermanent": "permanent_reference", "SetPT": "resolving_modification", "PT": "fixed_pt"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lacks %s role", name, role)
		}
	}
}

func TestRONCapturedBasePTIndependentDeaths(t *testing.T) {
	g, a, b := ronGame()
	factory, err := Compile(hostUntapRecord(t, "LEG", "Brine Hag"))
	if err != nil {
		t.Fatal(err)
	}
	hags := make([]*mage.Permanent, 2)
	dealers := make([]*mage.Permanent, 2)
	for i := range 2 {
		card := factory()
		card.SetOwner(a.PlayerID())
		hags[i] = g.PutOnBattlefield(card, a.PlayerID())
		dealers[i] = g.PutOnBattlefield(mage.NewCreature("Dealer", "", 4, 5), b.PlayerID())
		g.DealDamageToPermanent(hags[i], 1, dealers[i].ID())
	}
	if err := mage.ApplyEffect(g, mage.DestroyEach(mage.MatchingPermanents(mage.HasSubType("Hag"))), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.PutTriggersOnStack()
	clone.PutTriggersOnStack()
	if g.StackSize() != 2 || clone.StackSize() != 2 {
		t.Fatal("simultaneous deaths lost independent pending triggers")
	}
	g.ResolveTopOfStack()
	changed := 0
	for _, dealer := range dealers {
		if g.FindPermanent(dealer.ID()).CurrentPower(g) == 0 {
			changed++
		}
	}
	if changed != 1 {
		t.Fatal("one trigger changed another death's damage sources")
	}
	g.ResolveStack()
	clone.ResolveStack()
	for _, dealer := range dealers {
		if g.FindPermanent(dealer.ID()).CurrentPower(g) != 0 || clone.FindPermanent(dealer.ID()).CurrentPower(clone) != 0 {
			t.Fatal("independent captures lost a recipient")
		}
	}
}
