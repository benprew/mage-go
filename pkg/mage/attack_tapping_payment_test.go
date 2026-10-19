package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttackTappingPayment(t *testing.T) {
	for _, state := range []string{"paid", "declined", "tapped", "phase", "return", "left", "abilities", "control", "late", "clone"} {
		t.Run(state, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: state != "declined"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetTurn(2)
			g.SetStep(core.BeginCombat)
			body := PayActionOrElse(You, StaticAbilityPayment(This, CantAttack(This), core.EndOfCombat), AttacksWithoutTappingUntil(You, IsCreature, PermanentIsUntapped(This), core.EndOfCombat), nil)
			c := NewCreature("Source", "", 5, 4, WithAbility(Activate(Costs(), body)))
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if source.CanDeclareAsAttacker(g) != (state == "declined") {
				t.Fatal("payment restriction")
			}
			clone := g.Clone()
			switch state {
			case "tapped":
				g.TapPermanent(g.FindPermanent(source.ID()))
			case "phase":
				g.PhaseOut(source.ID())
			case "return", "left":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "return" {
					g.PutOnBattlefield(c, a.PlayerID())
				}
			case "abilities":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
				g.AddContinuousEffect(RemoveAllAbilities(attacker.ID(), core.EndOfTurn))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			case "late":
				attacker = g.PutOnBattlefield(NewCreature("Late", "", 2, 4), a.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
			}
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			wantTap := state == "declined" || state == "tapped" || state == "phase" || state == "return" || state == "left"
			if !g.GetCombat().IsAttacking(attacker.ID()) || g.FindPermanent(attacker.ID()).Tapped != wantTap {
				t.Fatal("attack tapping", g.FindPermanent(attacker.ID()).Tapped, wantTap)
			}
			if g.FindPermanent(attacker.ID()).HasKeyword(core.Vigilance) {
				t.Fatal("rule grants vigilance")
			}
			if state == "phase" {
				g.PhaseIn([]uuid.UUID{source.ID()})
				if !g.attackDoesNotTap(g.FindPermanent(attacker.ID())) {
					t.Fatal("phase cycle")
				}
			}
			if state == "tapped" {
				g.UntapPermanent(g.FindPermanent(source.ID()))
				if !g.attackDoesNotTap(g.FindPermanent(attacker.ID())) {
					t.Fatal("untap did not restore rule")
				}
			}
			g.GetEffects().RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			if g.attackDoesNotTap(g.FindPermanent(attacker.ID())) {
				t.Fatal("rule duration")
			}
			if state == "paid" || state == "clone" {
				if !g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) || !clone.attackDoesNotTap(clone.FindPermanent(attacker.ID())) {
					t.Fatal("grant expiry or clone isolation")
				}
			}
		})
	}
}

func TestAttackTappingPaymentDefinitions(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		AttacksWithoutTappingUntil(nil, IsCreature, PermanentIsUntapped(This), core.EndOfCombat),
		AttacksWithoutTappingUntil(You, IsCreature, nil, core.EndOfCombat),
		AttacksWithoutTappingUntil(You, IsCreature, PermanentIsUntapped(This), core.WhileOnBattlefield),
		PayActionOrElse(You, StaticAbilityPayment(nil, CantAttack(This), core.EndOfCombat), GainLifeFor(You, Fixed(1)), nil),
		PayActionOrElse(You, StaticAbilityPayment(This, nil, core.EndOfCombat), GainLifeFor(You, Fixed(1)), nil),
		PayActionOrElse(You, StaticAbilityPayment(This, CantAttack(This), core.WhileOnBattlefield), GainLifeFor(You, Fixed(1)), nil),
	} {
		c := NewCreature("Invalid", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid definition accepted")
		}
	}
}

