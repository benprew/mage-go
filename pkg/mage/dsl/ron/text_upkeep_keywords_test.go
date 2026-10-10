package ron

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type upkeepKeywordPaymentChooser struct {
	mage.Player
	accept bool
}

func (p *upkeepKeywordPaymentChooser) ChooseMayAbility(string) bool { return p.accept }

func TestRONTextUpkeepKeywordsRecords(t *testing.T) {
	for _, tc := range []struct {
		name     string
		sets     []string
		keywords []core.Attr
	}{
		{"Force of Nature", []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}, []core.Attr{core.Trample}},
		{"Phantasmal Forces", []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}, []core.Attr{core.Flying}},
		{"Serendib Efreet", []string{"ARN", "3ED"}, []core.Attr{core.Flying}},
		{"Lord of the Pit", []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}, []core.Attr{core.Flying, core.Trample}},
		{"Junún Efreet", []string{"ARN"}, []core.Attr{core.Flying}},
		{"Palladia-Mors", []string{"LEG"}, []core.Attr{core.Flying, core.Trample}},
		{"Cosmic Horror", []string{"LEG"}, []core.Attr{core.FirstStrike}},
		{"Yawgmoth Demon", []string{"ATQ"}, []core.Attr{core.Flying, core.FirstStrike}},
	} {
		for _, set := range tc.sets {
			for _, paid := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/paid=%v", set, tc.name, paid), func(t *testing.T) {
					a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: paid}
					b := mage.NewBasePlayer("B")
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					card := repositoryCard(t, set, tc.name)
					card.SetOwner(a.PlayerID())
					p := g.PutOnBattlefield(card, a.PlayerID())
					id := p.RuntimeAbilities[0].AbilityID()
					consumerSet := set
					if set == "ARN" || set == "LEG" || set == "ATQ" {
						consumerSet = "LEA"
					}
					castManaTapObjectWords(t, g, a, p, consumerSet, "Sleight of Mind", "Green", "Blue")
					castManaTapObjectWords(t, g, a, p, consumerSet, "Magical Hack", "Swamp", "Forest")
					change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
					if err := g.RevisePrintedUpkeepTrigger(g.FindPermanent(p.ID()), id, change); err == nil {
						t.Fatal("real consumer did not install a text layer")
					}
					for _, keyword := range tc.keywords {
						if !g.FindPermanent(p.ID()).HasKeyword(keyword) {
							t.Fatal("consumer changed fixed keyword")
						}
					}
					if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
						t.Fatal("consumer changed trigger identity")
					}
					var victim *mage.Permanent
					if tc.name == "Lord of the Pit" && paid {
						victim = g.PutOnBattlefield(mage.NewCreature("Payment", "", 1, 1), a.PlayerID())
					}
					if tc.name == "Yawgmoth Demon" && paid {
						victim = g.PutOnBattlefield(mage.NewArtifact("Payment", ""), a.PlayerID())
					}
					color, amount := core.Green, 4
					if tc.name == "Phantasmal Forces" {
						color, amount = core.Blue, 1
					}
					if tc.name == "Junún Efreet" {
						color, amount = core.Black, 2
					}
					a.ManaPool().Clear()
					a.ManaPool().Add(color, amount)
					if tc.name == "Palladia-Mors" {
						a.ManaPool().Clear()
						for _, c := range []core.Color{core.Red, core.Green, core.White} {
							a.ManaPool().Add(c, 1)
						}
						amount = 3
					}
					if tc.name == "Cosmic Horror" {
						a.ManaPool().Clear()
						a.ManaPool().Add(core.Black, 3)
						a.ManaPool().Add(core.Colorless, 3)
						amount = 6
					}
					queueRONUpkeep(g, b)
					if g.StackSize() != 0 {
						t.Fatal("opponent upkeep triggered")
					}
					queueRONUpkeep(g, a)
					if g.StackSize() != 1 || a.offers != 0 || a.Life() != 20 || a.ManaPool().TotalMana() != amount {
						t.Fatal("payment or damage ran before resolution")
					}
					clone := g.Clone()
					clone.SetPlayerAt(0, &upkeepKeywordPaymentChooser{Player: clone.GetPlayer(a.PlayerID()), accept: paid})
					g.ResolveStack()
					wantLife, present, mana := 20, true, amount
					switch tc.name {
					case "Force of Nature":
						if paid {
							mana = 0
						} else {
							wantLife = 12
						}
					case "Phantasmal Forces", "Junún Efreet", "Palladia-Mors":
						if paid {
							mana = 0
						} else {
							present = false
						}
					case "Serendib Efreet":
						wantLife = 19
					case "Cosmic Horror":
						if paid {
							mana = 0
						} else {
							present, wantLife = false, 13
						}
					case "Yawgmoth Demon":
						if !paid {
							wantLife = 18
						}
						if g.FindPermanent(p.ID()).Tapped != !paid || (victim != nil && g.FindPermanent(victim.ID()) != nil) {
							t.Fatal("optional sacrifice lost tap or payment")
						}
					case "Lord of the Pit":
						if !paid {
							wantLife = 13
						}
						if a.offers != 0 || (victim != nil && g.FindPermanent(victim.ID()) != nil) {
							t.Fatal("mandatory sacrifice became optional or failed")
						}
					}
					if a.Life() != wantLife || b.Life() != 20 || (g.FindPermanent(p.ID()) != nil) != present || a.ManaPool().TotalMana() != mana {
						t.Fatalf("upkeep outcome: life=%d mana=%d present=%v", a.Life(), a.ManaPool().TotalMana(), g.FindPermanent(p.ID()) != nil)
					}
					if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.FindPermanent(p.ID()) == nil {
						t.Fatal("resolution changed clone")
					}
					clone.ResolveStack()
					if clone.GetPlayer(a.PlayerID()).Life() != wantLife || (clone.FindPermanent(p.ID()) != nil) != present {
						t.Fatal("clone lost payment or body")
					}
					if present && g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
						blocker := g.PutOnBattlefield(mage.NewCreature("Reach blocker", "", 1, 1, mage.WithKeyword(core.Reach)), b.PlayerID())
						power := g.FindPermanent(p.ID()).CurrentPower(g)
						g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
						g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: p.ID()}})
						g.ExecuteCombatDamage()
						if b.Life() != 20-(power-1) {
							t.Fatal("text layers changed trample damage")
						}
					}
				})
			}
		}
	}
}

