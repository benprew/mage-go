package ron

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONClampedDamageDrainLife(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, scenario := range []string{"player", "creature", "walker", "partial", "prevented", "zero", "illegal", "phase", "copy"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				factory, err := Compile(repositoryRecord(t, set, "Drain Life", set+".ron"))
				if err != nil {
					t.Fatal(err)
				}
				g, a, b := ronGame()
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				target := b.PlayerID()
				want := 3
				var victim *mage.Permanent
				if scenario == "creature" || scenario == "illegal" || scenario == "phase" {
					victim = g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 3), b.PlayerID())
					target = victim.ID()
				}
				if scenario == "walker" {
					victim = g.PutOnBattlefield(mage.NewPlaneswalker("Walker", "", 3), b.PlayerID())
					target = victim.ID()
				}
				b.SetLife(3)
				x := 5
				if scenario == "zero" {
					x = 0
					want = 0
				}
				if scenario == "partial" {
					g.AddPreventionShield(target, 4)
					want = 1
				}
				if scenario == "prevented" {
					g.AddPreventionShield(target, 5)
					want = 0
				}
				a.ManaPool().Add(core.Black, x+1)
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, x); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 20 || b.Life() != 3 {
					t.Fatal("early damage or gain")
				}
				if scenario == "illegal" {
					g.BouncePermanentToHand(victim)
					b.RemoveFromHand(victim.Card.ID())
					g.PutOnBattlefield(victim.Card, b.PlayerID())
					want = 0
				}
				if scenario == "phase" {
					g.PhaseOut(target)
					want = 0
				}
				if scenario == "copy" {
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if a.Life() != 20+want {
					t.Fatalf("gain %d want %d", a.Life()-20, want)
				}
				if scenario == "player" || scenario == "copy" {
					damage := 5
					if scenario == "copy" {
						damage = 10
					}
					if b.Life() != 3-damage {
						t.Fatal("damage was clamped")
					}
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20+want {
					t.Fatal("clone")
				}
			})
		}
	}
}

func TestRONClampedDamageInvalid(t *testing.T) {
	for _, body := range []string{
		`GainLife(TheClampedAmountOfDamageDealtThisWay)`,
		`SpellDealsDamage(ThisSpell,Integer(1),Player(You)),GainLife(TheClampedAmountOfDamageDealtThisWay())`,
		`SpellDealsDamage(ThisSpell,Integer(1),EachPlayer(AnyPlayer)),GainLife(TheClampedAmountOfDamageDealtThisWay)`,
		`If(PermanentPassesFilter(Ref_TargetPermanent,IsCardtype(Creature)),[SpellDealsDamage(ThisSpell,Integer(1),Player(You))]),GainLife(TheClampedAmountOfDamageDealtThisWay)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
	for _, ctx := range []scope{{damageOutcome: &damageOutcomePlan{}, enteringSource: true}, {damageOutcome: &damageOutcomePlan{}, drawReplacementBody: true}} {
		if _, err := compileInstanceValue(Node{Kind: Symbol, Name: "TheClampedAmountOfDamageDealtThisWay"}, ctx); err == nil {
			t.Fatal("invalid context")
		}
	}
}

func TestRONClampedDamageScopes(t *testing.T) {
	for _, scenario := range []string{"latest", "branch", "delayed", "instances"} {
		t.Run(scenario, func(t *testing.T) {
			body := `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),SpellDealsDamage(ThisSpell,Integer(0),Player(You)),GainLife(TheClampedAmountOfDamageDealtThisWay)`
			if scenario == "branch" {
				body = `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),If(PermanentPassesFilter(Ref_TargetPermanent,IsCardtype(Creature)),[SpellDealsDamage(ThisSpell,Integer(2),Player(You)),GainLife(TheClampedAmountOfDamageDealtThisWay)]),GainLife(TheClampedAmountOfDamageDealtThisWay)`
			}
			if scenario == "delayed" {
				body = `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheClampedAmountOfDamageDealtThisWay)]))`
			}
			if scenario == "instances" {
				body = `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),GainLife(TheClampedAmountOfDamageDealtThisWay)`
			}
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+body+`])))`)
			g, a, b := ronGame()
			victim := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 5), b.PlayerID())
			count := 1
			if scenario == "instances" {
				count = 2
			}
			for range count {
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			want := 20
			if scenario == "latest" || scenario == "delayed" {
				want = 17
			}
			if a.Life() != want {
				t.Fatalf("life %d want %d", a.Life(), want)
			}
			if scenario == "delayed" {
				clone := g.Clone()
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("lost delayed cap")
				}
				clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("clone lost delayed cap")
				}
			}
		})
	}
}

func TestRONClampedDamageAtomicLoad(t *testing.T) {
	data, err := fs.ReadFile(repositoryFixtures.files, "LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Drain Life"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing record")
	}
	files := fstest.MapFS{"valid.ron": {Data: []byte(record)}}
	registry := mage.NewRegistry()
	registry.Register("Retained", func() mage.Card { return mage.NewLand("Retained") })
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(record, `"Drain Life"`, `"Bad"`), "TheClampedAmountOfDamageDealtThisWay", "TheClampedAmountOfDamageDealtThisWay()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Retained") || registry.CardRegistered("Drain Life") {
		t.Fatal("partial collection published", err)
	}
	delete(files, "bad.ron")
	if err := LoadInto(registry, files, "*.ron"); err != nil || !registry.CardRegistered("Drain Life") {
		t.Fatal("complete record load", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TheClampedAmountOfDamageDealtThisWay" && signature.Role == "value" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing value role")
	}
}