func TestAttackTappingPaymentDeclarationPaths(t *testing.T) {
	for _, path := range []string{"native", "explicit"} {
		t.Run(path, func(t *testing.T) {
			a := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetTurn(2)
			g.SetActivePlayerIndex(0)
			g.SetStep(core.BeginCombat)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			p := g.PutOnBattlefield(NewCreature("Match", "", 2, 4), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			q := g.PutOnBattlefield(NewCreature("Other", "", 2, 4, WithKeyword(core.Flying)), a.PlayerID())
			q.RevokeBaseAttr(core.AttrSummonSick)
			if err := ApplyEffect(g, AttacksWithoutTappingUntil(You, Not(HasKeywordFilter(core.Flying)), PermanentIsUntapped(This), core.EndOfCombat), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			a.attackers = []uuid.UUID{p.ID(), q.ID()}
			if path == "native" {
				g.doDeclareAttackers()
			} else {
				g.ExecuteAttackers(a.PlayerID(), a.attackers)
			}
			if !g.GetCombat().IsAttacking(p.ID()) || !g.GetCombat().IsAttacking(q.ID()) || g.FindPermanent(p.ID()).Tapped || !g.FindPermanent(q.ID()).Tapped {
				t.Fatal("declaration path")
			}
			g.RunStepWithPriority(core.EndCombat)
			if g.attackDoesNotTap(g.FindPermanent(p.ID())) {
				t.Fatal("native combat expiry")
			}
		})
	}
}

type attackTappingRecheckPlayer struct {
	*BasePlayer
	change  func()
	choices int
}

func (p *attackTappingRecheckPlayer) ChooseMayAbility(string) bool {
	p.choices++
	if p.change != nil {
		p.change()
	}
	return true
}
func TestAttackTappingPaymentChoiceRecheck(t *testing.T) {
	for _, change := range []string{"left", "return", "phase", "cycle", "tapped"} {
		t.Run(change, func(t *testing.T) {
			a := &attackTappingRecheckPlayer{BasePlayer: NewBasePlayer("A")}
			g := NewGame(a, NewBasePlayer("B"))
			c := NewCreature("Source", "", 2, 4)
			source := g.PutOnBattlefield(c, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			a.change = func() {
				switch change {
				case "left", "return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "return" {
						g.PutOnBattlefield(c, a.PlayerID())
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				case "tapped":
					g.TapPermanent(g.FindPermanent(source.ID()))
				}
			}
			effect := PayActionOrElse(You, StaticAbilityPayment(This, CantAttack(This), core.EndOfCombat), GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(10)))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := 30
			if change == "cycle" || change == "tapped" {
				want = 21
			}
			if a.Life() != want || a.choices != 1 {
				t.Fatal("availability recheck", a.Life(), want)
			}
		})
	}
}

func TestAttackTappingPaymentWaitingSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := PayActionOrElse(You, StaticAbilityPayment(This, CantAttack(This), core.EndOfCombat), GainLifeFor(You, Fixed(1)), nil)
	c := NewArtifact("Source", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))
	source := g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("static payment source absent from delayed queries")
	}
}

type attackTappingDeclarationPlayer struct{ *playerRestrictionAttacker }

func (*attackTappingDeclarationPlayer) ChooseMayAbility(string) bool { return true }

func TestAttackTappingPaymentBeforeAttackCosts(t *testing.T) {
	for _, path := range []string{"native", "explicit"} {
		for _, amount := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/%d", path, amount), func(t *testing.T) {
				a := &attackTappingDeclarationPlayer{&playerRestrictionAttacker{BasePlayer: NewBasePlayer("A")}}
				g := NewGame(a, NewBasePlayer("B"))
				g.SetActivePlayerIndex(0)
				g.SetTurn(2)
				g.SetStep(core.BeginCombat)
				source := g.PutOnBattlefield(NewArtifact("Source", "", WithManaAbility(core.Green)), a.PlayerID())
				p := g.PutOnBattlefield(NewCreature("Taxed", "", 2, 4, WithStaticAbility(SourceCantAttackUnlessPays(GenericCost(amount)))), a.PlayerID())
				q := g.PutOnBattlefield(NewCreature("Free", "", 2, 4), a.PlayerID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				q.RevokeBaseAttr(core.AttrSummonSick)
				if err := ApplyEffect(g, AttacksWithoutTappingUntil(You, IsCreature, PermanentIsUntapped(This), core.EndOfCombat), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				a.attackers = []uuid.UUID{p.ID(), q.ID()}
				if path == "native" {
					g.doDeclareAttackers()
				} else {
					g.ExecuteAttackers(a.PlayerID(), a.attackers)
				}
				if g.GetCombat().IsAttacking(p.ID()) != (amount == 1) || !g.GetCombat().IsAttacking(q.ID()) || g.FindPermanent(p.ID()).Tapped || g.FindPermanent(q.ID()).Tapped {
					t.Fatal("attack costs changed earlier tapping decisions")
				}
				if g.FindPermanent(source.ID()).Tapped != (amount == 1) || a.ManaPool().TotalMana() != 0 {
					t.Fatal("payment or rollback")
				}
			})
		}
	}
}

func TestAttackTappingPaymentSavesConditionBindings(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), a.PlayerID())
	binding := BindPlayer("condition player")
	g.resolution.playerBindings = map[*PlayerBinding]uuid.UUID{binding: a.PlayerID()}
	effect := Sequence(SnapshotPlayer(You, binding), AttacksWithoutTappingUntil(You, IsCreature, PlayerControls(FromPlayer(binding), IsArtifact), core.EndOfCombat), ChoosePlayer(You, OpponentsOf(You), binding))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.resolution.playerBindings[binding] = b.PlayerID()
	if !g.attackDoesNotTap(attacker) {
		t.Fatal("condition lost its saved player")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(source)
	if !clone.attackDoesNotTap(clone.FindPermanent(attacker.ID())) {
		t.Fatal("condition clone changed")
	}
}
