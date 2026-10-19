package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func finiteRedirectFixture(t *testing.T, budget int) (*Game, *BasePlayer, *preventionAllocationPlayer, *Permanent, *Permanent) {
	t.Helper()
	a := NewBasePlayer("Owner")
	b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("Controller"), numbers: []int{1}}
	g := NewGame(a, b)
	card := NewCreature("Recipient", "", 6, 20)
	card.SetOwner(a.PlayerID())
	subject := g.PutOnBattlefield(card, b.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Damage source", "", 1, 20), b.PlayerID())
	if err := ApplyEffect(g, RedirectNextDamageToPlayer(This, You, Fixed(budget)), subject.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	return g, a, b, subject, source
}

func TestFiniteDamageRedirectBudget(t *testing.T) {
	g, a, b, subject, source := finiteRedirectFixture(t, 3)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if a.Life() != 17 || subject.Damage != 2 {
			t.Error("notification saw partial split")
		}
	})
	outcome := g.damage.dealDamageToPermanentFromSourceOutcome(g, subject, 5, g.CaptureDamageSource(source.ID()))
	if a.Life() != 17 || b.Life() != 20 || subject.Damage != 2 || outcome.amount != 5 || observed != 2 {
		t.Fatalf("life %d/%d damage %d result %+v notifications %d", a.Life(), b.Life(), subject.Damage, outcome, observed)
	}
	g.damage.SetOnDamageDealt(nil)
	g.DealDamageToPermanent(subject, 2, source.ID())
	if subject.Damage != 4 || a.Life() != 17 {
		t.Fatal("exhausted budget remained active")
	}
}

func TestFiniteDamageRedirectAllocation(t *testing.T) {
	for _, choice := range []int{0, 1, 99} {
		t.Run(string(rune('A'+choice)), func(t *testing.T) {
			g, a, b, subject, first := finiteRedirectFixture(t, 3)
			b.numbers = []int{choice}
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 20), b.PlayerID())
			outcomes := g.dealSimultaneousDamageOutcomes(PermanentDamageAssignment(subject, 3, g.CaptureDamageSource(first.ID())), PermanentDamageAssignment(subject, 3, g.CaptureDamageSource(second.ID())))
			if b.numberCalls != 1 || a.Life() != 17 || subject.Damage != 3 {
				t.Fatal("allocation used wrong chooser or budget")
			}
			totals := map[uuid.UUID]int{}
			for _, outcome := range outcomes {
				totals[outcome.source.ID()] += outcome.amount
			}
			if totals[first.ID()] != 3 || totals[second.ID()] != 3 {
				t.Fatalf("lost split outcomes: %+v", outcomes)
			}
			redirected := choice
			if choice == 99 {
				redirected = 3
			}
			got := 0
			for _, outcome := range outcomes {
				if outcome.source.ID() == first.ID() && outcome.playerID == a.PlayerID() {
					got += outcome.amount
				}
			}
			if got != redirected {
				t.Fatalf("redirected %d want %d", got, redirected)
			}
		})
	}
}

func TestFiniteDamageRedirectLifecycle(t *testing.T) {
	for _, state := range []string{"clone", "phase", "phase-cycle", "return", "cleanup", "control", "zero"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, subject, source := finiteRedirectFixture(t, 3)
			switch state {
			case "phase":
				g.PhaseOut(subject.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(subject.ID()))
			case "return":
				g.RemoveFromBattlefield(subject)
				subject = g.PutOnBattlefield(subject.Card, b.PlayerID())
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: subject.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "zero":
				g.DealDamageToPermanent(subject, 0, source.ID())
			}
			baseline := g.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				game.DealDamageToPermanent(game.FindPermanent(subject.ID()), 2, source.ID())
				want := 18
				if state == "phase" || state == "return" || state == "cleanup" {
					want = 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatalf("owner life %d want %d", game.GetPlayer(a.PlayerID()).Life(), want)
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone shared redirect budget")
			}
		})
	}
}