func TestRONTextUpkeepKeywordsPayments(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "wrong-color", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "declined"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			p := g.PutOnBattlefield(repositoryCard(t, "LEA", "Force of Nature"), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Green, core.Blue)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err != nil {
				t.Fatal(err)
			}
			color, amount := core.Green, 4
			if scenario == "wrong-color" {
				color = core.Blue
			}
			if scenario == "insufficient" {
				amount = 3
			}
			a.ManaPool().Add(color, amount)
			queueRONUpkeep(g, a)
			if a.offers != 0 || a.ManaPool().TotalMana() != amount {
				t.Fatal("payment ran at detection")
			}
			g.ResolveStack()
			if scenario == "paid" {
				if a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("fixed green mana symbols changed")
				}
			} else if a.Life() != 12 || a.ManaPool().TotalMana() != amount {
				t.Fatal("unpaid branch spent resources or lost damage")
			}
			if !g.FindPermanent(p.ID()).HasKeyword(core.Trample) {
				t.Fatal("payment changed fixed keyword")
			}
		})
	}
}

func TestRONTextUpkeepKeywordsInventory(t *testing.T) {
	expected := map[string]int{"Force of Nature": 6, "Phantasmal Forces": 6, "Serendib Efreet": 2, "Lord of the Pit": 6, "Junún Efreet": 1, "Palladia-Mors": 1, "Cosmic Horror": 1, "Yawgmoth Demon": 1, "Wall of Tombstones": 1}
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	rejected := map[string]int{}
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
			if !upkeepObjectTextRules(node.Fields["Rules"].Args) {
				continue
			}
			fixed := false
			for _, rule := range node.Fields["Rules"].Args {
				if _, exists := keywords[rule.Name]; exists && rule.Kind == Symbol {
					fixed = true
				}
			}
			if !fixed {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				if !strings.Contains(err.Error(), "activation history requires the resolving activated ability") {
					t.Fatal(err)
				}
				rejected[node.Fields["Name"].String]++
				continue
			}
			g, a, _ := ronGame()
			p := g.PutOnBattlefield(factory(), a.PlayerID())
			change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
			if err := g.AddPrintedUpkeepObjectTextEffect(p, core.EndOfTurn, change); err != nil {
				t.Fatal(err)
			}
			counts[node.Fields["Name"].String]++
		}
	}
	if !maps.Equal(counts, expected) {
		t.Fatalf("accepted keyword contexts: %v", counts)
	}
	if !maps.Equal(rejected, map[string]int{"Mishra's War Machine": 2}) {
		t.Fatalf("remaining body blockers: %v", rejected)
	}
}

