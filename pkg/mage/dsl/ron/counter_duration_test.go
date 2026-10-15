package ron

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCounterDurationTomb(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := chosenDiscardRecord(t, set, "Cyclopean Tomb")
			record.Fields = maps.Clone(record.Fields)
			rules := record.Fields["Rules"]
			rules.Args = rules.Args[:1]
			record.Fields["Rules"] = rules
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			source := g.PutOnBattlefield(card, a.PlayerID())
			land := g.PutOnBattlefield(mage.NewLand("Forest", mage.WithSubTypes("Forest")), b.PlayerID())
			g.SetActivePlayerIndex(0)
			g.SetStep(core.Upkeep)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			live := g.FindPermanent(land.ID())
			if live.Counters[core.Mire] != 1 || !live.HasSubType("Swamp") || live.HasSubType("Forest") {
				t.Fatal("activation did not set Swamp")
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ApplyContinuousEffects()
			if !g.FindPermanent(land.ID()).HasSubType("Swamp") {
				t.Fatal("source departure ended duration")
			}
			g.MutablePermanent(land.ID()).RemoveCounter(core.Mire, 1)
			g.MutablePermanent(land.ID()).AddCounter(core.Mire, 1)
			g.ApplyContinuousEffects()
			if !g.FindPermanent(land.ID()).HasSubType("Forest") {
				t.Fatal("counter replacement restarted duration")
			}
		})
	}
}

func TestRONCounterDurationInvalid(t *testing.T) {
	good := `ForAsLongAsPermanentHasACounterOfTypeOnIt(EachablePermanent,MireCounter)`
	for _, duration := range []string{`ForAsLongAsPermanentHasACounterOfTypeOnIt`, `ForAsLongAsPermanentHasACounterOfTypeOnIt()`, strings.Replace(good, `EachablePermanent`, `ThisPermanent`, 1), strings.Replace(good, `EachablePermanent`, `EachablePermanent()`, 1), strings.Replace(good, `MireCounter`, `MireCounter()`, 1), strings.Replace(good, `MireCounter`, `NoCounter`, 1), strings.Replace(good, `MireCounter`, `MireCounter,You`, 1)} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],`+duration+`)]))`)
	}
	assertConditionalRejected(t, `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(EachablePermanent,[AdjustPT(1,1)],`+good+`)]))`)
}

func TestRONCounterDurationTiming(t *testing.T) {
	for _, change := range []string{"none", "unpaid", "illegal", "returned", "phased", "copy", "pending-clone", "source-gone"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(And([PayMana([ManaCostGeneric(2)]),TapPermanent(ThisPermanent)]),Targeted([TargetPermanent(IsCardtype(Land))],ActionList([PutCounters([ACounterOfTypeOnPermanent(MireCounter,Ref_TargetPermanent)]),CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SetLandType(Swamp)],ForAsLongAsPermanentHasACounterOfTypeOnIt(EachablePermanent,MireCounter))])))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			landCard := mage.NewLand("Forest", mage.WithSubTypes("Forest"))
			land := g.PutOnBattlefield(landCard, b.PlayerID())
			mana := 2
			if change == "unpaid" {
				mana = 1
			}
			a.ManaPool().Add(core.Colorless, mana)
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{land.ID()})
			if change == "unpaid" {
				if err == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 1 || !g.GetStack().IsEmpty() {
					t.Fatal("unpaid activation did not roll back")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if land.Counters[core.Mire] != 0 || land.HasSubType("Swamp") {
				t.Fatal("action ran during payment")
			}
			clone := g.Clone()
			switch change {
			case "illegal":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), land.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "returned":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				g.PutOnBattlefield(landCard, b.PlayerID())
			case "phased":
				g.PhaseOut(land.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "source-gone":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			}
			g.ResolveStack()
			live := g.FindPermanentIncludingPhased(land.ID())
			active := change != "illegal" && change != "returned" && change != "phased"
			count := 0
			if active {
				count = 1
			}
			if change == "copy" {
				count = 2
			}
			if live.Counters[core.Mire] != count || live.HasSubType("Swamp") != active {
				t.Fatal("wrong resolution result", live.Counters[core.Mire], live.HasSubType("Swamp"))
			}
			clone.ResolveStack()
			if clone.FindPermanent(land.ID()).Counters[core.Mire] != 1 || !clone.FindPermanent(land.ID()).HasSubType("Swamp") {
				t.Fatal("pending clone shared result")
			}
		})
	}
}

func TestRONCounterDurationBoundaries(t *testing.T) {
	good := `ForAsLongAsPermanentHasACounterOfTypeOnIt(EachablePermanent,MireCounter)`
	for _, modification := range []string{`SetLandType(Swamp())`, `SetLandType(Angel)`, `SetLandType()`, `SetLandType(Swamp,Forest)`} {
		assertConditionalRejected(t, `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[`+modification+`],`+good+`)]))`)
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],` + good + `)])`,
		`Activated(CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],` + good + `),ActionList([]))`,
		`PermanentLayerEffect(ThisPermanent,[` + good + `])`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONCounterDurationInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],ForAsLongAsPermanentHasACounterOfTypeOnIt(EachablePermanent,MireCounter))]))])`
	report, err := InspectFS(fstest.MapFS{"duration.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, signature := range report.Signatures {
		if !strings.Contains(signature.Context, "ForAsLongAsPermanentHasACounterOfTypeOnIt") {
			continue
		}
		want, ok := map[string]string{"EachablePermanent": "permanent_reference", "MireCounter": "permanent_counter_type"}[signature.Name]
		if ok {
			if signature.Role != want {
				t.Fatalf("%s role=%s, want %s", signature.Name, signature.Role, want)
			}
			found[signature.Name] = true
		}
	}
	if len(found) != 2 {
		t.Fatal("missing duration operands", found)
	}
}
