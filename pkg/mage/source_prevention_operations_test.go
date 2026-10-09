package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func sourcePreventionOperationsEffect(source DamageSourceRef, gain bool) ResolvingEffect {
	if gain {
		return PreventNextDamageFromSelectedSourceWithOperations(source, You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(You))
	}
	return PreventNextDamageFromSelectedSourceWithOperations(source, You, DamageSourceFilter{}, PreventProposedDamage(HalfProposedDamageRoundedDown()))
}

func TestSourcePreventionOperationsAmounts(t *testing.T) {
	for _, gain := range []bool{false, true} {
		for _, packets := range [][]int{{0}, {1}, {2}, {5}, {1, 1}, {2, 3}} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 5, 10), b.PlayerID())
			effect := sourcePreventionOperationsEffect(This, gain)
			if err := ValidateCardDefinition("Shield", NewArtifact("Shield", "", WithAbility(Activate(Costs(), effect)))); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			source := g.CaptureDamageSource(p.ID())
			clone := g.Clone()
			total := 0
			var assignments []DamageAssignment
			for _, amount := range packets {
				total += amount
				assignments = append(assignments, PlayerDamageAssignment(a, amount, source))
			}
			g.DealSimultaneousDamage(assignments...)
			prevented := total / 2
			want := 20 - total + prevented
			if gain {
				prevented = total
				want = 20 + total
			}
			if a.Life() != want {
				t.Fatalf("gain=%v packets=%v life=%d want=%d", gain, packets, a.Life(), want)
			}
			if g.HasDealtDamageToPlayer(p.ID(), a.PlayerID()) != (total > prevented) {
				t.Fatal("wrong actual damage history")
			}
			before := a.Life()
			g.DealDamageToPlayer(a, 4, p.ID())
			wantLoss := 4
			if prevented == 0 {
				wantLoss = 2
				if gain {
					wantLoss = -4
				}
			}
			if a.Life() != before-wantLoss {
				t.Fatalf("wrong consumption gain=%v total=%d life=%d", gain, total, a.Life())
			}
			clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 4, p.ID())
			cloneWant := 18
			if gain {
				cloneWant = 24
			}
			if clone.GetPlayer(a.PlayerID()).Life() != cloneWant {
				t.Fatal("clone shared shield state")
			}
		}
	}
}

func TestSourcePreventionOperationsIdentity(t *testing.T) {
	for _, gain := range []bool{false, true} {
		for _, state := range []string{"depart", "return", "phase", "phase-cycle", "control", "cleanup"} {
			t.Run(state, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Source", "{R}", 5, 10)
				p := g.PutOnBattlefield(card, b.PlayerID())
				saved := g.CaptureDamageSource(p.ID())
				if err := ApplyEffect(g, sourcePreventionOperationsEffect(This, gain), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				switch state {
				case "depart", "return":
					g.RemoveFromBattlefield(p)
					if state == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
					g.zones.ClearLKI()
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: a.PlayerID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "cleanup":
					g.effects.ClearReplacementsEndOfTurn()
				}
				if state == "return" {
					g.DealDamageToPlayer(a, 1, p.ID())
					if a.Life() != 19 {
						t.Fatal("returned source inherited shield")
					}
				}
				before := a.Life()
				g.DealDamageToPlayerFromSource(b, 2, saved)
				g.DealDamageToPlayerFromSource(a, 5, saved)
				want := before - 3
				if gain {
					want = before + 5
				}
				if state == "cleanup" {
					want = before - 5
				}
				if a.Life() != want {
					t.Fatalf("gain=%v life=%d want=%d", gain, a.Life(), want)
				}
			})
		}
	}
}

func TestSourcePreventionOperationsDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		PreventNextDamageFromSelectedSourceWithOperations(nil, You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount())),
		PreventNextDamageFromSelectedSourceWithOperations(This, nil, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount())),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, nil),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, PreventProposedDamage(nil)),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, GainLifeFromPreventedDamage(You), PreventProposedDamage(ProposedDamageAmount())),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), PreventProposedDamage(ProposedDamageAmount())),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(nil)),
		PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(FromPlayer(BindPlayer("unbound gain player")))),
		sourcePreventionOperationsEffect(FromDamageSource(BindDamageSource("unbound")), true),
	} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), effect)))); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestSourcePreventionOperationsOrder(t *testing.T) {
	for _, gain := range []bool{false, true} {
		for choice := range 2 {
			a := &sourceFilterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: choice}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 5, 10), b.PlayerID())
			if err := ApplyEffect(g, sourcePreventionOperationsEffect(This, gain), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.AddPreventionShield(a.PlayerID(), 2)
			g.DealDamageToPlayer(a, 5, p.ID())
			want := 19
			if gain {
				want = 25
			}
			if choice == 1 {
				want = 18
				if gain {
					want = 23
				}
			}
			if a.Life() != want || a.prompts == 0 {
				t.Fatalf("gain=%v order=%d life=%d want=%d prompts=%d", gain, choice, a.Life(), want, a.prompts)
			}
		}
	}
}

