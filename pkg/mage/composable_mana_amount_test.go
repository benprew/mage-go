package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedManaAmountActivation(t *testing.T) {
	for _, n := range []int{0, 3} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		saved := BindPermanent("source")
		ability := Activate(Costs(TapCost(This)), Sequence(Snapshot(This, saved), AddManaAmount(Black, PowerOf(From(saved)))))
		source := NewCreature("Mana", "", n, 4, WithAbility(ability), WithKeyword(Haste))
		p := g.PutOnBattlefield(source, a.PlayerID())
		p.GrantBaseAttr(AttrCantActivateNonManaAbilities)
		events, amount := 0, 0
		watcher := NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *GameEvent, _ GameReader, _, _ uuid.UUID) bool { events++; amount = evt.Amount; return false })
		g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(watcher)), a.PlayerID())
		if err := ValidateCardDefinition(source.Name(), source); err != nil {
			t.Fatal(err)
		}
		if got := ManaProductionsForAbility(ability); got != nil {
			t.Fatalf("dynamic amount advertised fixed production: %v", got)
		}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.StackSize() != 0 || a.ManaPool().Count(Black) != n {
			t.Fatal("mana ability did not resolve immediately with saved amount")
		}
		wantEvents := 0
		if n > 0 {
			wantEvents = 1
		}
		if events != wantEvents || amount != n {
			t.Fatalf("events=%d amount=%d", events, amount)
		}
	}
}

func TestTypedManaAmountMainPhaseCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	calls := 0
	source := NewEnchantment("Phase mana", "", WithAbility(At(YourFirstMainPhase, AddManaAmount(Black, countedLifeAmount{&calls}))))
	g.PutOnBattlefield(source, a.PlayerID())
	g.FireEvent(GameEvent{Type: EvtMainPhase, PlayerID: a.PlayerID(), Flag: true})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 || calls != 0 {
		t.Fatal("main-phase mana bypassed stack or evaluated early")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	clone := g.Clone()
	clone.ResolveStack()
	if calls != 2 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Black) != 20 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Black) != 20 {
		t.Fatal("copied mana lost controller or evaluated amount twice")
	}
	if a.ManaPool().Count(Black) != 0 || b.ManaPool().Count(Black) != 0 {
		t.Fatal("clone changed original pools")
	}
}

func TestTypedManaAmountValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, ValueAtResolution(nil), FromNumber(BindNumber("missing")), PowerOf(DeclarePermanentTarget("undeclared", IsCreature))} {
		card := NewSorcery("Invalid mana", "", WithAbility(Spell(AddManaAmount(Black, amount))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	for _, color := range []Color{AnyColor, Color(-1), Color(99)} {
		card := NewSorcery("Invalid color", "", WithAbility(Spell(AddManaAmount(color, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("nonconcrete color accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.ManaPool().Add(Black, 2)
	for _, n := range []int{-2, 0} {
		if err := ApplyEffect(g, AddManaAmount(Black, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.ManaPool().Count(Black) != 2 {
		t.Fatal("nonpositive production changed pool")
	}
}

func TestTypedManaAmountPlanning(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), Sequence(AddManaAmount(Blue, Fixed(1)), AddManaAmount(Green, Fixed(2))))
	source := NewLand("Fixed mana", WithAbility(ability))
	p := g.PutOnBattlefield(source, a.PlayerID())
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1, Green: 2}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(p.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Green) != 2 || p.Tapped || a.ManaPool().Count(Green) != 0 {
		t.Fatal("fixed planning or clone isolation failed")
	}
	calls := 0
	dynamic := Activate(Costs(TapCost(This)), AddManaAmount(Black, countedLifeAmount{&calls}))
	other := NewLand("Dynamic mana", WithAbility(dynamic))
	q := g.PutOnBattlefield(other, a.PlayerID())
	if _, ok := manaSourceAbilityForPlanning(dynamic, q.ID(), 0, g); ok || calls != 0 {
		t.Fatal("planner evaluated or advertised an unknown amount")
	}
}

func TestTypedManaAmountNestedResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	nested := &StackObject{ID: uuid.New(), Controller: b.PlayerID(), IsAbility: true, Effects: []Effect{AddManaAmount(Blue, Fixed(7))}}
	body := Sequence(AddManaAmount(Black, Fixed(2)), FuncEffect("resolve nested mana", EffectProperties{}, func(game *Game, _, _ uuid.UUID, _ []uuid.UUID) error { game.ResolveStackObject(nested); return nil }), AddManaAmount(Black, Fixed(3)))
	outer := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Effects: []Effect{body}}
	if got := g.resolveStackObject(outer); got != 5 {
		t.Fatalf("outer production=%d, want 5", got)
	}
	if a.ManaPool().Count(Black) != 5 || b.ManaPool().Count(Blue) != 7 || g.resolution.manaProduced != 0 {
		t.Fatal("nested production escaped its resolution")
	}
}

func TestTypedManaAmountDirectResolutionHasNoTrackingState(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AddManaAmount(Black, Fixed(3)), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Black) != 3 || g.resolution.manaProduced != 0 || g.resolution.manaTracking {
		t.Fatal("direct effect retained production tracking")
	}
}
