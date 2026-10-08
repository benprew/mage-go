package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSelectedSourceRedirectIdentity(t *testing.T) {
	for _, state := range []string{"live", "source-return", "source-phase", "recipient-return", "recipient-phase", "phase-cycle", "control", "cleanup"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20), b.PlayerID())
			target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
			binding := BindDamageSource("selected")
			saved := g.CaptureDamageSource(source.ID())
			g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
			if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			baseline := g.Clone()
			g.zones.ClearLKI()
			switch state {
			case "source-return":
				g.RemoveFromBattlefield(source)
				g.PutOnBattlefield(source.Card, b.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "recipient-return":
				g.RemoveFromBattlefield(target)
				g.PutOnBattlefield(target.Card, b.PlayerID())
			case "recipient-phase":
				g.PhaseOut(target.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			}
			for _, game := range []*Game{g, g.Clone()} {
				wantRefs := 2
				if state == "cleanup" {
					wantRefs = 0
				}
				if len(game.WaitingRedirectionPermanentSources()) != wantRefs {
					t.Fatal("lost waiting references")
				}
				if state == "recipient-phase" {
					rule := game.effects.replacements[0]
					if rule.Matches(NewDamageToCreatureAction(source.ID(), target.ID(), 2, false).withDamageSource(saved), game) {
						t.Fatal("phased subject redirected")
					}
					game.PhaseIn([]uuid.UUID{target.ID()})
				}
				if state == "source-return" {
					game.DealDamageToPermanent(game.FindPermanent(target.ID()), 1, source.ID())
					if game.GetPlayer(a.PlayerID()).Life() != 20 {
						t.Fatal("returned source inherited rule")
					}
				}
				damage := 0
				if p := game.FindPermanent(target.ID()); p != nil {
					damage = p.Damage
				}
				game.DealSimultaneousDamage(PermanentDamageAssignment(game.FindPermanent(target.ID()), 2, saved), PermanentDamageAssignment(game.FindPermanent(target.ID()), 3, saved))
				redirects := state != "recipient-return" && state != "cleanup"
				wantLife := 20
				if redirects {
					wantLife = 15
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantLife {
					t.Fatalf("life=%d want=%d", game.GetPlayer(a.PlayerID()).Life(), wantLife)
				}
				wantDamage := damage
				if !redirects {
					wantDamage += 5
				}
				if game.FindPermanent(target.ID()).Damage != wantDamage {
					t.Fatal("wrong recipient damage")
				}
				if len(game.WaitingRedirectionPermanentSources()) != 0 && redirects {
					t.Fatal("consumed rule kept references")
				}
				game.DealDamageToPermanentFromSource(game.FindPermanent(target.ID()), 1, saved)
				if game.FindPermanent(target.ID()).Damage != wantDamage+1 {
					t.Fatal("rule did not consume")
				}
			}
			if baseline.GetPlayer(a.PlayerID()).Life() != 20 || baseline.FindPermanent(target.ID()).Damage != 0 || len(baseline.WaitingRedirectionPermanentSources()) != 2 {
				t.Fatal("clone shared state")
			}
		})
	}
}

func TestSelectedSourceRedirectSpellContinuity(t *testing.T) {
	for _, consumed := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
		card := NewCreature("Selected", "", 1, 20)
		obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
		g.stack.Push(obj)
		binding := BindDamageSource("selected")
		spell := g.CaptureSpellDamageSource(obj.ID)
		g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: spell}
		if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if consumed {
			g.DealDamageToPermanentFromSource(target, 2, spell)
		}
		g.ResolveTopOfStack()
		p := g.FindPermanent(card.ID())
		if p == nil {
			t.Fatal("spell did not enter")
		}
		saved := g.CaptureDamageSource(p.ID())
		for _, game := range []*Game{g, g.Clone()} {
			before := game.GetPlayer(a.PlayerID()).Life()
			game.DealDamageToPermanentFromSource(game.FindPermanent(target.ID()), 3, saved)
			want := before - 3
			if consumed {
				want = before
			}
			if game.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("wrong spell continuation")
			}
		}
	}
}

