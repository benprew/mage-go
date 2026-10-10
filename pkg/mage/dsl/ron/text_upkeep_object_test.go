package ron

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextUpkeepObjectPendingTargets(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer),Targeted([TargetPermanent(IsColor(Red))],ActionList([TapPermanent(Ref_TargetPermanent),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])))`)
	g, a, b := ronModalGame(0)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	blue := g.PutOnBattlefield(textColorCard(core.Blue), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{red.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending target text changed")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	a.targets = []uuid.UUID{green.ID()}
	queueRONUpkeep(g, b)
	if g.StackSize() != 3 {
		t.Fatal("revised target not chosen")
	}
	clone := g.Clone()
	change, _ = mage.ReplaceColorWord(core.Green, core.Blue)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{blue.ID()}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if !g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(green.ID()).Tapped || !g.FindPermanent(blue.ID()).Tapped || a.Life() != 23 || b.Life() != 21 {
		t.Fatalf("queued/copy facts: A=%d B=%d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 21 || clone.FindPermanent(blue.ID()).Tapped {
		t.Fatal("clone shared later revision or actions")
	}
	g.UntapPermanent(blue)
	a.targets = []uuid.UUID{blue.ID()}
	queueRONUpkeep(g, b)
	g.BouncePermanentToHand(blue)
	b.RemoveFromHand(blue.Card.ID())
	returned := g.PutOnBattlefield(blue.Card, b.PlayerID())
	g.ResolveStack()
	if returned.Tapped || b.Life() != 21 {
		t.Fatal("illegal returned target permitted body")
	}
}

func TestRONTextUpkeepObjectIdentity(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(IsLandType(Swamp)),Player(You))]))`)
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	change, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err != nil {
		t.Fatal(err)
	}
	queueRONUpkeep(g, a)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err == nil {
		t.Fatal("phased revision accepted")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 {
		t.Fatal("phased source detected upkeep")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 20 {
		t.Fatal("pending controller changed")
	}
	g.PhaseIn(phased)
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("former controller upkeep triggered")
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 18 {
		t.Fatal("phase cycle lost text or control")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AddPrintedUpkeepObjectTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, change); err == nil {
		t.Fatal("removed rule revised")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("removed rule triggered")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.ApplyContinuousEffects()
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 16 {
		t.Fatal("restoration lost revision")
	}
	stale := g.FindPermanent(p.ID())
	g.BouncePermanentToHand(stale)
	if lki := g.LKIAbilities(p.ID()); len(lki) != len(stale.RuntimeAbilities) || lki[0] != stale.RuntimeAbilities[0] {
		t.Fatal("departure lost the final upkeep rule")
	}
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedUpkeepObjectTextEffect(stale, core.EndOfTurn, change); err == nil {
		t.Fatal("old incarnation revised return")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 17 {
		t.Fatal("return did not use printed rules")
	}
}

func TestRONTextUpkeepObjectChosenIdentity(t *testing.T) {
	for _, name := range []string{"Black Vise", "The Rack"} {
		for _, state := range []string{"control", "phase", "return", "remove"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				set := "ATQ"
				if name == "Black Vise" {
					set = "LEA"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
				if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err != nil {
					t.Fatal(err)
				}
				fillRONHand(b, 1)
				if name == "Black Vise" {
					fillRONHand(b, 5)
				}
				queueRONUpkeep(g, b)
				switch state {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.ApplyContinuousEffects()
				case "phase":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "return":
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
				case "remove":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				g.ResolveStack()
				if b.Life() != 18 || a.Life() != 20 {
					t.Fatal("pending choice or damage source changed")
				}
				if state == "remove" {
					g.RunStepWithPriority(core.Cleanup)
					g.ApplyContinuousEffects()
				}
				if state == "return" {
					queueRONUpkeep(g, b)
					if g.StackSize() != 0 {
						t.Fatal("return inherited entry choice")
					}
					queueRONUpkeep(g, a)
				} else {
					queueRONUpkeep(g, a)
					if g.StackSize() != 0 {
						t.Fatal("entry choice followed controller")
					}
					queueRONUpkeep(g, b)
				}
				if g.StackSize() != 1 {
					t.Fatal("saved entry choice lost")
				}
			})
		}
	}
}

func TestRONTextUpkeepObjectOrderAndExpiry(t *testing.T) {
	for _, prior := range []bool{false, true} {
		for _, phased := range []bool{false, true} {
			t.Run(fmt.Sprintf("prior=%v/phased=%v", prior, phased), func(t *testing.T) {
				g, a, b := ronGame()
				p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Karma"), a.PlayerID())
				id := p.RuntimeAbilities[0].AbilityID()
				for _, land := range []string{"Swamp", "Forest", "Forest", "Island", "Island", "Island"} {
					g.PutOnBattlefield(repositoryCard(t, "LEA", land), b.PlayerID())
				}
				baseline := core.SwampLandType
				if prior {
					change, _ := mage.ReplaceBasicLandWord(baseline, core.PlainsLandType)
					if err := g.RevisePrintedUpkeepTrigger(p, id, change); err != nil {
						t.Fatal(err)
					}
					baseline = core.PlainsLandType
				}
				first, _ := mage.ReplaceBasicLandWord(baseline, core.ForestLandType)
				second, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
				if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, first); err != nil {
					t.Fatal(err)
				}
				if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, second); err != nil {
					t.Fatal(err)
				}
				if err := g.RevisePrintedUpkeepTrigger(p, id, first); err == nil {
					t.Fatal("direct revision changed a layered result")
				}
				for range 3 {
					g.ApplyContinuousEffects()
				}
				queueRONUpkeep(g, b)
				clone := g.Clone()
				var phaseIDs []uuid.UUID
				if phased {
					phaseIDs = g.PhaseOut(p.ID())
				}
				g.RunStepWithPriority(core.Cleanup)
				if phased {
					g.PhaseIn(phaseIDs)
				}
				g.ApplyContinuousEffects()
				g.ResolveStack()
				if b.Life() != 17 {
					t.Fatal("expiry changed a pending body")
				}
				queueRONUpkeep(g, b)
				g.ResolveStack()
				want := 16
				if prior {
					want = 17
				}
				if b.Life() != want {
					t.Fatalf("surviving layer used expired text: got %d want %d", b.Life(), want)
				}
				clone.ResolveStack()
				clone.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
				clone.PutTriggersOnStack()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 14 {
					t.Fatal("clone inherited expiry or baseline mutation")
				}
				if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
					t.Fatal("layer passes changed declaration identity")
				}
			})
		}
	}
}

