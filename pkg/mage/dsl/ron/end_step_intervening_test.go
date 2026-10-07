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

func TestRONEndStepInterveningPestilence(t *testing.T) {
	for _, scenario := range []string{"empty", "creature-before", "creature-after", "phased-creature", "phase-in-after", "source-phase-before", "source-phase-after", "source-return", "control", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEB", "Pestilence")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			var victim *mage.Permanent
			if strings.Contains(scenario, "creature") || scenario == "phase-in-after" {
				v := mage.NewCreature("Subject", "", 2, 2)
				v.SetOwner(b.PlayerID())
				victim = g.PutOnBattlefield(v, b.PlayerID())
				if scenario != "creature-before" && scenario != "creature-after" {
					g.PhaseOut(victim.ID())
				}
				if scenario == "creature-after" {
					g.BouncePermanentToHand(victim)
				}
			}
			if scenario == "source-phase-before" {
				g.PhaseOut(p.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			triggered := scenario != "creature-before" && scenario != "source-phase-before"
			if (g.StackSize() == 1) != triggered {
				t.Fatalf("stack size %d", g.StackSize())
			}
			if scenario == "creature-before" {
				g.DestroyPermanent(victim)
			}
			if scenario == "creature-after" {
				g.PutOnBattlefield(victim.Card, b.PlayerID())
			}
			if scenario == "phase-in-after" {
				g.PhaseIn([]uuid.UUID{victim.ID()})
			}
			if scenario == "source-phase-after" {
				g.PhaseOut(p.ID())
			}
			if scenario == "source-return" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
			}
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("clone changed source")
			}
			g.ResolveStack()
			sacrificed := triggered && scenario != "creature-after" && scenario != "phase-in-after" && scenario != "source-phase-after" && scenario != "source-return" && scenario != "control"
			for _, game := range []*mage.Game{g, clone} {
				if (game.FindPermanentIncludingPhased(p.ID()) == nil) != sacrificed {
					t.Fatalf("sacrifice=%v", sacrificed)
				}
			}
		})
	}
}

func TestRONEndStepInterveningDervish(t *testing.T) {
	for _, scenario := range []string{"damage", "none", "self", "zero", "prevented", "after-event", "control-before", "control-after", "return-before", "return-after", "phase-before", "phase-after", "phase-cycle", "copy", "copy-opponent", "cleanup"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEG", "Whirling Dervish")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			recipient := b
			if scenario == "self" {
				recipient = a
			}
			amount := 1
			if scenario == "zero" {
				amount = 0
			}
			if scenario == "prevented" {
				g.PreventAllDamageFrom(p.ID())
			}
			if scenario != "none" && scenario != "after-event" {
				g.DealDamageToPlayer(recipient, amount, p.ID())
			}
			if scenario == "return-before" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			}
			if scenario == "control-before" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "phase-before" {
				g.PhaseOut(p.ID())
			}
			if scenario == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			triggered := scenario != "none" && scenario != "self" && scenario != "zero" && scenario != "prevented" && scenario != "after-event" && scenario != "return-before" && scenario != "control-before" && scenario != "phase-before" && scenario != "cleanup"
			if (g.StackSize() == 1) != triggered {
				t.Fatalf("stack size %d", g.StackSize())
			}
			if scenario == "after-event" {
				g.DealDamageToPlayer(b, 1, p.ID())
			}
			if scenario == "control-after" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "return-after" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(c.ID())
				p = g.PutOnBattlefield(c, a.PlayerID())
			}
			if scenario == "phase-after" {
				g.PhaseOut(p.ID())
			}
			if scenario == "phase-cycle" {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			if scenario == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			if scenario == "copy-opponent" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P1] != 0 {
				t.Fatal("clone changed counters")
			}
			g.ResolveStack()
			want := 0
			if triggered && scenario != "return-after" && scenario != "phase-after" {
				want = 1
			}
			if scenario == "copy" {
				want = 2
			}
			for _, game := range []*mage.Game{g, clone} {
				if got := int(game.FindPermanentIncludingPhased(p.ID()).Counters[core.P1P1]); got != want {
					t.Fatalf("counters=%d want %d", got, want)
				}
			}
		})
	}
}

