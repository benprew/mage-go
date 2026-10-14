package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestructionReplacementNext(t *testing.T) {
	for _, change := range []string{"none", "clone", "return", "phase", "source-exit", "cleanup", "no-regeneration", "indestructible"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("subject", IsCreature)
			source := g.PutOnBattlefield(NewArtifact("Maker", "", WithAbility(Activate(Costs(), ReplaceNextDestructionThisTurn(target), WithTargets(target)))), a.PlayerID())
			subjectCard := NewCreature("Subject", "", 3, 3)
			subjectCard.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(subjectCard, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{p.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p = g.FindPermanent(p.ID())
			p.Damage = 2
			p.DeathtouchDamage = true
			clone := g.Clone()
			clone.DestroyPermanent(clone.FindPermanent(p.ID()))
			if clone.FindPermanent(p.ID()) == nil || g.FindPermanent(p.ID()).Damage != 2 {
				t.Fatal("clone shared replacement state")
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(p.Card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
				g.DestroyPermanent(p)
				g.PhaseIn([]uuid.UUID{p.ID()})
				p = g.FindPermanent(p.ID())
			case "source-exit":
				g.RemoveFromBattlefield(source)
			case "cleanup":
				g.effects.ClearReplacementsEndOfTurn()
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				g.DestroyPermanent(p)
				p.RevokeBaseAttr(core.Indestructible)
			}
			if change == "no-regeneration" {
				g.destroyPermanents([]*Permanent{p}, true)
			} else {
				g.DestroyPermanent(p)
			}
			saved := g.FindPermanent(p.ID())
			if change == "return" || change == "cleanup" {
				if saved != nil {
					t.Fatal("expired shield applied")
				}
				return
			}
			if saved == nil || saved.Damage != 0 || saved.DeathtouchDamage || saved.Tapped {
				t.Fatal("cancel destruction did not only remove damage")
			}
			g.DestroyPermanent(saved)
			if g.FindPermanent(p.ID()) != nil {
				t.Fatal("shield was not consumed")
			}
		})
	}
}

func TestDestructionReplacementStatic(t *testing.T) {
	for _, change := range []string{"none", "return", "phase", "remove-ability", "no-regeneration", "prohibited", "zero-toughness", "sacrifice", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 3, WithAbility(RegenerateInsteadOfDestroy()))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.Damage = 2
			p.DeathtouchDamage = true
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
				if g.effects.ApplyReplacements(NewDestroyPermanentAction(uuid.Nil, p.ID()), g) == nil {
					t.Fatal("phased source applied")
				}
				g.PhaseIn([]uuid.UUID{p.ID()})
				p = g.FindPermanent(p.ID())
			case "remove-ability":
				e := RemoveAllAbilities(p.ID(), core.EndOfTurn)
				g.AddContinuousEffect(e)
				g.ApplyContinuousEffects()
			case "prohibited":
				if err := ApplyEffect(g, Until(core.EndOfTurn, GrantKeywordModification(This, core.CantRegenerate)), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "zero-toughness":
				if err := ApplyEffect(g, Until(core.EndOfTurn, SetBasePowerToughness(This, Fixed(3), Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
				g.CheckStateBasedActions()
			case "sacrifice":
				g.DoSacrifice(p)
			case "clone":
				clone := g.Clone()
				clone.DestroyPermanent(clone.FindPermanent(p.ID()))
				if g.FindPermanent(p.ID()).Damage != 2 {
					t.Fatal("clone shared damage")
				}
			}
			if change == "no-regeneration" {
				g.destroyPermanents([]*Permanent{p}, true)
			} else {
				g.DestroyPermanent(p)
			}
			dead := change == "remove-ability" || change == "no-regeneration" || change == "prohibited" || change == "zero-toughness" || change == "sacrifice"
			live := g.FindPermanent(p.ID())
			if dead {
				if live != nil {
					t.Fatal("invalid regeneration")
				}
				return
			}
			if live == nil || !live.Tapped || live.Damage != 0 || live.DeathtouchDamage {
				t.Fatal("missing regeneration")
			}
			g.UntapPermanent(live)
			live.Damage = 2
			g.DestroyPermanent(live)
			if g.FindPermanent(p.ID()) == nil {
				t.Fatal("static replacement was consumed")
			}
		})
	}
}

func TestDestructionReplacementCombatAndBatch(t *testing.T) {
	for _, regenerate := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove-damage", true: "regenerate"}[regenerate], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var options []CardOption
			if regenerate {
				options = append(options, WithAbility(RegenerateInsteadOfDestroy()))
			}
			subjects := make([]*Permanent, 0, 3)
			for _, name := range []string{"First", "Second", "Unprotected"} {
				opts := options
				if name == "Unprotected" {
					opts = nil
				}
				card := NewCreature(name, "", 3, 3, opts...)
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				p.Damage = 2
				p.DeathtouchDamage = true
				g.combat.AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
				if !regenerate && name != "Unprotected" {
					if err := ApplyEffect(g, ReplaceNextDestructionThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				subjects = append(subjects, p)
			}
			g.DestroyPermanents(subjects)
			for _, p := range subjects[:2] {
				live := g.FindPermanent(p.ID())
				if live == nil || live.Damage != 0 || live.DeathtouchDamage || live.Tapped != regenerate || g.combat.IsAttacking(p.ID()) == regenerate {
					t.Fatal("wrong batch/combat replacement")
				}
			}
			if g.FindPermanent(subjects[2].ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("unprotected member survived")
			}
		})
	}
}