func TestRONTextUpkeepKeywordsWall(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "LEG", "Wall of Tombstones"), a.PlayerID())
	castManaTapObjectWords(t, g, a, p, "LEA", "Sleight of Mind", "Black", "Blue")
	castManaTapObjectWords(t, g, a, p, "LEA", "Magical Hack", "Swamp", "Forest")
	change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
	if err := g.RevisePrintedUpkeepTrigger(g.FindPermanent(p.ID()), p.RuntimeAbilities[0].AbilityID(), change); err == nil {
		t.Fatal("consumer did not install text layer")
	}
	for range 2 {
		a.AddToGraveyard(mage.NewCreature("Dead creature", "", 1, 1))
	}
	a.AddToGraveyard(mage.NewArtifact("Dead artifact", ""))
	b.AddToGraveyard(mage.NewCreature("Other graveyard", "", 1, 1))
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("opponent upkeep triggered")
	}
	queueRONUpkeep(g, a)
	if p.CurrentToughness(g) != 1 {
		t.Fatal("toughness changed before resolution")
	}
	g.ResolveStack()
	current := g.FindPermanent(p.ID())
	if current.CurrentPower(g) != 0 || current.CurrentToughness(g) != 3 || !current.HasKeyword(core.Defender) {
		t.Fatal("revised upkeep lost graveyard count or defender")
	}
}