func TestRONTextUpkeepObjectRecords(t *testing.T) {
	for _, tc := range []struct {
		name string
		sets []string
	}{
		{"Black Vise", []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}},
		{"The Rack", []string{"ATQ", "3ED"}}, {"Ivory Tower", []string{"ATQ", "3ED"}},
	} {
		for _, set := range tc.sets {
			for _, size := range []int{0, 3, 4, 7} {
				t.Run(fmt.Sprintf("%s/%s/%d", set, tc.name, size), func(t *testing.T) {
					g, a, b := ronGame()
					card := repositoryCard(t, set, tc.name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Colorless, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					p := g.FindPermanent(card.ID())
					index := 0
					if tc.name != "Ivory Tower" {
						index = 1
					}
					id := p.RuntimeAbilities[index].AbilityID()
					choice := p.ChosenPlayer
					for _, consumer := range []string{"Magical Hack", "Sleight of Mind"} {
						from, to := "Swamp", "Forest"
						if consumer == "Sleight of Mind" {
							from, to = "Red", "Green"
						}
						consumerSet := set
						if set == "ATQ" {
							consumerSet = "LEA"
						}
						castManaTapObjectWords(t, g, a, p, consumerSet, consumer, from, to)
					}
					change, _ := mage.ReplaceColorWord(core.Red, core.Green)
					if err := g.RevisePrintedUpkeepTrigger(g.FindPermanent(p.ID()), id, change); err == nil {
						t.Fatal("consumer did not install an upkeep text layer")
					}
					if p = g.FindPermanent(p.ID()); p.RuntimeAbilities[index].AbilityID() != id || p.ChosenPlayer != choice {
						t.Fatal("revision changed declaration or entry choice")
					}
					event, other := b, a
					if tc.name == "Ivory Tower" {
						event, other = a, b
					}
					queueRONUpkeep(g, other)
					if g.StackSize() != 0 {
						t.Fatal("wrong player triggered")
					}
					queueRONUpkeep(g, event)
					if g.StackSize() != 1 || a.Life() != 20 || b.Life() != 20 {
						t.Fatal("incorrect trigger timing")
					}
					fillRONHand(event, size)
					fillRONHand(other, 10)
					g.DestroyPermanent(p)
					clone := g.Clone()
					clone.ResolveStack()
					g.ResolveStack()
					amount := max(0, size-4)
					if tc.name == "The Rack" {
						amount = max(0, 3-size)
					}
					want := 20 - amount
					if tc.name == "Ivory Tower" {
						want = 20 + amount
					}
					if event.Life() != want || other.Life() != 20 || clone.GetPlayer(event.PlayerID()).Life() != want {
						t.Fatalf("resolution/clone: got %d, want %d", event.Life(), want)
					}
					if err := g.AddPrintedUpkeepTextEffect(p, id, core.EndOfTurn, change); err == nil {
						t.Fatal("departed source revised")
					}
				})
			}
		}
	}
}