func TestRONEndStepInterveningDeclarations(t *testing.T) {
	for _, condition := range []string{
		`NoPermanentsPassFilter(IsCardtype(Creature))`, `NoPermanentsPassFilter(IsCardtype(Creature), InGraveyard)`,
		`NoPermanentsPassFilter(IsCardtype(Creature), OnTheBattlefield())`,
		`PermanentPassesFilter(Ref_TargetPermanent, DealtDamageToAnyPlayerThisTurn(Opponent))`,
		`PermanentPassesFilter(ThisPermanent, DealtDamageToAnyPlayerThisTurn(AnyPlayer))`,
		`PermanentPassesFilter(ThisPermanent, DealtDamageToAnyPlayerThisTurn(Opponent()))`,
		`PermanentPassesFilter(ThisPermanent, DealtDamageToAnyPlayerThisTurn)`,
		`ACreatureOrPlaneswalkerDiedThisTurn(IsCardtype(Planeswalker))`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),%s,ActionList([GainLife(Integer(1))]))])`, condition)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", condition)
		}
	}
	factory := permanentScopeFactory(t, "Artifact", `TriggerI(AtTheBeginningOfAPlayersEndStep(SinglePlayer(You)),NoPermanentsPassFilter(IsCardtype(Creature),OnTheBattlefield),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`)
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := mage.ValidateCardDefinition(p.Card.Name(), p.Card); err != nil {
		t.Fatal(err)
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("opponent step matched own step")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 20 {
		t.Fatal("event player changed")
	}
}

func TestRONEndStepInterveningNativeLoad(t *testing.T) {
	fs := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		var records []string
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name == "Pestilence" || name == "Whirling Dervish" {
				records = append(records, strings.Split(string(data), "\n")[n.Position.Line-1])
			}
		}
		fs[set+".ron"] = &fstest.MapFile{Data: []byte(strings.Join(records, "\n"))}
	}
	registry, err := LoadFS(fs, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("reprint count")
	}
	g, a, b := ronGame()
	for _, name := range registry.RegisteredCardNames() {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast succeeded")
		}
		color, amount := core.Black, 4
		if name == "Whirling Dervish" {
			color, amount = core.Green, 2
		}
		a.ManaPool().Add(color, amount)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	d := g.FilterBattlefield(mage.IsCreature)[0]
	g.DealDamageToPlayer(b, 1, d.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(d.ID()).Counters[core.P1P1] != 1 || len(g.FilterBattlefield(mage.IsEnchantment)) != 1 {
		t.Fatal("loaded cards did not execute")
	}
}

func TestRONEndStepInterveningPestilenceDamage(t *testing.T) {
	for _, scenario := range []string{"normal", "prevent-player", "source-leaves", "source-control", "source-phase", "phased-recipient"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "LEB", "Pestilence")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			d := repositoryCard(t, "LEG", "Whirling Dervish")
			d.SetOwner(b.PlayerID())
			protected := g.PutOnBattlefield(d, b.PlayerID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 4), a.PlayerID())
			g.SetStep(core.EndStep)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err == nil {
				t.Fatal("unpaid activation succeeded")
			}
			if g.StackSize() != 0 || a.Life() != 20 || victim.Damage != 0 {
				t.Fatal("failed activation changed state")
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if victim.Damage != 0 || a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("payment or resolution timing")
			}
			switch scenario {
			case "prevent-player":
				g.AddPreventionShield(b.PlayerID(), 1)
			case "source-leaves":
				g.BouncePermanentToHand(p)
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phase":
				g.PhaseOut(p.ID())
			case "phased-recipient":
				g.PhaseOut(victim.ID())
			}
			callbacks := 0
			g.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
				callbacks++
				wantB := 19
				if scenario == "prevent-player" {
					wantB = 20
				}
				if a.Life() != 19 || b.Life() != wantB {
					t.Error("damage events saw partial player results")
				}
				if scenario != "phased-recipient" && g.FindPermanent(victim.ID()).Damage != 1 {
					t.Error("damage events saw partial creature results")
				}
			})
			g.ResolveStack()
			wantCallbacks := 3
			if scenario == "prevent-player" || scenario == "phased-recipient" {
				wantCallbacks = 2
			}
			if callbacks != wantCallbacks || g.FindPermanent(protected.ID()).Damage != 0 {
				t.Fatalf("callbacks %d or protection failed", callbacks)
			}
			wantDamage := 1
			if scenario == "phased-recipient" {
				wantDamage = 0
			}
			if g.FindPermanentIncludingPhased(victim.ID()).Damage != wantDamage {
				t.Fatal("recipient damage")
			}
		})
	}
}

func TestRONEndStepInterveningDervishProtectionAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEG", "Whirling Dervish"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Whirling Dervish"), a.PlayerID())
	black := g.PutOnBattlefield(mage.NewCreature("Black", "{B}", 2, 2), b.PlayerID())
	g.DealDamageToPermanent(first, 2, black.ID())
	if g.FindPermanent(first.ID()).Damage != 0 || mage.CanBlock(black, first, g) {
		t.Fatal("black protection")
	}
	bolt := repositoryCard(t, "LEB", "Terror")
	bolt.SetOwner(b.PlayerID())
	b.AddToHand(bolt)
	b.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{first.ID()}, 0); err == nil {
		t.Fatal("black spell targeted protected creature")
	}
	aura := repositoryCard(t, "LEB", "Unholy Strength")
	aura.SetOwner(b.PlayerID())
	b.AddToHand(aura)
	if err := g.CastSpellByID(b.PlayerID(), aura.ID(), []uuid.UUID{first.ID()}, 0); err == nil {
		t.Fatal("black Aura targeted protected creature")
	}
	for range 3 {
		g.DealDamageToPlayer(b, 1, first.ID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("damage count changed trigger count")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[core.P1P1] != 1 || g.FindPermanent(second.ID()).Counters[core.P1P1] != 0 {
		t.Fatal("instances shared history")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[core.P1P1] != 2 {
		t.Fatal("opponent end step did not trigger")
	}
}

func TestRONEndStepInterveningAPNAPAndAbilityRemoval(t *testing.T) {
	for _, scenario := range []string{"normal", "removed-before", "removed-after"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			cards := []mage.Card{repositoryCard(t, "LEB", "Pestilence"), repositoryCard(t, "LEB", "Pestilence")}
			cards[0].SetOwner(a.PlayerID())
			cards[1].SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(cards[0], a.PlayerID())
			second := g.PutOnBattlefield(cards[1], b.PlayerID())
			if cards[0].Abilities()[0] == cards[1].Abilities()[0] {
				t.Fatal("shared trigger instance")
			}
			remove := func() {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), first.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
			}
			if scenario == "removed-before" {
				remove()
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			want := 2
			if scenario == "removed-before" {
				want = 1
			}
			if g.StackSize() != want || g.StackPeek().Controller != b.PlayerID() {
				t.Fatal("trigger detection or APNAP")
			}
			if scenario == "removed-after" {
				remove()
			}
			g.ResolveTopOfStack()
			if g.FindPermanent(second.ID()) != nil || g.FindPermanent(first.ID()) == nil {
				t.Fatal("triggers shared source")
			}
			g.ResolveStack()
			if (g.FindPermanent(first.ID()) != nil) != (scenario == "removed-before") {
				t.Fatal("ability removal changed pending trigger")
			}
		})
	}
}

func TestRONEndStepInterveningInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),NoPermanentsPassFilter(IsCardtype(Creature),OnTheBattlefield),ActionList([])),TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),PermanentPassesFilter(ThisPermanent,DealtDamageToAnyPlayerThisTurn(Opponent)),ActionList([]))],CardPT:(Power:1,Toughness:1))`
	report, err := InspectFS(fstest.MapFS{"inventory.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 0 {
		t.Fatal(report.Issues)
	}
	for name, role := range map[string]string{"NoPermanentsPassFilter": "trigger_condition", "IsCardtype": "permanent_filter", "OnTheBattlefield": "zone", "DealtDamageToAnyPlayerThisTurn": "permanent_filter", "Opponent": "player_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
	for _, rule := range []string{
		`TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer()),NoPermanentsPassFilter(IsCardtype(Creature),OnTheBattlefield),ActionList([]))`,
		`TriggerI(AtTheBeginningOfAPlayersEndStep(SinglePlayer(Opponent)),NoPermanentsPassFilter(IsCardtype(Creature),OnTheBattlefield),ActionList([]))`,
		`TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer),NoPermanentsPassFilter(IsCardtype(Creature),OnTheBattlefield),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))])),Activated(PayMana([]),ActionList([PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}