func TestRONTextUpkeepKeywordsLifecycle(t *testing.T) {
	factory := textCopyFactory(t, "Keyword upkeep", `Flying,Trample,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([PermanentDealsDamage(ThisPermanent,TheNumberOfPermanentsOnTheBattlefield(IsLandType(Swamp)),Player(You))]))`)
	g, a, b := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	id := p.RuntimeAbilities[0].AbilityID()
	for _, land := range []string{"Swamp", "Forest", "Forest", "Island", "Island", "Island"} {
		g.PutOnBattlefield(repositoryCard(t, "LEA", land), b.PlayerID())
	}
	queueRONUpkeep(g, a)
	castManaTapObjectWords(t, g, a, p, "LEA", "Magical Hack", "Swamp", "Forest")
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatal("consumer changed pending text")
	}
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedUpkeepObjectTextEffect(g.FindPermanent(p.ID()), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	queueRONUpkeep(g, a)
	clone := g.Clone()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ApplyContinuousEffects()
	phased := g.PhaseOut(p.ID())
	queueRONUpkeep(g, b)
	if g.StackSize() != 1 {
		t.Fatal("phased upkeep detected")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.PhaseIn(phased)
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if a.Life() != 16 || b.Life() != 20 {
		t.Fatal("pending controller or temporary text changed")
	}
	clone.ResolveStack()
	queueRONUpkeep(clone, clone.GetPlayer(a.PlayerID()))
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 13 {
		t.Fatal("clone shared expiry or control")
	}
	current := g.FindPermanent(p.ID())
	if current.RuntimeAbilities[0].AbilityID() != id || !current.HasKeyword(core.Flying) || !current.HasKeyword(core.Trample) {
		t.Fatal("phase cycle changed declarations")
	}
	ground := g.PutOnBattlefield(mage.NewCreature("Ground blocker", "", 1, 1), a.PlayerID())
	reach := g.PutOnBattlefield(mage.NewCreature("Reach blocker", "", 1, 1, mage.WithKeyword(core.Reach)), a.PlayerID())
	if mage.CanBlock(ground, current, g) || !mage.CanBlock(reach, current, g) {
		t.Fatal("text layers changed flying legality")
	}
	queueRONUpkeep(g, a)
	if g.StackSize() != 0 {
		t.Fatal("former controller triggered")
	}
	queueRONUpkeep(g, b)
	g.AddPreventionShield(b.PlayerID(), 1)
	g.ResolveStack()
	if b.Life() != 19 {
		t.Fatal("expiry lost surviving layer or prevention")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	current = g.FindPermanent(p.ID())
	if current.HasKeyword(core.Flying) || current.HasKeyword(core.Trample) {
		t.Fatal("text layer restored removed keywords")
	}
	if err := g.AddPrintedUpkeepObjectTextEffect(current, core.EndOfTurn, change); err == nil {
		t.Fatal("removed trigger revised")
	}
	queueRONUpkeep(g, b)
	if g.StackSize() != 0 {
		t.Fatal("removed trigger detected upkeep")
	}
	g.RunStepWithPriority(core.Cleanup)
	g.ApplyContinuousEffects()
	queueRONUpkeep(g, b)
	stale := g.FindPermanent(p.ID())
	if !stale.HasKeyword(core.Flying) || !stale.HasKeyword(core.Trample) {
		t.Fatal("restoration lost fixed keywords")
	}
	g.BouncePermanentToHand(stale)
	if snapshot := g.LKI(p.ID()).Snapshot; !snapshot.HasKeyword(core.Flying) || !snapshot.HasKeyword(core.Trample) || snapshot.RuntimeAbilities[0] != stale.RuntimeAbilities[0] {
		t.Fatal("departure lost final declarations")
	}
	g.ResolveStack()
	if b.Life() != 17 {
		t.Fatal("pending body lost departure facts")
	}
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.AddPrintedUpkeepObjectTextEffect(stale, core.EndOfTurn, change); err == nil {
		t.Fatal("old visit revised returned source")
	}
	queueRONUpkeep(g, a)
	g.ResolveStack()
	if a.Life() != 15 || !returned.HasKeyword(core.Flying) || !returned.HasKeyword(core.Trample) {
		t.Fatal("return did not use printed rules")
	}
}

func TestRONTextUpkeepKeywordsBoundaries(t *testing.T) {
	for _, keyword := range []string{`Flying()`, `Trample(Integer(1))`, `UnknownKeyword`} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[` + keyword + `,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-keyword.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid keyword accepted", keyword)
		}
	}
	for _, sibling := range []string{`Landwalk(IsLandType(Swamp))`, `Protection(FromColor(Colors([Red])))`, `Activated(PayMana([]),ActionList([GainLife(Integer(1))]))`} {
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(textCopyFactory(t, "Foreign sibling", sibling+`,Flying,TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([GainLife(Integer(1))]))`)(), a.PlayerID())
		change, _ := mage.ReplaceColorWord(core.Red, core.Blue)
		if err := g.AddPrintedUpkeepObjectTextEffect(p, core.Indefinite, change); err == nil {
			t.Fatal("unsupported sibling accepted", sibling)
		}
	}
}
