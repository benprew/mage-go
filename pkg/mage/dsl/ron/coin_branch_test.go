package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCoinBottle(t *testing.T) {
	for _, win := range []bool{false, true} {
		for _, copyAbility := range []bool{false, true} {
			t.Run(fmt.Sprintf("win=%v/copy=%v", win, copyAbility), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "ARN", "Bottle of Suleiman")
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				g.SetCoinFlipResults([]bool{win, win, !win})
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid casting accepted")
				}
				a.ManaPool().Add(core.Colorless, 4)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil || g.FindPermanent(card.ID()) == nil {
					t.Fatal("failed activation spent its sacrifice")
				}
				a.ManaPool().Add(core.Colorless, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil || a.Life() != 20 || len(g.FilterBattlefield(mage.IsToken)) != 0 {
					t.Fatal("wrong sacrifice cost or resolution timing")
				}
				if copyAbility && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				g.ResolveStack()
				wantLife, count := 15, 0
				if win {
					wantLife, count = 20, 1
				}
				if copyAbility && win {
					count++
				}
				wantB := 20
				if copyAbility {
					wantB = wantLife
				}
				if g.GetPlayer(a.PlayerID()).Life() != wantLife || g.GetPlayer(b.PlayerID()).Life() != wantB || len(g.FilterBattlefield(mage.IsToken)) != count {
					t.Fatal("wrong branch or resolving controller")
				}
				for _, token := range g.FilterBattlefield(mage.IsToken) {
					if token.CurrentPower(g) != 5 || token.CurrentToughness(g) != 5 || !token.HasSubType("Djinn") || !token.HasType(core.TypeArtifact) || !token.HasType(core.TypeCreature) || !token.HasKeyword(core.Flying) || len(token.Colors()) != 0 || token.Card.Owner() != token.ControllerID() {
						t.Fatal("wrong Djinn characteristics")
					}
				}
				if win && (len(g.FilterBattlefield(mage.And(mage.IsToken, mage.ControlledBy(a.PlayerID())))) != 1 || (copyAbility && len(g.FilterBattlefield(mage.And(mage.IsToken, mage.ControlledBy(b.PlayerID())))) != 1)) {
					t.Fatal("token used the source owner instead of the resolving controller")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 || len(clone.FilterBattlefield(mage.IsToken)) != 0 {
					t.Fatal("clone shared branch effects")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != wantLife || len(clone.FilterBattlefield(mage.IsToken)) != count {
					t.Fatal("clone lost pending outcome")
				}
			})
		}
	}
}

func TestRONCoinCaptain(t *testing.T) {
	for _, change := range []string{"win", "loss", "shroud", "return", "phase", "copy", "source-depart", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Orcish Captain"), a.PlayerID())
			card := mage.NewCreature("Orc", "", 3, 3, mage.WithSubTypes("Orc"))
			victim := g.PutOnBattlefield(card, b.PlayerID())
			wrong := g.PutOnBattlefield(mage.NewCreature("Wrong", "", 3, 3), a.PlayerID())
			g.SetCoinFlipResults([]bool{true, false, true})
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{wrong.ID()}); err == nil {
				t.Fatal("non-Orc accepted")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if victim.CurrentPower(g) != 3 {
				t.Fatal("flip ran before resolution")
			}
			legal := true
			switch change {
			case "loss":
				g.SetCoinFlipResults([]bool{false, true})
			case "shroud":
				g.FindPermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				legal = false
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(card, b.PlayerID())
				legal = false
			case "phase":
				g.PhaseOut(victim.ID())
				legal = false
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "source-depart":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			clone := g.Clone()
			g.ResolveStack()
			p := g.FindPermanentIncludingPhased(victim.ID())
			power, toughness := 3, 3
			if legal {
				if change == "loss" {
					toughness = 1
				} else {
					power = 5
					if change == "copy" {
						toughness = 1
					}
				}
			}
			if p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
				t.Fatalf("got %d/%d, want %d/%d", p.CurrentPower(g), p.CurrentToughness(g), power, toughness)
			}
			if !legal && !g.FlipCoin(a.PlayerID()) {
				t.Fatal("illegal target consumed a flip")
			}
			if clone.FindPermanentIncludingPhased(victim.ID()).CurrentPower(clone) != 3 {
				t.Fatal("clone shared modifiers")
			}
			clone.ResolveStack()
			if p := clone.FindPermanentIncludingPhased(victim.ID()); p.CurrentPower(clone) != power || p.CurrentToughness(clone) != toughness {
				t.Fatal("clone lost branch or binding")
			}
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if p = g.FindPermanentIncludingPhased(victim.ID()); p.CurrentPower(g) != 3 || p.CurrentToughness(g) != 3 {
				t.Fatal("branch bonus did not expire")
			}
		})
	}
}

