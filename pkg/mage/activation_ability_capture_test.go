package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationAbilityCaptureIdentity(t *testing.T) {
	for _, scenario := range []string{"normal", "reattach", "source-left", "source-control", "host-control", "host-return", "host-phase", "phase-cycle", "copy", "ability-copy", "absent", "two"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindAbility("activated")
			watcher := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetCreature()), WithAbility(WhenEvent(
				ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana().WithAbility(binding),
				Sequence(CounterAbility(FromAbility(binding)), GainLifeFor(You, Fixed(1)))))), a.PlayerID())
			card := NewCreature("Host", "", 1, 5, WithAbility(Activate(Costs(Tap()), GainLifeFor(You, Fixed(5)))))
			host := g.PutOnBattlefield(card, b.PlayerID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), b.PlayerID())
			g.Attach(watcher.ID(), host.ID())
			if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			wantA, wantB := 21, 20
			switch scenario {
			case "reattach":
				g.Attach(watcher.ID(), other.ID())
			case "source-left":
				g.RemoveFromBattlefield(watcher)
			case "source-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: watcher.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "host-control":
				g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "ability-copy":
				if g.CopyStackObjectDirect(original, a.PlayerID(), false) == nil {
					t.Fatal("ability copy failed")
				}
				wantA = 26
			case "host-return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(card, b.PlayerID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "absent":
				g.CounterStackObject(original.ID)
			case "two":
				g.UntapPermanent(host)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				wantA = 22
			}
			g.PutTriggersOnStack()
			if scenario == "copy" {
				g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				wantB = 21
			}
			clone := g.Clone()
			g.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("live resolution changed clone")
			}
			clone.ResolveStack()
			for _, game := range []*Game{g, clone} {
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB || game.StackSize() != 0 {
					t.Fatalf("life=%d,%d want=%d,%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
			}
		})
	}
}

func TestActivationAbilityCaptureFiltersAndRollback(t *testing.T) {
	for _, scenario := range []string{"tap", "no-tap", "mana", "targeted-mana", "other", "unpaid", "sick", "phased", "removed"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindAbility("ability")
			watcher := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetCreature()), WithAbility(WhenEvent(ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana().WithAbility(binding), CounterAbility(FromAbility(binding))))), a.PlayerID())
			cost := Costs(Tap())
			if scenario == "no-tap" {
				cost = Costs(ManaPayment("0"))
			}
			if scenario == "unpaid" {
				cost = Costs(Tap(), ManaPayment("{1}"))
			}
			var ability Ability = Activate(cost, GainLifeFor(You, Fixed(3)))
			if scenario == "mana" {
				ability = NewManaAbility(core.Green)
			}
			if scenario == "targeted-mana" {
				ability = Activate(cost, AddManaAmount(core.Green, Fixed(1)), WithTargets(DeclarePlayerTarget("target")))
			}
			host := g.PutOnBattlefield(NewCreature("Host", "", 1, 5, WithAbility(ability)), b.PlayerID())
			if scenario != "sick" {
				host.RevokeBaseAttr(core.AttrSummonSick)
			}
			g.Attach(watcher.ID(), host.ID())
			if scenario == "other" {
				g.Attach(watcher.ID(), g.PutOnBattlefield(NewCreature("Other", "", 1, 5), b.PlayerID()).ID())
			}
			if scenario == "phased" {
				g.PhaseOut(host.ID())
			}
			if scenario == "removed" {
				g.AddContinuousEffect(RemoveAllAbilities(watcher.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
			}
			var targets []uuid.UUID
			if scenario == "targeted-mana" {
				targets = []uuid.UUID{a.PlayerID()}
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, targets)
			invalid := scenario == "unpaid" || scenario == "sick" || scenario == "phased"
			if (err != nil) != invalid {
				t.Fatalf("activation error=%v", err)
			}
			base := g.StackSize()
			g.PutTriggersOnStack()
			want := 0
			if scenario == "tap" || scenario == "targeted-mana" {
				want = 1
			}
			if g.StackSize() != base+want {
				t.Fatalf("trigger count=%d want=%d", g.StackSize()-base, want)
			}
			if invalid && (host.Tapped || g.StackSize() != 0) {
				t.Fatal("failed activation changed state")
			}
			g.ResolveStack()
			if want == 1 && (b.Life() != 20 || b.ManaPool().TotalMana() != 0) {
				t.Fatal("countered ability resolved")
			}
		})
	}
}

