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

func TestRONTemporaryPowerDeclarations(t *testing.T) {
	for _, power := range []int{-2, 0, 5} {
		factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [SetPower(Integer(%d))], UntilEndOfTurn)])))`, power))
		first, second := factory(), factory()
		if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
			t.Fatal(err)
		}
		if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
			t.Fatal("shared target declaration")
		}
	}
	for _, modification := range []string{`SetPower`, `SetPower()`, `SetPower(0)`, `SetPower(Integer())`, `SetPower(Integer(0, 1))`, `SetPower(Integer(You))`, `SetPower(Integer(0), Integer(1))`, `SetPower(ValueX)`, `SetPower(TheChosenNumber)`, `SetPower(PowerOfPermanent(ThisPermanent))`} {
		src := fmt.Sprintf(`Card(Name:"Invalid", Typeline:(Supertypes:[], Cardtypes:[Artifact], Subtypes:[]), Rules:[Activated(PayMana([]), ActionList([CreatePermanentLayerEffectUntil(ThisPermanent, [%s], UntilEndOfTurn)]))])`, modification)
		nodes, err := Parse(strings.NewReader(src), "invalid-power.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", modification)
		}
	}
}

func TestRONTemporaryPowerRealCards(t *testing.T) {
	for _, name := range []string{"Island of Wak-Wak", "Singing Tree"} {
		for _, change := range []string{"none", "source-leaves", "returned", "phased", "phase-cycle", "loses-qualification", "control", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "ARN", name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				creature := repositoryCard(t, "LEB", "Air Elemental")
				creature.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(creature, b.PlayerID())
				target.RevokeBaseAttr(core.AttrSummonSick)
				if name == "Singing Tree" {
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{target.ID()})
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || target.CurrentPower(g) != 4 || g.StackSize() != 1 {
					t.Fatal("activation timing or tap payment")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
					t.Fatal("tapped source activated")
				}
				clone := g.Clone()
				current := target.ID()
				switch change {
				case "source-leaves":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(creature.ID())
					current = g.PutOnBattlefield(creature, b.PlayerID()).ID()
				case "phased":
					g.PhaseOut(target.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "loses-qualification":
					if name == "Singing Tree" {
						g.GetCombat().RemoveFromCombat(target.ID())
					} else {
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveKeyword(mage.This, core.Flying)), target.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("ability copy failed")
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{current})
				want := 0
				if change == "returned" || change == "phased" || change == "loses-qualification" || (name == "Singing Tree" && (change == "phase-cycle" || change == "control")) {
					want = 4
				}
				check := func(game *mage.Game, power, toughness int) {
					t.Helper()
					p := game.FindPermanent(current)
					if p == nil || p.CurrentPower(game) != power || p.CurrentToughness(game) != toughness {
						t.Fatalf("P/T want %d/%d", power, toughness)
					}
				}
				check(g, want, 4)
				if clone.FindPermanent(target.ID()).CurrentPower(clone) != 4 {
					t.Fatal("live resolution changed clone")
				}
				clone.ResolveStack()
				if clone.FindPermanent(target.ID()).CurrentPower(clone) != 0 {
					t.Fatal("clone lost target")
				}
				if want == 0 {
					g.GetCombat().RemoveFromCombat(current)
					g.PhaseIn(g.PhaseOut(current))
					check(g, 0, 4)
					p := g.FindPermanent(current)
					p.AddCounter(core.P1P1, 1)
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(3), mage.Fixed(2))), current, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					check(g, 4, 7)
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetBasePower(mage.This, mage.Fixed(2))), current, b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					check(g, 6, 7)
					g.RunStepWithPriority(core.Cleanup)
					check(g, 5, 5)
					if clone.FindPermanent(target.ID()).CurrentPower(clone) != 0 {
						t.Fatal("cleanup changed clone")
					}
				}
			})
		}
	}
}

func TestRONTemporaryPowerLegality(t *testing.T) {
	for _, name := range []string{"Island of Wak-Wak", "Singing Tree"} {
		for _, reason := range []string{"plain", "noncreature", "shroud", "phased", "wrong-controller", "sick"} {
			t.Run(name+"/"+reason, func(t *testing.T) {
				g, a, b := ronGame()
				source := g.PutOnBattlefield(repositoryCard(t, "ARN", name), a.PlayerID())
				if reason != "sick" {
					source.RevokeBaseAttr(core.AttrSummonSick)
				}
				target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
				if name == "Singing Tree" && reason != "plain" {
					g.GetCombat().AddAttacker(target.ID(), a.PlayerID())
				}
				if reason == "plain" && name == "Island of Wak-Wak" {
					target.RevokeBaseAttr(core.Flying)
				}
				if reason == "noncreature" {
					target = g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), b.PlayerID())
				}
				if reason == "shroud" {
					target.GrantBaseAttr(core.Shroud)
				}
				if reason == "phased" {
					g.PhaseOut(target.ID())
				}
				payer := a
				if reason == "wrong-controller" {
					payer = b
				}
				if reason == "sick" && name == "Island of Wak-Wak" {
					source.Tapped = true
				}
				wasTapped := source.Tapped
				if err := g.ActivateAbilityByIndex(payer.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
					t.Fatal("illegal activation accepted")
				}
				if g.FindPermanent(source.ID()).Tapped != wasTapped || g.StackSize() != 0 {
					t.Fatal("failed activation changed state")
				}
			})
		}
	}
}

func TestRONTemporaryPowerStrictLoadAndNativeGame(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "ARN.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		if name := node.Fields["Name"].String; name == "Island of Wak-Wak" || name == "Singing Tree" {
			files[name+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	tree, err := registry.CreateCard("Singing Tree")
	if err != nil {
		t.Fatal(err)
	}
	tree.SetOwner(a.PlayerID())
	a.AddToHand(tree)
	if err := g.CastSpellByID(a.PlayerID(), tree.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast accepted")
	}
	a.ManaPool().Add(core.Colorless, 3)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), tree.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if p := g.FindPermanent(tree.ID()); p.CurrentPower(g) != 0 || p.CurrentToughness(g) != 3 {
		t.Fatal("tree metadata")
	}
	land, err := registry.CreateCard("Island of Wak-Wak")
	if err != nil {
		t.Fatal(err)
	}
	land.SetOwner(a.PlayerID())
	a.AddToHand(land)
	if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	target.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{target.ID()})
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.ExecuteCombatDamage()
	if a.Life() != 20 {
		t.Fatal("zero-power attacker dealt damage")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.FindPermanent(target.ID()).CurrentPower(g) != 4 {
		t.Fatal("cleanup did not restore power")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad", Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection published a registry")
	}
}

func TestRONTemporaryPowerSharedContexts(t *testing.T) {
	for _, power := range []int{-2, 0, 5} {
		action := fmt.Sprintf(`CreatePermanentLayerEffectUntil(ThisPermanent, [SetPower(Integer(%d)), AdjustPT(1, 2)], UntilEndOfTurn)`, power)
		for _, rule := range []string{`Activated(PayMana([]), ActionList([%s]))`, `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([%s]))`} {
			t.Run(fmt.Sprintf("%d/%s", power, rule), func(t *testing.T) {
				g, a, b := ronGame()
				nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Power context", Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]), CardPT:(Power:4,Toughness:4), Rules:[%s])`, fmt.Sprintf(rule, action))), "power-context.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				source := g.PutOnBattlefield(card, a.PlayerID())
				if strings.HasPrefix(rule, "Activated") {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				p := g.FindPermanent(source.ID())
				if p.CurrentPower(g) != power+1 {
					t.Fatal("fixed power did not precede bonus")
				}
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				if g.FindPermanent(source.ID()).CurrentPower(g) != power+1 {
					t.Fatal("installed effect ended on control change")
				}
			})
		}
	}
}

func TestRONTemporaryPowerSpellAndInventory(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent, [SetPower(Integer(5))], UntilEndOfTurn)])))`)
	target := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	spell := factory()
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()).CurrentPower(g) != 5 || g.FindPermanent(target.ID()).CurrentToughness(g) != 4 {
		t.Fatal("spell base power")
	}
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range report.Signatures {
		if entry.Name == "Integer" && entry.Role == "value" {
			for _, example := range entry.Examples {
				if (example.Card == "Island of Wak-Wak" || example.Card == "Singing Tree") && strings.Contains(example.Path, "Rules") {
					count++
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("temporary power value missing from semantic inventory")
	}
}
