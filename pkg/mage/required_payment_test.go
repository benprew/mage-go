package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type requiredPaymentPlayer struct {
	*BasePlayer
	offers int
}

func (p *requiredPaymentPlayer) ChooseMayAbility(string) bool { p.offers++; return false }

func TestRequiredPayment(t *testing.T) {
	for _, branch := range []string{"paid", "unpaid"} {
		for _, situation := range []string{"available", "missing", "phased", "opponent", "rollback", "replaced"} {
			t.Run(branch+"/"+situation, func(t *testing.T) {
				a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				victim := NewCreature("Payment", "", 1, 1)
				victim.SetOwner(b.PlayerID())
				controller := a.PlayerID()
				if situation == "opponent" {
					controller = b.PlayerID()
				}
				p := g.PutOnBattlefield(victim, controller)
				if situation == "missing" {
					g.RemoveFromBattlefield(p)
				}
				if situation == "phased" {
					g.PhaseOut(p.ID())
				}
				costs := Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature"))
				if situation == "rollback" {
					costs = Costs(SacrificeCreatureCost(), SacrificeCreatureCost())
				}
				if situation == "replaced" {
					g.AddExileIfWouldGoToGraveyardThisTurn(victim.ID(), uuid.New())
				}
				body := GainLifeFor(You, Fixed(3))
				var effect = IfRequiredPaid(You, costs, body)
				if branch == "unpaid" {
					effect = UnlessRequiredPaid(You, costs, body)
				}
				if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				paid := situation == "available" || situation == "replaced"
				want := 20
				if paid == (branch == "paid") {
					want += 3
				}
				if a.Life() != want || a.offers != 0 {
					t.Fatalf("life=%d offers=%d", a.Life(), a.offers)
				}
				if paid != (g.FindPermanentIncludingPhased(p.ID()) == nil) && situation != "missing" {
					t.Fatal("payment state")
				}
				if situation == "available" && len(b.Graveyard()) != 1 {
					t.Fatal("owner did not receive sacrifice")
				}
				if situation == "replaced" && len(g.GetExile()) != 1 {
					t.Fatal("replacement did not apply")
				}
			})
		}
	}
}

func TestRequiredPaymentNestedAndClone(t *testing.T) {
	a := &requiredPaymentPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	g.PutOnBattlefield(NewCreature("Only", "", 1, 1), a.PlayerID())
	cost := Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice a creature"))
	effect := IfRequiredPaid(You, cost, Sequence(UnlessRequiredPaid(You, cost, GainLifeFor(You, Fixed(2))), GainLifeFor(You, Fixed(3))))
	card := NewSorcery("Payment", "", WithAbility(Spell(effect)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || len(g.AllBattlefield()) != 1 || clone.GetPlayer(a.PlayerID()).Life() != 25 {
		t.Fatal("clone or nested result")
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 25 || a.offers != 0 {
		t.Fatal("nested payment result")
	}
}

func TestRequiredPaymentDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		IfRequiredPaid(nil, Costs(SacrificeCreatureCost()), GainLifeFor(You, Fixed(1))),
		IfRequiredPaid(You, CostGroup{}, GainLifeFor(You, Fixed(1))),
		UnlessRequiredPaid(You, Costs(SacrificeCreatureCost()), nil),
		IfRequiredPaid(You, Costs(ManaCostOf("{1}")), GainLifeFor(You, Fixed(1))),
	} {
		card := NewSorcery("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid required payment")
		}
	}
}

func TestRequiredPaymentReplacementEventsAndOrdering(t *testing.T) {
	for _, mode := range []int{0, 1} {
		t.Run(string(rune('0'+mode)), func(t *testing.T) {
			a := &choiceRecordingPlayer{BasePlayer: NewBasePlayer("A"), chosenMode: mode}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(1)))), WithAbility(WhenEvent(CreatureDies(), GainLifeFor(You, Fixed(10)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			victim := NewCreature("Victim", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(), GainLifeFor(You, Fixed(2)))))
			victim.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(victim, a.PlayerID())
			first, second := uuid.New(), uuid.New()
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), first)
			g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), second)
			g.AddRegenerationShield(p.ID())
			effect := IfRequiredPaid(You, Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice")), GainLifeFor(You, Fixed(3)))
			clone := g.Clone()
			if err := ApplyEffect(g, effect, watcher.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if len(g.GetExile()) != 1 || len(b.Graveyard()) != 0 || a.Life() != 23 || clone.FindPermanent(p.ID()) == nil {
				t.Fatal("payment, destination or clone")
			}
			want := first
			if mode == 1 {
				want = second
			}
			if g.GetExile()[0].ExiledBy != want {
				t.Fatal("affected controller choice or replacement recheck")
			}
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatalf("sacrifice and source LKI triggers=%d; want 2 and no death trigger", g.StackSize())
			}
			g.ResolveStack()
			if a.Life() != 26 {
				t.Fatal("sacrifice trigger resolution")
			}
		})
	}
}