func TestSourcePreventionOperationsLifeGain(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 5, 10), b.PlayerID())
	if err := ApplyEffect(g, sourcePreventionOperationsEffect(This, true), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.AddReplacement(&damageDuringLifeGain{replacementBase: replacementBase{sourceID: p.ID()}, playerID: a.PlayerID()})
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, g.CaptureDamageSource(p.ID())), PlayerDamageAssignment(a, 3, g.CaptureDamageSource(p.ID())))
	if a.Life() != 24 || g.PlayerLifeGainedThisTurn(a.PlayerID()) != 5 {
		t.Fatal("life gain reused consumed shield or split its outcome")
	}
}

func TestSourcePreventionOperationsSpellIdentity(t *testing.T) {
	for _, gain := range []bool{false, true} {
		for _, state := range []string{"spell", "depart", "recast", "continued", "continued-return", "copy", "recheck"} {
			t.Run(state, func(t *testing.T) {
				g, _, b, obj := spellSourceFixture(t)
				saved := g.CaptureSpellDamageSource(obj.ID)
				binding := BindDamageSource("chosen")
				g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
				if err := sourcePreventionOperationsEffect(FromDamageSource(binding), gain).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				delete(g.resolution.damageSourceBindings, binding)
				if state == "depart" {
					g.stack.Pop()
				}
				if state == "recast" {
					g.CounterStackObject(obj.ID)
					next := *obj
					next.ID = uuid.New()
					g.stack.Push(&next)
					g.DealDamageToPlayerFromSource(b, 1, g.CaptureSpellDamageSource(next.ID))
					if b.Life() != 19 {
						t.Fatal("new cast used old shield")
					}
				}
				if state == "continued" || state == "continued-return" || state == "copy" {
					if state == "copy" {
						copied := g.CopyStackObjectDirect(obj, obj.Controller, false)
						g.DealDamageToPlayerFromSource(b, 1, g.CaptureSpellDamageSource(copied.ID))
						if b.Life() != 19 {
							t.Fatal("copied spell inherited shield")
						}
						g.stack.Pop()
					}
					card := NewCreature("Permanent spell", "{R}", 5, 10)
					card.SetID(obj.SourceID)
					card.SetOwner(obj.Controller)
					obj.Card = card
					g.ResolveTopOfStack()
					p := g.FindPermanent(card.ID())
					if p == nil {
						t.Fatal("permanent spell did not enter")
					}
					saved = g.CaptureDamageSource(p.ID())
					if state == "continued-return" {
						g.RemoveFromBattlefield(p)
						g.PutOnBattlefield(card, obj.Controller)
						g.DealDamageToPlayer(b, 1, card.ID())
						if b.Life() != 19 {
							t.Fatal("returned continuation inherited shield")
						}
						g.zones.ClearLKI()
					}
					if len(g.WaitingPreventionPermanentSources()) != 1 {
						t.Fatal("continued source missing from waiting query")
					}
				} else if len(g.WaitingPreventionSpellSources()) != 1 {
					t.Fatal("spell source missing from waiting query")
				}
				if state == "recheck" {
					g.effects.ClearReplacementsEndOfTurn()
					g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: saved}
					var ops []SourcePreventionOperation
					if gain {
						ops = []SourcePreventionOperation{PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(You)}
					} else {
						ops = []SourcePreventionOperation{PreventProposedDamage(HalfProposedDamageRoundedDown())}
					}
					if err := PreventNextDamageFromSelectedSourceWithOperations(FromDamageSource(binding), You, DamageSourceHasColor(core.Red), ops...).Apply(&EffectContext{Game: g, Controller: b.PlayerID()}); err != nil {
						t.Fatal(err)
					}
					g.ChangeObjectColor(obj.ID, core.Blue)
					g.DealDamageToPlayerFromSource(b, 1, saved)
					if b.Life() != 19 {
						t.Fatal("source filter did not recheck")
					}
					g.ChangeObjectColor(obj.ID, core.Red)
				}
				for _, game := range []*Game{g, g.Clone()} {
					recipient := game.GetPlayer(b.PlayerID())
					before := recipient.Life()
					game.DealDamageToPlayerFromSource(recipient, 5, saved)
					want := before - 3
					if gain {
						want = before + 5
					}
					if recipient.Life() != want {
						t.Fatalf("gain=%v life=%d want=%d", gain, recipient.Life(), want)
					}
					game.DealDamageToPlayerFromSource(recipient, 2, saved)
					if recipient.Life() != want-2 {
						t.Fatal("wrong spell event consumption")
					}
				}
			})
		}
	}
}

