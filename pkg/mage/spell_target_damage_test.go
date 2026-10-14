package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellTargetDamageFacts(t *testing.T) {
	g, a, _, obj := spellSourceFixture(t)
	card := NewCreature("Victim", "", 1, 10)
	victim := g.PutOnBattlefield(card, a.PlayerID())
	obj.Targets = []uuid.UUID{victim.ID(), a.PlayerID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
	saved := g.CaptureSpellDamageSource(obj.ID)
	if !saved.TargetsPermanent(victim) {
		t.Fatal("live spell lost its permanent target")
	}
	cloned := cloneDamageSource(saved)
	obj.Targets = nil
	if !saved.TargetsPermanent(victim) || !cloned.TargetsPermanent(victim) {
		t.Fatal("source facts changed with stack targets")
	}
	obj.Targets = []uuid.UUID{victim.ID()}
	g.GetStack().Pop()
	end := g.resolution.Begin(obj)
	resolving := g.CaptureDamageSource(obj.SourceID)
	if !resolving.TargetsPermanent(victim) || !g.Clone().CaptureDamageSource(obj.SourceID).TargetsPermanent(victim) {
		t.Fatal("resolution or clone lost targets after stack departure")
	}
	end()
	if g.CaptureDamageSource(obj.SourceID).TargetsPermanent(victim) {
		t.Fatal("cleared resolution kept a spell")
	}
	g.PhaseOut(victim.ID())
	if !saved.TargetsPermanent(g.FindPermanentIncludingPhased(victim.ID())) {
		t.Fatal("phase changed saved identity")
	}
	g.PhaseIn([]uuid.UUID{victim.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if saved.TargetsPermanent(returned) || saved.TargetsPermanent(nil) {
		t.Fatal("target followed a new visit")
	}
	unknown := saved
	unknown.stackID = uuid.Nil
	if unknown.TargetsPermanent(victim) {
		t.Fatal("unknown spell identity matched")
	}
}

func TestSpellTargetDamagePrevention(t *testing.T) {
	for _, change := range []string{"targeted", "untargeted", "ability", "combat", "unpreventable", "return", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, obj := spellSourceFixture(t)
			card := NewCreature("Victim", "", 1, 10, WithAbility(Static(PreventDamageFromSpellsTargeting(This))))
			victim := g.PutOnBattlefield(card, a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 10), b.PlayerID())
			obj.Targets = []uuid.UUID{victim.ID()}
			obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
			saved := g.CaptureSpellDamageSource(obj.ID)
			want := 0
			switch change {
			case "untargeted":
				obj.Targets = nil
				saved = g.CaptureSpellDamageSource(obj.ID)
				want = 1
			case "ability":
				saved = g.CaptureDamageSource(other.ID())
				want = 1
			case "combat":
				g.damage.SetResolvingCombatDamage(true)
				want = 1
			case "unpreventable":
				g.MutablePermanent(victim.ID()).GrantBaseAttr(core.AttrDamageCantBePreventedOrRedirected)
				want = 1
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(card, a.PlayerID())
				want = 1
			case "clone":
				g = g.Clone()
			}
			for range 2 {
				g.DealSimultaneousDamage(PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, saved), PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, saved), PlayerDamageAssignment(g.GetPlayer(a.PlayerID()), 1, saved))
			}
			if g.FindPermanent(victim.ID()).Damage != 2*want || g.FindPermanent(other.ID()).Damage != 2 || g.GetPlayer(a.PlayerID()).Life() != 18 {
				t.Fatal("targeted spell prevention or packet scope")
			}
		})
	}
}

func TestSpellTargetDamageInvalid(t *testing.T) {
	for _, rule := range []StaticRule{
		PreventDamageFromSpellsTargeting(nil),
		PreventDamageFromSpellsTargeting(DeclarePermanentTarget("target", IsCreature)),
		PreventDamageFromSpellsTargetingIf(This, nil),
	} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Static(rule)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSpellTargetDamageCopySnapshots(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 10), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 10), b.PlayerID())
	obj.Targets = []uuid.UUID{first.ID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{first.ID(): first.incarnationID}
	saved := g.CaptureSpellDamageSource(obj.ID)
	copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy")
	}
	copiedSource := g.CaptureSpellDamageSource(copied.ID)
	if !copiedSource.TargetsPermanent(first) || copiedSource.SameObject(saved) {
		t.Fatal("copy target facts or stack identity")
	}
	copied.Targets = []uuid.UUID{second.ID()}
	copied.targetIncarnations[second.ID()] = second.incarnationID
	if !g.CaptureSpellDamageSource(copied.ID).TargetsPermanent(second) || g.CaptureSpellDamageSource(copied.ID).TargetsPermanent(first) || !saved.TargetsPermanent(first) {
		t.Fatal("copy target revision changed other snapshots")
	}
	binding := BindDamageSource("saved")
	spell := BindSpell("spell")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{spell: obj.ID}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	if err := SnapshotDamageSource(DamageSourceOfSpell(FromSpell(spell)), binding).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.resolution.damageSourceBindings[binding].permanentTargets[0] = capturedPermanent{id: second.ID(), incarnation: second.incarnationID}
	ctx.Game = clone
	if !FromDamageSource(binding).resolveDamageSource(ctx).TargetsPermanent(first) {
		t.Fatal("saved binding clone shared target facts")
	}
	g.GetStack().RemoveByID(obj.ID)
	if !saved.TargetsPermanent(first) {
		t.Fatal("departure lost target facts")
	}
	obj.IsAbility = true
	if damageSourceFromSpellObject(obj).TargetsPermanent(first) {
		t.Fatal("ability source accepted as spell")
	}
	obj.IsAbility = false
	obj.targetIncarnations = nil
	if damageSourceFromSpellObject(obj).TargetsPermanent(first) {
		t.Fatal("UUID-only target accepted as exact visit")
	}
}

