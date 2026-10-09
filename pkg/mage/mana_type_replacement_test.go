package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaTypeReplacementProductionPaths(t *testing.T) {
	for _, path := range []string{"tap", "direct", "typed", "planned", "planned-body"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			var card Card
			switch path {
			case "typed":
				card = NewLand("Mixed", WithAbility(Activate(Costs(TapCost(This)), Sequence(AddManaAmount(White, Fixed(2)), AddManaAmount(Green, Fixed(1))))))
			case "planned-body":
				history := BindActivationHistory("mana")
				card = NewLand("Mixed", WithAbility(NewAction(ActionActivated, WithCost(TapCost(This)), WithEffect(Sequence(AddManaAmount(White, Fixed(2)), IfActivationsAtLeast(history, 1, AtNext(EachEndStep, GainLifeFor(You, Fixed(1)))))), WithActivationHistory(history))))
			default:
				card = NewLand("Mixed", WithMultiManaAbility(ManaProduction{Color: White, Amount: 2}, ManaProduction{Color: Green, Amount: 1}))
			}
			land := g.PutOnBattlefield(card, a.PlayerID())
			effect := ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue)
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			var event GameEvent
			g.PutOnBattlefield(NewEnchantment("Observer", "", WithAbility(NewTriggered(EvtTappedForMana, false, GainLifeFor(You, Fixed(1))).SetCondition(func(e *GameEvent, _ GameReader, _, _ uuid.UUID) bool { event = *e; return false }))), b.PlayerID())
			var err error
			switch path {
			case "tap":
				err = g.TapForMana(a.PlayerID(), land.ID())
			case "planned", "planned-body":
				err = g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 2})
			default:
				err = g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			want := 3
			if path == "planned-body" {
				want = 2
			}
			if a.ManaPool().Count(Blue) != want || a.ManaPool().Count(White) != 0 || a.ManaPool().Count(Green) != 0 {
				t.Fatalf("pool = %v", a.ManaPool())
			}
			if event.Amount != want || !slices.Equal(event.ManaTypes, []Color{Blue}) {
				t.Fatalf("event = %+v", event)
			}
			if path == "planned-body" {
				g.RunStepWithPriority(EndStep)
				if a.Life() != 21 {
					t.Fatal("planned body lost delay")
				}
			}
		})
	}
}

func TestManaTypeReplacementWhiteOnly(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Mixed", WithMultiManaAbility(ManaProduction{Color: White, Amount: 2}, ManaProduction{Color: Green, Amount: 1})), a.PlayerID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Green: 1, Generic: 2}); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Colorless) != 2 || a.ManaPool().Count(Green) != 1 || a.ManaPool().Count(White) != 0 {
		t.Fatalf("pool = %v", a.ManaPool())
	}
}

func TestManaTypeReplacementDeclarations(t *testing.T) {
	for _, e := range []ResolvingEffect{ReplaceTappedManaTypeForPlayerThisTurn(nil, IsLand, Blue), ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, AnyColor), ReplaceTappedManaTypeUntilLeaves(nil, White, Colorless), ReplaceTappedManaTypeUntilLeaves(This, AnyColor, Colorless), ReplaceTappedManaTypeUntilLeaves(This, White, AnyColor)} {
		c := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), e)))
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("accepted invalid mana replacement")
		}
	}
}

func TestManaTypeReplacementAnyColorPlanning(t *testing.T) {
	for _, combination := range []bool{false, true} {
		t.Run(map[bool]string{false: "one color", true: "combination"}[combination], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Choice", WithMultiManaAbility(ManaProduction{Color: AnyColor, Amount: 2, AnyCombination: combination})), a.PlayerID())
			if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Generic: 1}); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(White) != 0 {
				t.Fatal("planned white mana escaped its replacement")
			}
			g.MutablePermanent(land.ID()).Tapped = false
			a.ManaPool().Clear()
			if err := g.AutoTapForCost(a.PlayerID(), ManaCost{White: 1}); err == nil {
				t.Fatal("planned unavailable white mana")
			}
			if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("failed plan changed state")
			}
		})
	}
}

type manaReplacementChooser struct {
	*BasePlayer
	choice, calls int
}

func (p *manaReplacementChooser) ChooseMode(_ []string, _ string) int { p.calls++; return p.choice }

