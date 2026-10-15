package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type forcedManaPlayer struct {
	*BasePlayer
	choose func([]string) int
	calls  int
	amount int
}

func (p *forcedManaPlayer) ChooseMode(options []string, _ string) int {
	p.calls++
	if p.choose != nil {
		return p.choose(options)
	}
	return 0
}
func (*forcedManaPlayer) ChooseMayAbility(string) bool          { panic("unexpected optional activation") }
func (p *forcedManaPlayer) ChooseNumber(_, _ int, _ string) int { return p.amount }
func (p *forcedManaPlayer) ChooseNonnegativeNumber(string) int  { return p.amount }

func TestForcedManaCostsAndOrder(t *testing.T) {
	a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	converter := g.PutOnBattlefield(NewLand("Converter", WithAbility(Activate(Costs(ManaPayment("{R}")), AddMana(Blue, 3)))), a.PlayerID())
	zero := g.PutOnBattlefield(NewLand("Zero", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Green, Fixed(0))))), a.PlayerID())
	red := g.PutOnBattlefield(NewLand("Red", WithManaAbility(Red)), a.PlayerID())
	sacrifice := g.PutOnBattlefield(NewLand("Sacrifice", WithAbility(Activate(Costs(SacrificeSourceCost()), AddMana(White, 2)))), a.PlayerID())
	a.choose = func(options []string) int {
		for i, option := range options {
			if strings.Contains(option, "Red") {
				return i
			}
		}
		return 0
	}
	if err := ApplyEffect(g, ActivateManaAbilityOfEach(You, IsLand), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(red.ID()).Tapped || !g.FindPermanent(zero.ID()).Tapped || g.FindPermanent(converter.ID()).Tapped || g.FindPermanent(sacrifice.ID()) != nil || a.ManaPool().Count(Blue) != 3 || a.ManaPool().Count(White) != 2 || a.ManaPool().Count(Red) != 0 || g.StackSize() != 0 {
		t.Fatal("cost, zero output, order or one activation per source")
	}
}

func TestForcedManaRechecks(t *testing.T) {
	for _, change := range []string{"phase", "returned", "control", "ability", "cost", "tapped"} {
		t.Run(change, func(t *testing.T) {
			a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("First", WithManaAbility(Blue)), a.PlayerID())
			g.PutOnBattlefield(NewLand("Second", WithManaAbility(Red)), a.PlayerID())
			a.choose = func(_ []string) int {
				switch change {
				case "phase":
					g.PhaseOut(land.ID())
				case "returned":
					card := land.Card
					g.RemoveFromBattlefield(land)
					g.PutOnBattlefield(card, a.PlayerID())
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
					g.ApplyContinuousEffects()
				case "ability":
					g.AddContinuousEffect(RemoveAllAbilities(land.ID(), Indefinite))
					g.ApplyContinuousEffects()
				case "cost":
					g.effects.Rules.activationCostModifiers = append(g.effects.Rules.activationCostModifiers, activationCostModifier{sourceID: land.ID(), incarnation: land.incarnationID, filter: Named("First"), increase: ManaCost{Generic: 9}})
				case "tapped":
					g.TapPermanent(land)
				}
				a.choose = nil
				return 0
			}
			if err := ApplyEffect(g, ActivateManaAbilityOfEach(You, IsLand), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(Blue) != 0 || a.ManaPool().Count(Red) != 1 || b.ManaPool().TotalMana() != 0 {
				t.Fatal("stale activation changed game state or stopped other sources")
			}
		})
	}
}

