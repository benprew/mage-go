package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLandEntryAnkh(t *testing.T) {
	for _, change := range []string{"present", "control", "departed", "returned", "phased", "phase cycle", "source departed", "source returned", "source phased", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Ankh of Mishra")
			card.SetOwner(a.PlayerID())
			g.PutOnBattlefield(card, a.PlayerID())
			land := mage.NewLand("Land")
			land.SetOwner(a.PlayerID())
			g.PutOnBattlefield(land, b.PlayerID())
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("early damage")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("entry trigger")
			}
			switch change {
			case "control":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), land.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
				if change == "returned" {
					g.PutOnBattlefield(land, a.PlayerID())
				}
			case "phased", "phase cycle":
				g.PhaseOut(land.ID())
				if change == "phase cycle" {
					g.PhaseIn([]uuid.UUID{land.ID()})
				}
			case "source departed", "source returned":
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				if change == "source returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "source phased":
				g.PhaseOut(card.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			wantA, wantB := 20, 18
			if change == "control" {
				wantA, wantB = 18, 20
			}
			if change == "copy" {
				wantB = 16
			}
			if change == "returned" {
				wantA = 18
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life=%d,%d want=%d,%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONLandEntryChantEvents(t *testing.T) {
	for _, tc := range []struct{ name, subtype string }{{"Thelon's Chant", "Swamp"}, {"Tourach's Chant", "Forest"}} {
		t.Run(tc.name, func(t *testing.T) {
			n := hostUntapRecord(t, "FEM", tc.name)
			if _, err := Compile(n); err == nil {
				t.Fatal("full Chant unexpectedly accepted")
			}
			event := n.Fields["Rules"].Args[1].Args[0]
			nodes, err := Parse(strings.NewReader(`Card(Name:"Chant event",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(Placeholder,ActionList([PermanentDealsDamage(ThisPermanent,Integer(3),Player(Trigger_ThatPlayer))]))])`), "chant-event.ron")
			if err != nil {
				t.Fatal(err)
			}
			nodes[0].Fields["Rules"].Args[0].Args[0] = event
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			watcher := factory()
			if err := mage.ValidateCardDefinition(watcher.Name(), watcher); err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.PutOnBattlefield(watcher, a.PlayerID())
			g.PutOnBattlefield(mage.NewLand("Setup", mage.WithSubTypes(tc.subtype)), b.PlayerID())
			g.PutOnBattlefieldBy(mage.NewLand("Wrong", mage.WithSubTypes("Mountain")), b.PlayerID(), a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("unknown actor or wrong subtype triggered")
			}
			land := mage.NewLand("Matching", mage.WithSubTypes(tc.subtype))
			g.PutOnBattlefieldBy(land, b.PlayerID(), a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("actor entry trigger")
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy")
			}
			g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
			g.PutOnBattlefieldBy(land, a.PlayerID(), b.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			if a.Life() != 14 || b.Life() != 17 {
				t.Fatalf("actor capture: %d,%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONLandEntryInvalid(t *testing.T) {
	for _, event := range []string{
		`WhenAPermanentEntersTheBattlefield(IsCardtype(Creature))`,
		`WhenAPermanentEntersTheBattlefield(IsCardtype(Land()))`,
		`WhenAPermanentEntersTheBattlefield(IsCardtype(Land),AnyPlayer)`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(Opponent,IsLandType(Swamp))`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer(),IsLandType(Swamp))`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer,IsLandType(Goblin))`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer,IsLandType(UnknownLand))`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer,IsCardtype(Land))`,
		`WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer,IsLandType(Swamp()))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(` + event + `,ActionList([]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", event)
		}
	}
	for _, rules := range []string{
		`TriggerA(WhenAPermanentEntersTheBattlefield(IsCardtype(Land)),ActionList([TapPermanent(Trigger_ThatPermanent())]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(IsCardtype(Land)),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
		`TriggerA(WhenAPermanentEntersTheBattlefield(IsCardtype(Land)),ActionList([])),Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatPermanent)]))`,
		`TriggerA(WhenAPlayerPutsAPermanentOnTheBattlefield(AnyPlayer,IsLandType(Swamp)),ActionList([])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rules+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted scope leak %s", rules)
		}
	}
}

func TestRONLandEntryReprints(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		files[set+".ron"] = &fstest.MapFile{Data: damageReceivedRecord(t, set, "Ankh of Mishra")}
		card := repositoryCard(t, set, "Ankh of Mishra")
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprints not canonical")
	}
	files["FEM.ron"] = &fstest.MapFile{Data: append(append(damageReceivedRecord(t, "FEM", "Thelon's Chant"), '\n'), damageReceivedRecord(t, "FEM", "Tourach's Chant")...)}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Records != 8 || report.Compiled != 6 || len(report.Issues) != 2 {
		t.Fatalf("inventory: %+v %v", report, err)
	}
	for _, issue := range report.Issues {
		if !strings.Contains(issue.Message, "optional payment cost (PutCounters)") {
			t.Fatalf("unexpected Chant blocker: %+v", issue)
		}
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "AnyPlayer" && strings.Contains(sig.Context, "WhenAPlayerPutsAPermanentOnTheBattlefield") {
			found = true
			if sig.Role != "player_filter" {
				t.Fatalf("actor filter role: %s", sig.Role)
			}
		}
	}
	if !found {
		t.Fatal("missing actor inventory role")
	}
	delete(files, "FEM.ron")
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("collection rejection not atomic")
	}
}

func TestRONLandEntryCastAndNativeLand(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEB", "Ankh of Mishra")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 {
		t.Fatal("unpaid casting changed state")
	}
	a.ManaPool().Add(core.Colorless, 1)
	g.SetStep(core.Upkeep)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 2 {
		t.Fatal("illegal timing spent mana")
	}
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || g.StackSize() != 1 {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
		t.Fatal("cast or payment")
	}
	g.SetActivePlayerIndex(1)
	land := repositoryCard(t, "LEB", "Swamp")
	land.SetOwner(b.PlayerID())
	b.AddToHand(land)
	g.AddPreventionShield(b.PlayerID(), 1)
	if err := g.PlayLand(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 19 || g.FindPermanent(land.ID()) == nil {
		t.Fatal("native entry or prevention")
	}
	other := repositoryCard(t, "LEB", "Swamp")
	other.SetOwner(b.PlayerID())
	b.AddToHand(other)
	if err := g.PlayLand(b.PlayerID(), other.ID()); err == nil || b.Life() != 19 || g.FindPermanent(other.ID()) != nil {
		t.Fatal("illegal land triggered")
	}
}

func TestRONLandEntrySourcesAndPendingEvents(t *testing.T) {
	for _, disabled := range []string{"active", "phased", "removed abilities"} {
		t.Run(disabled, func(t *testing.T) {
			g, a, b := ronGame()
			first := g.PutOnBattlefield(repositoryCard(t, "LEB", "Ankh of Mishra"), a.PlayerID())
			g.PutOnBattlefield(repositoryCard(t, "LEB", "Ankh of Mishra"), b.PlayerID())
			if disabled == "phased" {
				g.PhaseOut(first.ID())
			}
			if disabled == "removed abilities" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(first.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			g.PutOnBattlefield(mage.NewArtifact("Nonland", ""), a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("nonland triggered")
			}
			one := g.PutOnBattlefield(mage.NewLand("One"), a.PlayerID())
			g.PutOnBattlefield(mage.NewLand("Two"), b.PlayerID())
			g.PutTriggersOnStack()
			count := 1
			if disabled == "active" {
				count = 2
			}
			if g.StackSize() != 2*count {
				t.Fatal("instance or pending event count")
			}
			g.PhaseOut(one.ID())
			g.PhaseIn([]uuid.UUID{one.ID()})
			if g.StackSize() != 2*count {
				t.Fatal("phasing fired entry")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed pending events")
			}
			g.ResolveStack()
			if a.Life() != 20-2*count || b.Life() != 20-2*count {
				t.Fatal("pending events shared subjects")
			}
		})
	}
}
