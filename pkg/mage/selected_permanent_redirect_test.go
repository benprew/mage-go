package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSelectedPermanentRedirectIdentity(t *testing.T) {
	for _, state := range []string{"live", "source-return", "source-depart", "target-return", "target-depart", "target-phase", "phase-cycle", "control", "not-creature", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
			target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), a.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			binding := BindDamageSource("chosen")
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
			if err := RedirectNextDamageToPermanentFromSelectedSource(FromDamageSource(binding), You, This, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			baseline := g.Clone()
			switch state {
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, b.PlayerID())
			case "source-depart":
				g.RemoveFromBattlefield(source)
			case "target-return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(target.Card, a.PlayerID())
			case "target-depart":
				g.RemoveFromBattlefield(target)
			case "target-phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "not-creature":
				g.AddContinuousEffect(FuncContinuousEffect(core.LayerType, core.EndOfTurn, func(game *Game, _ uuid.UUID) error { game.RevokeAttr(target.ID(), core.AttrIsCreature); return nil }))
				g.ApplyContinuousEffects()
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				if state == "source-return" {
					game.DealDamageToPlayer(game.GetPlayer(a.PlayerID()), 1, source.ID())
					if game.GetPlayer(a.PlayerID()).Life() != 19 {
						t.Fatal("new visit inherited rule")
					}
				}
				before := game.GetPlayer(a.PlayerID()).Life()
				game.DealSimultaneousDamage(PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 2, saved), PlayerDamageAssignment(game.GetPlayer(a.PlayerID()), 3, saved))
				redirects := state != "target-return" && state != "target-depart" && state != "target-phase" && state != "not-creature" && state != "cleanup"
				want := before - 5
				if redirects {
					want = before
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatalf("life=%d want=%d", game.GetPlayer(a.PlayerID()).Life(), want)
				}
				if redirects && game.FindPermanent(target.ID()).Damage != 5 {
					t.Fatal("batch did not retain all damage")
				}
				if game.GetPlayer(b.PlayerID()).Life() < 25 {
					t.Fatal("source lifelink was lost")
				}
				if redirects {
					game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
					if game.GetPlayer(a.PlayerID()).Life() != want-1 {
						t.Fatal("consumed rule applied again")
					}
				}
			}
			if baseline.FindPermanent(target.ID()).Damage != 0 || len(baseline.WaitingRedirectionPermanentSources()) != 2 {
				t.Fatal("clone mutated original rule")
			}
		})
	}
}

func TestSelectedPermanentRedirectRechecks(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
		target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), a.PlayerID())
		binding := BindDamageSource("chosen")
		saved := g.CaptureDamageSource(source.ID())
		g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
		if err := RedirectNextDamageToPermanentFromSelectedSource(FromDamageSource(binding), You, This, DamageSourceHasColor(core.Red)).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if err := ApplyEffect(g, PreventNextDamage(You, Fixed(3)), target.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPlayer(a, 3, source.ID())
		wantDamage := 3
		wantRefs := 0
		if choice == 1 {
			wantDamage = 0
			wantRefs = 2
		}
		if a.Life() != 20 || target.Damage != wantDamage || len(g.WaitingRedirectionPermanentSources()) != wantRefs || a.prompts != 1 {
			t.Fatal("replacement order or recheck failed")
		}
	}
}

func TestSelectedPermanentRedirectSpellContinuity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), a.PlayerID())
	card := NewCreature("Selected", "", 1, 20)
	obj := newStackObject(b.PlayerID(), card.ID(), card, nil, nil, 0, false)
	g.stack.Push(obj)
	binding := BindDamageSource("spell")
	saved := g.CaptureSpellDamageSource(obj.ID)
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	if err := RedirectNextDamageToPermanentFromSelectedSource(FromDamageSource(binding), You, This, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if refs := g.WaitingRedirectionSpellSources(); len(refs) != 1 || !refs[0].SameObject(saved) {
		t.Fatal("missing waiting spell")
	}
	g.ResolveTopOfStack()
	source := g.FindPermanent(card.ID())
	if source == nil {
		t.Fatal("selected spell did not enter")
	}
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 20 || target.Damage != 2 {
		t.Fatal("spell did not continue to its permanent")
	}
}

func TestSelectedPermanentRedirectInvalid(t *testing.T) {
	for _, effect := range []Effect{
		RedirectNextDamageToPermanentFromSelectedSource(nil, You, This, DamageSourceFilter{}),
		RedirectNextDamageToPermanentFromSelectedSource(This, nil, This, DamageSourceFilter{}),
		RedirectNextDamageToPermanentFromSelectedSource(This, You, nil, DamageSourceFilter{}),
		RedirectNextDamageToPermanentFromSelectedSource(FromDamageSource(BindDamageSource("missing")), You, This, DamageSourceFilter{}),
	} {
		if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSelectedPermanentRedirectBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), a.PlayerID())
	binding := BindDamageSource("source")
	saved := g.CaptureDamageSource(source.ID())
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	if err := RedirectNextDamageToPermanentFromSelectedSource(FromDamageSource(binding), You, This, DamageSourceHasColor(core.Red)).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	rule := g.effects.replacements[0]
	for _, amount := range []int{0, -1} {
		if rule.Matches(NewDamageToPlayerAction(source.ID(), a.PlayerID(), amount, false).withDamageSource(saved), g) {
			t.Fatal("nonpositive damage matched")
		}
	}
	if rule.Matches(NewDamageToPlayerAction(source.ID(), b.PlayerID(), 1, false).withDamageSource(saved), g) {
		t.Fatal("wrong player matched")
	}
	g.MutablePermanent(source.ID()).ColorOverride = &[]core.Color{core.Blue}
	if rule.Matches(NewDamageToPlayerAction(source.ID(), a.PlayerID(), 1, false).withDamageSource(saved), g) {
		t.Fatal("current filter was ignored")
	}
	g.MutablePermanent(source.ID()).ColorOverride = &[]core.Color{core.Red}
	if !rule.Matches(NewDamageToPlayerAction(source.ID(), a.PlayerID(), 1, false).withDamageSource(saved), g) || !rule.IsActive(g) {
		t.Fatal("mismatch consumed replacement")
	}
	for range 2 {
		if len(g.WaitingRedirectionPermanentSources()) != 2 {
			t.Fatal("waiting query changed rule")
		}
	}
}