func TestFiniteDamageRedirectOrdering(t *testing.T) {
	for _, order := range []string{"prevent-first", "redirect-first", "double-first", "double-after", "blocked"} {
		t.Run(order, func(t *testing.T) {
			g, a, _, subject, source := finiteRedirectFixture(t, 2)
			switch order {
			case "prevent-first":
				g.AddPreventionShield(subject.ID(), 2)
				g.effects.PrependReplacement(g.effects.replacements[len(g.effects.replacements)-1])
			case "redirect-first":
				g.AddPreventionShield(a.PlayerID(), 1)
			case "double-first":
				g.effects.PrependReplacement(&continuedDamageDouble{})
			case "double-after":
				g.effects.AddReplacement(&continuedDamageDouble{})
			case "blocked":
				subject.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			}
			g.DealDamageToPermanent(subject, 4, source.ID())
			life, marked := 18, 2
			switch order {
			case "prevent-first":
				marked = 0
			case "redirect-first":
				life = 19
			case "double-first":
				marked = 6
			case "double-after":
				life = 16
				marked = 4
			case "blocked":
				life = 20
				marked = 4
			}
			if a.Life() != life || subject.Damage != marked {
				t.Fatalf("life %d damage %d want %d/%d", a.Life(), subject.Damage, life, marked)
			}
		})
	}
}

