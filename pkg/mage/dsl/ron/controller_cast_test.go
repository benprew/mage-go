package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONControllerCastEnchantress(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "opponent", "artifact", "failed", "phased", "phase-cycle", "removed", "departed", "returned", "control-before", "control-after", "spell-control", "countered", "spell-copy", "trigger-copy", "two-sources"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: scenario != "decline"}
			b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEA", "Verduran Enchantress")
			card.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			for range 4 {
				a.AddToLibrary(repositoryCard(t, "LEA", "Forest"))
				b.AddToLibrary(repositoryCard(t, "LEA", "Island"))
			}
			if scenario == "phased" || scenario == "phase-cycle" {
				group := g.PhaseOut(source.ID())
				if scenario == "phase-cycle" {
					g.PhaseIn(group)
				}
			}
			if scenario == "removed" {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
			}
			if scenario == "two-sources" {
				g.PutOnBattlefield(repositoryCard(t, "LEA", "Verduran Enchantress"), a.PlayerID())
			}
			caster := a.BasePlayer
			if scenario == "opponent" || scenario == "control-before" {
				caster = b.BasePlayer
				g.SetActivePlayerIndex(1)
			}
			if scenario == "control-before" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			spell := repositoryCard(t, "LEA", "Wild Growth")
			if scenario == "artifact" {
				spell = repositoryCard(t, "LEA", "Sol Ring")
			}
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), caster.PlayerID())
			if scenario == "failed" {
				land.Tapped = true
			}
			spell.SetOwner(caster.PlayerID())
			caster.AddToHand(spell)
			if scenario != "failed" {
				caster.ManaPool().Add(core.Green, 1)
				caster.ManaPool().Add(core.Colorless, 1)
			}
			targets := []uuid.UUID{land.ID()}
			if scenario == "artifact" {
				targets = nil
			}
			err := g.CastSpellByID(caster.PlayerID(), spell.ID(), targets, 0)
			if scenario == "failed" {
				if err == nil || g.StackSize() != 0 || len(caster.Hand()) != 1 {
					t.Fatal("failed casting changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			victim := g.StackPeek()
			if scenario == "departed" || scenario == "returned" {
				g.BouncePermanentToHand(source)
				if scenario == "returned" {
					g.PutOnBattlefield(source.Card, b.PlayerID())
				}
			}
			if scenario == "control-after" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), source.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "spell-control" {
				victim.Controller = b.PlayerID()
			}
			if scenario == "countered" {
				g.CounterStackObject(victim.ID)
			}
			if scenario == "spell-copy" {
				g.CopyStackObjectDirect(victim, a.PlayerID(), false)
			}
			g.PutTriggersOnStack()
			triggers := 1
			if scenario == "opponent" || scenario == "artifact" || scenario == "phased" || scenario == "removed" {
				triggers = 0
			}
			if scenario == "two-sources" {
				triggers = 2
			}
			spells := 1
			if scenario == "countered" {
				spells = 0
			}
			if scenario == "spell-copy" {
				spells = 2
			}
			if g.StackSize() != spells+triggers || a.offers != 0 || b.offers != 0 {
				t.Fatalf("stack=%d, want %d; premature optional choice", g.StackSize(), spells+triggers)
			}
			if scenario == "trigger-copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				triggers++
			}
			beforeA, beforeB := len(a.Hand()), len(b.Hand())
			clone := g.Clone()
			for range triggers {
				g.ResolveTopOfStack()
			}
			gainA, gainB := triggers, 0
			if scenario == "decline" {
				gainA = 0
			}
			if scenario == "control-before" {
				gainA, gainB = 0, 1
			}
			if scenario == "trigger-copy" {
				gainA, gainB = 1, 1
			}
			if len(a.Hand()) != beforeA+gainA || len(b.Hand()) != beforeB+gainB {
				t.Fatalf("draw gains=(%d,%d), want (%d,%d)", len(a.Hand())-beforeA, len(b.Hand())-beforeB, gainA, gainB)
			}
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != beforeA || len(clone.GetPlayer(b.PlayerID()).Hand()) != beforeB {
				t.Fatal("live resolution changed clone")
			}
			clone.SetPlayerAt(0, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: a.accept})
			clone.SetPlayerAt(1, &ronPaymentPlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*mage.SearchPlayer).BasePlayer, accept: b.accept})
			for range triggers {
				clone.ResolveTopOfStack()
			}
			if len(clone.GetPlayer(a.PlayerID()).Hand()) != beforeA+gainA || len(clone.GetPlayer(b.PlayerID()).Hand()) != beforeB+gainB {
				t.Fatal("clone lost pending controller or body")
			}
		})
	}
}

func TestRONControllerCastInvalid(t *testing.T) {
	for _, selector := range []string{"You", "SinglePlayer", "SinglePlayer()", "SinglePlayer(You,Opponent)", "SinglePlayer(You())", "SinglePlayer(Opponent)", "SinglePlayer(Trigger_ThatPlayer)", "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(ActionPlayer)", "SinglePlayer(ThisPermanent)", "AnyPlayer()", "Opponent()"} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(`+selector+`,AnySpell),ActionList([DrawACard]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", selector)
		}
	}
	for _, body := range []string{`MayAction`, `MayAction()`, `MayAction(DrawACard,DrawACard)`, `MayAction(DrawACard())`, `MayAction(DrawACard(Integer(1)))`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You),AnySpell),ActionList([`+body+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONControllerCastScopedOptionalDraw(t *testing.T) {
	a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("B"), accept: false}
	g := mage.NewGame(a, b)
	a.AddToLibrary(mage.NewLand("A card"))
	b.AddToLibrary(mage.NewLand("B card"))
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerActions(Ref_TargetPlayer,[MayAction(DrawACard)]),MayAction(DrawACard)])))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	if a.offers != 1 || b.offers != 1 || len(a.Hand()) != 1 || len(b.Hand()) != 0 {
		t.Fatal("optional draws lost player scope or independent choices")
	}
}

func TestRONControllerCastRecords(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Verduran Enchantress"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
		card := repositoryCard(t, set, "Verduran Enchantress")
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		other := repositoryCard(t, set, "Verduran Enchantress")
		if card.Abilities()[0].AbilityID() == other.Abilities()[0].AbilityID() {
			t.Fatal("shared ability declaration")
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 6 || report.UniqueCards != 1 || len(report.Issues) != 0 {
		t.Fatalf("reprint inventory: %v, %v", report, err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerA(WhenAPlayerCastsASpell(SinglePlayer(You),AnySpell),ActionList([MayAction(DrawACard())]))])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("unsupported optional draw published a partial registry")
	}
	if LoadInto(registry, files, "*.ron") == nil {
		t.Fatal("invalid replacement accepted")
	}
	if _, err := registry.CreateCard("Verduran Enchantress"); err != nil {
		t.Fatal("invalid replacement erased previous registry")
	}
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range nodes {
		name := n.Fields["Name"].String
		if name == "Mana Vortex" {
			if _, err := Compile(n); err != nil {
				t.Fatalf("self-cast record rejected: %v", err)
			}
		}
	}
}
