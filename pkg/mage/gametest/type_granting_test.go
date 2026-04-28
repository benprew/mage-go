package gametest

// Engine tests for type-granting / subtype-changing continuous effects
// (CR 614 layer 4) and color-changing continuous effects (CR 614 layer 5).
// Covers GrantSubTypeToControlled (Allosaurus Shepherd static),
// GrantSubTypeToAll, GrantSubTypeToTarget, BecomesSubType (Wishful Merfolk),
// and BecomesColor (Scuttlemutt).

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

var typeGrantingOnce sync.Once

func registerTypeGrantingTestCards() {
	typeGrantingOnce.Do(func() {
		reg := func(name string, f func() mage.Card) {
			if !mage.CardRegistered(name) {
				mage.Register(name, f)
			}
		}

		// Static effect that says "all Elf creatures you control are also
		// Dinosaurs in addition to their other types" (Allosaurus Shepherd
		// static fragment).
		reg("Test Elf Dinosaur Lord", func() mage.Card {
			return mage.NewCreature("Test Elf Dinosaur Lord", "{G}", 1, 1,
				mage.WithSubTypes("Elf", "Shaman"),
				mage.WithStaticAbility(
					mage.GrantSubTypeToControlled("Dinosaur",
						mage.And(mage.IsCreature, mage.HasSubType("Elf"))),
				),
			)
		})

		// Plain Elf creature used as a recipient of the lord's static.
		reg("Test Plain Elf", func() mage.Card {
			return mage.NewCreature("Test Plain Elf", "{G}", 1, 1,
				mage.WithSubTypes("Elf"),
			)
		})

		// Creature with an activated ability that turns it into a Human
		// until end of turn (Wishful Merfolk pattern).
		reg("Test Becomes Human", func() mage.Card {
			return mage.NewCreature("Test Becomes Human", "{1}{U}", 2, 2,
				mage.WithSubTypes("Merfolk"),
				mage.WithActivatedAbility(
					mage.FuncEffect(
						"this creature becomes a Human until end of turn",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							eff := mage.BecomesSubType(sourceID, "Human", core.EndOfTurn)
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
							return nil
						}),
					mage.ManaCostOf("{U}"),
				),
			)
		})

		// Vanilla green creature target for color change tests.
		reg("Test Green Bear", func() mage.Card {
			return mage.NewCreature("Test Green Bear", "{1}{G}", 2, 2,
				mage.WithSubTypes("Bear"),
			)
		})

		// Activated ability: target creature becomes blue until end of
		// turn (Scuttlemutt-like color change).
		reg("Test Color Changer", func() mage.Card {
			return mage.NewCreature("Test Color Changer", "{2}", 3, 3,
				mage.WithSubTypes("Scarecrow"),
				mage.WithActivatedAbility(
					mage.FuncEffect(
						"target creature becomes blue until end of turn",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							eff := mage.BecomesColor(targets[0], core.Blue, core.EndOfTurn)
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
							return nil
						}),
					mage.TapSourceCost(),
					mage.WithTarget(mage.TargetPermanent(mage.IsCreature)),
				),
			)
		})
	})
}

// TestGrantSubTypeToControlled verifies a static "your Elves are also
// Dinosaurs" effect grants the Dinosaur subtype to controlled Elves while
// keeping their original subtypes intact.
func TestGrantSubTypeToControlled(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Elf Dinosaur Lord")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Plain Elf")
	g.Effects.Apply(g.Game)

	elf := g.FindPermanentByName("Test Plain Elf", g.getPlayerID(PlayerA))
	if elf == nil {
		t.Fatal("plain Elf not found")
	}
	if !elf.HasSubType("Elf") {
		t.Errorf("expected Elf to retain its Elf subtype")
	}
	if !elf.HasSubType("Dinosaur") {
		t.Errorf("expected Elf to gain Dinosaur subtype from controlled-grant")
	}

	// The lord itself is also an Elf, so it should also gain Dinosaur.
	lord := g.FindPermanentByName("Test Elf Dinosaur Lord", g.getPlayerID(PlayerA))
	if lord == nil {
		t.Fatal("lord not found")
	}
	if !lord.HasSubType("Dinosaur") {
		t.Errorf("expected lord (an Elf) to also gain Dinosaur")
	}
}

