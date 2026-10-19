package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCastSpellHistoryIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewSorcery("History", "", Spell(GainLifeFor(You, Fixed(0))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	first := g.StackPeek()
	source := g.CaptureSpellDamageSource(first.ID)
	copied := g.CopyStackObjectDirect(first, b.PlayerID(), false)
	g.DealDamageToPlayerFromSource(b, 8, g.CaptureSpellDamageSource(copied.ID))
	g.AddPreventionShield(b.PlayerID(), 2)
	g.DealDamageToPlayerFromSource(b, 5, source)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 30), b.PlayerID())
	g.DealSimultaneousDamage(PermanentDamageAssignment(p, 2, source), PlayerDamageAssignment(a, 2, source))
	g.ResolveStack()
	binding := BindDamageSource("cast spell")
	effect := ChooseSpellCastThisTurn(You, You, DamageSourceHasType(core.TypeSorcery), binding)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	value := DamageDealtThisTurnBySource(FromDamageSource(binding))
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("actual damage = %d, want 7", got)
	}
	if !g.resolution.damageSourceBindings[binding].SameObject(source) {
		t.Fatal("copy selected as a cast")
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	a.RemoveFromGraveyard(card.ID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	second := g.CaptureSpellDamageSource(g.StackPeek().ID)
	if second.SameObject(source) {
		t.Fatal("recast identity reused")
	}
	g.DealDamageToPlayerFromSource(b, 1, second)
	if value.resolveValue(ctx) != 7 || value.resolveValue(&cloneCtx) != 7 {
		t.Fatal("recast or clone changed total")
	}
	clone.DealDamageToPlayerFromSource(clone.GetPlayer(b.PlayerID()), 1, source)
	if value.resolveValue(&cloneCtx) != 8 || value.resolveValue(ctx) != 7 {
		t.Fatal("clone shares totals")
	}
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	g.damage.ClearEndOfTurn()
	if value.resolveValue(ctx) != 0 || value.resolveValue(&cloneCtx) != 8 {
		t.Fatal("cleanup or clone history")
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.damageSourceBindings[binding]; ok {
		t.Fatal("empty history retained choice")
	}
}

func TestCastSpellHistoryDefinition(t *testing.T) {
	binding := BindDamageSource("cast")
	for _, effect := range []ResolvingEffect{
		ChooseSpellCastThisTurn(nil, You, DamageSourceHasType(core.TypeSorcery), binding),
		ChooseSpellCastThisTurn(You, nil, DamageSourceHasType(core.TypeSorcery), binding),
		ChooseSpellCastThisTurn(You, You, DamageSourceHasType(core.TypeSorcery), nil),
		GainLifeFor(You, DamageDealtThisTurnBySource(nil)),
		GainLifeFor(You, DamageDealtThisTurnBySource(FromDamageSource(binding))),
	} {
		card := NewInstant("Invalid", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	card := NewInstant("Valid", "", Spell(Sequence(ChooseSpellCastThisTurn(You, You, DamageSourceHasType(core.TypeSorcery), binding), GainLifeFor(You, DamageDealtThisTurnBySource(FromDamageSource(binding))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

type castHistoryChooser struct {
	Player
	index   int
	options []DamageSource
}

func (p *castHistoryChooser) ChooseDamageSource(sources []DamageSource, _ *Game) int {
	p.options = sources
	return p.index
}

func TestCastSpellHistoryCounteredAndChoices(t *testing.T) {
	base, b := NewBasePlayer("A"), NewBasePlayer("B")
	a := &castHistoryChooser{Player: base, index: 1}
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	binding := BindDamageSource("cast")
	effect := ChooseSpellCastThisTurn(You, You, DamageSourceHasType(core.TypeSorcery), binding)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	saved := make([]DamageSource, 0, 2)
	for i := range 2 {
		card := NewSorcery("Same name", "", Spell(GainLifeFor(You, Fixed(0))))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		obj := g.StackPeek()
		source := g.CaptureSpellDamageSource(obj.ID)
		saved = append(saved, source)
		if i == 1 {
			g.DealDamageToPlayerFromSource(b, 3, source)
		}
		g.CounterStackObject(obj.ID)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(a.options) != 2 || !a.options[1].SameObject(saved[1]) || DamageDealtThisTurnBySource(FromDamageSource(binding)).resolveValue(ctx) != 3 {
		t.Fatal("countered cast history lost")
	}
	a.options[1].card.(*BaseCard).name = "mutated"
	a.index = 0
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if a.options[1].Name() != "Same name" || a.options[1].card.Name() != "Same name" || DamageDealtThisTurnBySource(FromDamageSource(binding)).resolveValue(ctx) != 0 {
		t.Fatal("choice aliases history or zero damage omitted")
	}
	a.index = 99
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if !g.resolution.damageSourceBindings[binding].SameObject(saved[0]) {
		t.Fatal("required choice fallback")
	}
	otherCtx := *ctx
	otherCtx.Controller = b.PlayerID()
	if err := effect.Apply(&otherCtx); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.resolution.damageSourceBindings[binding]; ok {
		t.Fatal("other caster inherited choice")
	}
}

func TestCastSpellHistoryRedirectAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := NewSorcery("History", "", Spell(GainLifeFor(You, Fixed(0))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	source := g.CaptureSpellDamageSource(g.StackPeek().ID)
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 30), b.PlayerID())
	ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}
	if err := RedirectNextDamageToPlayer(This, You, Fixed(2)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPermanentFromSource(p, 5, source)
	binding := BindDamageSource("cast")
	if err := ChooseSpellCastThisTurn(You, You, DamageSourceHasType(core.TypeSorcery), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	value := DamageDealtThisTurnBySource(FromDamageSource(binding))
	if got := value.resolveValue(ctx); got != 4 {
		t.Fatalf("redirected and prevented damage = %d", got)
	}
	g.PhaseOut(p.ID())
	g.DealDamageToPermanentFromSource(p, 3, source)
	if value.resolveValue(ctx) != 4 {
		t.Fatal("phased recipient took damage")
	}
}
