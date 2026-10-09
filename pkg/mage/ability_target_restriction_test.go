package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantBeTargetedByAbilitiesDefinition(t *testing.T) {
	filter := HasColorCardFilter(core.White)
	target := DeclarePermanentTarget("unbound", IsCreature)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "source", Card: NewCreature("Good", "", 2, 5, WithAbility(Static(CantBeTargetedByAbilities(This, filter))))},
		{Name: "host", Card: NewAura("Good", "", WithAbility(Static(CantBeTargetedByAbilities(AttachedPermanent, filter))))},
		{Name: "nil", Card: NewCreature("Bad", "", 2, 5, WithAbility(Static(CantBeTargetedByAbilities(nil, filter)))), WantErr: true},
		{Name: "unbound", Card: NewCreature("Bad", "", 2, 5, WithAbility(Static(CantBeTargetedByAbilities(target, filter)))), WantErr: true},
		{Name: "filter", Card: NewCreature("Bad", "", 2, 5, WithAbility(Static(CantBeTargetedByAbilities(This, CardFilter{})))), WantErr: true},
	})
}

func TestCantBeTargetedByAbilitiesSourceFacts(t *testing.T) {
	for _, kind := range []string{"color", "type"} {
		for _, change := range []string{"before", "after", "leave", "return", "phase", "copy"} {
			t.Run(kind+"/"+change, func(t *testing.T) {
				g, a, b := newSpellCopyGame()
				filter := HasColorCardFilter(core.White)
				if kind == "type" {
					filter = NewCardFilter("artifact", func(c Card) bool { return c.HasType(core.TypeArtifact) })
				}
				protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(CantBeTargetedByAbilities(This, filter)))), b.PlayerID())
				target := DeclarePermanentTarget("creature", IsCreature)
				card := NewCreature("Source", "", 1, 5, WithAbility(Activate(Costs(ManaPayment("{1}")), DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
				source := g.PutOnBattlefield(card, a.PlayerID())
				mark := func() {
					if kind == "color" {
						g.ChangeObjectColor(source.ID(), core.White)
					} else {
						if err := ApplyEffect(g, Until(core.EndOfTurn, AddCardType(This, core.TypeArtifact)), source.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
				}
				a.ManaPool().Add(core.Colorless, 1)
				if change == "before" {
					mark()
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{protected.ID()}); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
						t.Fatal("illegal activation spent resources")
					}
					return
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{protected.ID()}); err != nil {
					t.Fatal(err)
				}
				mark()
				if change == "leave" || change == "return" {
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "return" {
						g.PutOnBattlefield(card, a.PlayerID())
					}
				}
				if change == "phase" {
					g.PhaseOut(source.ID())
				}
				if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				clone := g.Clone()
				for _, game := range []*Game{g, clone} {
					game.ResolveStack()
					if game.FindPermanent(protected.ID()).Damage != 0 {
						t.Fatal("current or retained source facts ignored")
					}
				}
			})
		}
	}
}

func TestCantBeTargetedByAbilitiesTriggeredAndPartial(t *testing.T) {
	g, a, b := newSpellCopyGame()
	protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(CantBeTargetedByAbilities(This, HasColorCardFilter(core.White))))), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), b.PlayerID())
	targets := DeclareAnyNumberDamageTargets("targets")
	card := NewCreature("Source", "", 1, 5, WithAbility(At(YourUpkeep, DealDamageEach(This, targets, Fixed(2)), WithTargets(targets))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{protected.ID(), other.ID()}}
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("trigger not placed")
	}
	g.ChangeObjectColor(source.ID(), core.White)
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveStack()
	if g.FindPermanent(protected.ID()).Damage != 0 || g.FindPermanent(other.ID()).Damage != 4 {
		t.Fatal("partial target legality")
	}
}

func TestCantBeTargetedByAbilitiesPendingSourceIdentity(t *testing.T) {
	for _, change := range []string{"leave", "return", "phase"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(CantBeTargetedByAbilities(This, HasColorCardFilter(core.White))))), b.PlayerID())
			target := DeclarePermanentTarget("creature", IsCreature)
			card := NewCreature("Source", "", 1, 5, WithAbility(At(YourUpkeep, DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.ChangeObjectColor(source.ID(), core.White)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			if change == "phase" {
				g.PhaseOut(source.ID())
			} else {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			a.chooseQueue = [][]uuid.UUID{{protected.ID()}}
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("trigger selected a restricted target from a later or missing source")
			}
		})
	}
}

func TestCantBeTargetedByAbilitiesCopyTargetsAndCardSource(t *testing.T) {
	g, a, b := newSpellCopyGame()
	protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 10, WithAbility(Static(CantBeTargetedByAbilities(This, HasColorCardFilter(core.White))))), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 10), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewCreature("Source", "", 1, 5, WithAbility(Activate(Costs(), DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ChangeObjectColor(source.ID(), core.White)
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	b.chooseQueue = [][]uuid.UUID{{other.ID()}}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || len(b.calls) == 0 {
		t.Fatal("no retarget choice")
	}
	for _, id := range b.calls[len(b.calls)-1].possible {
		if id == protected.ID() {
			t.Fatal("retarget offered illegal target")
		}
	}
	g.ResolveStack()
	if g.FindPermanent(other.ID()).Damage != 4 {
		t.Fatal("legal copy targets failed")
	}
	saved := NewInstant("Card source", "{W}")
	obj := &StackObject{IsAbility: true, SourceID: saved.ID(), TargetSource: saved}
	returned := g.PutOnBattlefield(saved, a.PlayerID())
	g.ChangeObjectColor(returned.ID(), core.Red)
	if protected.CanBeTargetedBy(g.resolutionTargetSource(obj), a.PlayerID(), g) {
		t.Fatal("card-source snapshot followed a later permanent")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(1), Fixed(1))), protected.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(protected.ID()).CurrentPower(g) != 3 {
		t.Fatal("nontargeted effect was restricted")
	}
}
