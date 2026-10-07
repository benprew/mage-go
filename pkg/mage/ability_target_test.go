package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivatedAbilityTargetContract(t *testing.T) {
	for _, change := range []string{"live", "departed", "returned", "phased", "phase-cycle", "type-lost", "copy", "clone", "gone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			sourceCard := NewArtifact("Source", "", WithAbility(Activate(Costs(ManaCostOf("{1}")), GainLifeFor(You, Fixed(3)))))
			sourceCard.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(sourceCard, b.PlayerID())
			b.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
			counter := NewInstant("Counter", "", Spell(CounterAbility(target), WithTargets(target)))
			counter.SetOwner(a.PlayerID())
			a.AddToHand(counter)
			if !slices.Contains(target.Possible(a.PlayerID(), counter, g), original.ID) {
				t.Fatal("missing activated ability")
			}
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{source.ID()}, 0); err == nil {
				t.Fatal("source ID accepted as ability ID")
			}
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{original.ID}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			switch change {
			case "departed", "returned":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if change == "returned" {
					b.RemoveFromHand(sourceCard.ID())
					g.PutOnBattlefield(sourceCard, b.PlayerID())
					g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "type-lost":
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
			case "copy":
				if g.CopyStackObjectDirect(original, a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
			case "clone":
				clone.ResolveStack()
			case "gone":
				g.CounterStackObject(original.ID)
			}
			g.ResolveStack()
			want := 20
			if change == "type-lost" {
				want = 23
			}
			if b.Life() != want {
				t.Fatalf("life=%d want %d", b.Life(), want)
			}
			if b.ManaPool().TotalMana() != 0 {
				t.Fatal("counter refunded costs")
			}
			if change != "clone" {
				clone.ResolveStack()
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone lost target identity")
			}
		})
	}
}

func TestActivatedAbilityTargetValidation(t *testing.T) {
	target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	for _, card := range []Card{
		NewInstant("Unbound", "", Spell(CounterAbility(target))),
		NewInstant("Empty", "", Spell(CounterAbility(DeclareActivatedAbilityTarget("", core.TypeArtifact)), WithTargets(target))),
		NewInstant("Nil", "", Spell(CounterAbility(nil), WithTargets(target))),
	} {
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted", card.Name())
		}
	}
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Trigger source", "", WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1))))), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if slices.Contains(target.Possible(a.PlayerID(), nil, g), g.StackPeek().ID) {
		t.Fatal("trigger accepted")
	}
	mana := g.PutOnBattlefield(NewArtifact("Mana", "", WithAbility(Activate(Costs(), AddManaToPlayer(You, core.Colorless, Fixed(1))))), b.PlayerID())
	before := g.StackSize()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), mana.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != before || slices.Contains(target.Possible(a.PlayerID(), nil, g), source.ID()) {
		t.Fatal("mana ability accepted")
	}
}

func TestActivatedAbilityTargetGraveyard(t *testing.T) {
	for _, exileCost := range []bool{false, true} {
		t.Run(map[bool]string{false: "live", true: "exile-cost"}[exileCost], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			cost := ManaCostOf("{0}")
			if exileCost {
				cost = ExileSelfFromGraveyardCost()
			}
			card := NewArtifact("Graveyard source", "", WithGraveyardActivatedAbility(GainLifeFor(You, Fixed(3)), cost))
			card.SetOwner(b.PlayerID())
			card.Abilities()[0].SetSource(card.ID())
			b.AddToGraveyard(card)
			g.SetActivePlayerIndex(1)
			g.SetStep(core.PrecombatMain)
			if err := g.ActivateGraveyardAbility(b.PlayerID(), card.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
			counter := NewInstant("Counter", "", Spell(CounterAbility(target), WithTargets(target)))
			counter.SetOwner(a.PlayerID())
			a.AddToHand(counter)
			if !slices.Contains(target.Possible(a.PlayerID(), counter, g), obj.ID) {
				t.Fatal("graveyard activated ability missing")
			}
			if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{obj.ID}, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			if b.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("graveyard ability was not countered")
			}
		})
	}
}

func TestActivatedAbilityTargetCopiesAndController(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(3))))), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	first := g.StackPeek()
	copied := g.CopyStackObjectDirect(first, a.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy failed")
	}
	target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	if !slices.Contains(target.Possible(a.PlayerID(), nil, g), copied.ID) {
		t.Fatal("activated copy rejected")
	}
	counter := NewInstant("Counter", "", Spell(Sequence(GainLifeFor(ControllerOfAbility(target), Fixed(2)), CounterAbility(target)), WithTargets(target)))
	counter.SetOwner(a.PlayerID())
	a.AddToHand(counter)
	if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{first.ID}, 0); err != nil {
		t.Fatal(err)
	}
	b.chooseQueue = [][]uuid.UUID{{copied.ID}}
	cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != copied.ID {
		t.Fatal("retarget failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed live players")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("copies did not keep separate targets and controllers")
	}
}

func TestActivatedAbilityTargetSacrificeCost(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(3))))), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	counter := NewInstant("Counter", "", Spell(CounterAbility(target), WithTargets(target)))
	counter.SetOwner(a.PlayerID())
	a.AddToHand(counter)
	if err := g.CastSpellByID(a.PlayerID(), counter.ID(), []uuid.UUID{obj.ID}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.Life() != 20 || g.FindPermanent(source.ID()) != nil {
		t.Fatal("sacrifice LKI or cost refund")
	}
}

func TestActivatedAbilityTargetSourceTypesAndRollback(t *testing.T) {
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 4, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(3))))), b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	card := NewInstant("Counter", "{G}", Spell(CounterAbility(target), WithTargets(target)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{obj.ID}, 0); err == nil {
		t.Fatal("nonartifact source accepted")
	}
	if a.ManaPool().TotalMana() != 1 || len(a.Hand()) != 1 || g.StackSize() != 1 {
		t.Fatal("invalid target changed payment")
	}
	g.MutablePermanent(source.ID()).GrantBaseAttr(core.AttrIsArtifact)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{obj.ID}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("current source type not used")
	}
	invalid := DeclareActivatedAbilityTarget("ability", core.CardType(-1))
	if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", Spell(CounterAbility(invalid), WithTargets(invalid)))) == nil {
		t.Fatal("invalid source type accepted")
	}
	if ValidateCardDefinition("Nil controller", NewInstant("Nil controller", "", Spell(GainLifeFor(ControllerOfAbility(nil), Fixed(1))))) == nil {
		t.Fatal("nil controller accepted")
	}
}

func TestActivatedAbilityTargetGraveyardMana(t *testing.T) {
	g, a, b := newSpellCopyGame()
	card := NewArtifact("Graveyard mana", "", WithGraveyardActivatedAbility(AddManaToPlayer(You, core.Colorless, Fixed(1)), ManaCostOf("{0}")))
	card.SetOwner(b.PlayerID())
	card.Abilities()[0].SetSource(card.ID())
	b.AddToGraveyard(card)
	g.SetActivePlayerIndex(1)
	g.SetStep(core.PrecombatMain)
	if err := g.ActivateGraveyardAbility(b.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	target := DeclareActivatedAbilityTarget("ability", core.TypeArtifact)
	if g.StackSize() != 0 || b.ManaPool().Count(core.Colorless) != 1 || len(target.Possible(a.PlayerID(), nil, g)) != 0 {
		t.Fatal("graveyard mana ability must resolve without a stack target")
	}
}
