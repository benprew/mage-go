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

const coinRoundBody = `RepeatableActions([EachPlayerAction(Or([SinglePlayer(You),SinglePlayer(Ref_TargetPlayer)]),FlipACoin),SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(CoinCameUpTails)),Unless(AllCoinsCameUpHeads,[RepeatThisProcess])])`

func TestRONCoinRoundManaClash(t *testing.T) {
	for _, active := range []int{0, 1} {
		for _, first := range [][]bool{{true, true}, {true, false}, {false, true}, {false, false}} {
			t.Run(fmt.Sprint(active, first), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "DRK", "Mana Clash")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				g.SetCoinFlipResults(append(append([]bool{}, first...), true, true, false))
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
					t.Fatal("unpaid cast")
				}
				if g.StackSize() != 0 || len(a.Hand()) != 1 || a.Life() != 20 || b.Life() != 20 {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{a.PlayerID()}, 0); err == nil {
					t.Fatal("controller was a legal opponent target")
				}
				if a.ManaPool().Count(core.Red) != 1 {
					t.Fatal("illegal target spent mana")
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("announcement dealt damage")
				}
				g.SetActivePlayerIndex(active)
				callbacks := 0
				g.SetOnDamageDealt(func(source, _ string, amount int, combat bool) {
					callbacks++
					if source != "Mana Clash" || amount != 1 || combat || a.Life() != 20-boolInt(!first[0]) || b.Life() != 20-boolInt(!first[1]) {
						t.Error("damage did not use one spell-source batch")
					}
				})
				g.ResolveStack()
				if a.Life() != 20-boolInt(!first[0]) || b.Life() != 20-boolInt(!first[1]) || callbacks != boolInt(!first[0])+boolInt(!first[1]) || len(a.Graveyard()) != 1 {
					t.Fatal("round outcome or termination")
				}
				if first[0] && first[1] {
					for range 2 {
						if !g.FlipCoin(a.PlayerID()) {
							t.Fatal("all heads did not end the first round")
						}
					}
				}
				if g.FlipCoin(a.PlayerID()) {
					t.Fatal("wrong number of flips")
				}
			})
		}
	}
}

func TestRONCoinRoundPreventionAndMultipleRounds(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Mana Clash")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	g.AddPreventionShield(a.PlayerID(), 1)
	g.AddPreventionShield(b.PlayerID(), 2)
	g.SetCoinFlipResults([]bool{false, false, true, false, false, true, true, true, false})
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 || g.FlipCoin(a.PlayerID()) {
		t.Fatal("prevention changed the repeat condition or round results leaked")
	}
}

func TestRONCoinRoundCopiesAndClones(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Mana Clash")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil {
		t.Fatal("spell copy")
	}
	g.SetCoinFlipResults([]bool{false, true, true, true, true, false, true, true, false})
	clone := g.Clone()
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 19 || g.StackSize() != 1 {
		t.Fatalf("copy controller/target: life %d/%d stack %d targets %v", a.Life(), b.Life(), g.StackSize(), copied.Targets)
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 18 || g.FlipCoin(a.PlayerID()) {
		t.Fatal("original inherited copy results")
	}
	if clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("original changed clone")
	}
	clone.ResolveStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 18 || clone.FlipCoin(a.PlayerID()) {
		t.Fatal("clone lost targets or random queue")
	}
}

func TestRONCoinRoundInvalid(t *testing.T) {
	for _, body := range []string{
		`RepeatableActions`, `RepeatableActions([])`, `RepeatableActions([FlipACoin])`,
		strings.Replace(coinRoundBody, "FlipACoin", "FlipACoin()", 1),
		strings.Replace(coinRoundBody, "FlipACoin", "DrawACard", 1),
		strings.Replace(coinRoundBody, "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(ActionPlayer)", 1),
		strings.Replace(coinRoundBody, "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(Ref_TargetPermanent)", 1),
		strings.Replace(coinRoundBody, "SinglePlayer(You)", "SinglePlayer(You,You)", 1),
		strings.Replace(coinRoundBody, "ThisSpell", "ThisPermanent", 1),
		strings.Replace(coinRoundBody, "Integer(1)", "ValueX", 1),
		strings.Replace(coinRoundBody, "CoinCameUpTails", "CoinCameUpTails()", 1),
		strings.Replace(coinRoundBody, "AllCoinsCameUpHeads", "AllCoinsCameUpHeads()", 1),
		strings.Replace(coinRoundBody, "RepeatThisProcess", "RepeatThisProcess()", 1),
		strings.Replace(coinRoundBody, "[RepeatThisProcess]", "[RepeatThisProcess,DrawACard]", 1),
		strings.Replace(coinRoundBody, "EachPlayer(CoinCameUpTails)", "Player(You)", 1),
		strings.Replace(coinRoundBody, "Unless(AllCoinsCameUpHeads,[RepeatThisProcess])", "Unless(AllCoinsCameUpHeads,[RepeatThisProcess]),DrawACard", 1),
		`FlipACoin`, `SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(CoinCameUpTails))`,
		`Unless(AllCoinsCameUpHeads,[RepeatThisProcess])`,
	} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(Opponent)],ActionList([` + body + `])))])`
		nodes, err := Parse(strings.NewReader(src), "bad-coin-round.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil || !strings.Contains(err.Error(), "bad-coin-round.ron:") {
			t.Fatalf("accepted or unpositioned %s: %v", body, err)
		}
	}
}

func TestRONCoinRoundNative(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.HasPrefix(line, `Card(Name: "Mana Clash"`) {
			record = line
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, fstest.MapFS{"clash.ron": {Data: []byte(record)}}, "*.ron"); err != nil {
		t.Fatal(err)
	}
	one, _ := registry.CreateCard("Mana Clash")
	two, _ := registry.CreateCard("Mana Clash")
	if one.ID() == two.ID() || one.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == two.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factories share declarations")
	}
	bad := strings.Replace(record, "AllCoinsCameUpHeads", "NoOneTookAnActionThisWay", 1)
	if err := LoadInto(registry, fstest.MapFS{"clash.ron": {Data: []byte(record)}, "bad.ron": {Data: []byte(strings.Replace(bad, `"Mana Clash"`, `"Bad Clash"`, 1))}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONCoinRoundIllegalCopy(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Mana Clash")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy")
	}
	g.SetCoinFlipResults([]bool{false, true, true, true, false})
	g.ResolveTopOfStack()
	if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 1 {
		t.Fatal("illegal copy target allowed resolution")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 || g.FlipCoin(a.PlayerID()) {
		t.Fatal("illegal copy consumed the original's randomness")
	}
}

func TestRONCoinRoundInventory(t *testing.T) {
	entries := map[string]*InventorySignature{}
	walkInventory(hostUntapRecord(t, "DRK", "Mana Clash"), "Mana Clash", "Card", "card", "collection", false, entries)
	want := map[string]string{"RepeatableActions": "action", "EachPlayerAction": "action", "FlipACoin": "player_action", "Ref_TargetPlayer": "player_reference", "CoinCameUpTails": "player_filter", "AllCoinsCameUpHeads": "condition", "RepeatThisProcess": "action"}
	for _, entry := range entries {
		if role, ok := want[entry.Name]; ok {
			if entry.Role != role {
				t.Errorf("%s role %s, want %s", entry.Name, entry.Role, role)
			}
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 {
		t.Fatal("missing nested signatures", want)
	}
}