func TestRONCoinCastingAndInstances(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/FEM.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "FEM.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Goblin Kites", "Orcish Captain"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			var factory mage.CardFactory
			for _, node := range nodes {
				if node.Fields["Name"].String == name {
					factory, err = Compile(node)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			if factory == nil {
				t.Fatal("missing card record")
			}
			card := factory()
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid casting accepted")
			}
			a.ManaPool().Add(core.Red, 1)
			if name == "Goblin Kites" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p := g.FindPermanent(card.ID())
			if name == "Orcish Captain" && (!p.HasSubType("Orc") || !p.HasSubType("Warrior") || p.CurrentPower(g) != 1 || p.CurrentToughness(g) != 1) {
				t.Fatal("wrong captain metadata")
			}
			if name == "Goblin Kites" && !p.HasType(core.TypeEnchantment) {
				t.Fatal("wrong kites metadata")
			}
			other := g.PutOnBattlefield(factory(), b.PlayerID())
			if p.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == other.Card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declarations")
			}
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 2, mage.WithSubTypes("Orc")), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 2, mage.WithSubTypes("Orc")), b.PlayerID())
			color := core.Colorless
			if name == "Goblin Kites" {
				color = core.Red
				a.ManaPool().Add(color, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{second.ID()}); err == nil {
					t.Fatal("opponent creature accepted")
				}
			} else {
				a.ManaPool().Add(color, 1)
			}
			b.ManaPool().Add(color, 1)
			g.SetCoinFlipResults([]bool{true, false, true})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, []uuid.UUID{second.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if name == "Goblin Kites" {
				if len(g.DelayedTriggers()) != 2 {
					t.Fatal("pending activations lost delays")
				}
				g.RunStepWithPriority(core.EndStep)
				if (g.FindPermanent(first.ID()) == nil) == (g.FindPermanent(second.ID()) == nil) {
					t.Fatal("independent delayed references lost")
				}
			} else if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()).CurrentPower(g) != 5 {
				t.Fatal("pending activations lost independent targets or outcomes")
			}
		})
	}
}

func TestRONCoinBottleDamageAndLKI(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Bottle of Suleiman")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 1)
	g.SetCoinFlipResults([]bool{false, true})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	returned := g.PutOnBattlefield(card, b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 3)
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 || g.FindPermanent(returned.ID()) == nil || !g.FlipCoin(a.PlayerID()) {
		t.Fatal("damage bypassed prevention, followed the returned source, or flipped twice")
	}
}

