package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronStoragePlayer struct {
	*ronCounterPayer
	untap      bool
	untapCalls int
}

func (p *ronStoragePlayer) ChooseMayAbility(_ string) bool { p.untapCalls++; return p.untap }

func TestRONStorageLands(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color core.Color
	}{
		{"Bottomless Vault", core.Black}, {"Dwarven Hold", core.Red}, {"Hollow Trees", core.Green}, {"Icatian Store", core.White}, {"Sand Silos", core.Blue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A"), choice: 2}}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "FEM", tc.name)
			if err := mage.ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			c.SetOwner(a.PlayerID())
			g.SetStep(core.PrecombatMain)
			a.AddToHand(c)
			if err := g.PlayLand(a.PlayerID(), c.ID()); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(c.ID())
			if p == nil || !p.Tapped {
				t.Fatal("entry must be tapped")
			}
			g.RunStepWithPriority(core.Untap)
			if !g.FindPermanent(p.ID()).Tapped || a.untapCalls != 1 {
				t.Fatal("declined untap")
			}
			g.RunStepWithPriority(core.Upkeep)
			if g.FindPermanent(p.ID()).Counters[core.Storage] != 1 {
				t.Fatal("native upkeep did not add storage")
			}
			for range 2 {
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("opponent upkeep triggered")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("own upkeep missing trigger")
				}
				g.ResolveStack()
			}
			if g.FindPermanent(p.ID()).Counters[core.Storage] != 3 {
				t.Fatal("storage accumulation")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err == nil {
				t.Fatal("tapped activation accepted")
			}
			a.untap = true
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(p.ID()).Tapped || a.untapCalls != 2 {
				t.Fatal("accepted untap")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("untapped source triggered")
			}
			clone := g.Clone()
			if !mage.IsActivatedManaAbility(c.Abilities()[2]) {
				t.Fatal("mana classification")
			}
			if mage.ManaProductionsForAbilityInGame(c.Abilities()[2], g, p.ID()) != nil || a.calls != 0 {
				t.Fatal("solver consumed choice")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 0 || a.ManaPool().Count(tc.color) != 2 || b.ManaPool().TotalMana() != 0 || g.FindPermanent(p.ID()).Counters[core.Storage] != 1 || !g.FindPermanent(p.ID()).Tapped {
				t.Fatal("immediate payment and mana")
			}
			if clone.FindPermanent(p.ID()).Tapped || clone.FindPermanent(p.ID()).Counters[core.Storage] != 3 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("clone changed")
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			a.choice = 0
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(tc.color) != 2 || g.FindPermanent(p.ID()).Counters[core.Storage] != 1 {
				t.Fatal("zero payment")
			}
			g.UntapPermanent(g.FindPermanent(p.ID()))
			a.choice = 2
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 2, nil); err == nil {
				t.Fatal("excess payment accepted")
			}
			if g.FindPermanent(p.ID()).Tapped || g.FindPermanent(p.ID()).Counters[core.Storage] != 1 {
				t.Fatal("failed cost changed state")
			}
		})
	}
}