func TestManaTypeReplacementOrderAndPurity(t *testing.T) {
	for _, path := range []string{"manual-first-blue", "manual-first-red", "plan-blue", "plan-red"} {
		t.Run(path, func(t *testing.T) {
			a := &manaReplacementChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("White", WithManaAbility(White)), a.PlayerID())
			for _, color := range []Color{Blue, Red} {
				if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, color), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				if !g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) || !g.CanAfford(a.PlayerID(), ManaCost{Red: 1}, nil) || g.CanAfford(a.PlayerID(), ManaCost{White: 1}, nil) {
					t.Fatal("wrong projected colors")
				}
			}
			if a.calls != 0 || land.Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("planning made a choice or mutation")
			}
			want := Red
			switch path {
			case "manual-first-blue":
				if err := g.TapForMana(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
			case "manual-first-red":
				a.choice = 1
				want = Blue
				if err := g.TapForMana(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
			case "plan-blue":
				want = Blue
				if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1}); err != nil {
					t.Fatal(err)
				}
			case "plan-red":
				if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Red: 1}); err != nil {
					t.Fatal(err)
				}
			}
			if a.ManaPool().Count(want) != 1 || a.ManaPool().TotalMana() != 1 {
				t.Fatalf("pool = %v, want %v", a.ManaPool(), want)
			}
			if path[:4] == "plan" && a.calls != 0 {
				t.Fatal("automatic order made a fresh choice")
			}
		})
	}
}

func TestManaTypeReplacementRestrictionsAndBonuses(t *testing.T) {
	for _, path := range []string{"manual", "planned"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			land := g.PutOnBattlefield(NewLand("Restricted", WithMultiManaAbility(ManaProduction{Color: White, Amount: 2, Restriction: CreatureSpellsOnly{}})), a.PlayerID())
			g.PutOnBattlefield(NewEnchantment("Flare", "", WithAbility(NewManaFlareAbility(IsLand))), b.PlayerID())
			if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			ctx := SpellPaymentContext{IsCreature: true}
			if g.CanAfford(a.PlayerID(), ManaCost{Generic: 2}, nil) || !g.CanAfford(a.PlayerID(), ManaCost{Generic: 3}, &ctx) {
				t.Fatal("restriction or matched bonus lost in planning")
			}
			if path == "manual" {
				if err := g.TapForMana(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := g.mana.AutoTapForCost(g, a.PlayerID(), ManaCost{Generic: 3}, AutoTapHint{}, &ctx); err != nil {
					t.Fatal(err)
				}
			}
			if a.ManaPool().Count(Colorless) != 3 || a.ManaPool().CanPay(ManaCost{Generic: 2}, nil) || !a.ManaPool().CanPay(ManaCost{Generic: 3}, &ctx) {
				t.Fatal("production lost restriction or bonus used wrong type")
			}
		})
	}
}

func TestManaTypeReplacementTimingAndIdentity(t *testing.T) {
	for _, change := range []string{"control", "phased", "phase-cycle", "return", "cleanup", "ability-removal", "cloned-cleanup"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("White", WithManaAbility(White))
			card.SetOwner(a.PlayerID())
			land := g.PutOnBattlefield(card, a.PlayerID())
			if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: uuid.New(), TargetID: land.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phased":
				g.PhaseOut(land.ID())
				if g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
					t.Fatal("phased source used")
				}
				g.PhaseIn([]uuid.UUID{land.ID()})
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(land.ID()))
			case "return":
				g.BouncePermanentToHand(g.FindPermanent(land.ID()))
				a.RemoveFromHand(card.ID())
				land = g.PutOnBattlefield(card, a.PlayerID())
			case "cleanup":
				g.RunStepWithPriority(Cleanup)
			case "ability-removal":
				g.AddContinuousEffect(RemoveAllAbilities(land.ID(), EndOfTurn))
				if err := g.TapForMana(a.PlayerID(), land.ID()); err == nil {
					t.Fatal("removed mana ability activated")
				}
				g.RunStepWithPriority(Cleanup)
			case "cloned-cleanup":
				clone := g.Clone()
				clone.RunStepWithPriority(Cleanup)
				if err := clone.TapForMana(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				if clone.GetPlayer(a.PlayerID()).ManaPool().Count(Colorless) != 1 || land.Tapped {
					t.Fatal("clone changed duration or source")
				}
			}
			player := a.PlayerID()
			want := Blue
			if change == "control" {
				player = b.PlayerID()
				want = Colorless
			}
			if change == "cleanup" || change == "ability-removal" {
				want = Colorless
			}
			if err := g.AutoTapForCost(player, ManaCost{Generic: 1}); err != nil {
				t.Fatal(err)
			}
			if g.GetPlayer(player).ManaPool().Count(want) != 1 {
				t.Fatalf("wrong identity/duration, pool %v", g.GetPlayer(player).ManaPool())
			}
			if change == "return" {
				g.MutablePermanent(land.ID()).Tapped = false
				g.RunStepWithPriority(Cleanup)
				if err := g.TapForMana(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().Count(White) != 1 {
					t.Fatal("returned land inherited old exact rule")
				}
			}
		})
	}
}