func TestForcedManaRollbackAndBoundaries(t *testing.T) {
	a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	a.SetLife(3)
	bad := g.PutOnBattlefield(NewLand("Bad", WithAbility(Activate(Costs(TapCost(This), LifePayCost(2), LifePayCost(2)), AddMana(White, 5)))), a.PlayerID())
	sick := g.PutOnBattlefield(NewCreature("Sick", "", 1, 1, WithManaAbility(Green), WithCardType(TypeLand)), a.PlayerID())
	phased := g.PutOnBattlefield(NewLand("Phased", WithManaAbility(Blue)), a.PlayerID())
	g.PhaseOut(phased.ID())
	for _, effect := range []ResolvingEffect{ActivateManaAbilityOfEach(You, IsLand), ActivateManaAbilityOfEach(FromPlayer(BindPlayer("absent")), IsLand)} {
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if a.Life() != 3 || g.FindPermanent(bad.ID()).Tapped || g.FindPermanent(sick.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("failed cost or absent source changed state")
	}
	card := NewSorcery("Invalid", "", WithAbility(Spell(ActivateManaAbilityOfEach(nil, IsLand))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("nil player accepted")
	}
}

func TestForcedManaVariableOutputAndReplacement(t *testing.T) {
	for _, amount := range []int{0, 2, 3} {
		a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A"), amount: amount}
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("storage payment")
		land := g.PutOnBattlefield(NewLand("Storage", WithAbility(Activate(Costs(TapCost(This), RemoveAnyCountersCost(Storage, paid)), AddManaAmount(Black, FromNumber(paid))))), a.PlayerID())
		land.AddCounter(Storage, 3)
		if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		var event GameEvent
		g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(EvtTappedForMana, false).SetCondition(func(e *GameEvent, _ GameReader, _, _ uuid.UUID) bool { event = *e; return false }))), a.PlayerID())
		if !g.canForceManaAbility(a.PlayerID(), land, 0) || a.calls != 0 || land.Tapped || land.Counters[Storage] != 3 {
			t.Fatal("activation query made a choice or changed state")
		}
		if err := ApplyEffect(g, ActivateManaAbilityOfEach(You, IsLand), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().Count(Blue) != amount || a.ManaPool().Count(Black) != 0 || land.Counters[Storage] != 3-amount || !land.Tapped || event.Amount != amount {
			t.Fatal("variable choice, replacement or mana event was lost")
		}
	}
}

func TestForcedManaIndependentX(t *testing.T) {
	for _, amount := range []int{0, 2, 99} {
		a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A"), amount: amount}
		g := NewGame(a, NewBasePlayer("B"))
		land := g.PutOnBattlefield(NewLand("X", WithAbility(Activate(Costs(ManaPayment("{X}"), TapCost(This)), AddMana(Blue, 4)))), a.PlayerID())
		a.ManaPool().Add(Red, 2)
		g.SetXValue(17)
		if err := ApplyEffect(g, ActivateManaAbilityOfEach(You, IsLand), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		wantRed := 2
		if amount == 2 {
			wantRed = 0
		}
		if !g.FindPermanent(land.ID()).Tapped || a.ManaPool().Count(Blue) != 4 || a.ManaPool().Count(Red) != wantRed || g.XValue() != 17 {
			t.Fatalf("X=%d: outer X, chosen payment or fallback changed", amount)
		}
	}
}

func TestForcedManaNonmanaX(t *testing.T) {
	for _, life := range []bool{false, true} {
		a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A"), amount: 2}
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("X payment")
		cost := RemoveXCountersFromSourceCostInto(Charge, paid)
		if life {
			cost = LifePayXCostInto(paid)
		}
		land := g.PutOnBattlefield(NewLand("X", WithAbility(Activate(Costs(cost), AddManaAmount(Blue, FromNumber(paid))))), a.PlayerID())
		land.AddCounter(Charge, 3)
		g.SetXValue(17)
		if err := ApplyEffect(g, ActivateManaAbilityOfEach(You, IsLand), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().Count(Blue) != 2 || g.XValue() != 17 || life && a.Life() != 18 || !life && land.Counters[Charge] != 1 {
			t.Fatal("nonmana X used a different activation's value")
		}
	}
}

func TestForcedManaWaitingSourceAndClone(t *testing.T) {
	a := &forcedManaPlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	sourceCard := NewArtifact("Source", "")
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	land := g.PutOnBattlefield(NewLand("A land", WithManaAbility(Blue)), a.PlayerID())
	g.PutOnBattlefield(NewLand("B land", WithManaAbility(Red)), b.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, ActivateManaAbilityOfEach(ControllerOfWithLKI(This), IsLand)), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	saved := g.CaptureDamageSource(source.ID())
	g.DestroyPermanent(source)
	g.PutOnBattlefield(source.Card, b.PlayerID())
	for range 2 {
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.calls != 0 || land.Tapped {
			t.Fatal("waiting query lost source identity or made a choice")
		}
	}
	clone := g.Clone()
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.ManaPool().Count(Blue) != 1 || b.ManaPool().TotalMana() != 0 || clone.FindPermanent(land.ID()).Tapped {
		t.Fatal("delayed action lost LKI player or changed clone")
	}
	clone.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	clone.PutTriggersOnStack()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("delayed clone shared activation state")
	}
}