func TestDestructionReplacementOrderingAndRecheck(t *testing.T) {
	for _, selection := range []string{"shield", "static", "remove-static", "no-regeneration", "cancel"} {
		t.Run(selection, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 3, WithAbility(RegenerateInsteadOfDestroy()))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.Damage = 2
			if err := ApplyEffect(g, ReplaceNextDestructionThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			action := NewDestroyPermanentAction(uuid.Nil, p.ID())
			calls := 0
			result, _ := g.effects.prepareReplacementsChoosing(action, g, func(candidates []ReplacementEffect, action Action, game *Game) int {
				calls++
				if calls == 1 && len(candidates) != 2 {
					t.Fatalf("contenders=%d", len(candidates))
				}
				for i, r := range candidates {
					if _, ok := r.(*staticRegenerationReplacement); ok && selection != "shield" && selection != "cancel" {
						if selection == "remove-static" {
							p.RuntimeAbilities = nil
						}
						if selection == "no-regeneration" {
							action.(*DestroyPermanentAction).cannotRegenerate = true
						}
						return i
					}
				}
				return 0
			})
			if result != nil || p.Damage != 0 {
				t.Fatal("destruction was not replaced")
			}
			if p.Tapped != (selection == "static") {
				t.Fatal("replacement order or recheck failed")
			}
			shield := g.effects.replacements[0].(*nextDestructionReplacement)
			if shield.consumed == (selection == "static") {
				t.Fatal("wrong shield consumption")
			}
		})
	}
}

func TestDestructionReplacementDeclarations(t *testing.T) {
	card := NewInstant("Invalid", "", WithAbility(Spell(ReplaceNextDestructionThisTurn(nil))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil subject accepted")
	}
	card = NewInstant("Invalid static", "", WithAbility(RegenerateInsteadOfDestroy()))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nonbattlefield static source accepted")
	}
}

func TestDestructionReplacementWaitingSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 3, 3)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, ReplaceNextDestructionThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	refs := g.WaitingPersistentReplacementPermanentSources()
	if len(refs) != 1 || refs[0].Incarnation() != p.incarnationID {
		t.Fatal("waiting destruction lost exact subject")
	}
	returned := g.PutOnBattlefield(card, a.PlayerID())
	clone := g.Clone()
	g.effects.ClearReplacementsEndOfTurn()
	if len(g.WaitingPersistentReplacementPermanentSources()) != 0 || len(clone.WaitingPersistentReplacementPermanentSources()) != 1 {
		t.Fatal("waiting reference lifetime or clone failure")
	}
	refs = clone.WaitingPersistentReplacementPermanentSources()
	if refs[0].Incarnation() == returned.incarnationID {
		t.Fatal("returned object supplied waiting reference")
	}
}