func TestManaTypeReplacementNonTapAndZero(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Converter", WithAbility(Activate(Costs(ManaPayment("{1}")), AddManaAmount(White, Fixed(1))))), a.PlayerID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	a.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(White) != 1 || land.Tapped {
		t.Fatal("non-tap mana was replaced")
	}
	if err := ApplyEffect(g, AddManaAmount(Red, Fixed(0)), land.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Blue) != 0 {
		t.Fatal("zero or spell mana became tapping production")
	}
}

func TestManaTypeReplacementCouldProduce(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("White", WithManaAbility(White)), a.PlayerID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	land.Tapped = true
	production := g.ManaTypesAbilityCouldProduce(land.RuntimeAbilities[0], land.ID())
	if len(production) != 1 || production[0] != Blue {
		t.Fatalf("could produce = %+v", production)
	}
	observer := g.PutOnBattlefield(NewArtifact("Colors", "", WithStaticAbility(SourceHasManaAbilitiesOpponentLandsCouldProduce())), b.PlayerID())
	found := false
	for _, ability := range observer.RuntimeAbilities {
		for _, prod := range ManaProductionsForAbility(ability) {
			if prod.Color != Blue {
				t.Fatal("could-produce service granted old type")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("could-produce service ignored replacement")
	}
}

func TestManaTypeReplacementRepeatedOrders(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("White", WithManaAbility(White)), a.PlayerID())
	for range 20 {
		for _, color := range []Color{Blue, Red} {
			if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, color), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := g.ManaTypesAbilityCouldProduce(land.RuntimeAbilities[0], land.ID()); len(got) != 2 || !slices.Contains(got, Blue) || !slices.Contains(got, Red) {
		t.Fatalf("types = %v", got)
	}
}

func TestManaTypeReplacementWaitingSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewLand("White", WithManaAbility(White))
	card.SetOwner(a.PlayerID())
	land := g.PutOnBattlefield(card, a.PlayerID())
	saved := g.CaptureDamageSource(land.ID())
	if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(land.ID())
	references := g.WaitingManaReplacementPermanentSources()
	if len(references) != 1 || !references[0].SameObject(saved) {
		t.Fatal("phased waiting subject lost identity")
	}
	found := false
	for _, source := range g.DamageSourcesFor(a.PlayerID()) {
		if source.SameObject(saved) {
			found = true
		}
	}
	if !found {
		t.Fatal("legal damage choices omitted waiting subject")
	}
	clone := g.Clone()
	g.PhaseIn([]uuid.UUID{land.ID()})
	g.BouncePermanentToHand(g.FindPermanent(land.ID()))
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if len(g.WaitingManaReplacementPermanentSources()) != 0 || len(clone.WaitingManaReplacementPermanentSources()) != 1 {
		t.Fatal("departure duration or clone isolation")
	}
}

func TestManaTypeReplacementStalePlanAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("White", WithManaAbility(White)), a.PlayerID())
	solution, err := g.mana.PlanManaForCost(g, a.PlayerID(), ManaCost{White: 1}, AutoTapHint{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, ReplaceTappedManaTypeUntilLeaves(This, White, Colorless), land.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := g.mana.ApplyManaSolution(g, a.PlayerID(), solution); err == nil || g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("stale plan spent resources")
	}
	spell := NewSorcery("Unpaid", "{1}{W}", WithAbility(Spell(GainLifeFor(You, Fixed(1)))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("unavailable payment cast spell")
	}
	if g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed cast changed live resources")
	}
}