func TestRONCoinNestedSequencing(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([FlipACoin_OnWinAndLose([PlayerAction(Ref_TargetPlayer, GainLife(Integer(2))), FlipACoin_OnLose([PlayerAction(Ref_TargetPlayer, LoseLife(Integer(1)))])], [PlayerAction(Ref_TargetPlayer, LoseLife(Integer(4)))]), PlayerAction(Ref_TargetPlayer, GainLife(Integer(3)))])))`)
	for _, outcomes := range [][]bool{{true, false, true}, {false, true}} {
		g, a, b := ronGame()
		g.SetCoinFlipResults(outcomes)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		want := 19
		if outcomes[0] {
			want = 24
		}
		if b.Life() != want || a.Life() != 20 || !g.FlipCoin(a.PlayerID()) {
			t.Fatal("nested coin used the wrong branch, order, target, or number of flips")
		}
	}
}

func TestRONCoinKites(t *testing.T) {
	for _, change := range []string{"win", "loss", "return", "phase", "phase-cycle", "control", "source-depart", "shroud-after", "too-large-before", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "FEM", "Goblin Kites"), a.PlayerID())
			card := mage.NewCreature("Victim", "", 2, 2)
			card.SetOwner(a.PlayerID())
			victim := g.PutOnBattlefield(card, a.PlayerID())
			g.SetCoinFlipResults([]bool{false, true})
			if change == "copy" {
				g.SetCoinFlipResults([]bool{false, false, true})
			}
			a.ManaPool().Add(core.Red, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "too-large-before" {
				g.FindPermanent(victim.ID()).AddCounter(core.P1P1, 1)
			}
			g.ResolveStack()
			if change == "too-large-before" {
				if len(g.DelayedTriggers()) != 0 || g.FlipCoin(a.PlayerID()) {
					t.Fatal("illegal resolution installed delay or consumed RNG")
				}
				return
			}
			if !g.FindPermanent(victim.ID()).HasKeyword(core.Flying) || len(g.DelayedTriggers()) != 1 {
				t.Fatal("missing grant or delay")
			}
			switch change {
			case "win":
				g.SetCoinFlipResults([]bool{true, false})
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-depart":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "shroud-after":
				g.FindPermanent(victim.ID()).GrantBaseAttr(core.Shroud)
				g.FindPermanent(victim.ID()).AddCounter(core.P1P1, 3)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || len(g.StackPeek().TargetSpecs) != 0 {
				t.Fatal("delay did not use a captured reference")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			present := change == "win" || change == "return" || change == "phase" || change == "control"
			if (g.FindPermanentIncludingPhased(victim.ID()) != nil) != present || len(g.DelayedTriggers()) != 0 {
				t.Fatal("wrong delayed outcome or incarnation")
			}
			if g.FlipCoin(a.PlayerID()) != (change != "win") {
				t.Fatal("delayed creation or skipped actions changed the number of flips")
			}
			if clone.FindPermanentIncludingPhased(victim.ID()) == nil {
				t.Fatal("clone shared sacrifice")
			}
			clone.ResolveStack()
			if (clone.FindPermanentIncludingPhased(victim.ID()) != nil) != present {
				t.Fatal("clone lost delay or outcome")
			}
			if present {
				g.RunStepWithPriority(core.Cleanup)
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanentIncludingPhased(victim.ID()) == nil {
					t.Fatal("one-shot delay repeated")
				}
			}
		})
	}
}

func TestRONCoinInvalidAndInventory(t *testing.T) {
	for _, body := range []string{
		`FlipACoin_OnLose`, `FlipACoin_OnLose([], [])`, `FlipACoin_OnLose(DrawACard)`,
		`FlipACoin_OnLose([])`, `FlipACoin_OnWinAndLose([DrawACard])`,
		`FlipACoin_OnWinAndLose([DrawACard], [])`,
		`FlipACoin_OnLose([SacrificePermanent(Ref_TargetPermanent)])`,
		`FlipACoin_OnLose([MayCost(PayMana([ManaCostGeneric(1)]))])`,
		`FlipACoin_OnLose([If(CostWasPaid, [DrawACard])])`,
		`FlipACoin_OnLose([Targeted([TargetPlayer(AnyPlayer)], ActionList([DrawACard]))])`,
		`FlipACoin_OnLose([MayCost(PayMana([ManaCostGeneric(1)])), If(CostWasPaid, [DrawACard])]), If(CostWasPaid, [DrawACard])`,
	} {
		t.Run(body, func(t *testing.T) {
			src := fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [SpellActions(ActionList([%s]))])`, body)
			nodes, err := Parse(strings.NewReader(src), "invalid-coin.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "invalid-coin.ron:") {
				t.Fatalf("invalid declaration accepted or unpositioned: %v", err)
			}
		})
	}
	src := `Card(Name: "Coin", Typeline: (Supertypes: [], Cardtypes: [Sorcery], Subtypes: []), Rules: [SpellActions(ActionList([FlipACoin_OnWinAndLose([DrawACard], [FlipACoin_OnLose([DrawACard])])]))])`
	report, err := InspectFS(fstest.MapFS{"coin.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("nested branches rejected: %v", report.Issues)
	}
	for _, entry := range report.Signatures {
		if (entry.Name == "DrawACard" || entry.Name == "FlipACoin_OnLose") && entry.Role != "action" {
			t.Fatalf("lost branch role: %+v", entry)
		}
	}
}