func TestSourcePreventionOperationsLifeOutcomes(t *testing.T) {
	for _, state := range []string{"double", "replace", "zero", "lich", "trigger"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 5, 10), b.PlayerID())
			watcher := g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, ValueAtResolution(EventAmountValue())), false))), a.PlayerID())
			for range 5 {
				a.AddToLibrary(NewLand("Card"))
			}
			switch state {
			case "double":
				g.AddLifeGainModifier(watcher.ID(), DoubleLifeGainForController(watcher.ID(), a.PlayerID()))
			case "replace":
				g.effects.AddReplacement(&lifeOutcomeReplacement{amount: 7})
			case "zero":
				g.effects.AddReplacement(&lifeOutcomeReplacement{})
			case "lich":
				g.SetLichActive(a.PlayerID(), watcher.ID())
			}
			if err := ApplyEffect(g, sourcePreventionOperationsEffect(This, true), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, g.CaptureDamageSource(p.ID())), PlayerDamageAssignment(a, 3, g.CaptureDamageSource(p.ID())))
			want := 5
			switch state {
			case "double":
				want = 10
			case "replace":
				want = 7
			case "zero", "lich":
				want = 0
			}
			if a.Life() != 20+want || g.PlayerLifeGainedThisTurn(a.PlayerID()) != want || g.HasDealtDamageToPlayer(p.ID(), a.PlayerID()) {
				t.Fatal("actual gain did not use shared life service")
			}
			if state == "lich" && len(a.Hand()) != 5 {
				t.Fatal("life replacement did not draw")
			}
			g.PutTriggersOnStack()
			count := 0
			if want > 0 {
				count = 1
			}
			if g.StackSize() != count {
				t.Fatal("simultaneous prevention split life gain events")
			}
			g.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("life trigger did not retain actual outcome")
			}
		})
	}
}

type observePreventionGainTiming struct {
	replacementBase
	playerID     uuid.UUID
	permanent    capturedPermanent
	beforeLife   int
	beforeDamage int
}

func (*observePreventionGainTiming) IsActive(GameReader) bool { return true }
func (r *observePreventionGainTiming) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*LifeGainAction)
	return ok
}
func (r *observePreventionGainTiming) Clone() ReplacementEffect { c := *r; return &c }
func (r *observePreventionGainTiming) Replace(a Action, g *Game) Action {
	r.beforeLife = g.GetPlayer(r.playerID).Life()
	if p := g.FindPermanent(r.permanent.id); p != nil {
		r.beforeDamage = p.Damage
	}
	return a
}

func TestSourcePreventionOperationsAdditionalEffectTiming(t *testing.T) {
	for _, batch := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Source", "", 5, 10), b.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), a.PlayerID())
		effect := PreventNextDamageFromSelectedSourceWithOperations(This, You, DamageSourceFilter{}, PreventProposedDamage(HalfProposedDamageRoundedDown()), GainLifeFromPreventedDamage(You))
		if batch {
			effect = sourcePreventionOperationsEffect(This, true)
		}
		if err := ApplyEffect(g, effect, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		observer := &observePreventionGainTiming{playerID: a.PlayerID(), permanent: capturedPermanent{victim.ID(), victim.incarnationID}}
		g.effects.AddReplacement(observer)
		if batch {
			other := g.PutOnBattlefield(NewCreature("Other", "", 5, 10), b.PlayerID())
			g.DealSimultaneousDamage(PlayerDamageAssignment(a, 5, g.CaptureDamageSource(p.ID())), PlayerDamageAssignment(a, 3, g.CaptureDamageSource(other.ID())), PermanentDamageAssignment(victim, 2, g.CaptureDamageSource(other.ID())))
		} else {
			g.DealDamageToPlayer(a, 5, p.ID())
		}
		wantDamage := 0
		if batch {
			wantDamage = 2
		}
		if observer.beforeLife != 17 || observer.beforeDamage != wantDamage {
			t.Fatalf("batch=%v additional effect ran before damage: life=%d damage=%d", batch, observer.beforeLife, observer.beforeDamage)
		}
	}
}

func TestSourcePreventionOperationsGainPlayerAndWaiting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Provider", "", 5, 10), a.PlayerID())
	q := g.PutOnBattlefield(NewCreature("Selected source", "", 5, 10), b.PlayerID())
	binding := BindDamageSource("chosen")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: g.CaptureDamageSource(q.ID())}
	effect := PreventNextDamageFromSelectedSourceWithOperations(FromDamageSource(binding), You, DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(ControllerOfWithLKI(This)))
	body := AtNext(EachEndStep, effect)
	if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	old := g.CaptureDamageSource(p.ID())
	g.RemoveFromBattlefield(p)
	g.zones.ClearLKI()
	for _, game := range []*Game{g, g.Clone()} {
		found := false
		for _, source := range game.WaitingDelayedPermanentSources() {
			if source.SameObject(old) {
				found = true
			}
		}
		if !found {
			t.Fatal("explicit gain player lost its waiting source")
		}
	}

	g2 := NewGame(NewBasePlayer("C"), NewBasePlayer("D"))
	c, d := g2.PlayerAt(0), g2.PlayerAt(1)
	source := g2.PutOnBattlefield(NewCreature("Source", "", 5, 10), d.PlayerID())
	effect = PreventNextDamageFromSelectedSourceWithOperations(This, ControllerOf(This), DamageSourceFilter{}, PreventProposedDamage(ProposedDamageAmount()), GainLifeFromPreventedDamage(You))
	if err := ApplyEffect(g2, effect, source.ID(), c.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g2.DealDamageToPlayer(d, 5, source.ID())
	if c.Life() != 25 || d.Life() != 20 {
		t.Fatal("gain player followed protected player")
	}
}
