package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONSourceTappedDurationCards(t *testing.T) {
	for _, tc := range []struct {
		set, name              string
		cost, power, toughness int
	}{
		{"ATQ", "Tawnos's Weaponry", 2, 2, 2}, {"FEM", "Spirit Shield", 2, 1, 3}, {"FEM", "Zelyon Sword", 3, 3, 1},
	} {
		for _, change := range []string{"none", "untap-before", "retap-before", "return-before", "phase-before", "control", "illegal", "target-return", "target-phase", "target-phase-in", "untap-after", "phase-after", "return-after", "target-phase-after", "copy"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				sourceCard := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(tc.name, sourceCard); err != nil {
					t.Fatal(err)
				}
				sourceCard.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(sourceCard, a.PlayerID())
				targetCard := repositoryCard(t, "LEB", "Llanowar Elves")
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				land := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forest"), b.PlayerID())
				a.ManaPool().Add(core.Colorless, tc.cost-1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				if g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != tc.cost-1 || !g.GetStack().IsEmpty() {
					t.Fatal("cost did not roll back")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{land.ID()}); err == nil {
					t.Fatal("land target accepted")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || target.CurrentPower(g) != 1 || target.CurrentToughness(g) != 1 {
					t.Fatal("announcement timing or payment")
				}
				switch change {
				case "untap-before", "retap-before":
					g.UntapPermanent(g.FindPermanent(source.ID()))
					if change == "retap-before" {
						g.TapPermanent(g.FindPermanent(source.ID()))
					}
				case "return-before":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					b.RemoveFromHand(sourceCard.ID())
					g.TapPermanent(g.PutOnBattlefield(sourceCard, a.PlayerID()))
				case "phase-before":
					g.PhaseOut(source.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "illegal":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Shroud)), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "target-return":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(targetCard.ID())
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
				case "target-phase":
					g.PhaseOut(target.ID())
				case "target-phase-in":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("ability copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				active := change != "untap-before" && change != "return-before" && change != "phase-before" && change != "illegal" && change != "target-return" && change != "target-phase"
				check := func(game *mage.Game, active bool, copies int) {
					t.Helper()
					live := game.FindPermanentIncludingPhased(target.ID())
					power, toughness := 1, 1
					if active {
						power += (tc.power - 1) * copies
						toughness += (tc.toughness - 1) * copies
					}
					if live.CurrentPower(game) != power || live.CurrentToughness(game) != toughness {
						t.Fatalf("P/T %d/%d, want %d/%d", live.CurrentPower(game), live.CurrentToughness(game), power, toughness)
					}
				}
				copies := 1
				if change == "copy" {
					copies = 2
				}
				check(g, active, copies)
				check(clone, false, copies)
				clone.ResolveStack()
				check(clone, active, copies)
				switch change {
				case "untap-after":
					g.UntapPermanent(g.FindPermanent(source.ID()))
					g.TapPermanent(g.FindPermanent(source.ID()))
					active = false
				case "phase-after":
					g.PhaseIn(g.PhaseOut(source.ID()))
					active = false
				case "return-after":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					b.RemoveFromHand(sourceCard.ID())
					g.TapPermanent(g.PutOnBattlefield(sourceCard, a.PlayerID()))
					active = false
				case "target-phase-after":
					g.PhaseIn(g.PhaseOut(target.ID()))
				}
				g.PhaseIn([]uuid.UUID{target.ID(), source.ID()})
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				check(g, active, copies)
				if change == "untap-after" || change == "phase-after" || change == "return-after" {
					check(clone, true, copies)
				}
			})
		}
	}
}

func TestRONSourceTappedDurationUntapChoiceAndInstances(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		cost      int
	}{{"ATQ", "Tawnos's Weaponry", 2}, {"FEM", "Spirit Shield", 2}, {"FEM", "Zelyon Sword", 3}} {
		for _, change := range []string{"decline", "accept", "gone-target", "control", "stun", "remove-ability"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}, untap: false}
				b := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("B")}, untap: true}
				g := mage.NewGame(a, b)
				first := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				second := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
				a.ManaPool().Add(core.Colorless, tc.cost*2)
				for _, source := range []*mage.Permanent{first, second} {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				initialP, initialT := g.FindPermanent(target.ID()).CurrentPower(g), g.FindPermanent(target.ID()).CurrentToughness(g)
				if initialP+initialT <= 2 {
					t.Fatal("instances did not add bonuses")
				}
				g.UntapPermanent(g.FindPermanent(second.ID()))
				g.ApplyContinuousEffects()
				if g.FindPermanent(target.ID()).CurrentPower(g)+g.FindPermanent(target.ID()).CurrentToughness(g) != (initialP+initialT+2)/2 {
					t.Fatal("instances shared a lifetime")
				}
				switch change {
				case "accept":
					a.untap = true
				case "gone-target":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.SetActivePlayerIndex(1)
				case "stun":
					a.untap = true
					g.MutablePermanent(first.ID()).AddCounter(core.Stun, 1)
				case "remove-ability":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), first.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.RunStepWithPriority(core.Untap)
				g.ApplyContinuousEffects()
				wantTapped := change == "decline" || change == "gone-target" || change == "stun"
				if g.FindPermanent(first.ID()).Tapped != wantTapped {
					t.Fatal("optional untap or replacement contract")
				}
				if change == "control" && (b.untapCalls != 1 || a.untapCalls != 0) {
					t.Fatal("wrong controller chose untap")
				}
				if change == "gone-target" && a.untapCalls != 1 {
					t.Fatal("target departure removed optional untap choice")
				}
				if live := g.FindPermanent(target.ID()); live != nil {
					sum := 2
					if wantTapped {
						sum = (initialP + initialT + 2) / 2
					}
					if live.CurrentPower(g)+live.CurrentToughness(g) != sum {
						t.Fatal("untap did not end bonus")
					}
				}
			})
		}
	}
}