func TestFiniteDamageRedirectInvalid(t *testing.T) {
	for _, effect := range []ResolvingEffect{RedirectNextDamageToPlayer(nil, You, Fixed(1)), RedirectNextDamageToPlayer(This, nil, Fixed(1)), RedirectNextDamageToPlayer(This, You, nil), RedirectNextDamageToPlayer(From(BindPermanent("missing")), You, Fixed(1))} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestFiniteDamageRedirectResults(t *testing.T) {
	g, a, b, subject, source := finiteRedirectFixture(t, 2)
	results, amount, player, permanent, savedSource := BindDamageResults("split"), BindNumber("amount"), BindPlayer("recipient"), BindPermanent("recipient"), BindDamageSource("source")
	effect := DealDamageFrom(DamageSourceOf(This), From(permanent), Fixed(4), DamageOutcomesInto(results), DamageAmountInto(amount), DamagePlayerInto(player), DamageSourceInto(savedSource))
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{permanent: {subject.ID(), subject.incarnationID}}
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: b.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	outcomes := g.resolution.damageResultsBindings[results]
	if len(outcomes) != 2 || g.resolution.numberBindings[amount] != 4 || a.Life() != 18 || subject.Damage != 2 {
		t.Fatalf("incorrect split results %+v", outcomes)
	}
	if _, ok := g.resolution.playerBindings[player]; ok {
		t.Fatal("ambiguous recipient supplied a player")
	}
	if !g.resolution.damageSourceBindings[savedSource].SameObject(g.CaptureDamageSource(source.ID())) {
		t.Fatal("split lost source result")
	}
	clone := g.Clone()
	clone.resolution.damageResultsBindings[results][0].source.card.SetOwner(a.PlayerID())
	if g.resolution.damageResultsBindings[results][0].source.card.Owner() == a.PlayerID() {
		t.Fatal("clone shared result source")
	}
}

func TestFiniteDamageRedirectFactsAndWaiting(t *testing.T) {
	g, a, b, subject, source := finiteRedirectFixture(t, 2)
	savedSubject := g.CaptureDamageSource(subject.ID())
	g.RemoveFromBattlefield(source)
	source = g.PutOnBattlefield(NewCreature("Lifelink source", "", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.RemoveFromBattlefield(source)
	g.PutOnBattlefield(source.Card, a.PlayerID())
	cause := damageCause{stackID: uuid.New()}
	action := NewDamageToCreatureAction(saved.ID(), subject.ID(), 4, true).withDamageSource(saved)
	action.cause = cause
	actions, _ := g.effects.prepareDamageBatchReplacements([]Action{action}, g)
	if len(actions) != 2 {
		t.Fatal("missing split")
	}
	for _, packet := range actions {
		if !damageActionSource(packet).SameObject(saved) || damageActionCause(packet).stackID != cause.stackID {
			t.Fatal("source or cause lost")
		}
		switch p := packet.(type) {
		case *DamageToPlayerAction:
			if !p.IsCombatDamage() {
				t.Fatal("combat fact lost")
			}
		case *DamageToCreatureAction:
			if !p.IsCombatDamage() {
				t.Fatal("combat fact lost")
			}
		}
	}
	g.damage.executePreparedDamageActions(g, actions)
	if a.Life() != 18 || b.Life() != 24 || subject.Damage != 2 || g.DamageTakenByPlayer(a.PlayerID()) != 2 {
		t.Fatal("split lost LKI lifelink, history or damage")
	}
	if len(g.WaitingRedirectionPermanentSources()) != 0 {
		t.Fatal("consumed reference remained")
	}
	if err := ApplyEffect(g, RedirectNextDamageToPlayer(This, You, Fixed(2)), subject.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(subject)
	for range 2 {
		refs := g.WaitingRedirectionPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(savedSubject) || a.Life() != 18 {
			t.Fatal("waiting read lost exact absent subject or changed state")
		}
	}
}

func TestFiniteDamageRedirectBudgetRetention(t *testing.T) {
	for _, kind := range []string{"prevented", "blocked", "partial", "two-rules", "local"} {
		t.Run(kind, func(t *testing.T) {
			g, a, _, subject, source := finiteRedirectFixture(t, 3)
			switch kind {
			case "prevented":
				g.AddPreventionShield(subject.ID(), 2)
				g.effects.PrependReplacement(g.effects.replacements[len(g.effects.replacements)-1])
				g.DealDamageToPermanent(subject, 2, source.ID())
			case "blocked":
				subject.GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				g.DealDamageToPermanent(subject, 2, source.ID())
				subject.RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
			case "partial":
				g.DealDamageToPermanent(subject, 1, source.ID())
			case "two-rules":
				if err := ApplyEffect(g, RedirectNextDamageToPlayer(This, You, Fixed(1)), subject.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "local":
				if err := ApplyEffect(g, DealDamageWithPreventionFrom(DamageSourceOf(This), This, Fixed(5), Fixed(1)), subject.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.Life() != 18 || subject.Damage != 2 {
					t.Fatalf("instruction prevention lost redirected packet: %d/%d", a.Life(), subject.Damage)
				}
				return
			}
			g.DealDamageToPermanent(subject, 4, source.ID())
			life, damage := 17, 1
			switch kind {
			case "blocked":
				damage = 3
			case "partial":
				damage = 2
			case "two-rules":
				life = 16
				damage = 0
			}
			if a.Life() != life || subject.Damage != damage {
				t.Fatalf("life %d damage %d want %d/%d", a.Life(), subject.Damage, life, damage)
			}
		})
	}
}

func TestFiniteDamageRedirectAPNAP(t *testing.T) {
	for active := range 2 {
		order := []string{}
		a := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("A"), order: &order, numbers: []int{1}}
		b := &preventionAllocationPlayer{BasePlayer: NewBasePlayer("B"), order: &order, numbers: []int{1}}
		g := NewGame(a, b)
		g.SetActivePlayerIndex(active)
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 20), a.PlayerID())
		second := g.PutOnBattlefield(NewCreature("Second", "", 1, 20), b.PlayerID())
		source := g.CaptureDamageSource(first.ID())
		for _, p := range []*Permanent{first, second} {
			if err := ApplyEffect(g, RedirectNextDamageToPlayer(This, You, Fixed(2)), p.ID(), p.ControllerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.DealSimultaneousDamage(PermanentDamageAssignment(second, 2, source), PermanentDamageAssignment(first, 2, source), PermanentDamageAssignment(second, 2, source), PermanentDamageAssignment(first, 2, source))
		if len(order) != 2 || order[0] != g.ActivePlayerObj().Name() || order[1] != g.NonActivePlayerObj().Name() {
			t.Fatalf("allocation order %v", order)
		}
		if a.Life() != 18 || b.Life() != 18 || first.Damage != 2 || second.Damage != 2 {
			t.Fatal("budgets crossed subjects")
		}
	}
}

func TestFiniteDamageRedirectOneSourceEvent(t *testing.T) {
	g, a, _, subject, source := finiteRedirectFixture(t, 2)
	if err := ApplyEffect(g, RedirectNextDamageToPlayer(This, You, Fixed(2)), subject.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, amount int, _ bool) {
		observed++
		if amount != 4 || a.Life() != 16 || subject.Damage != 0 {
			t.Error("partial damage event")
		}
	})
	outcome := g.damage.dealDamageToPermanentFromSourceOutcome(g, subject, 4, g.CaptureDamageSource(source.ID()))
	if observed != 1 || outcome.amount != 4 || outcome.playerID != a.PlayerID() {
		t.Fatalf("one source event became %d notifications, outcome %+v", observed, outcome)
	}
}
