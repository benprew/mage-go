package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedCollectionLifeSavedAmount(t *testing.T) {
	for _, power := range []int{-2, 0, 3} {
		t.Run(Fixed(power).Text(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := NewCreature("Amount", "", power, 4)
			g.PutOnBattlefield(source, a.PlayerID())
			saved := BindPermanent("amount")
			body := Sequence(Snapshot(This, saved), LoseLifeEach(EachPlayer, PowerOf(From(saved))))
			card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			events := 0
			watcher := NewTriggered(core.EvtLifeLost, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, game GameReader, _, _ uuid.UUID) bool {
				events++
				want := 20 - max(0, power)
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != want || evt.Amount != power {
					t.Error("event observed an incomplete life action")
				}
				return true
			})
			g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
			clone := g.Clone()
			if err := ApplyEffect(clone, body, source.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 20 - max(0, power)
			if clone.GetPlayer(a.PlayerID()).Life() != want || clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("saved amount was not applied to both players")
			}
			wantEvents := 0
			if power > 0 {
				wantEvents = 2
			}
			if events != wantEvents {
				t.Fatalf("events = %d, want %d", events, wantEvents)
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
		})
	}
}

func TestTypedCollectionLifeTargetAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	player := DeclarePlayerTarget("player")
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(LoseLifeEach(OpponentsOf(player), PowerOf(amount)), WithTargets(player, amount))
	card := NewSorcery("Life", "", WithAbility(ability))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: card.ID(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{b.PlayerID(), source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 20 {
		t.Fatalf("life = %d/%d", a.Life(), b.Life())
	}
}

type countedLifeAmount struct{ calls *int }

func (countedLifeAmount) Text() string { return "counted amount" }
func (v countedLifeAmount) resolveValue(ctx *EffectContext) int {
	*v.calls++
	return ctx.Game.GetPlayer(ctx.Controller).Life()
}

func TestTypedCollectionLifeAmountCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	amount := countedLifeAmount{&calls}
	if err := ApplyEffect(g, LoseLifeEach(OpponentsOf(ControllerOf(This)), amount), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("empty set evaluated its amount")
	}
	if err := ApplyEffect(g, LoseLifeEach(EachPlayer, amount), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || a.Life() != 0 || b.Life() != 0 {
		t.Fatalf("calls=%d, life=%d/%d", calls, a.Life(), b.Life())
	}
}

func TestTypedCollectionLifeAmountValidation(t *testing.T) {
	saved := BindPermanent("missing")
	target := DeclarePermanentTarget("undeclared", IsCreature)
	for _, amount := range []ResolutionValue{nil, PowerOf(nil), PowerOf(From(saved)), PowerOf(target), ValueAtResolution(nil)} {
		card := NewSorcery("Invalid life", "", WithAbility(Spell(LoseLifeEach(EachPlayer, amount))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "abilities[0].effects[0].amount") {
			t.Fatalf("validation = %v", err)
		}
	}
	info := InspectDefinition(LoseLifeEach(EachPlayer, PowerOf(This)))
	if info.Properties.Outcome != OutcomeDetriment || !info.Properties.Mass || len(info.Children) != 2 {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestTypedCollectionLifeCopiedAmount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Amount", "", 3, 4)
	g.PutOnBattlefield(source, a.PlayerID())
	amount := DeclarePermanentTarget("amount", IsCreature)
	ability := Spell(LoseLifeEach(OpponentsOf(You), PowerOf(amount)), WithTargets(amount))
	card := NewSorcery("Life", "", WithAbility(ability))
	original := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{source.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(original)
	clone := g.Clone()
	copied := clone.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	clone.ResolveStackObject(copied)
	if clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("copy lost its controller or amount target")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("copy changed original game")
	}
	g.ResolveStackObject(original)
	if a.Life() != 20 || b.Life() != 17 {
		t.Fatal("original lost its controller or amount target")
	}
}
