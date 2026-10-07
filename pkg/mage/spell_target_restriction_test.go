package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestCantBeTargetedBySpellsDefinition(t *testing.T) {
	target := DeclarePermanentTarget("target", IsCreature)
	filter := HasSubTypeCardFilter("Aura")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "source", Card: NewCreature("Good", "", 2, 3, WithAbility(Static(CantBeTargetedBySpells(This, filter))))},
		{Name: "host", Card: NewAura("Good", "", WithAbility(Static(CantBeTargetedBySpells(AttachedPermanent, filter))))},
		{Name: "nil", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpells(nil, filter)))), WantErr: true},
		{Name: "unbound", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpells(target, filter)))), WantErr: true},
		{Name: "empty filter", Card: NewCreature("Bad", "", 2, 3, WithAbility(Static(CantBeTargetedBySpells(This, CardFilter{})))), WantErr: true},
	})
}

func TestCantBeTargetedBySpellsLiveAndCopies(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "remove", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			g.SetStep(core.PrecombatMain)
			victimCard := NewCreature("Victim", "", 2, 5, WithAbility(Static(CantBeTargetedBySpells(This, HasSubTypeCardFilter("Aura")))))
			victim := g.PutOnBattlefield(victimCard, b.PlayerID())
			aura := NewAura("Test Aura", "{W}")
			aura.SetOwner(a.PlayerID())
			a.AddToHand(aura)
			a.ManaPool().Add(core.White, 1)
			if len(TargetCreature().Possible(a.PlayerID(), SpellTargetSource(aura), g)) != 0 {
				t.Fatal("restricted candidate")
			}
			if err := g.CastSpellByName(a.PlayerID(), aura.Name(), []uuid.UUID{victim.ID()}); err == nil || len(a.Hand()) != 1 || a.ManaPool().Count(core.White) != 1 || g.StackSize() != 0 {
				t.Fatal("restricted cast or rollback")
			}
			ids := g.PhaseOut(victim.ID())
			g.PhaseIn(ids)
			if err := g.CastSpellByName(a.PlayerID(), aura.Name(), []uuid.UUID{victim.ID()}); err == nil {
				t.Fatal("phase cycle lost rule")
			}
			clone := g.Clone()
			switch change {
			case "phase":
				g.PhaseOut(victim.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				g.PutOnBattlefield(victimCard, b.PlayerID())
			case "remove":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), victim.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: victim.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			allowed := change == "remove"
			err := g.CastSpellByName(a.PlayerID(), aura.Name(), []uuid.UUID{victim.ID()})
			if (err == nil) != allowed {
				t.Fatalf("allowed=%v err=%v", allowed, err)
			}
			if allowed {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				g.ResolveStack()
				if g.FindPermanent(aura.ID()) != nil {
					t.Fatal("restored rule did not reject original and copy")
				}
			}
			if err := clone.CastSpellByName(a.PlayerID(), aura.Name(), []uuid.UUID{victim.ID()}); err == nil {
				t.Fatal("clone rule changed")
			}
		})
	}
}

func TestCantBeTargetedBySpellsAbilitySourceDeparture(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 2, 7, WithAbility(Static(CantBeTargetedBySpells(This, HasSubTypeCardFilter("Aura"))))), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	sourceCard := NewAura("Ability Aura", "", WithAbility(Activate(Costs(), DealDamageFrom(This, target, Fixed(2)), WithTargets(target))))
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.FindPermanent(victim.ID()).Damage != 4 {
			t.Fatal("Aura ability was treated as a spell")
		}
	}
}

func TestCantBeTargetedBySpellsColorAndPartialLegality(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		g, a, b := newSpellCopyGame()
		g.SetStep(core.PrecombatMain)
		victims := DeclareAnyNumberDamageTargets("recipients")
		spell := NewInstant("Damage", "{R}", Spell(DealDamageEach(This, victims, Fixed(2)), WithTargets(victims)))
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.Red, 1)
		protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 8, WithAbility(Static(CantBeTargetedBySpells(This, HasColorCardFilter(core.Blue))))), b.PlayerID())
		other := g.PutOnBattlefield(NewCreature("Other", "", 2, 8), b.PlayerID())
		targets := []uuid.UUID{protected.ID()}
		if mixed {
			targets = append(targets, other.ID())
		}
		if err := g.CastSpellByName(a.PlayerID(), spell.Name(), targets); err != nil {
			t.Fatal(err)
		}
		if !g.ChangeObjectColor(g.StackPeek().ID, core.Blue) {
			t.Fatal("color change failed")
		}
		copiedSpell := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
		if copiedSpell == nil {
			t.Fatal("copy failed")
		}
		if !g.ChangeObjectColor(copiedSpell.ID, core.Blue) {
			t.Fatal("copy color change failed")
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		for _, game := range []*Game{g, clone} {
			if game.FindPermanent(protected.ID()).Damage != 0 {
				t.Fatal("current spell color was ignored")
			}
			want := 0
			if mixed {
				want = 4
			}
			if game.FindPermanent(other.ID()).Damage != want {
				t.Fatal("legal target did not resolve independently")
			}
		}
	}
}

func TestCantBeTargetedBySpellsDoesNotPreventAttachment(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 8, WithAbility(Static(CantBeTargetedBySpells(This, HasSubTypeCardFilter("Aura"))))), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Attach", ""), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	if g.FindPermanent(aura.ID()).AttachedTo != host.ID() {
		t.Fatal("nontargeted attachment was restricted")
	}
}

func TestCantBeTargetedBySpellsRetargetCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	protected := g.PutOnBattlefield(NewCreature("Protected", "", 2, 5, WithAbility(Static(CantBeTargetedBySpells(This, HasSubTypeCardFilter("Aura"))))), b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 5), b.PlayerID())
	aura := NewAura("Aura", "{W}")
	aura.SetOwner(a.PlayerID())
	a.AddToHand(aura)
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByName(a.PlayerID(), aura.Name(), []uuid.UUID{other.ID()}); err != nil {
		t.Fatal(err)
	}
	b.chooseQueue = [][]uuid.UUID{{other.ID()}}
	copiedSpell := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copiedSpell == nil || len(b.calls) == 0 {
		t.Fatal("missing copy choice")
	}
	for _, id := range b.calls[len(b.calls)-1].possible {
		if id == protected.ID() {
			t.Fatal("copy offered restricted target")
		}
	}
	g.ResolveStack()
	if g.FindPermanent(aura.ID()) == nil {
		t.Fatal("legal Aura did not enter")
	}
}