func TestRONStorageInterveningIf(t *testing.T) {
	for _, change := range []string{"none", "untap", "retap", "leave", "return", "phase-before", "phase-after", "control", "copy", "counter"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, "FEM", "Sand Silos")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			if change == "phase-before" {
				g.PhaseOut(p.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			if change == "phase-before" {
				if g.StackSize() != 0 {
					t.Fatal("phased source triggered")
				}
				return
			}
			if g.StackSize() != 1 {
				t.Fatal("missing trigger")
			}
			current := p.ID()
			switch change {
			case "untap", "retap":
				g.UntapPermanent(g.FindPermanent(p.ID()))
				if change == "retap" {
					g.TapPermanent(g.FindPermanent(p.ID()))
				}
			case "leave", "return":
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				if change == "return" {
					a.RemoveFromHand(c.ID())
					current = g.PutOnBattlefield(c, a.PlayerID()).ID()
				}
			case "phase-after":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "counter":
				g.GetStack().RemoveByID(g.StackPeek().ID)
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(current) != nil && g.FindPermanentIncludingPhased(current).Counters[core.Storage] != 0 {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			want := 0
			if change == "none" || change == "retap" || change == "control" {
				want = 1
			}
			if change == "copy" {
				want = 2
			}
			for _, game := range []*mage.Game{g, clone} {
				if q := game.FindPermanentIncludingPhased(current); q != nil && q.Counters[core.Storage] != want {
					t.Fatalf("counters=%d want %d", q.Counters[core.Storage], want)
				}
			}
		})
	}
}

func TestRONStorageInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent, [])`,
		`PermanentRuleEffect(ThisPermanent)`,
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), PermanentPassesFilter(ThisPermanent, IsTapped))`,
		`MayChooseNotToUntapDuringUntap`,
		`Activated(PermanentRuleEffect(ThisPermanent, [MayChooseNotToUntapDuringUntap]), ActionList([GainLife(Integer(1))]))`,
		`Activated(PayMana([]), ActionList([MayChooseNotToUntapDuringUntap]))`,
		`PermanentRuleEffect(Ref_TargetPermanent, [MayChooseNotToUntapDuringUntap])`,
		`PermanentRuleEffect(ThisPermanent(), [MayChooseNotToUntapDuringUntap])`,
		`PermanentRuleEffect(ThisPermanent, [MayChooseNotToUntapDuringUntap()])`,
		`PermanentRuleEffect(ThisPermanent, [MayChooseNotToUntapDuringUntap, DoesntUntapDuringControllersUntap])`,
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), PermanentPassesFilter(Ref_TargetPermanent, IsTapped), ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), PermanentPassesFilter(ThisPermanent, IsTapped()), ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(AnyPlayer), PermanentPassesFilter(ThisPermanent, IsTapped), ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheBeginningOfAPlayersEndStep(AnyPlayer), PermanentPassesFilter(ThisPermanent, IsTapped), ActionList([GainLife(Integer(1))]))`,
		`TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), PermanentPassesFilter(ThisPermanent, IsUntapped), ActionList([GainLife(Integer(1))]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONStorageInventory(t *testing.T) {
	source := `Card(Name:"Storage",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[MayChooseNotToUntapDuringUntap]),TriggerI(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),PermanentPassesFilter(ThisPermanent,IsTapped),ActionList([GainLife(Integer(1))]))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"MayChooseNotToUntapDuringUntap": "permanent_rule_operation", "PermanentPassesFilter": "trigger_condition", "IsTapped": "permanent_filter", "AtTheBeginningOfAPlayersUpkeep": "event", "GainLife": "action"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

func TestRONStorageUntapInstances(t *testing.T) {
	for _, scenario := range []string{"separate-choices", "control", "phase-in", "restriction", "remove-abilities"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("A")}, untap: false}
			b := &ronStoragePlayer{ronCounterPayer: &ronCounterPayer{BasePlayer: mage.NewBasePlayer("B")}, untap: true}
			g := mage.NewGame(a, b)
			c := repositoryCard(t, "FEM", "Sand Silos")
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			q := g.PutOnBattlefield(repositoryCard(t, "FEM", "Sand Silos"), b.PlayerID())
			g.MutablePermanent(p.ID()).AddCounter(core.Storage, 2)
			switch scenario {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(1)
			case "phase-in":
				g.PhaseOut(p.ID())
			case "restriction":
				g.SkipNextUntap(p.ID())
				a.untap = true
			case "remove-abilities":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Untap)
			wantTapped := scenario == "separate-choices" || scenario == "phase-in" || scenario == "restriction"
			if g.FindPermanent(p.ID()).Tapped != wantTapped || g.FindPermanent(p.ID()).Counters[core.Storage] != 2 {
				t.Fatal("untap choice, restriction, or counters")
			}
			if !clone.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("untap changed clone")
			}
			if scenario == "control" {
				if b.untapCalls != 2 || a.untapCalls != 0 || g.FindPermanent(q.ID()).Tapped {
					t.Fatal("current controller did not choose")
				}
			} else if !g.FindPermanent(q.ID()).Tapped || b.untapCalls != 0 {
				t.Fatal("inactive player untapped")
			}
			if scenario == "restriction" || scenario == "remove-abilities" {
				if a.untapCalls != 0 {
					t.Fatal("invalid choice prompt")
				}
			}
			if scenario == "separate-choices" {
				g.SetActivePlayerIndex(1)
				g.RunStepWithPriority(core.Untap)
				if g.FindPermanent(q.ID()).Tapped || !g.FindPermanent(p.ID()).Tapped || b.untapCalls != 1 {
					t.Fatal("factory choices shared")
				}
			}
		})
	}
}
