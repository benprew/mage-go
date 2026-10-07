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

func TestRONTemporaryCharacteristicHalfdane(t *testing.T) {
	for _, change := range []string{"none", "target-return", "target-phase", "target-shroud", "target-phase-cycle", "source-return", "source-phase", "control", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Halfdane")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Colorless, 1)
			a.ManaPool().Add(core.White, 1)
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Black, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			targetCard := mage.NewCreature("Values", "", 4, 7)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing upkeep trigger")
			}
			if p := g.FindPermanent(card.ID()); p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 {
				t.Fatal("values changed before resolution")
			}
			target.AddCounter(core.P1P1, 2)
			switch change {
			case "target-return":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(targetCard.ID())
				g.PutOnBattlefield(targetCard, b.PlayerID())
			case "target-phase":
				g.PhaseOut(target.ID())
			case "target-shroud":
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
			case "target-phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "source-return":
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(card.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: target.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			wantP, wantT := 6, 9
			if change == "target-return" || change == "target-phase" || change == "target-shroud" || change == "source-return" || change == "source-phase" {
				wantP, wantT = 3, 3
			}
			check := func(game *mage.Game, power, toughness int) {
				t.Helper()
				p := game.FindPermanentIncludingPhased(card.ID())
				if p == nil || p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
					t.Fatalf("P/T want %d/%d, got %d/%d", power, toughness, p.CurrentPower(game), p.CurrentToughness(game))
				}
			}
			if change == "source-phase" {
				g.PhaseIn([]uuid.UUID{card.ID()})

			}
			check(g, wantP, wantT)
			if change != "source-phase" {
				check(clone, 3, 3)
			}
			clone.ResolveStack()
			if change == "source-phase" {
				clone.PhaseIn([]uuid.UUID{card.ID()})
			}
			check(clone, wantP, wantT)
			if wantP == 6 {
				g.MutablePermanent(target.ID()).AddCounter(core.P1P1, 3)
				check(g, 6, 9)
				g.MutablePermanent(card.ID()).AddCounter(core.P1P1, 1)
				check(g, 7, 10)
				g.PhaseIn(g.PhaseOut(card.ID()))
				check(g, 7, 10)
				g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Upkeep)
				check(g, 7, 10)
				g.SetActivePlayerIndex(0)
				g.RunStepWithPriority(core.Upkeep)
				check(g, 4, 4)
				check(clone, 6, 9)
			}
		})
	}
}

func TestRONTemporaryCharacteristicInvalid(t *testing.T) {
	for _, value := range []string{`PowerOfPermanent()`, `ToughnessOfPermanent(You)`, `ToughnessOfPermanent(Ref_TargetPermanent())`, `ToughnessOfPermanent(Ref_TargetPlayer)`, `Minus(Integer(1))`, `ManualPT(Integer(1))`} {
		rule := `Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[SetPT(ManualPT(` + value + `,Integer(2)))],UntilEndOfTurn)]))`
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}

func TestRONTemporaryCharacteristicValues(t *testing.T) {
	for _, expr := range []string{`PowerOfPermanent(ActionPermanent)`, `Minus(ToughnessOfPermanent(ActionPermanent),Integer(3))`, `Minus(Minus(ToughnessOfPermanent(Ref_TargetPermanent),Integer(1)),Integer(2))`} {
		g, a, b := ronGame()
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPTX(PlusX,Zero,`+expr+`),AdjustPTX(PlusX,Zero,PowerOfPermanent(ActionPermanent))],UntilEndOfTurn)])))`)
		spell := factory()
		other := factory()
		if spell.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == other.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
			t.Fatal("shared targets")
		}
		if err := mage.ValidateCardDefinition(spell.Name(), spell); err != nil {
			t.Fatal(err)
		}
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 3, 6), b.PlayerID())
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.MutablePermanent(target.ID()).AddCounter(core.P1P1, 1)
		clone := g.Clone()
		g.ResolveStack()
		p := g.FindPermanent(target.ID())
		if p.CurrentPower(g) != 12 || p.CurrentToughness(g) != 7 {
			t.Fatal("values were not captured together at resolution")
		}
		g.MutablePermanent(target.ID()).AddCounter(core.P1P1, 2)
		if p = g.FindPermanent(target.ID()); p.CurrentPower(g) != 14 {
			t.Fatal("captured values changed with later counters")
		}
		if clone.FindPermanent(target.ID()).CurrentPower(clone) != 4 {
			t.Fatal("resolution changed clone")
		}
		clone.ResolveStack()
		if clone.FindPermanent(target.ID()).CurrentPower(clone) != 12 {
			t.Fatal("clone lost values")
		}
		g.PhaseIn(g.PhaseOut(target.ID()))
		if g.FindPermanent(target.ID()).CurrentPower(g) != 14 {
			t.Fatal("phase cycle lost captured bonus")
		}
		g.RunStepWithPriority(core.Cleanup)
		if g.FindPermanent(target.ID()).CurrentPower(g) != 6 {
			t.Fatal("bonus survived cleanup")
		}
	}
}

func TestRONTemporaryCharacteristicLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEG.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		if node.Fields["Name"].String == "Halfdane" {
			files["half.ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
		}
	}
	reg, err := LoadFS(files, "*.ron")
	if err != nil || reg == nil {
		t.Fatalf("strict load: %v", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unsupported])`)}
	if reg, err = LoadFS(files, "*.ron"); err == nil || reg != nil {
		t.Fatal("invalid collection published registry")
	}
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"ManualPT": "resolution_pt", "PowerOfPermanent": "value", "ToughnessOfPermanent": "value"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name != name || signature.Role != role {
				continue
			}
			for _, example := range signature.Examples {
				if example.Card == "Halfdane" {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("missing inventory role %s=%s", name, role)
		}
	}
}

func TestRONTemporaryCharacteristicInvalidShapes(t *testing.T) {
	for _, modification := range []string{`SetPT(ManualPT)`, `SetPT(ManualPT())`, `SetPT(ManualPT(Integer(1)))`, `SetPT(ManualPT(Integer(1),Integer(2),Integer(3)))`, `SetPT(ManualPT(Integer(1),ToughnessOfPermanent(Ref_TargetPermanent)))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[` + modification + `],UntilEndOfTurn)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modification)
		}
	}
	for _, duration := range []string{`UntilEndOfNextUpkeep`, `UntilEndOfNextUpkeep()`, `UntilEndOfNextUpkeep(You())`, `UntilEndOfNextUpkeep(Opponent)`, `UntilEndOfNextUpkeep(You,You)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:3),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],` + duration + `)]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", duration)
		}
	}
}

func TestRONTemporaryCharacteristicSourceInformation(t *testing.T) {
	for _, change := range []string{"departure", "return", "phasing"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Information source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:5),Rules:[Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[SetPT(ManualPT(PowerOfPermanent(ThisPermanent),ToughnessOfPermanent(ThisPermanent)))],UntilEndOfTurn)])))])`), "information.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.AddCounter(core.P1P1, 2)
			target := g.PutOnBattlefield(mage.NewCreature("Recipient", "", 1, 8), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "phasing" {
				g.PhaseOut(source.ID())
			} else {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID()).AddCounter(core.P1P1, 9)
				}
			}
			g.ResolveStack()
			p := g.FindPermanent(target.ID())
			if p.CurrentPower(g) != 4 || p.CurrentToughness(g) != 7 {
				t.Fatal("temporary values lost original source information")
			}
		})
	}
}
