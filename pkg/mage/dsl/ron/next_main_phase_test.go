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

func TestRONNextMainPhaseManaDrain(t *testing.T) {
	for _, scenario := range []string{"precombat", "postcombat", "uncounterable", "zero", "illegal", "recast", "copy", "copy-delay", "counter-delay"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			mana := `[ManaCostX,ManaCostX,ManaCostU]`
			x, want := 2, 5
			if scenario == "zero" {
				mana = `[]`
				x, want = 0, 0
			}
			nodes, err := Parse(strings.NewReader(`Card(Name:"Victim",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),ManaCost:`+mana+`,Rules:[SpellActions(ActionList([GainLife(Integer(1))]))])`), "victim.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			victim := factory()
			if scenario == "uncounterable" {
				mage.WithUncounterable()(victim.(*mage.BaseCard))
			}
			victim.SetOwner(b.PlayerID())
			b.AddToHand(victim)
			b.ManaPool().Add(core.Blue, 5)
			if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, x); err != nil {
				t.Fatal(err)
			}
			spell := g.StackPeek()
			saved := g.CaptureSpellDamageSource(spell.ID)
			drain := repositoryCard(t, "LEG", "Mana Drain")
			drain.SetOwner(a.PlayerID())
			a.AddToHand(drain)
			if err := g.CastSpellByID(a.PlayerID(), drain.ID(), []uuid.UUID{spell.ID}, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 1 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("failed payment changed state")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), drain.ID(), []uuid.UUID{spell.ID}, 0); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().TotalMana() != 0 || len(g.DelayedTriggers()) != 0 {
				t.Fatal("wrong payment or early schedule")
			}
			if scenario == "illegal" {
				g.GetStack().RemoveByID(spell.ID)
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveTopOfStack()
			if scenario == "copy" {
				g.ResolveTopOfStack()
			}
			if scenario == "illegal" {
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("illegal target created a schedule")
				}
				return
			}
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("missing independent delayed instruction")
			}
			if (g.GetStack().FindByID(spell.ID) != nil) != (scenario == "uncounterable") {
				t.Fatal("wrong counter outcome")
			}
			if scenario == "uncounterable" {
				g.ResolveTopOfStack()
				if b.Life() != 21 {
					t.Fatal("uncounterable spell did not resolve")
				}
			}
			if scenario == "recast" {
				returned, ok := b.RemoveFromGraveyard(victim.ID())
				if !ok {
					t.Fatal("countered spell missing")
				}
				b.AddToHand(returned)
				b.ManaPool().Add(core.Blue, 19)
				if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, 9); err != nil {
					t.Fatal(err)
				}
			}
			b.ManaPool().Clear()
			for range 2 {
				if sources := g.WaitingDelayedSpellSources(); len(sources) != 1 || !sources[0].SameObject(saved) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("waiting target spell lost its saved identity or query ran the body", sources)
				}
			}
			controller, opponent := a.PlayerID(), b.PlayerID()
			if scenario == "copy" {
				controller, opponent = b.PlayerID(), a.PlayerID()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: opponent, Flag: true})
			if len(g.DelayedTriggers()) != 1 {
				t.Fatal("opponent's main consumed schedule")
			}
			g.RunStepWithPriority(core.Cleanup)
			clone := g.Clone()
			for _, game := range []*mage.Game{clone, g} {
				game.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: controller, Flag: scenario != "postcombat"})
				if len(game.DelayedTriggers()) != 0 || game.GetPlayer(controller).ManaPool().TotalMana() != 0 {
					t.Fatal("detection ran mana body or kept schedule")
				}
				game.PutTriggersOnStack()
				if len(game.WaitingDelayedSpellSources()) != 0 || len(game.StackBoundSpellSources()) != 1 {
					t.Fatal("waiting spell did not transfer to the stack frame")
				}
				if len(game.StackPeek().TargetSpecs) != 0 {
					t.Fatal("saved spell became a new target")
				}
				if scenario == "copy-delay" {
					if game.CopyStackObjectDirect(game.StackPeek(), opponent, false) == nil {
						t.Fatal("delayed trigger copy failed")
					}
					game.ResolveTopOfStack()
				}
				if scenario == "counter-delay" {
					game.GetStack().RemoveByID(game.StackPeek().ID)
				}
				game.ResolveTopOfStack()
				amount := want
				if scenario == "counter-delay" {
					amount = 0
				}
				if game.GetPlayer(controller).ManaPool().TotalMana() != amount || !game.GetPlayer(controller).ManaPool().CanPay(core.ParseManaCost(strings.Repeat("{C}", amount)), nil) {
					t.Fatal("wrong delayed mana amount or type")
				}
				otherAmount := 0
				if scenario == "copy-delay" {
					otherAmount = want
				}
				if game.GetPlayer(opponent).ManaPool().TotalMana() != otherAmount {
					t.Fatal("mana went to the wrong player")
				}
				game.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: controller})
				game.PutTriggersOnStack()
				if len(game.DelayedTriggers()) != 0 || game.GetPlayer(controller).ManaPool().TotalMana() != amount {
					t.Fatal("schedule repeated")
				}
				if game == clone && (len(g.DelayedTriggers()) != 1 || g.GetPlayer(controller).ManaPool().TotalMana() != 0) {
					t.Fatal("clone changed original")
				}
			}
		})
	}
}

