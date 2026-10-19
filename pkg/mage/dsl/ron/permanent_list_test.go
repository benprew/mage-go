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

type ronPermanentListPayer struct {
	*mage.BasePlayer
	pay     bool
	choices int
	check   func()
}

func (p *ronPermanentListPayer) ChooseMayAbility(string) bool {
	p.choices++
	if p.check != nil {
		p.check()
	}
	return p.pay
}

func TestRONPermanentListRohgahh(t *testing.T) {
	for _, scenario := range []string{"paid", "declined", "unaffordable", "source-left", "source-return", "source-phase", "subject-phase", "source-control", "copy", "clone", "countered"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronPermanentListPayer{BasePlayer: mage.NewBasePlayer("A"), pay: scenario == "paid" || scenario == "unaffordable"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetTurn(1)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Rohgahh of Kher Keep")
			addRONHandCard(a.BasePlayer, card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Colorless, 2)
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Red, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if source == nil || source.CurrentPower(g) != 5 || source.CurrentToughness(g) != 5 {
				t.Fatal("cast metadata")
			}
			own := g.PutOnBattlefield(repositoryCard(t, "LEG", "Kobolds of Kher Keep"), a.PlayerID())
			enemy := g.PutOnBattlefield(repositoryCard(t, "LEG", "Kobolds of Kher Keep"), b.PlayerID())
			g.TapPermanent(enemy)
			unrelated := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 4), a.PlayerID())
			if own.CurrentPower(g) != 2 || own.CurrentToughness(g) != 3 || enemy.CurrentPower(g) != 0 {
				t.Fatal("static bonus scope")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			if scenario != "unaffordable" {
				a.ManaPool().Add(core.Red, 3)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 || source.Tapped || a.choices != 0 {
				t.Fatal("trigger timing")
			}
			switch scenario {
			case "source-left":
				g.RemoveFromBattlefield(source)
			case "source-return":
				g.RemoveFromBattlefield(source)
				source = g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "subject-phase":
				g.PhaseOut(own.ID())
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "countered":
				g.CounterStackObject(g.StackPeek().ID)
			}
			clone := g.Clone()
			g.ResolveStack()
			if scenario == "countered" {
				if g.FindPermanent(own.ID()).ControllerID() != a.PlayerID() || a.choices != 0 {
					t.Fatal("countered trigger acted")
				}
				return
			}
			keep := scenario == "paid" || scenario == "subject-phase"
			want := b.PlayerID()
			if keep {
				want = a.PlayerID()
			}
			if scenario != "subject-phase" && g.FindPermanent(own.ID()).ControllerID() != want {
				t.Fatal("wrong control branch")
			}
			if scenario != "subject-phase" && g.FindPermanent(own.ID()).Tapped != (scenario != "paid") {
				t.Fatal("wrong tap branch")
			}
			if g.FindPermanent(unrelated.ID()).Tapped || g.FindPermanent(unrelated.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("unrelated creature affected")
			}
			if scenario == "paid" {
				if a.ManaPool().TotalMana() != 0 || g.FindPermanent(source.ID()).Tapped {
					t.Fatal("payment did not skip loss")
				}
			}
			if scenario == "source-return" && (g.FindPermanent(source.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(source.ID()).Tapped) {
				t.Fatal("returned source substituted")
			}
			if scenario == "source-phase" {
				g.PhaseIn([]uuid.UUID{source.ID()})
				if g.FindPermanent(source.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("phased source affected")
				}
			}
			if scenario == "subject-phase" {
				g.PhaseIn([]uuid.UUID{own.ID()})
				if g.FindPermanent(own.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("phased subject affected")
				}
			}
			if scenario != "paid" && scenario != "source-left" && scenario != "source-return" && scenario != "source-phase" {
				if g.FindPermanent(enemy.ID()).CurrentPower(g) != 2 {
					t.Fatal("bonus did not follow control")
				}
			}
			if scenario == "clone" {
				clone.ResolveStack()
				if clone.FindPermanent(own.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("pending clone lost set")
				}
				g.RemoveFromBattlefield(g.FindPermanent(own.ID()))
				if clone.FindPermanent(own.ID()) == nil {
					t.Fatal("clone shares battlefield")
				}
			}
			if scenario != "source-phase" && scenario != "source-left" && scenario != "source-return" && scenario != "paid" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(enemy.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("control ended with source")
				}
			}
		})
	}
}

