package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticUntapCards(t *testing.T) {
	for _, name := range []string{"Stasis", "Winter Orb"} {
		for _, active := range []int{0, 1} {
			for _, scenario := range []string{"normal", "tapped", "departed", "phased", "abilities", "recipient-abilities", "control", "returned", "clone"} {
				t.Run(name+"/"+string(rune('A'+active))+"/"+scenario, func(t *testing.T) {
					_, a, b := ronGame()
					g := mage.NewGame(&ronPaymentPlayer{BasePlayer: a, accept: true}, &ronPaymentPlayer{BasePlayer: b, accept: true})
					g.SetTurn(3)
					g.SetStep(core.EndStep)
					g.SetActivePlayerIndex(active)
					card := repositoryCard(t, "LEA", name)
					card.SetOwner(a.PlayerID())
					source := g.PutOnBattlefield(card, a.PlayerID())
					player := g.ActivePlayerObj()
					first := g.PutOnBattlefield(mage.NewLand("First"), player.PlayerID())
					second := g.PutOnBattlefield(mage.NewLand("Second"), player.PlayerID())
					creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 4), player.PlayerID())
					for _, p := range []*mage.Permanent{first, second, creature} {
						g.TapPermanent(p)
					}
					blocked := true
					switch scenario {
					case "tapped":
						g.TapPermanent(source)
						blocked = name == "Stasis"
					case "departed":
						g.DestroyPermanent(source)
						blocked = false
					case "phased":
						g.PhaseOut(source.ID())
						blocked = name == "Winter Orb" && active == 0
					case "abilities":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						blocked = false
					case "recipient-abilities":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), second.ID(), player.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "returned":
						g.BouncePermanentToHand(source)
						a.RemoveFromHand(card.ID())
						g.PutOnBattlefield(card, a.PlayerID())
					}
					clone := g.Clone()
					if scenario == "clone" {
						clone.DestroyPermanent(clone.FindPermanent(source.ID()))
						clone.RunStepWithPriority(core.Untap)
						if clone.FindPermanent(second.ID()).Tapped {
							t.Fatal("clone retained removed source")
						}
					}
					g.RunStepWithPriority(core.Untap)
					if g.FindPermanent(second.ID()).Tapped != blocked {
						t.Fatal("wrong second-land untap")
					}
					if name == "Stasis" && blocked {
						if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(creature.ID()).Tapped || g.GetStep() != core.EndStep {
							t.Fatal("Stasis did not skip the whole step")
						}
					} else if g.FindPermanent(first.ID()).Tapped || g.FindPermanent(creature.ID()).Tapped {
						t.Fatal("wrong unrestricted untap")
					}
					if !g.UntapPermanent(g.FindPermanent(second.ID())) && blocked {
						t.Fatal("ordinary untap effect blocked")
					}
				})
			}
		}
	}
}

func TestRONStaticUntapCasting(t *testing.T) {
	for _, name := range []string{"Stasis", "Winter Orb"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEA", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast did not roll back")
			}
			a.ManaPool().Add(core.Blue, 1)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || g.FindPermanent(card.ID()) != nil {
				t.Fatal("casting cost or entry timing")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("permanent did not enter")
			}
		})
	}
}

func TestRONStasisUpkeep(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "unaffordable", "other-player", "control", "source-left", "source-returned", "pending-clone", "copied-trigger"} {
		t.Run(scenario, func(t *testing.T) {
			_, a, b := ronGame()
			g := mage.NewGame(&ronPaymentPlayer{BasePlayer: a, accept: scenario != "declined"}, &ronPaymentPlayer{BasePlayer: b, accept: true})
			g.SetTurn(3)
			card := repositoryCard(t, "LEA", "Stasis")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario == "other-player" {
				g.SetActivePlayerIndex(1)
			}
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			}
			payer := g.ActivePlayerObj()
			if scenario == "paid" || scenario == "declined" || scenario == "control" || scenario == "pending-clone" {
				payer.ManaPool().Add(core.Blue, 1)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: payer.PlayerID()})
			g.PutTriggersOnStack()
			if scenario == "other-player" {
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("missing upkeep trigger")
			}
			if scenario == "source-left" || scenario == "source-returned" {
				g.BouncePermanentToHand(source)
				if scenario == "source-returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			clone := g.Clone()
			if scenario == "copied-trigger" {
				b.ManaPool().Add(core.Blue, 1)
				if g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false) == nil {
					t.Fatal("trigger copy failed")
				}
				g.ResolveTopOfStack()
				if g.FindPermanent(source.ID()) == nil || b.ManaPool().TotalMana() != 0 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("copied trigger did not use its controller")
				}
			}
			g.ResolveStack()
			survives := scenario == "paid" || scenario == "control" || scenario == "source-returned" || scenario == "pending-clone"
			if (g.FindPermanent(source.ID()) != nil) != survives {
				t.Fatal("upkeep payment or source identity")
			}
			if scenario == "paid" || scenario == "control" {
				if payer.ManaPool().TotalMana() != 0 {
					t.Fatal("upkeep cost was not paid")
				}
			}
			if scenario == "declined" && payer.ManaPool().TotalMana() != 1 {
				t.Fatal("declined payment spent mana")
			}
			if scenario == "pending-clone" {
				clone.ResolveStack()
				if clone.FindPermanent(source.ID()) != nil || g.FindPermanent(source.ID()) == nil {
					t.Fatal("clone payment changed original trigger")
				}
			}
		})
	}
}