func TestRONNextMainPhaseSeparateSpellValues(t *testing.T) {
	g, a, b := ronGame()
	for _, x := range []int{2, 4} {
		victim := mage.NewInstant("Victim", "{X}{U}")
		victim.SetOwner(b.PlayerID())
		b.AddToHand(victim)
		b.ManaPool().Add(core.Blue, x+1)
		if err := g.CastSpellByID(b.PlayerID(), victim.ID(), nil, x); err != nil {
			t.Fatal(err)
		}
		spell := g.StackPeek()
		drain := repositoryCard(t, "LEG", "Mana Drain")
		drain.SetOwner(a.PlayerID())
		a.AddToHand(drain)
		a.ManaPool().Add(core.Blue, 2)
		if err := g.CastSpellByID(a.PlayerID(), drain.ID(), []uuid.UUID{spell.ID}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	if len(g.DelayedTriggers()) != 2 {
		t.Fatal("independent spells lost schedules")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: a.PlayerID(), Flag: false})
	g.PutTriggersOnStack()
	clone := g.Clone()
	for _, game := range []*mage.Game{clone, g} {
		game.ResolveTopOfStack()
		amount := game.GetPlayer(a.PlayerID()).ManaPool().TotalMana()
		if amount != 3 && amount != 5 {
			t.Fatal("wrong saved value", amount)
		}
		game.ResolveTopOfStack()
		if game.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 8 {
			t.Fatal("saved spell values shared a frame")
		}
	}
}

func TestRONNextMainPhaseSpellBindings(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		g, a, b := ronGame()
		body := `AddManaRepeated(ManaProduceU,ManaValueOfSpell(Trigger_ThatSpell))`
		if delayed {
			body = `CreateFutureTrigger(AtTheBeginningOfPlayersNextMainPhase(You),ActionList([` + body + `]))`
		}
		factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPlayerCastsASpell(AnyPlayer,AnySpell),ActionList([`+body+`]))`)
		first, second := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
		if first.Card.Abilities()[0] == second.Card.Abilities()[0] {
			t.Fatal("shared ability instance")
		}
		victim := mage.NewInstant("Victim", "{X}{U}")
		victim.SetOwner(a.PlayerID())
		a.AddToHand(victim)
		a.ManaPool().Add(core.Blue, 3)
		if err := g.CastSpellByID(a.PlayerID(), victim.ID(), nil, 2); err != nil {
			t.Fatal(err)
		}
		spell := g.StackPeek()
		g.PutTriggersOnStack()
		g.GetStack().RemoveByID(spell.ID)
		g.ResolveStack()
		if delayed {
			if a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || len(g.DelayedTriggers()) != 2 {
				t.Fatal("delayed event binding ran early")
			}
			for _, id := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
				g.FireEvent(core.GameEvent{Type: core.EvtMainPhase, PlayerID: id})
				g.ResolveStack()
			}
		}
		if !a.ManaPool().CanPay(core.ParseManaCost("{U}{U}{U}"), nil) || !b.ManaPool().CanPay(core.ParseManaCost("{U}{U}{U}"), nil) || a.ManaPool().TotalMana() != 3 || b.ManaPool().TotalMana() != 3 {
			t.Fatal("event spell identity or player frame lost")
		}
	}
}

func TestRONNextMainPhaseInvalidBoundAmounts(t *testing.T) {
	for _, amount := range []string{`ManaValueOfSpell`, `ManaValueOfSpell()`, `ManaValueOfSpell(Ref_TargetSpell())`, `ManaValueOfSpell(ThisPermanent)`, `ManaValueOfSpell(Trigger_ThatSpell)`, `ManaValueOfSpell(Ref_TargetSpell,Ref_TargetSpell)`} {
		for _, production := range []string{`ManaProduceC`, `AnyManaColor`} {
			nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([AddManaRepeated(`+production+`,`+amount+`)])))])`), "bad.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid bound amount accepted", amount)
			}
		}
	}
	for _, production := range []string{`AnyManaColor`, `And([ManaProduceC,ManaProduceU])`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(AnySpell)],ActionList([AddManaRepeated(`+production+`,ManaValueOfSpell(Ref_TargetSpell))])))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unsupported production accepted", production)
		}
	}
}