// TestGrantSubTypeToControlled_RevertsWhenSourceLeaves verifies the
// granted subtype goes away once the static source leaves play.
func TestGrantSubTypeToControlled_RevertsWhenSourceLeaves(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	lordID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Elf Dinosaur Lord")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Plain Elf")
	g.Effects.Apply(g.Game)

	lord := g.FindPermanent(lordID)
	if lord == nil {
		t.Fatal("lord not found")
	}
	g.RemoveFromBattlefield(lord)
	g.Effects.Apply(g.Game)

	elf := g.FindPermanentByName("Test Plain Elf", g.getPlayerID(PlayerA))
	if elf == nil {
		t.Fatal("plain Elf not found")
	}
	if elf.HasSubType("Dinosaur") {
		t.Errorf("expected Elf to lose Dinosaur after lord left play")
	}
	if !elf.HasSubType("Elf") {
		t.Errorf("expected Elf to keep its printed Elf subtype")
	}
}

// TestGrantSubTypeToControlled_DoesNotApplyToOpponentElves verifies the
// "you control" restriction.
func TestGrantSubTypeToControlled_DoesNotApplyToOpponentElves(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Elf Dinosaur Lord")
	g.AddCard(core.ZoneBattlefield, PlayerB, "Test Plain Elf")
	g.Effects.Apply(g.Game)

	elf := g.FindPermanentByName("Test Plain Elf", g.getPlayerID(PlayerB))
	if elf == nil {
		t.Fatal("opponent Elf not found")
	}
	if elf.HasSubType("Dinosaur") {
		t.Errorf("opponent Elf should not gain Dinosaur from your static")
	}
}

// TestBecomesSubType_WishfulMerfolk verifies an activated ability that
// makes the creature lose its current subtypes and become a Human.
func TestBecomesSubType_WishfulMerfolk(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Becomes Human")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Becomes Human")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	perm := g.FindPermanentByName("Test Becomes Human", g.getPlayerID(PlayerA))
	if perm == nil {
		t.Fatal("permanent not found")
	}
	if !perm.HasSubType("Human") {
		t.Errorf("expected creature to be a Human after activation")
	}
	if perm.HasSubType("Merfolk") {
		t.Errorf("expected creature to lose Merfolk subtype after activation")
	}
	if !perm.HasType(core.TypeCreature) {
		t.Errorf("expected creature to remain a creature")
	}
}

// TestBecomesSubType_ExpiresAtEOT verifies the EndOfTurn duration.
func TestBecomesSubType_ExpiresAtEOT(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Becomes Human")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Island")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Becomes Human")
	g.StopAt(2, core.Untap)
	g.Execute()

	perm := g.FindPermanentByName("Test Becomes Human", g.getPlayerID(PlayerA))
	if perm == nil {
		t.Fatal("permanent not found")
	}
	if perm.HasSubType("Human") {
		t.Errorf("Human subtype should expire at end of turn")
	}
	if !perm.HasSubType("Merfolk") {
		t.Errorf("Merfolk subtype should return after EOT cleanup")
	}
}

// TestBecomesColor verifies a Scuttlemutt-style color override applies to
// a target creature until end of turn.
func TestBecomesColor(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Color Changer")
	targetID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Green Bear")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Color Changer",
		"Test Green Bear")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	target := g.FindPermanent(targetID)
	if target == nil {
		t.Fatal("target not found")
	}
	colors := target.Colors()
	if len(colors) != 1 || colors[0] != core.Blue {
		t.Errorf("expected target colors == [Blue]; got %v", colors)
	}
}

// TestBecomesColor_ExpiresAtEOT verifies the color override goes away
// at end of turn.
func TestBecomesColor_ExpiresAtEOT(t *testing.T) {
	registerTypeGrantingTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Test Color Changer")
	targetID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Green Bear")

	g.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Color Changer",
		"Test Green Bear")
	g.StopAt(2, core.Untap)
	g.Execute()

	target := g.FindPermanent(targetID)
	if target == nil {
		t.Fatal("target not found")
	}
	colors := target.Colors()
	hasBlue := false
	hasGreen := false
	for _, c := range colors {
		if c == core.Blue {
			hasBlue = true
		}
		if c == core.Green {
			hasGreen = true
		}
	}
	if hasBlue {
		t.Errorf("Blue should be gone after EOT; got %v", colors)
	}
	if !hasGreen {
		t.Errorf("expected Grizzly Bears to be Green again; got %v", colors)
	}
}
