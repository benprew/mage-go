package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const ronDamageLifeFloor = `PlayerEffect(You,[DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead(Integer(1))])`

func TestRONDamageLifeFloorAli(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ARN", "Ali from Cairo")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Red, 4)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("paid cast failed")
	}
	source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 1, mage.WithKeyword(core.Lifelink)), b.PlayerID())
	g.DealDamageToPlayer(a, 25, source.ID())
	if a.Life() != 1 || b.Life() != 45 {
		t.Fatalf("life %d/%d", a.Life(), b.Life())
	}
	g.DealDamageToPlayer(a, 3, source.ID())
	if a.Life() != 1 || b.Life() != 48 {
		t.Fatal("damage at the floor was lost")
	}
	if err := mage.ApplyEffect(g, mage.LoseLifeFor(mage.You, mage.Fixed(1)), card.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 0 {
		t.Fatal("direct life loss was replaced")
	}
}

func TestRONDamageLifeFloorLifetime(t *testing.T) {
	for _, change := range []string{"live", "departure", "return", "phase", "phase-cycle", "control", "remove-ability", "expiry", "cleanup", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "Ali from Cairo")
			source := g.PutOnBattlefield(card, a.PlayerID())
			want := 1
			switch change {
			case "departure", "return":
				g.RemoveFromBattlefield(source)
				want = -5
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
					want = 1
				}
			case "phase", "phase-cycle":
				ids := g.PhaseOut(source.ID())
				want = -5
				if change == "phase-cycle" {
					g.PhaseIn(ids)
					want = 1
				}
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = -5
			case "remove-ability", "expiry":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
				want = -5
				if change == "expiry" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					want = 1
				}
			case "cleanup":
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
				clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 25, source.ID())
				if clone.GetPlayer(a.PlayerID()).Life() != -5 || a.Life() != 20 {
					t.Fatal("clone state shared")
				}
			}
			g.DealDamageToPlayer(a, 25, source.ID())
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if change == "control" {
				g.DealDamageToPlayer(b, 25, source.ID())
				if b.Life() != 1 {
					t.Fatal("current controller was not protected")
				}
			}
		})
	}
}

func TestRONDamageLifeFloorInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.ReplaceAll(ronDamageLifeFloor, "Integer(1)", "Integer(0)"),
		strings.ReplaceAll(ronDamageLifeFloor, "Integer(1)", "Integer(-1)"),
		strings.ReplaceAll(ronDamageLifeFloor, "Integer(1)", "ValueX"),
		strings.ReplaceAll(ronDamageLifeFloor, "Integer(1)", "Integer(1,2)"),
		strings.ReplaceAll(ronDamageLifeFloor, "You,", "Opponent,"),
		strings.ReplaceAll(ronDamageLifeFloor, "You,", "You(),"),
		`Activated(PayMana([]),ActionList([` + ronDamageLifeFloor + `]))`,
		`PlayerEffect(You,[DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead(Integer(1)),DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead(Integer(2))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[`+rule+`],CardPT:(Power:0,Toughness:1))`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONDamageLifeFloorCollection(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Ali from Cairo"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing complete record")
	}
	registry := mage.NewRegistry()
	files := fstest.MapFS{"ali.ron": {Data: []byte(record)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if !registry.CardRegistered("Ali from Cairo") {
		t.Fatal("missing card")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(record, "Ali from Cairo", "Invalid"), "Integer(1)", "Integer(0)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 || !registry.CardRegistered("Ali from Cairo") {
		t.Fatal("failed collection replaced registry")
	}
}