func TestRequiredPaymentReplacementPreventsMove(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Victim", "", 1, 1), a.PlayerID())
	g.effects.AddReplacement(&testPrioritizedReplacement{
		matchFunc:   func(a Action) bool { _, ok := a.(*SacrificePermanentAction); return ok },
		replaceFunc: func(Action, *Game) Action { return nil },
	})
	effect := IfRequiredPaid(You, Costs(SacrificeMatchingIncludingSourceCost(IsCreature, "Sacrifice")), GainLifeFor(You, Fixed(3)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 23 || g.FindPermanent(p.ID()) == nil {
		t.Fatal("started payment must remain successful after replacement")
	}
}

func TestRequiredPaymentSourceIdentity(t *testing.T) {
	for _, situation := range []string{"self", "return", "phased", "control", "copy"} {
		t.Run(situation, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			effect := UnlessRequiredPaid(You, Costs(SacrificeCreatureCost()), GainLifeFor(You, Fixed(3)))
			card := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(), effect)))
			p := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch situation {
			case "return":
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(card.ID())
				p = g.PutOnBattlefield(card, a.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				g.PutOnBattlefield(NewCreature("Payment", "", 1, 1), b.PlayerID())
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			g.ResolveStack()
			want := 23
			if situation == "return" {
				want = 20
			}
			if a.Life() != want {
				t.Fatal("source incarnation")
			}
			if situation == "return" && g.FindPermanent(p.ID()) != nil {
				t.Fatal("returned source was excluded")
			}
			if situation == "copy" && (b.Life() != 20 || len(b.Graveyard()) != 1) {
				t.Fatal("copy payer")
			}
		})
	}
}

func TestRequiredPaymentSimultaneousSacrifices(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 1, 1), a.PlayerID())
	watcher := NewArtifact("Watcher", "", WithAbility(NewTriggered(core.EvtSacrifice, false, GainLifeFor(You, Fixed(1))).SetCondition(func(_ *core.GameEvent, g GameReader, _, _ uuid.UUID) bool {
		return g.FindPermanent(first.ID()) != nil && g.FindPermanent(second.ID()) != nil
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	effect := IfRequiredPaid(You, Costs(SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two creatures")), GainLifeFor(You, Fixed(3)))
	if err := ApplyEffect(g, effect, watcher.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil {
		t.Fatal("events lost the simultaneous pre-move snapshot")
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatal("batch trigger count")
	}
}

type requiredSacrificeSourceReplacement struct{ replacementBase }

func (r *requiredSacrificeSourceReplacement) Matches(a Action, _ GameReader) bool {
	sacrifice, ok := a.(*SacrificePermanentAction)
	return ok && sacrifice.Destination() == core.ZoneGraveyard
}
func (r *requiredSacrificeSourceReplacement) Replace(a Action, _ *Game) Action {
	return a.(*SacrificePermanentAction).WithExile(r.sourceID)
}
func (r *requiredSacrificeSourceReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}
func (r *requiredSacrificeSourceReplacement) Clone() ReplacementEffect { c := *r; return &c }

func TestRequiredPaymentSimultaneousReplacementSource(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewCreature("Replacement source", "", 1, 1), a.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 1), a.PlayerID())
	g.effects.AddReplacement(&requiredSacrificeSourceReplacement{replacementBase: replacementBase{sourceID: source.ID()}})
	effect := IfRequiredPaid(You, Costs(SacrificeNMatchingIncludingSourceCost(2, IsCreature, "Sacrifice two")), GainLifeFor(You, Fixed(1)))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.GetExile()) != 2 || len(a.Graveyard()) != 0 || g.FindPermanent(victim.ID()) != nil || a.Life() != 21 {
		t.Fatal("simultaneous replacement lost its departing source")
	}
}
