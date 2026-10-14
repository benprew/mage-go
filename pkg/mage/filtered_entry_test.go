package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestFilteredEntryLifecycle(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewEnchantment("Entry rule", "", WithAbility(EntersTappedFor(OpponentsOf(You), Or(IsCreature, IsArtifact, IsLand))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	for _, tc := range []struct {
		card             Card
		opponent, tapped bool
	}{
		{NewCreature("Creature", "", 2, 2), true, true},
		{NewArtifact("Artifact", ""), true, true},
		{NewLand("Land"), true, true},
		{NewEnchantment("Enchantment", ""), true, false},
		{NewCreature("Own", "", 2, 2), false, false},
	} {
		controller := a.PlayerID()
		if tc.opponent {
			controller = b.PlayerID()
		}
		p := g.PutOnBattlefield(tc.card, controller)
		if p.Tapped != tc.tapped {
			t.Fatalf("%s tapped=%v", p.Name(), p.Tapped)
		}
	}
	clone := g.Clone()
	phased := g.PhaseOut(source.ID())
	if g.PutOnBattlefield(NewLand("Phased rule"), b.PlayerID()).Tapped {
		t.Fatal("phased source applied")
	}
	if !clone.PutOnBattlefield(NewLand("Clone rule"), b.PlayerID()).Tapped {
		t.Fatal("clone lost rule")
	}
	g.PhaseIn(phased)
	if !g.PutOnBattlefield(NewLand("Restored"), b.PlayerID()).Tapped {
		t.Fatal("phase in lost rule")
	}
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
	if g.PutOnBattlefield(NewLand("No ability"), b.PlayerID()).Tapped {
		t.Fatal("removed ability applied")
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if !g.PutOnBattlefield(NewLand("Ability restored"), b.PlayerID()).Tapped {
		t.Fatal("ability did not return")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if g.PutOnBattlefield(NewLand("Departed"), b.PlayerID()).Tapped {
		t.Fatal("departed source applied")
	}
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.incarnationID == source.incarnationID {
		t.Fatal("return reused visit")
	}
	if g.PutOnBattlefield(NewLand("New controller"), b.PlayerID()).Tapped || !g.PutOnBattlefield(NewLand("New opponent"), a.PlayerID()).Tapped {
		t.Fatal("rule retained old controller")
	}
}

func TestFilteredEntryDefinition(t *testing.T) {
	for _, c := range []Card{
		NewArtifact("Nil", "", WithAbility(EntersTappedFor(nil, AnyPermanent))),
		NewInstant("Wrong zone", "", WithAbility(EntersTappedFor(EachPlayer, AnyPermanent))),
	} {
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

func entryAlways() StateCondition {
	return StateConditionFunc("always", nil, func(StateReader, StateContext) bool { return true })
}

func TestFilteredEntryCharacteristics(t *testing.T) {
	for _, mode := range []string{"existing", "self", "general"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(EntersTappedFor(OpponentsOf(You), IsCreature))), a.PlayerID())
			var options []CardOption
			switch mode {
			case "existing":
				g.PutOnBattlefield(NewArtifact("Animator", "", WithAbility(Static(While(entryAlways(), AddEachCardType(MatchingPermanents(IsEnchantment), core.TypeCreature))))), a.PlayerID())
			case "self":
				options = []CardOption{WithAbility(Static(While(entryAlways(), AddCardType(This, core.TypeCreature))))}
			default:
				options = []CardOption{WithAbility(Static(While(entryAlways(), AddEachCardType(MatchingPermanents(IsEnchantment), core.TypeCreature))))}
			}
			p := g.PutOnBattlefield(NewEnchantment("Entrant", "", options...), b.PlayerID())
			if p.Tapped != (mode != "general") {
				t.Fatal("wrong entry characteristics", mode, p.Tapped)
			}
			if !p.HasType(core.TypeCreature) {
				t.Fatal("layer was lost after entry")
			}
		})
	}
}

func TestFilteredEntryCopyAndBatch(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	rule := NewEnchantment("Rule", "", WithAbility(EntersTappedFor(EachPlayer, IsArtifact)))
	g.PutOnBattlefield(rule, b.PlayerID())
	donorPlan, err := NewCardDefinitionPlan(func() Card { return NewEnchantment("Donor", "") })
	if err != nil {
		t.Fatal(err)
	}
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, b.PlayerID())
	physicalPlan, err := NewCardDefinitionPlan(func() Card {
		return NewArtifact("Physical", "", WithAbility(MayCopyAsEnters(NewPermanentFilter("donor", func(p *Permanent, _ *Game) bool { return p.Name() == "Donor" }))))
	})
	if err != nil {
		t.Fatal(err)
	}
	physical, err := physicalPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	copied := g.PutOnBattlefield(physical, a.PlayerID())
	if copied.Tapped || copied.HasType(core.TypeArtifact) {
		t.Fatal("matching ran before copy replacement")
	}
	var observed []bool
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if event.ToZone == core.ZoneBattlefield && event.SourceID != uuid.Nil {
			if p := reader.FindPermanent(event.SourceID); p != nil && p.IsToken {
				tokens := reader.(*Game).FilterBattlefield(IsToken)
				observed = append(observed, len(tokens) == 2 && tokens[0].Tapped && tokens[1].Tapped)
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, b.PlayerID())
	if err := ApplyEffect(g, CreateTokens(2, "Token", 1, 1, []core.CardType{core.TypeArtifact, core.TypeCreature}, nil), watcher.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(observed) != 2 || !observed[0] || !observed[1] {
		t.Fatal("entry observers saw an incomplete batch", observed)
	}
}

type filteredEntryChooser struct {
	*BasePlayer
	choice, choices int
	before          func()
}

func (p *filteredEntryChooser) ChooseMode(_ []string, _ string) int {
	p.choices++
	if p.before != nil {
		p.before()
	}
	return p.choice
}

func TestFilteredEntryReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &filteredEntryChooser{BasePlayer: NewBasePlayer("A"), choice: 1 - choice}
		b := &filteredEntryChooser{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(EntersTappedFor(OpponentsOf(You), IsArtifact))), a.PlayerID())
		g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action {
			action.(*battlefieldEntryAction).permanent.Tapped = false
			return action
		}})
		p := g.PutOnBattlefield(NewArtifact("Entrant", ""), b.PlayerID())
		if p.Tapped != (choice == 0) || a.choices != 0 || b.choices != 1 {
			t.Fatal("wrong affected player or replacement order", choice, p.Tapped, a.choices, b.choices)
		}
	}
}