func TestRONSourceTappedDurationInvalidAndInventory(t *testing.T) {
	for _, duration := range []string{`ForAsLongAsPermanentRemainsTapped`, `ForAsLongAsPermanentRemainsTapped()`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent, ThisPermanent)`, `ForAsLongAsPermanentRemainsTapped(ThisPermanent())`, `ForAsLongAsPermanentRemainsTapped(Ref_TargetPermanent)`, `ForAsLongAsPermanentRemainsTapped(HostPermanent)`, `ForAsLongAsPermanentRemainsTapped(You)`} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],%s)]))])`, duration)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", duration)
		}
	}
	for _, typ := range []string{"Instant", "Artifact"} {
		rule := `SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,1)],ForAsLongAsPermanentRemainsTapped(ThisPermanent))])))`
		if typ == "Artifact" {
			rule = `EachPermanentLayerEffect(AnyPermanent,[ForAsLongAsPermanentRemainsTapped(ThisPermanent)])`
		}
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[]),Rules:[%s])`, typ, rule)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted wrong duration context")
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[AdjustPT(1,1)],ForAsLongAsPermanentRemainsTapped(ThisPermanent))]))])`
	report, err := InspectFS(fstest.MapFS{"duration.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "ThisPermanent" && strings.Contains(signature.Context, "ForAsLongAsPermanentRemainsTapped") {
			found = true
			if signature.Role != "permanent_reference" {
				t.Fatalf("duration operand role %s", signature.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing duration operand")
	}
}

func TestRONSourceTappedDurationTriggerAndDefinition(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,2),AddAbility([Flying])],ForAsLongAsPermanentRemainsTapped(ThisPermanent))])))`)
	first, second := factory(), factory()
	trigger := first.Abilities()[0].(*mage.GenericTriggered)
	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	invalid := mage.NewArtifact("Missing declaration", "", mage.WithAbility(mage.NewAction(mage.ActionActivated, mage.WithEffects(trigger.Effects()...))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("duration accepted a removed target declaration")
	}
	if trigger.Targets()[0] == second.Abilities()[0].(*mage.GenericTriggered).Targets()[0] {
		t.Fatal("factory shared target declarations")
	}
	g, a, b := ronGame()
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Llanowar Elves"), b.PlayerID())
	source := g.PutOnBattlefield(first, a.PlayerID())
	g.TapPermanent(source)
	g.ResolveStack()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 2 || !g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
		t.Fatal("trigger lost duration or typed target")
	}
	g.UntapPermanent(g.FindPermanent(source.ID()))
	if g.FindPermanent(target.ID()).CurrentPower(g) != 1 || g.FindPermanent(target.ID()).HasKeyword(core.Flying) {
		t.Fatal("mixed trigger layers did not expire together")
	}
}