func TestActivationAbilityCaptureDefinition(t *testing.T) {
	binding := BindAbility("ability")
	valid := NewEnchantment("Valid", "", WithAbility(WhenEvent(ActivatedAbilityEvent().WithAbility(binding), CounterAbility(FromAbility(binding)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		ActivatedAbilityEvent().WithAbility(nil),
		ActivatedAbilityEvent().WithAbility(BindAbility("")),
		ActivatedAbilityEvent().WithAbility(&AbilityBinding{}),
		ActivatedAbilityEvent().WithAbility(binding).WithAbility(binding),
		ActivatedAbilityEvent().WithTapCost().WithoutTapCost(),
		ActivatedAbilityEvent().NonMana().NonMana(),
		ActivatedAbilityEvent().FromAttachedPermanent().FromAttachedPermanent(),
	} {
		card := NewEnchantment("Invalid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid event accepted")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(ActivatedAbilityEvent(), CounterAbility(FromAbility(binding)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound read accepted")
	}
}

func TestActivationAbilityCaptureMissingAndExactEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindAbility("saved")
	card := NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(3)))))
	host := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), host.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	ability := g.StackPeek()
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if _, ok := FromAbility(binding).resolveStackObject(ctx); ok {
		t.Fatal("missing capture supplied ability")
	}
	if err := (&eventAbilityCapture{binding: binding, id: ability.ID}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if obj, ok := FromAbility(binding).resolveStackObject(ctx); !ok || obj.ID != ability.ID {
		t.Fatal("saved ability unavailable")
	}
	if err := (&eventAbilityCapture{binding: binding}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromAbility(binding).resolveStackObject(ctx); ok {
		t.Fatal("empty event reused old capture")
	}
	spell := newStackObject(a.PlayerID(), uuid.New(), NewInstant("Spell", ""), nil, nil, 0, false)
	g.PushStack(spell)
	if err := (&eventAbilityCapture{binding: binding, id: spell.ID}).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromAbility(binding).resolveStackObject(ctx); ok {
		t.Fatal("spell supplied ability")
	}
	aura := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetCreature()), WithAbility(WhenEvent(ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana().WithAbility(binding), CounterAbility(FromAbility(binding))))), b.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Host", "", 1, 5), a.PlayerID())
	g.Attach(aura.ID(), creature.ID())
	oldVisit := creature.incarnationID
	g.RemoveFromBattlefield(creature)
	returned := g.PutOnBattlefield(creature.Card, a.PlayerID())
	g.Attach(aura.ID(), returned.ID())
	g.FireEvent(core.GameEvent{Type: core.EvtAbilityActivated, SourceID: returned.ID(), SourceIncarnation: oldVisit, ActivatedAbilityID: ability.ID, PlayerID: a.PlayerID(), Flag: true})
	base := g.StackSize()
	g.PutTriggersOnStack()
	if g.StackSize() != base {
		t.Fatal("returned host supplied old event visit")
	}
}

func TestActivationAbilityCaptureDelayedFrame(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	binding := BindAbility("saved")
	watcher := g.PutOnBattlefield(NewAura("Watcher", "", WithCastTarget(TargetCreature()), WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana().WithAbility(binding),
		AtNext(EachEndStep, Sequence(CounterAbility(FromAbility(binding)), GainLifeFor(You, Fixed(1))))))), a.PlayerID())
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 5, WithAbility(Activate(Costs(Tap()), GainLifeFor(You, Fixed(3))))), b.PlayerID())
	host.RevokeBaseAttr(core.AttrSummonSick)
	g.Attach(watcher.ID(), host.ID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.ResolveTopOfStack()
	g.RemoveFromBattlefield(watcher)
	g.UntapPermanent(host)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), host.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
		game.PutTriggersOnStack()
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 23 {
			t.Fatal("delayed binding followed a later activation")
		}
	}
}