func TestRONTextUpkeepObjectKarma(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, set, "Karma"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
			queueRONUpkeep(g, b)
			castManaTapObjectWords(t, g, a, p, set, "Magical Hack", "Swamp", "Forest")
			g.ResolveStack()
			if b.Life() != 19 {
				t.Fatal("pending body lost old land word")
			}
			queueRONUpkeep(g, b)
			g.ResolveStack()
			if b.Life() != 17 {
				t.Fatal("new body lost word or event player")
			}
			queueRONUpkeep(g, a)
			g.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("amount used source controller instead of event player")
			}
		})
	}
}

func TestRONTextUpkeepObjectInventory(t *testing.T) {
	expected := map[string]int{"Black Vise": 6, "The Rack": 2, "Ivory Tower": 2, "Karma": 6, "Copper Tablet": 5, "Erhnam Djinn": 1, "Juzám Djinn": 1, "Gabriel Angelfire": 1, "Halfdane": 1, "Storm World": 1}
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if _, selected := expected[name]; !selected {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
				t.Fatalf("%s/%s: %v", file, name, err)
			}
			counts[name]++
		}
	}
	if !maps.Equal(counts, expected) {
		t.Fatalf("accepted contexts: %v", counts)
	}
}

func TestRONTextUpkeepObjectBoundaries(t *testing.T) {
	g, a, _ := ronGame()
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	for _, name := range []string{"Wild Growth", "Conversion", "Force of Nature", "Bog Wraith", "Mana Flare"} {
		p := g.PutOnBattlefield(repositoryCard(t, "LEA", name), a.PlayerID())
		if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatalf("accepted foreign context %s", name)
		}
	}
	for _, rule := range []string{
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(TheChosenPlayer)),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(AnyPlayer()),ActionList([GainLife(Integer(1))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PlayerAction(Trigger_ThatSpell,GainLife(Integer(1)))]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([If(CostWasPaid,[GainLife(Integer(1))])]))`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([CreatePermanentLayerEffect(ThisPermanent,[ReplaceColorWordVariableWithNewColorWordVariable(TheFirstChosenColorWord,TheSecondChosenColorWord)])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid binding", rule)
		}
	}
}

func TestRONTextUpkeepObjectPayments(t *testing.T) {
	rule := `TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayCost(PayMana([ManaCostG,ManaCostG])),Unless(CostWasPaid,[PermanentDealsDamage(ThisPermanent,Integer(3),Player(You))])]))`
	for _, scenario := range []string{"paid", "declined", "wrong-color", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", rule)(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			color, amount := core.Green, 2
			if scenario == "wrong-color" {
				color = core.Blue
			}
			if scenario == "insufficient" {
				amount = 1
			}
			a.ManaPool().Add(color, amount)
			queueRONUpkeep(g, a)
			if a.offers != 0 || a.ManaPool().TotalMana() != amount {
				t.Fatal("pre-resolution payment")
			}
			clone := g.Clone()
			g.ResolveStack()
			if scenario == "paid" {
				if a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("fixed mana symbols changed")
				}
			} else if a.Life() != 17 || a.ManaPool().TotalMana() != amount {
				t.Fatal("unpaid branch spent resources")
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != amount {
				t.Fatal("payment changed clone")
			}
		})
	}
}

func TestRONTextUpkeepObjectPrevention(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Karma"), a.PlayerID())
	for range 3 {
		g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	}
	castManaTapObjectWords(t, g, a, p, "LEA", "Magical Hack", "Swamp", "Forest")
	g.AddPreventionShield(b.PlayerID(), 2)
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("revised damage bypassed prevention")
	}
	queueRONUpkeep(g, b)
	g.ResolveStack()
	if b.Life() != 16 {
		t.Fatal("prevention did not expire on use")
	}
}