func TestRONNextMainPhaseInvalid(t *testing.T) {
	valid := `CreateFutureTrigger(AtTheBeginningOfPlayersNextMainPhase(You),ActionList([GainLife(Integer(1))]))`
	for _, action := range []string{
		strings.Replace(valid, "(You)", "", 1),
		strings.Replace(valid, "(You)", "(You,You)", 1),
		strings.Replace(valid, "(You)", "(Opponent)", 1),
		strings.Replace(valid, "(You)", "(You())", 1),
		strings.Replace(valid, "[GainLife(Integer(1))]", "[]", 1),
		strings.Replace(valid, "GainLife(Integer(1))", "AddManaRepeated(ManaProduceC,ManaValueOfSpell(Ref_TargetSpell))", 1),
		`AddManaRepeated(ManaProduceC,ManaValueOfSpell(ThisPermanent))`,
		`AddManaRepeated(ManaProduceC,ManaValueOfSpell(Ref_TargetSpell()))`,
		`AddManaRepeated(ManaProduceC,ManaValueOfSpell())`,
		`AddManaRepeated(ManaProduceC,ManaValueOfSpell(Ref_TargetSpell,Ref_TargetSpell))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}

func TestRONNextMainPhaseLoad(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Mana Drain", "drain.ron")
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("factory instances share targets")
	}
	registry := mage.NewRegistry()
	raw, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.Contains(line, `Name: "Mana Drain"`) {
			record = line
			break
		}
	}
	if record == "" {
		t.Fatal("missing raw record")
	}
	files := fstest.MapFS{"a.ron": {Data: []byte(record)}}
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unsupported])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONNextMainPhaseInventory(t *testing.T) {
	node := repositoryRecord(t, "LEG", "Mana Drain", "inventory.ron")
	entries := map[string]*InventorySignature{}
	walkInventory(node, "Mana Drain", "Card", "card", "collection", true, entries)
	for name, role := range map[string]string{
		"AtTheBeginningOfPlayersNextMainPhase": "delayed_step_event",
		"You":                                  "player_reference",
		"ManaValueOfSpell":                     "value",
		"Ref_TargetSpell":                      "spell_reference",
	} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s missing role %s", name, role)
		}
	}
}
