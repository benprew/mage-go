package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestDamageOutcomeCondition(t *testing.T) {
	for _, scenario := range []string{"normal", "partial", "prevented", "redirected", "zero", "wrong-source", "wrong-player", "departure", "return", "phase", "overwrite"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 5)
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), a.PlayerID())
			results := BindDamageResults("instruction")
			amount := 3
			if scenario == "zero" {
				amount = 0
			}
			if scenario == "partial" {
				g.AddPreventionShield(a.PlayerID(), 2)
			}
			if scenario == "prevented" {
				g.AddPreventionShield(a.PlayerID(), 3)
			}
			if scenario == "redirected" {
				g.effects.AddReplacement(&playerDamageRedirectReplacement{controllerID: a.PlayerID(), redirectPermID: other.ID()})
			}
			ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
			if err := DealDamageFrom(This, You, Fixed(amount), DamageOutcomesInto(results)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			saved := BindDamageSource("saved source")
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{saved: damageSourceFromPermanent(source)}
			expected := FromDamageSource(saved)
			var player PlayerRef = You
			switch scenario {
			case "wrong-source":
				g.resolution.damageSourceBindings[saved] = damageSourceFromPermanent(other)
			case "wrong-player":
				player = ActivePlayer
				g.SetActivePlayerIndex(1)
			case "departure":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				g.resolution.damageSourceBindings[saved] = damageSourceFromPermanent(returned)
			case "phase":
				g.PhaseOut(source.ID())
			case "overwrite":
				if err := DealDamageFrom(This, You, Fixed(0), DamageOutcomesInto(results)).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			before := a.Life()
			clone := g.Clone()
			body := IfDamageDealtToPlayer(results, expected, player, GainLifeFor(You, Fixed(1)))
			if err := body.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if scenario == "normal" || scenario == "partial" || scenario == "departure" || scenario == "phase" {
				want = 1
			}
			if a.Life() != before+want || clone.GetPlayer(a.PlayerID()).Life() != before {
				t.Fatal("outcome match or clone isolation")
			}
			if scenario == "redirected" && g.FindPermanent(other.ID()).Damage != 3 {
				t.Fatal("redirection did not use the actual recipient")
			}
			cloneCtx := *ctx
			cloneCtx.Game = clone
			if err := body.Apply(&cloneCtx); err != nil {
				t.Fatal(err)
			}
			if clone.GetPlayer(a.PlayerID()).Life() != before+want {
				t.Fatal("clone lost outcomes")
			}
		})
	}
}

func TestDamageOutcomeConditionDefinition(t *testing.T) {
	results := BindDamageResults("result")
	producer := DealDamageFrom(This, You, Fixed(1), DamageOutcomesInto(results))
	body := IfDamageDealtToPlayer(results, This, You, GainLifeFor(You, Fixed(1)))
	for _, tc := range []struct {
		name   string
		effect Effect
		valid  bool
	}{
		{"valid", Sequence(producer, body), true},
		{"unbound", body, false},
		{"nil-result", IfDamageDealtToPlayer(nil, This, You, GainLifeFor(You, Fixed(1))), false},
		{"nil-source", Sequence(producer, IfDamageDealtToPlayer(results, nil, You, GainLifeFor(You, Fixed(1)))), false},
		{"nil-player", Sequence(producer, IfDamageDealtToPlayer(results, This, nil, GainLifeFor(You, Fixed(1)))), false},
		{"nil-body", Sequence(producer, IfDamageDealtToPlayer(results, This, You, nil)), false},
		{"duplicate-output", DealDamageFrom(This, You, Fixed(1), DamageOutcomesInto(results), DamageOutcomesInto(results)), false},
		{"nil-player-set", DealDamageEach(This, PlayerDamageRecipients(nil), Fixed(1)), false},
	} {
		card := NewArtifact(tc.name, "", WithAbility(Activate(Costs(), tc.effect)))
		if err := ValidateCardDefinition(tc.name, card); (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestDamageOutcomeConditionWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	results := BindDamageResults("result")
	body := Sequence(DealDamageFrom(This, You, Fixed(0), DamageOutcomesInto(results)), IfDamageDealtToPlayer(results, This, You, GainLifeFor(You, Fixed(1))))
	if err := ApplyEffect(g, AtNext(EachEndStep, body), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	before := a.Life()
	found := false
	for _, ref := range g.WaitingDelayedPermanentSources() {
		if ref.ID() == source.ID() && ref.Incarnation() != uuid.Nil {
			found = true
		}
	}
	if !found || a.Life() != before {
		t.Fatal("waiting query lost identity or executed damage")
	}
}

func TestDamageOutcomeConditionBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	results, total := BindDamageResults("batch"), BindNumber("total")
	g.AddPreventionShield(a.PlayerID(), 1)
	g.AddPreventionShield(b.PlayerID(), 2)
	effect := Sequence(DealDamageEach(This, PlayerDamageRecipients(EachPlayer), Fixed(2), DamageResultsInto(results), DamageTotalInto(total)), IfDamageDealtToPlayer(results, This, You, GainLifeFor(You, FromNumber(total))))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("batch outcomes did not use actual damage")
	}
}

func TestDamageOutcomeConditionWaitingPlayerSet(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	saved := BindPermanent("saved")
	body := DealDamageEach(DamageSourceOf(From(saved)), PlayerDamageRecipients(OpponentsOf(ControllerOfWithLKI(This))), Fixed(0))
	if err := ApplyEffect(g, Sequence(Snapshot(This, saved), AtNext(EachEndStep, body)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	refs := g.WaitingDelayedPermanentSources()
	if len(refs) != 1 || refs[0].ID() != source.ID() || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("player-set source read or query mutation")
	}
}
