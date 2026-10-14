package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDamageCausePrevention(t *testing.T) {
	for _, state := range []string{"spell", "ability", "untargeted", "combat", "none", "clone", "return", "phase-cycle", "unpreventable"} {
		t.Run(state, func(t *testing.T) {
			g, a, b, obj := spellSourceFixture(t)
			card := NewCreature("Victim", "", 1, 20)
			victim := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(NewCreature("Other source", "", 1, 20), b.PlayerID())
			if err := ApplyEffect(g, PreventDamageCausedByTargetingThisTurn(This), victim.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			obj.Targets = []uuid.UUID{victim.ID()}
			obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
			if state == "ability" {
				obj.IsAbility = true
			}
			if state == "untargeted" {
				obj.Targets = nil
			}
			end := g.resolution.Begin(obj)
			defer end()
			want := 0
			switch state {
			case "none":
				g.resolution.Clear()
				want = 2
			case "untargeted":
				want = 2
			case "combat":
				g.damage.SetResolvingCombatDamage(true)
				want = 2
			case "clone":
				g = g.Clone()
			case "return":
				g.RemoveFromBattlefield(victim)
				victim = g.PutOnBattlefield(card, a.PlayerID())
				want = 2
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "unpreventable":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 2
			}
			g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(source.ID())), PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, g.CaptureDamageSource(source.ID())), PermanentDamageAssignment(g.FindPermanent(source.ID()), 1, g.CaptureDamageSource(source.ID())), PlayerDamageAssignment(g.GetPlayer(a.PlayerID()), 1, g.CaptureDamageSource(source.ID())))
			if got := g.FindPermanent(victim.ID()).Damage; got != want {
				t.Fatalf("victim damage %d, want %d", got, want)
			}
			if g.FindPermanent(source.ID()).Damage != 1 || g.GetPlayer(a.PlayerID()).Life() != 19 {
				t.Fatal("prevention escaped recipient")
			}
		})
	}
}

func TestDamageCauseSnapshots(t *testing.T) {
	g, a, _, obj := spellSourceFixture(t)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), a.PlayerID())
	obj.Targets = []uuid.UUID{victim.ID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	copied := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy")
	}
	original := damageCauseFromStack(obj)
	copyCause := damageCauseFromStack(copied)
	if original.stackID == copyCause.stackID || len(copyCause.targets) != 1 {
		t.Fatal("copy cause identity")
	}
	copied.Targets = nil
	if len(copyCause.targets) != 1 || len(damageCauseFromStack(copied).targets) != 0 || len(original.targets) != 1 {
		t.Fatal("target snapshots shared")
	}
	g.GetStack().RemoveByID(obj.ID)
	end := g.resolution.Begin(obj)
	obj.Targets = nil
	clone := g.Clone()
	g.resolution.damageCause.targets[0] = capturedPermanent{}
	if clone.resolution.damageCause.targets[0].id != victim.ID() {
		t.Fatal("clone shared cause targets")
	}
	end()
	if g.resolution.damageCause.stackID != uuid.Nil || len(g.resolution.damageCause.targets) != 0 {
		t.Fatal("cleared frame kept cause")
	}
	if damageCauseFromStack(nil).stackID != uuid.Nil || damageCauseFromStack(&StackObject{ID: uuid.New()}).stackID != uuid.Nil {
		t.Fatal("invalid object has cause")
	}
	obj.ID = uuid.Nil
	if damageCauseFromStack(obj).stackID != uuid.Nil {
		t.Fatal("absent stack identity matched")
	}
}

func TestDamageCauseReplacementRecheck(t *testing.T) {
	for _, batch := range []bool{false, true} {
		for _, change := range []string{"amount", "redirect-in", "redirect-out", "return", "phase", "source-departure"} {
			t.Run(change+map[bool]string{true: "/batch", false: "/single"}[batch], func(t *testing.T) {
				g, a, b, obj := spellSourceFixture(t)
				card := NewCreature("Victim", "", 1, 20)
				victim := g.PutOnBattlefield(card, a.PlayerID())
				other := g.PutOnBattlefield(NewCreature("Other", "", 1, 20), b.PlayerID())
				if err := ApplyEffect(g, PreventDamageCausedByTargetingThisTurn(This), victim.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				obj.IsAbility = true
				obj.Targets = []uuid.UUID{victim.ID()}
				obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
				end := g.resolution.Begin(obj)
				defer end()
				source := g.CaptureDamageSource(other.ID())
				recipient := victim
				if change == "redirect-in" {
					recipient = other
				}
				g.AddReplacementEffect(&testPrioritizedReplacement{
					replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf},
					matchFunc:       func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok },
					replaceFunc: func(action Action, game *Game) Action {
						damage := action.(*DamageToCreatureAction)
						switch change {
						case "amount":
							return damage.WithAmount(3)
						case "redirect-in":
							return NewDamageToCreatureAction(damage.ActionSource(), victim.ID(), damage.Amount(), false).withDamageSource(damage.DamageSource())
						case "redirect-out":
							return NewDamageToCreatureAction(damage.ActionSource(), other.ID(), damage.Amount(), false).withDamageSource(damage.DamageSource())
						case "return":
							game.RemoveFromBattlefield(game.FindPermanent(victim.ID()))
							game.PutOnBattlefield(card, a.PlayerID())
						case "phase":
							game.PhaseOut(victim.ID())
						case "source-departure":
							game.RemoveFromBattlefield(game.FindPermanent(other.ID()))
						}
						return action
					},
				})
				if batch {
					g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(recipient.ID()), 2, source))
				} else {
					g.DealDamageToPermanentFromSource(g.FindPermanent(recipient.ID()), 2, source)
				}
				if g.FindPermanentIncludingPhased(victim.ID()).Damage != 0 {
					t.Fatal("cause or recipient recheck")
				}
				if change == "redirect-out" && g.FindPermanent(other.ID()).Damage != 2 {
					t.Fatal("redirect out retained prevention")
				}
			})
		}
	}
}

func TestDamageCauseDefinitionAndWaiting(t *testing.T) {
	invalid := NewInstant("Invalid", "", WithAbility(Spell(PreventDamageCausedByTargetingThisTurn(nil))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("nil subject accepted")
	}
	g, a, _, _ := spellSourceFixture(t)
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), a.PlayerID())
	if err := ApplyEffect(g, PreventDamageCausedByTargetingThisTurn(This), victim.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	sources := g.WaitingPreventionPermanentSources()
	if len(sources) != 1 || sources[0].ID() != victim.ID() || victim.Damage != 0 {
		t.Fatal("waiting subject")
	}
}