func TestSelectedSourceRedirectInvalid(t *testing.T) {
	for _, effect := range []Effect{RedirectNextDamageFromSelectedSource(nil, This, You, DamageSourceFilter{}), RedirectNextDamageFromSelectedSource(This, nil, You, DamageSourceFilter{}), RedirectNextDamageFromSelectedSource(This, This, nil, DamageSourceFilter{}), RedirectNextDamageFromSelectedSource(FromDamageSource(BindDamageSource("missing")), This, You, DamageSourceFilter{})} {
		if err := ValidateCardDefinition("Invalid", NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSelectedSourceRedirectRechecks(t *testing.T) {
	for choice := range 2 {
		a := NewBasePlayer("A")
		b := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("Source", "{R}", 1, 20, WithKeyword(core.Lifelink)), a.PlayerID())
		target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
		if err := ApplyEffect(g, RedirectNextDamageFromSelectedSource(DamageSourceOf(This), From(BindPermanent("unbound")), You, DamageSourceFilter{}), source.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(g.WaitingRedirectionPermanentSources()) != 0 {
			t.Fatal("absent subject installed rule")
		}
		binding := BindDamageSource("selected")
		g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: g.CaptureDamageSource(source.ID())}
		if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceHasColor(core.Red)).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.MutablePermanent(source.ID()).ColorOverride = &[]core.Color{core.Blue}
		g.DealDamageToPermanent(target, 1, source.ID())
		if a.Life() != 21 || target.Damage != 1 || len(g.WaitingRedirectionPermanentSources()) != 2 {
			t.Fatal("property mismatch consumed rule")
		}
		g.MutablePermanent(source.ID()).ColorOverride = &[]core.Color{core.Red}
		if err := ApplyEffect(g, PreventDamageThisTurn(This, PermanentFilter{}), target.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.DealDamageToPermanent(target, 2, source.ID())
		if a.Life() != 21 || b.Life() != 20 || target.Damage != 1 {
			t.Fatal("redirection or prevention lost lifelink")
		}
		if b.prompts != 1 {
			t.Fatal("affected player did not choose order")
		}
		refs := g.WaitingRedirectionPermanentSources()
		want := 0
		if choice == 1 {
			want = 2
		}
		if len(refs) != want {
			t.Fatal("wrong replacement consumed")
		}
	}
}

func TestSelectedSourceRedirectWaitingSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
	card := NewInstant("Spell", "{R}")
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.stack.Push(obj)
	binding := BindDamageSource("selected")
	saved := g.CaptureSpellDamageSource(obj.ID)
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveTopOfStack()
	refs := g.WaitingRedirectionSpellSources()
	if len(refs) != 1 || !refs[0].SameObject(saved) {
		t.Fatal("lost departed spell reference")
	}
	for _, game := range []*Game{g, g.Clone()} {
		game.DealDamageToPermanentFromSource(game.FindPermanent(target.ID()), 2, saved)
		if game.GetPlayer(a.PlayerID()).Life() != 18 || len(game.WaitingRedirectionSpellSources()) != 0 {
			t.Fatal("spell rule did not consume")
		}
	}
	if len(g.Clone().WaitingRedirectionSpellSources()) != 0 {
		t.Fatal("clone restored consumed spell")
	}
}

func TestSelectedSourceRedirectDamageContract(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 20, WithKeyword(core.Lifelink)), b.PlayerID())
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
	binding := BindDamageSource("selected")
	saved := g.CaptureDamageSource(source.ID())
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
	if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	rule := g.effects.replacements[0]
	action := NewDamageToCreatureAction(source.ID(), target.ID(), 2, true).withDamageSource(saved)
	redirected := rule.Clone().Replace(action, g).(*DamageToPlayerAction)
	if !redirected.IsCombatDamage() || redirected.Amount() != 2 || !redirected.DamageSource().SameObject(saved) || !rule.IsActive(g) {
		t.Fatal("clone changed damage attribution or original consumption")
	}
	for _, action := range []Action{
		NewDamageToCreatureAction(source.ID(), target.ID(), 0, false).withDamageSource(saved),
		NewDamageToCreatureAction(source.ID(), target.ID(), -1, false).withDamageSource(saved),
		NewDamageToCreatureAction(source.ID(), other.ID(), 2, false).withDamageSource(saved),
		NewDamageToPlayerAction(source.ID(), a.PlayerID(), 2, false).withDamageSource(saved),
		NewDamageToCreatureAction(source.ID(), target.ID(), 2, false).withDamageSource(g.CaptureDamageSource(other.ID())),
	} {
		if rule.Matches(action, g) {
			t.Fatal("incompatible damage matched")
		}
	}
	g.MutablePermanent(target.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.DealDamageToPermanent(target, 1, source.ID())
	if target.Damage != 1 || !rule.IsActive(g) {
		t.Fatal("prohibition consumed redirection")
	}
	g.MutablePermanent(target.ID()).RevokeBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
	g.AddPreventionShield(a.PlayerID(), 1)
	g.damage.resolvingCombatDamage = true
	outcomes := g.dealSimultaneousDamageOutcomes(PermanentDamageAssignment(target, 2, saved), PermanentDamageAssignment(target, 3, saved), PermanentDamageAssignment(other, 1, saved))
	if a.Life() != 16 || b.Life() != 26 || target.Damage != 1 || other.Damage != 1 || len(outcomes) != 3 {
		t.Fatal("wrong actual simultaneous outcomes")
	}
	for _, outcome := range outcomes {
		if !outcome.source.SameObject(saved) {
			t.Fatal("lost damage attribution")
		}
	}
}

func TestSelectedSourceRedirectSpellCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := g.PutOnBattlefield(NewCreature("Target", "", 1, 20), b.PlayerID())
	card := NewCreature("Spell", "", 1, 20)
	obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: b.PlayerID()}
	g.stack.Push(obj)
	binding := BindDamageSource("selected")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: g.CaptureSpellDamageSource(obj.ID)}
	if err := RedirectNextDamageFromSelectedSource(FromDamageSource(binding), This, You, DamageSourceFilter{}).Apply(&EffectContext{Game: g, SourceID: target.ID(), Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	copyObj := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	var copied *Permanent
	for _, p := range g.AllBattlefield() {
		if p.IsToken && p.Name() == "Spell" {
			copied = p
		}
	}
	if copied == nil {
		t.Fatal("copy did not enter")
	}
	g.DealDamageToPermanent(target, 1, copied.ID())
	if a.Life() != 20 || target.Damage != 1 {
		t.Fatal("copy inherited selected source")
	}
	g.ResolveTopOfStack()
	g.DealDamageToPermanent(target, 2, card.ID())
	if a.Life() != 18 || target.Damage != 1 {
		t.Fatal("original lost continuation")
	}
}
