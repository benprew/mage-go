package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDamageOutcomeSyphonSoul(t *testing.T) {
	for _, scenario := range []string{"normal", "partial", "prevented", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Syphon Soul")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed hand")
			}
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("damage before resolution")
			}
			if scenario == "partial" {
				g.AddPreventionShield(b.PlayerID(), 1)
			}
			if scenario == "prevented" {
				g.AddPreventionShield(b.PlayerID(), 2)
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			wantA, wantB := 22, 18
			switch scenario {
			case "partial":
				wantA, wantB = 21, 19
			case "prevented", "copy":
				wantA, wantB = 20, 20
			}
			if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("wrong actual total, recipients or copy controller")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("clone lost local results")
			}
		})
	}
}

func TestRONDamageOutcomeInventoryAndAtomicLoad(t *testing.T) {
	src := `Card(Name:"Outcomes",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),If(PermanentDealtDamageToPlayerThisWay(ThisPermanent,You),[GainLife(TheAmountOfDamageDealtThisWay)])]))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(src)}}
	files["players.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(src, `Name:"Outcomes"`, `Name:"Player outcomes"`), `Player(You)`, `EachPlayer(Other(You))`))}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 {
		t.Fatal("inventory", err)
	}
	source, player := false, false
	exclusion := false
	for _, sig := range report.Signatures {
		if sig.Name == "You" && strings.Contains(sig.Context, "Other[0]") && sig.Role == "player_reference" {
			exclusion = true
		}
		if !strings.Contains(sig.Context, "PermanentDealtDamageToPlayerThisWay") {
			continue
		}
		if sig.Name == "ThisPermanent" && sig.Role == "permanent_reference" {
			source = true
		}
		if sig.Name == "You" && sig.Role == "player_reference" {
			player = true
		}
	}
	if !source || !player || !exclusion {
		t.Fatal("outcome operands lack typed inventory roles")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(src, "TheAmountOfDamageDealtThisWay", "TheAmountOfDamageDealtThisWay()"))}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 0 {
		t.Fatal("partial registry published")
	}
}

func TestRONDamageOutcomeWarMachine(t *testing.T) {
	for _, set := range []string{"ATQ", "3ED"} {
		for _, scenario := range []string{"normal", "partial", "prevented", "paid", "opponent-upkeep", "control", "depart", "return", "phase", "copy"} {
			t.Run(set+"/"+scenario, func(t *testing.T) {
				a := &ronChosenCostPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario == "paid"}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				card := repositoryCard(t, set, "Mishra's War Machine")
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if !source.HasKeyword(core.Banding) || source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 5 {
					t.Fatal("metadata")
				}
				a.AddToHand(mage.NewLand("Discard"))
				if scenario == "partial" {
					g.AddPreventionShield(a.PlayerID(), 2)
				}
				if scenario == "prevented" {
					g.AddPreventionShield(a.PlayerID(), 3)
				}
				player := a.PlayerID()
				if scenario == "opponent-upkeep" {
					player = b.PlayerID()
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: player})
				g.PutTriggersOnStack()
				if scenario == "opponent-upkeep" {
					if g.StackSize() != 0 {
						t.Fatal("wrong upkeep")
					}
					return
				}
				if g.StackSize() != 1 || source.Tapped {
					t.Fatal("trigger timing")
				}
				switch scenario {
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "depart", "return":
					g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					if scenario == "return" {
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				want := 17
				if scenario == "partial" {
					want = 19
				}
				if scenario == "paid" || scenario == "prevented" {
					want = 20
				}
				if scenario == "copy" {
					want = 14
				}
				if a.Life() != want || b.Life() != 20 {
					t.Fatal("saved trigger player or actual damage")
				}
				live := g.FindPermanentIncludingPhased(source.ID())
				wantTapped := scenario != "paid" && scenario != "prevented" && scenario != "return" && scenario != "phase"
				if live != nil && live.Tapped != wantTapped {
					t.Fatal("outcome condition or exact source visit")
				}
				if scenario == "paid" && (len(a.Hand()) != 0 || len(a.Graveyard()) != 1) {
					t.Fatal("discard payment")
				}
				if clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("shared clone result")
				}
				clone.ResolveStack()
				cloneWant := want
				if scenario == "paid" {
					cloneWant = 17
				}
				if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
					t.Fatal("clone damage")
				}
			})
		}
	}
}

func TestRONDamageOutcomeScopes(t *testing.T) {
	for _, scenario := range []string{"latest", "branch", "delayed", "target", "collection"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			body := `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),SpellDealsDamage(ThisSpell,Integer(0),Player(You)),GainLife(TheAmountOfDamageDealtThisWay)`
			if scenario == "branch" {
				body = `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),If(PermanentPassesFilter(Ref_TargetPermanent,IsCardtype(Creature)),[SpellDealsDamage(ThisSpell,Integer(2),Player(You)),GainLife(TheAmountOfDamageDealtThisWay)]),GainLife(TheAmountOfDamageDealtThisWay)`
			}
			if scenario == "delayed" {
				body = `SpellDealsDamage(ThisSpell,Integer(3),Player(You)),CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([GainLife(TheAmountOfDamageDealtThisWay)]))`
			}
			rule := `SpellActions(ActionList([` + body + `]))`
			var targets []uuid.UUID
			if scenario == "collection" {
				first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 5), a.PlayerID())
				g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 5), b.PlayerID())
				g.AddPreventionShield(first.ID(), 1)
				rule = `SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(IsCardtype(Creature))),GainLife(TheAmountOfDamageDealtThisWay)]))`
			}
			if scenario == "branch" {
				victim := g.PutOnBattlefield(mage.NewCreature("Branch", "", 1, 5), b.PlayerID())
				targets = []uuid.UUID{victim.ID()}
				rule = `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([` + body + `])))`
			}
			if scenario == "target" {
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 5), b.PlayerID())
				targets = []uuid.UUID{victim.ID()}
				rule = `SpellActions(Targeted([AnyTarget],ActionList([SpellDealsDamage(ThisSpell,Integer(3),Ref_AnyTarget),GainLife(TheAmountOfDamageDealtThisWay)])))`
			}
			card := permanentScopeFactory(t, "Instant", rule)()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			want := 20
			if scenario == "latest" || scenario == "delayed" {
				want = 17
			}
			if scenario == "target" || scenario == "collection" {
				want = 23
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			if scenario == "delayed" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("delayed result lost")
				}
			}
		})
	}
}

func TestRONDamageOutcomeInvalid(t *testing.T) {
	for _, body := range []string{
		`GainLife(TheAmountOfDamageDealtThisWay)`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),GainLife(TheAmountOfDamageDealtThisWay())`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),GainLife(TheClampedAmountOfDamageDealtThisWay())`,
		`If(PermanentPassesFilter(ThisPermanent,IsCardtype(Artifact)),[PermanentDealsDamage(ThisPermanent,Integer(1),Player(You))]),GainLife(TheAmountOfDamageDealtThisWay)`,
		`If(PermanentDealtDamageToPlayerThisWay(ThisPermanent,You),[TapPermanent(ThisPermanent)])`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),If(PermanentDealtDamageToPlayerThisWay(ThisPermanent),[TapPermanent(ThisPermanent)])`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),If(PermanentDealtDamageToPlayerThisWay(ThisPermanent,ActionPlayer),[TapPermanent(ThisPermanent)])`,
		`PermanentDealsDamage(ThisPermanent,Integer(1),Player(You)),PermanentDealsMultipleDamage(ThisPermanent,[DamageToRecipients(Integer(2),Player(You))]),GainLife(TheAmountOfDamageDealtThisWay)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-outcome.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid result", body)
		}
	}
}

func TestRONDamageOutcomeContextBoundaries(t *testing.T) {
	condition, err := Parse(strings.NewReader(`If(PermanentDealtDamageToPlayerThisWay(ThisPermanent,You),[TapPermanent(ThisPermanent)])`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{
		{permanent: true, damageOutcome: &damageOutcomePlan{}, enteringSource: true},
		{permanent: true, damageOutcome: &damageOutcomePlan{}, drawReplacementBody: true},
	} {
		if _, err := compileInstanceValue(Node{Kind: Symbol, Name: "TheAmountOfDamageDealtThisWay"}, ctx); err == nil {
			t.Fatal("damage result escaped into entry or replacement context")
		}
		if _, err := compileDamageOutcomeBranch(condition[0], ctx); err == nil {
			t.Fatal("damage condition escaped into entry or replacement context")
		}
	}
}
