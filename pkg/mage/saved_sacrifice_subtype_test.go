package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSavedSacrificeSubtypeAction(t *testing.T) {
	for _, state := range []string{"normal", "exile", "cancel", "empty", "phase", "nonmatch", "replacement-type"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("Land", WithSubTypes("Island"))
			if state == "nonmatch" {
				card = NewLand("Land", WithSubTypes("Swamp"))
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			switch state {
			case "exile":
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
			case "cancel":
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			case "replacement-type":
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(a Action, g *Game) Action {
					g.MutablePermanent(p.ID()).SubTypeOverride = []string{"Swamp"}
					return a
				}})
			case "empty", "phase":
				g.PhaseOut(p.ID())
			}
			result := BindNumber("saved Island")
			body := SacrificeChosenBy(You, You, MatchingPermanents(IsLand), SacrificeSubtypeInto("Island", result))
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			g.resolution.numberBindings = map[*NumberBinding]int{result: 1}
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "normal" || state == "exile" {
				want = 1
			}
			if got := FromNumber(result).resolveValue(ctx); got != want {
				t.Fatalf("saved subtype %d want %d", got, want)
			}
			clone := g.Clone()
			g.resolution.numberBindings[result] = 9
			if got := FromNumber(result).resolveValue(&EffectContext{Game: clone, Controller: a.PlayerID()}); got != want {
				t.Fatal("clone shares results", got)
			}
		})
	}
}

func TestSavedSacrificeSubtypeCost(t *testing.T) {
	for _, state := range []string{"normal", "exile", "cancel", "rollback"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			result := BindNumber("paid Thrull")
			cost := SacrificeMatchingIncludingSourceCostWithSubtype(IsCreature, "Sacrifice a creature", "Thrull", result)
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(ManaCostOf("{1}"), cost), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2)))))), a.PlayerID())
			p := g.PutOnBattlefield(NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull")), a.PlayerID())
			if state == "exile" {
				g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), source.ID())
			}
			if state == "cancel" {
				g.effects.AddReplacement(&testPrioritizedReplacement{matchFunc: func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok }, replaceFunc: func(Action, *Game) Action { return nil }})
			}
			if state != "rollback" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if state == "rollback" {
				if err == nil || g.FindPermanent(p.ID()) == nil || g.StackSize() != 0 {
					t.Fatal("payment rollback", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 {
				t.Fatal("resolved while paying")
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 22
			if state == "cancel" {
				want = 20
			}
			if a.Life() != want {
				t.Fatal("paid facts", a.Life(), want)
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("clone lost paid facts")
			}
		})
	}
}

func TestSavedSacrificeSubtypeDefinition(t *testing.T) {
	result := BindNumber("saved subtype")
	producer := SacrificeChosenBy(You, You, MatchingPermanents(IsLand), SacrificeSubtypeInto("Island", result))
	read := IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1)))
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewSorcery("Good", "", Spell(Sequence(producer, read)))},
		{Name: "unbound", Card: NewSorcery("Bad", "", Spell(read)), WantErr: true},
		{Name: "nil output", Card: NewSorcery("Bad", "", Spell(SacrificeChosenBy(You, You, MatchingPermanents(IsLand), SacrificeSubtypeInto("Island", nil)))), WantErr: true},
		{Name: "empty subtype", Card: NewArtifact("Bad", "", WithAbility(Activate(Costs(SacrificeMatchingIncludingSourceCostWithSubtype(IsLand, "Sacrifice a land", "", result)), read))), WantErr: true},
		{Name: "nil paid output", Card: NewArtifact("Bad", "", WithAbility(Activate(Costs(SacrificeMatchingIncludingSourceCostWithSubtype(IsLand, "Sacrifice a land", "Island", nil)), read))), WantErr: true},
	})
}

func TestSavedSacrificeSubtypeWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewLand("Source", WithSubTypes("Island")), a.PlayerID())
	result := BindNumber("saved Island")
	body := Sequence(SacrificeChosenBy(ControllerOf(This), You, SingletonPermanent(This), SacrificeSubtypeInto("Island", result)), IfPositive(FromNumber(result), GainLifeFor(You, Fixed(2))))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || g.FindPermanent(source.ID()) == nil {
			t.Fatal("waiting query changed source")
		}
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(source.ID()) != nil || b.Life() != 22 {
		t.Fatal("saved result changed effect player")
	}
}

func TestSavedSacrificeSubtypeRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	chosen := g.PutOnBattlefield(NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull")), a.PlayerID())
	result := BindNumber("paid subtype")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), SacrificeMatchingIncludingSourceCostWithSubtype(IsCreature, "Sacrifice a creature", "Thrull", result)}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(chosen.ID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 || g.FindPermanentIncludingPhased(chosen.ID()) == nil {
		t.Fatal("stale subject did not roll back", err)
	}
}

func TestSavedSacrificeSubtypeIdentity(t *testing.T) {
	for _, state := range []string{"return", "phase", "control"} {
		t.Run(state, func(t *testing.T) {
			a := &chosenSacrificePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewLand("Other"), a.PlayerID())
			card := NewLand("Land", WithSubTypes("Island"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			a.choose = func(_ []*Permanent) *Permanent {
				switch state {
				case "return":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, a.PlayerID())
				case "phase":
					g.PhaseOut(p.ID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
				return p
			}
			result := BindNumber("saved Island")
			ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
			if err := SacrificeChosenBy(You, You, MatchingPermanents(IsLand), SacrificeSubtypeInto("Island", result)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if FromNumber(result).resolveValue(ctx) != 0 || g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("invalid visit supplied historical result")
			}
		})
	}
}
