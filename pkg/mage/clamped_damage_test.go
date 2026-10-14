package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestClampedDamageAmount(t *testing.T) {
	for _, scenario := range []string{"player", "creature", "marked", "walker", "partial", "prevented", "redirected", "departure", "return", "phase", "changed", "zero", "overwrite", "negative-life", "negative-toughness", "hybrid", "increased"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			victimCard := NewCreature("Victim", "", 1, 3)
			if scenario == "walker" || scenario == "hybrid" {
				victimCard = NewPlaneswalker("Walker", "", 3)
			}
			if scenario == "hybrid" {
				victimCard.AddType(core.TypeCreature)
			}
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			if scenario == "hybrid" {
				victim.SetCDAToughness(2)
			}
			if scenario == "negative-toughness" {
				victim.SetCDAToughness(-2)
			}
			if scenario == "increased" {
				g.effects.AddReplacement(&damageOutcomeReplacement{amount: 9})
			}
			saved := BindPermanent("recipient")
			results := BindDamageResults("damage")
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(&EffectContext{Game: g, SourceID: victim.ID(), Controller: b.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			var recipient DamageRecipientRef = From(saved)
			if scenario == "player" || scenario == "redirected" || scenario == "negative-life" {
				recipient = You
				a.SetLife(3)
				if scenario == "negative-life" {
					a.SetLife(-1)
				}
			}
			if scenario == "partial" {
				g.AddPreventionShield(victim.ID(), 4)
			}
			if scenario == "prevented" {
				g.AddPreventionShield(victim.ID(), 5)
			}
			if scenario == "marked" {
				victim.Damage = 2
			}
			if scenario == "redirected" {
				g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: a.PlayerID(), redirectPermID: victim.ID()})
				victim.Counters[core.P1P1] = 1
			}
			amount := 5
			if scenario == "zero" {
				amount = 0
			}
			if err := DealDamageFrom(This, recipient, Fixed(amount), DamageOutcomesInto(results)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "departure", "return":
				g.BouncePermanentToHand(g.FindPermanent(victim.ID()))
				if scenario == "return" {
					b.RemoveFromHand(victimCard.ID())
					returned := g.PutOnBattlefield(victimCard, b.PlayerID())
					returned.Counters[core.P1P1] = 9
				}
			case "phase":
				g.PhaseOut(victim.ID())
			case "changed":
				g.MutablePermanent(victim.ID()).Counters[core.P1P1] = 1
			case "overwrite":
				if err := DealDamageFrom(This, recipient, Fixed(0), DamageOutcomesInto(results)).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			value := ClampedDamageAmount(results)
			want := 3
			switch scenario {
			case "partial":
				want = 1
			case "prevented", "zero", "overwrite", "negative-life", "negative-toughness":
				want = 0
			case "hybrid":
				want = 2
			case "redirected", "changed":
				want = 4
			}
			if got := value.resolveValue(ctx); got != want {
				t.Fatalf("got %d want %d", got, want)
			}
			clone := g.Clone()
			copied := *ctx
			copied.Game = clone
			if got := value.resolveValue(&copied); got != want {
				t.Fatalf("clone got %d want %d", got, want)
			}
			if err := DealDamageFrom(This, recipient, Fixed(0), DamageOutcomesInto(results)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if value.resolveValue(ctx) != 0 || value.resolveValue(&copied) != want {
				t.Fatal("shared result state")
			}
		})
	}
}

func TestClampedDamageDefinition(t *testing.T) {
	results := BindDamageResults("damage")
	for _, tc := range []struct {
		name   string
		effect Effect
		valid  bool
	}{
		{"valid", Sequence(DealDamageFrom(This, You, Fixed(1), DamageOutcomesInto(results)), GainLifeFor(You, ClampedDamageAmount(results))), true},
		{"unbound", GainLifeFor(You, ClampedDamageAmount(results)), false},
		{"nil", GainLifeFor(You, ClampedDamageAmount(nil)), false},
	} {
		if err := ValidateCardDefinition(tc.name, NewArtifact(tc.name, "", WithAbility(Activate(Costs(), tc.effect)))); (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestClampedDamageDelayed(t *testing.T) {
	for _, event := range []bool{false, true} {
		t.Run(map[bool]string{false: "step", true: "departure"}[event], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 3), b.PlayerID())
			saved := BindPermanent("victim")
			results := BindDamageResults("damage")
			ctx := &EffectContext{Game: g, SourceID: victim.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			ctx.SourceID = source.ID()
			if err := DealDamageFrom(This, From(saved), Fixed(5), DamageOutcomesInto(results)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			body := GainLifeFor(You, ClampedDamageAmount(results))
			var delayed = AtNext(EachEndStep, body)
			if event {
				delayed = WhenLeavesThisTurn(From(saved), body)
			}
			if err := delayed.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if err := DealDamageFrom(This, You, Fixed(0), DamageOutcomesInto(results)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			before := a.Life()
			g.WaitingDelayedPermanentSources()
			if a.Life() != before || ClampedDamageAmount(results).resolveValue(ctx) != 0 {
				t.Fatal("waiting query changed results")
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				if event {
					game.BouncePermanentToHand(game.FindPermanent(victim.ID()))
				} else {
					game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				}
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != 23 {
					t.Fatal("lost delayed outcome")
				}
			}
		})
	}
}