func TestFilteredEntryChoiceRecheck(t *testing.T) {
	a := NewBasePlayer("A")
	b := &filteredEntryChooser{BasePlayer: NewBasePlayer("B"), choice: 1}
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(EntersTappedFor(OpponentsOf(You), IsArtifact))), a.PlayerID())
	g.effects.AddReplacement(&testPrioritizedReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, matchFunc: func(action Action) bool { _, ok := action.(*battlefieldEntryAction); return ok }, replaceFunc: func(action Action, _ *Game) Action { return action }})
	b.before = func() { g.RemoveFromBattlefield(g.FindPermanent(source.ID())) }
	if g.PutOnBattlefield(NewArtifact("Entrant", ""), b.PlayerID()).Tapped {
		t.Fatal("selected replacement used departed source")
	}
}

func TestFilteredEntryLiveControlAndProjectionPurity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(EntersTappedFor(OpponentsOf(You), IsCreature))), a.PlayerID())
	if err := ApplyEffect(g, Until(core.Indefinite, ChangeController(This, You)), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.PutOnBattlefield(NewCreature("New ally", "", 2, 2), b.PlayerID()).Tapped || !g.PutOnBattlefield(NewCreature("New opponent", "", 2, 2), a.PlayerID()).Tapped {
		t.Fatal("entry retained old source controller")
	}
	entrant := NewPermanent(NewCreature("Proposed", "", 2, 2), a.PlayerID())
	size, stack, life := len(g.AllBattlefield()), g.StackSize(), a.Life()
	projected, view := g.projectEntryPermanent(entrant)
	if projected.ID() != entrant.ID() || view.FindPermanent(entrant.ID()) == nil || g.FindPermanent(entrant.ID()) != nil {
		t.Fatal("projection published entrant")
	}
	if len(g.AllBattlefield()) != size || g.StackSize() != stack || a.Life() != life || source.Tapped {
		t.Fatal("projection changed live state")
	}
}