func TestRONPermanentListDeclarations(t *testing.T) {
	valid := `CreatePermanentsList(IsCardtype(Creature)),TapEachPermanent(ThePermanentsList),ChooseAPlayer(Opponent),PlayerAction(TheChosenPlayer,GainControlOfEachPermanent(ThePermanentsList))`
	if permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([`+valid+`]))`)() == nil {
		t.Fatal("no factory")
	}
	for _, bad := range []string{
		`TapEachPermanent(ThePermanentsList)`, `GainControlOfEachPermanent(ThePermanentsList)`,
		strings.Replace(valid, `CreatePermanentsList(IsCardtype(Creature))`, `CreatePermanentsList`, 1),
		strings.Replace(valid, `CreatePermanentsList(IsCardtype(Creature))`, `CreatePermanentsList()`, 1),
		strings.Replace(valid, `CreatePermanentsList(IsCardtype(Creature))`, `CreatePermanentsList(IsCardtype(Creature),IsTapped)`, 1),
		strings.ReplaceAll(valid, `ThePermanentsList`, `ThePermanentsList()`),
		`MayCost(PayMana([])),If(CostWasPaid,[CreatePermanentsList(IsCardtype(Creature))]),TapEachPermanent(ThePermanentsList)`,
		`ChooseAnAction([ActionList([CreatePermanentsList(IsCardtype(Creature))]),ActionList([TapEachPermanent(ThePermanentsList)])])`,
		strings.Replace(valid, `GainControlOfEachPermanent(ThePermanentsList)`, `GainControlOfEachPermanent()`, 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+bad+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}

func TestRONPermanentListScopesAndInstances(t *testing.T) {
	g, a, b := ronGame()
	payer := &ronPermanentListPayer{BasePlayer: a, pay: true}
	g.SetPlayerAt(0, payer)
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePermanentsList(IsCardtype(Creature)),MayCost(PayMana([])),If(CostWasPaid,[CreatePermanentsList(IsCardtype(Land)),TapEachPermanent(ThePermanentsList)]),UntapEachPermanent(ThePermanentsList)]))`)
	first, second := factory(), factory()
	if first.ID() == second.ID() || first.Abilities()[0].AbilityID() == second.Abilities()[0].AbilityID() {
		t.Fatal("shared factory identity")
	}
	source := g.PutOnBattlefield(first, a.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 4), b.PlayerID())
	land := g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
	g.TapPermanent(creature)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(creature.ID()).Tapped || !g.FindPermanent(land.ID()).Tapped {
		t.Fatal("inner list overwrote outer binding")
	}
}

func TestRONPermanentListLoadAndInventory(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Rohgahh of Kher Keep", "leg.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Rohgahh of Kher Keep", factory()); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(source), "\n") {
		if strings.Contains(line, `Name: "Rohgahh of Kher Keep"`) {
			record = line
		}
	}
	files := fstest.MapFS{"good.ron": {Data: []byte(record)}, "bad.ron": {Data: []byte(`Card(Name:"Bad",Rules:[UnknownRule])`)}}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
	delete(files, "bad.ron")
	if registry, err := LoadFS(files, "*.ron"); err != nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("full record failed", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"CreatePermanentsList": "action", "ThePermanentsList": "permanent_collection_reference", "GainControlOfEachPermanent": "player_action", "NamedCard": "card_name"}
	for name, role := range roles {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s/%s", role, name)
		}
	}
}

func TestRONPermanentListCapturesBeforeTap(t *testing.T) {
	g, a, b := ronGame()
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePermanentsList(And([IsCardtype(Creature),IsUntapped])),TapEachPermanent(ThePermanentsList),ChooseAPlayer(Opponent),PlayerAction(TheChosenPlayer,GainControlOfEachPermanent(ThePermanentsList))]))`)
	source := g.PutOnBattlefield(factory(), a.PlayerID())
	creature := g.PutOnBattlefield(mage.NewCreature("Creature", "", 1, 4, mage.WithKeyword(core.Shroud)), a.PlayerID())
	tapped := g.PutOnBattlefield(mage.NewCreature("Tapped", "", 1, 4), a.PlayerID())
	g.TapPermanent(tapped)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(creature.ID()).ControllerID() != b.PlayerID() || !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("tap changed saved membership or untargeted control used targeting")
	}
	if g.FindPermanent(tapped.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("excluded member joined")
	}
	late := g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 4), a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(late.ID()).ControllerID() != a.PlayerID() {
		t.Fatal("later entry joined control effect")
	}
}
