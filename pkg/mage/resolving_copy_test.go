package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func resolvingCopyCard(t *testing.T, build CardFactory) Card {
	t.Helper()
	plan, err := NewCardDefinitionPlan(build)
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	return card
}

func TestResolvingCopyRulesAndIdentity(t *testing.T) {
	g, a, b := newSpellCopyGame()
	old := resolvingCopyCard(t, func() Card {
		return NewCreature("Old", "{U}", 2, 5, WithKeyword(core.Flying), WithStaticAbility(BoostAllCreaturesIncludingSelf(1, 1, AnyPermanent)))
	})
	donor := resolvingCopyCard(t, func() Card {
		return NewCreature("Donor", "{R}", 3, 6, WithKeyword(core.Trample), WithStaticAbility(BoostAllCreaturesIncludingSelf(2, 2, AnyPermanent)), WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	})
	p := g.PutOnBattlefield(old, a.PlayerID())
	d := g.PutOnBattlefield(donor, b.PlayerID())
	p.Tapped, p.Damage = true, 2
	p.AddCounter(core.P1P1, 1)
	visit := p.incarnationID
	clone := g.Clone()
	if err := g.BecomeCopyOfPermanent(p, d, true); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	if p.Name() != "Donor" || p.incarnationID != visit || !p.Tapped || p.Damage != 2 || p.Counters[core.P1P1] != 1 || p.ControllerID() != a.PlayerID() {
		t.Fatal("copy changed object state")
	}
	if p.HasAttr(core.Flying) || !p.HasAttr(core.Trample) || p.CurrentPower(g) != 8 || g.FindPermanent(d.ID()).CurrentPower(g) != 7 {
		t.Fatal("old static rule remained or copied static rule missing", p.CurrentPower(g), d.CurrentPower(g))
	}
	if len(p.RuntimeAbilities) != 2 || p.RuntimeAbilities[1].AbilityID() == d.RuntimeAbilities[1].AbilityID() {
		t.Fatal("copy shared rules")
	}
	if len(p.Colors()) != 1 || p.Colors()[0] != core.Blue {
		t.Fatal("retained color")
	}
	if clone.FindPermanent(p.ID()).Name() != "Old" {
		t.Fatal("copy changed clone")
	}
	values, err := g.CaptureCopiableValues(p)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := values.Build()
	if err != nil || fresh.Name() != "Donor" || fresh.ManaCost().Red != 1 {
		t.Fatal("later copy", err)
	}
	g.BouncePermanentToHand(d)
	g.PhaseIn(g.PhaseOut(p.ID()))
	if g.FindPermanent(p.ID()).Name() != "Donor" {
		t.Fatal("copy followed donor or phase")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if a.Hand()[0].Name() != "Old" {
		t.Fatal("departure lost printed card")
	}
}

func TestResolvingCopyDeclarations(t *testing.T) {
	for _, e := range []ResolvingEffect{CopyPermanent(nil, This, false), CopyPermanent(This, nil, false)} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), e)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("nil copy reference accepted")
		}
	}
}

func TestResolvingCopyFailuresAndSelf(t *testing.T) {
	for _, change := range []string{"recipient missing", "recipient return", "recipient phase", "subject missing", "subject return", "subject phase", "unknown provenance", "self", "face-down donor", "face-down recipient"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			p := g.PutOnBattlefield(entryCopyPhysical(t), a.PlayerID())
			d := g.PutOnBattlefield(copiableTestCard(t), b.PlayerID())
			want := false
			switch change {
			case "recipient missing":
				g.BouncePermanentToHand(p)
			case "recipient return":
				g.BouncePermanentToHand(p)
				c := a.Hand()[0]
				a.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, a.PlayerID())
			case "recipient phase":
				g.PhaseOut(p.ID())
			case "subject missing":
				g.BouncePermanentToHand(d)
			case "subject return":
				g.BouncePermanentToHand(d)
				c := b.Hand()[0]
				b.RemoveFromHand(c.ID())
				g.PutOnBattlefield(c, b.PlayerID())
			case "subject phase":
				g.PhaseOut(d.ID())
			case "unknown provenance":
				d = g.PutOnBattlefield(NewCreature("Unknown", "", 2, 2), b.PlayerID())
			case "self":
				d = p
				want = true
			case "face-down donor":
				g.MutablePermanent(d.ID()).FaceDown = true
				want = true
			case "face-down recipient":
				g.MutablePermanent(p.ID()).FaceDown = true
				want = true
			}
			err := g.BecomeCopyOfPermanent(p, d, false)
			if (err == nil) != want {
				t.Fatal("copy validity", err)
			}
			if want && change == "face-down donor" {
				p = g.FindPermanent(p.ID())
				if p.FaceDown || p.Name() != "" || p.CurrentPower(g) != 2 || len(p.RuntimeAbilities) != 0 {
					t.Fatal("face-down values")
				}
			}
			if want && change == "face-down recipient" {
				p = g.FindPermanent(p.ID())
				if !p.FaceDown || p.CurrentPower(g) != 2 {
					t.Fatal("status changed")
				}
				g.turnFaceUp(p)
				if g.FindPermanent(p.ID()).CurrentPower(g) != 1 {
					t.Fatal("hidden copy lost")
				}
			}
		})
	}
}

func TestResolvingCopyLaterLayersAndPending(t *testing.T) {
	g, a, b := newSpellCopyGame()
	build := func() Card {
		return NewCreature("Trigger", "{U}", 2, 4, WithAbility(WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("upkeep")), GainLifeFor(You, Fixed(2)))))
	}
	p := g.PutOnBattlefield(resolvingCopyCard(t, build), a.PlayerID())
	d := g.PutOnBattlefield(copiableTestCard(t), b.PlayerID())
	g.SetStep(core.Upkeep)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("pending trigger")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, ModifyPT(This, Fixed(3), Fixed(1)), GrantKeywordModification(This, core.Haste)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.BecomeCopyOfPermanent(p, d, false); err != nil {
		t.Fatal(err)
	}
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != 4 || !p.HasAttr(core.Haste) {
		t.Fatal("later layers lost")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("pending old rule lost")
	}
}

func TestResolvingCopyWaiting(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	p := g.PutOnBattlefield(entryCopyPhysical(t), a.PlayerID())
	saved := g.CaptureDamageSource(p.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, CopyPermanent(This, This, false)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		sources := g.WaitingDelayedPermanentSources()
		if len(sources) != 1 || !sources[0].SameObject(saved) || g.FindPermanent(p.ID()).resolvingCopy != nil {
			t.Fatal("waiting query lost visit or copied rules")
		}
	}
	g.PhaseOut(p.ID())
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting phase LKI")
	}
	g.BouncePermanentToHand(g.FindPermanentIncludingPhased(p.ID()))
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting departure LKI")
	}
}