func TestSpellTargetDamageRecheck(t *testing.T) {
	for _, change := range []string{"enable", "disable", "redirect", "phase", "return"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, obj := spellSourceFixture(t)
			condition := PermanentIsUntapped(This)
			card := NewCreature("Victim", "", 1, 10, WithAbility(Static(PreventDamageFromSpellsTargetingIf(This, condition))))
			victim := g.PutOnBattlefield(card, a.PlayerID())
			g.MutablePermanent(victim.ID()).Tapped = change == "enable"
			other := g.PutOnBattlefield(NewCreature("Other", "", 1, 10), b.PlayerID())
			obj.Targets = []uuid.UUID{victim.ID()}
			obj.targetIncarnations = map[uuid.UUID]uuid.UUID{victim.ID(): victim.incarnationID}
			saved := g.CaptureSpellDamageSource(obj.ID)
			g.AddReplacementEffect(&testPrioritizedReplacement{
				replacementBase: replacementBase{duration: core.Indefinite, priority: ReplacementPrioritySelf},
				matchFunc:       func(action Action) bool { _, ok := action.(*DamageToCreatureAction); return ok },
				replaceFunc: func(action Action, game *Game) Action {
					switch change {
					case "enable":
						game.MutablePermanent(victim.ID()).Tapped = false
					case "disable":
						game.MutablePermanent(victim.ID()).Tapped = true
					case "redirect":
						damage := action.(*DamageToCreatureAction)
						return NewDamageToCreatureAction(damage.ActionSource(), other.ID(), damage.Amount(), false).withDamageSource(damage.DamageSource())
					case "phase":
						game.PhaseOut(victim.ID())
					case "return":
						game.RemoveFromBattlefield(game.FindPermanent(victim.ID()))
						game.PutOnBattlefield(card, a.PlayerID())
					}
					return action
				},
			})
			g.DealDamageToPermanentFromSource(g.FindPermanent(victim.ID()), 1, saved)
			want := 0
			if change == "disable" {
				want = 1
			}
			if g.FindPermanentIncludingPhased(victim.ID()).Damage != want {
				t.Fatal("condition or visit recheck")
			}
			otherDamage := 0
			if change == "redirect" {
				otherDamage = 1
			}
			if g.FindPermanent(other.ID()).Damage != otherDamage {
				t.Fatal("redirected recipient")
			}
		})
	}
}

func TestSpellTargetDamageHostAndWaiting(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	host := g.PutOnBattlefield(NewCreature("Host", "", 1, 10), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 10), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Rule", "", WithAbility(Static(PreventDamageFromSpellsTargeting(AttachedPermanent)))), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	obj.Targets = []uuid.UUID{host.ID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{host.ID(): host.incarnationID}
	saved := g.CaptureSpellDamageSource(obj.ID)
	g.DealDamageToPermanentFromSource(g.FindPermanent(host.ID()), 1, saved)
	if g.FindPermanent(host.ID()).Damage != 0 {
		t.Fatal("host prevention")
	}
	sources := g.WaitingPreventionPermanentSources()
	found := false
	for _, source := range sources {
		if source.ID() == host.ID() {
			found = true
		}
	}
	if !found || g.FindPermanent(host.ID()).Damage != 0 {
		t.Fatal("pure waiting subject query")
	}
	g.Attach(aura.ID(), other.ID())
	g.DealDamageToPermanentFromSource(g.FindPermanent(host.ID()), 1, saved)
	if g.FindPermanent(host.ID()).Damage != 1 {
		t.Fatal("prevention followed old host")
	}
	obj.Targets = []uuid.UUID{other.ID()}
	obj.targetIncarnations = map[uuid.UUID]uuid.UUID{other.ID(): other.incarnationID}
	changed := g.CaptureSpellDamageSource(obj.ID)
	g.DealDamageToPermanentFromSource(g.FindPermanent(other.ID()), 1, changed)
	if g.FindPermanent(other.ID()).Damage != 0 {
		t.Fatal("current host prevention")
	}
	g.PhaseOut(aura.ID())
	g.DealDamageToPermanentFromSource(g.FindPermanent(other.ID()), 1, changed)
	if g.FindPermanent(other.ID()).Damage != 1 {
		t.Fatal("phased rule remained active")
	}
}