func TestRONStaticUntapDeclarations(t *testing.T) {
	valid := `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Land))])])`
	for _, rule := range []string{
		`EachPlayerEffect(AnyPlayer(),[SkipsUntapStep])`, `EachPlayerEffect(You,[SkipsUntapStep])`, `EachPlayerEffect(AnyPlayer,[SkipsUntapStep()])`, `EachPlayerEffect(AnyPlayer,[SkipsUntapStep,SkipsUntapStep])`,
		strings.Replace(valid, "Integer(1)", "Integer(-1)", 1), strings.Replace(valid, "Integer(1)", "ValueX", 1), strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1), strings.Replace(valid, "ThisPermanent", "HostPermanent", 1), strings.Replace(valid, "AnyPlayer", "SinglePlayer(You)", 1), strings.Replace(valid, "IsUntapped", "IsUntapped()", 1), strings.Replace(valid, "CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Land))", "SkipsUntapStep", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	factory := permanentScopeFactory(t, "Artifact", valid)
	if err := mage.ValidateCardDefinition("Valid", factory()); err != nil {
		t.Fatal(err)
	}
}

func TestRONStaticUntapInteractions(t *testing.T) {
	_, a, b := ronGame()
	g := mage.NewGame(&ronPaymentPlayer{BasePlayer: a, accept: true}, b)
	g.SetTurn(3)
	g.SetStep(core.EndStep)
	stasisCard := repositoryCard(t, "LEA", "Stasis")
	stasisCard.SetOwner(b.PlayerID())
	stasis := g.PutOnBattlefield(stasisCard, b.PlayerID())
	orbCard := repositoryCard(t, "LEA", "Winter Orb")
	orbCard.SetOwner(a.PlayerID())
	orb := g.PutOnBattlefield(orbCard, a.PlayerID())
	g.TapPermanent(orb)
	first := g.PutOnBattlefield(mage.NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewLand("Second"), a.PlayerID())
	g.TapPermanent(first)
	g.TapPermanent(second)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(orb.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || g.GetStep() != core.EndStep {
		t.Fatal("Stasis did not prevent the entire untap operation")
	}
	g.DestroyPermanent(stasis)
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(orb.ID()).Tapped || g.FindPermanent(first.ID()).Tapped || g.FindPermanent(second.ID()).Tapped {
		t.Fatal("tapped Orb changed its decision during simultaneous untap")
	}
	g.TapPermanent(first)
	g.TapPermanent(second)
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("untapped Orb did not restrict the next step")
	}
	g.TapPermanent(g.FindPermanent(first.ID()))
	another := repositoryCard(t, "LEA", "Winter Orb")
	another.SetOwner(b.PlayerID())
	other := g.PutOnBattlefield(another, b.PlayerID())
	g.TapPermanent(g.FindPermanent(orb.ID()))
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(second.ID()).Tapped {
		t.Fatal("second instance lost its condition")
	}
	g.TapPermanent(g.FindPermanent(first.ID()))
	g.PhaseOut(other.ID())
	g.TapPermanent(g.FindPermanent(orb.ID()))
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(second.ID()).Tapped {
		t.Fatal("phased opponent Orb applied a restriction")
	}
}

func TestRONStaticUntapReprintsAndInventory(t *testing.T) {
	files, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
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
			if name != "Stasis" && name != "Winter Orb" {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, factory()); err != nil {
				t.Fatal(err)
			}
			count++
		}
	}
	if count != 12 {
		t.Fatalf("records=%d", count)
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[EachPlayerEffect(AnyPlayer,[SkipsUntapStep]),If(PermanentPassesFilter(ThisPermanent,IsUntapped),[EachPlayerEffect(AnyPlayer,[CantUntapMoreThanNumberPermanentsDuringTheirUntapStep(Integer(1),IsCardtype(Land))])])])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("valid inventory record did not compile")
	}
	for name, role := range map[string]string{"SkipsUntapStep": "player_rule_operation", "IsUntapped": "permanent_filter", "EachPlayerEffect": "rule", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s in %s", name, role)
		}
	}
}

func TestRONStasisNativeTurn(t *testing.T) {
	_, a, b := ronGame()
	g := mage.NewGame(&ronPaymentPlayer{BasePlayer: a, accept: true}, b)
	g.SetTurn(2)
	g.SetStep(core.EndStep)
	card := repositoryCard(t, "LEA", "Stasis")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Old creature", "", 2, 4), a.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Tapped land"), a.PlayerID())
	g.TapPermanent(land)
	g.SetTurn(3)
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(land.ID()).Tapped || !g.FindPermanent(creature.ID()).CanDeclareAsAttacker(g) {
		t.Fatal("skipped untap changed turn-start control rules")
	}
	a.ManaPool().Add(core.Blue, 1)
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(source.ID()) == nil || g.StackSize() != 0 {
		t.Fatal("paid native upkeep did not resolve after skipped untap")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.FindPermanent(source.ID()) != nil || !g.FindPermanent(land.ID()).Tapped {
		t.Fatal("unpaid later upkeep did not sacrifice Stasis")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(land.ID()).Tapped {
		t.Fatal("untap did not resume after native sacrifice")
	}
}
