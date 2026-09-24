package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestTypedRegenerationConsumption(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Subject", "", 3, 4)
	c.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(c, a.PlayerID())
	p.Damage = 2
	g.combat.AddAttacker(c.ID(), b.PlayerID())
	body := Sequence(Regenerate(This), Regenerate(This))
	if !IsRegenerationEffect(body) {
		t.Fatal("missing regeneration metadata")
	}
	if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if p.Tapped || p.Damage != 2 {
		t.Fatal("installation regenerated immediately")
	}
	clone := g.Clone()
	for range 2 {
		clone.DestroyPermanent(clone.FindPermanent(c.ID()))
		survivor := clone.FindPermanent(c.ID())
		if survivor == nil || !survivor.Tapped || survivor.Damage != 0 || clone.combat.IsAttacking(c.ID()) {
			t.Fatal("shield did not replace destruction")
		}
	}
	clone.DestroyPermanent(clone.FindPermanent(c.ID()))
	if clone.FindPermanent(c.ID()) != nil {
		t.Fatal("consumed shields still active")
	}
	if !g.HasRegenerationShield(c.ID()) || p.Tapped || p.Damage != 2 {
		t.Fatal("clone changed original")
	}
	g.effects.ClearReplacementsEndOfTurn()
	g.DestroyPermanent(p)
	if g.FindPermanent(c.ID()) != nil {
		t.Fatal("shield did not expire")
	}
}

func TestTypedRegenerationSavedLifetime(t *testing.T) {
	for _, change := range []string{"same", "return", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewArtifact("Subject", "")
			p := g.PutOnBattlefield(c, a.PlayerID())
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: c.ID(), Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(c, a.PlayerID())
			case "phase":
				g.PhaseOut(c.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(c.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: c.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			if err := Regenerate(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{c.ID()})
			}
			want := change != "return" && change != "phase"
			if g.HasRegenerationShield(c.ID()) != want {
				t.Fatal("incorrect saved subject lifetime")
			}
			if want {
				g.PhaseIn(g.PhaseOut(c.ID()))
				g.DestroyPermanent(g.FindPermanent(c.ID()))
				if g.FindPermanent(c.ID()) == nil {
					t.Fatal("phasing lost installed shield")
				}
			}
		})
	}
}

func TestTypedRegenerationCopyTargets(t *testing.T) {
	g, a, b := newSpellCopyGame()
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	target := DeclarePermanentTarget("subject", IsCreature)
	ability := Spell(Regenerate(target), WithTargets(target))
	card := NewInstant("Shield", "", WithAbility(ability))
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	b.chooseQueue = [][]uuid.UUID{{second.ID()}}
	cp := g.CopyStackObjectDirect(obj, b.PlayerID(), true)
	if cp == nil || cp.Targets[0] != second.ID() {
		t.Fatal("copy did not retarget")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ResolveStack()
	if g.HasRegenerationShield(first.ID()) || !g.HasRegenerationShield(second.ID()) {
		t.Fatal("incorrect copy or target identity")
	}
}

func TestTypedRegenerationValidation(t *testing.T) {
	saved := BindPermanent("missing")
	for _, body := range []Effect{Regenerate(nil), Regenerate(This), Regenerate(From(saved)), Regenerate(DeclarePermanentTarget("undeclared", IsCreature))} {
		card := NewInstant("Invalid shield", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted %T", body)
		}
	}
}

func TestTypedRegenerationSourceActivation(t *testing.T) {
	for _, scenario := range []string{"present", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			returned := scenario == "returned"
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Source", "", 2, 2, WithAbility(Activate(Costs(ManaPayment("{G}")), Regenerate(This))))
			g.PutOnBattlefield(c, a.PlayerID())
			a.ManaPool().Add(Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || g.HasRegenerationShield(c.ID()) || a.ManaPool().TotalMana() != 0 {
				t.Fatal("incorrect payment or stack timing")
			}
			if returned {
				g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
				g.PutOnBattlefield(c, a.PlayerID())
			}
			g.ResolveStack()
			if g.HasRegenerationShield(c.ID()) == returned {
				t.Fatal("source identity was not retained")
			}
		})
	}
}

func TestTypedRegenerationDoesNotReplaceOtherRemoval(t *testing.T) {
	for _, action := range []string{"no regeneration", "sacrifice", "exile", "prohibited"} {
		t.Run(action, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Subject", "", 2, 2)
			c.SetOwner(a.PlayerID())
			if action == "prohibited" {
				WithKeyword(CantRegenerate)(c)
			}
			g.PutOnBattlefield(c, a.PlayerID())
			if err := ApplyEffect(g, Regenerate(This), c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !g.HasRegenerationShield(c.ID()) {
				t.Fatal("shield installation failed")
			}
			var body Effect
			switch action {
			case "no regeneration":
				body = DestroyWithoutRegeneration(This)
			case "sacrifice":
				body = Sacrifice(This)
			case "exile":
				body = Exile(This)
			case "prohibited":
				body = Destroy(This)
			}
			if err := ApplyEffect(g, body, c.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(c.ID()) != nil {
				t.Fatal("shield prevented removal")
			}
		})
	}
}
