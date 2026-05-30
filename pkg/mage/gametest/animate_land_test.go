package gametest

// Engine tests for aura-animates-land and animate-target-land helpers.
// Covers AnimateAttachedLand (Vastwood Zendikon pattern),
// AnimateLandWhileSourceOnBattlefield (Awakener Druid pattern),
// AnimateTargetLand (Elemental Uprising pattern), and
// GrantManaAbilityToAttached (New Horizons pattern).

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var animateLandOnce sync.Once

func registerAnimateLandTestCards() {
	animateLandOnce.Do(func() {
		reg := func(name string, f func() mage.Card) {
			if !mage.CardRegistered(name) {
				mage.Register(name, f)
			}
		}

		// Aura that animates the enchanted land into a 6/4 green
		// Elemental creature with trample (mirrors Vastwood Zendikon).
		reg("Test Land Animator Aura", func() mage.Card {
			return mage.NewAura("Test Land Animator Aura", "{2}{G}",
				mage.WithCastTarget(mage.TargetLand()),
				mage.WithStaticAbility(
					mage.AnimateAttachedLand(mage.AnimateLandOptions{
						Power:     6,
						Toughness: 4,
						SubTypes:  []string{"Elemental"},
						Colors:    []core.Color{core.Green},
						Keywords:  []core.Attr{core.Trample},
					}),
				),
			)
		})

		// Creature whose ETB animates target Forest into a 4/5 green
		// Treefolk creature for as long as it remains on the
		// battlefield (mirrors Awakener Druid).
		reg("Test Awakener", func() mage.Card {
			return mage.NewCreature("Test Awakener", "{2}{G}", 1, 1,
				mage.WithSubTypes("Druid"),
				mage.WithAbility(mage.EntersBattlefieldTrigger(
					mage.FuncEffect(
						"target Forest becomes a 4/5 green Treefolk creature",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							eff := mage.AnimateLandWhileSourceOnBattlefield(targets[0],
								mage.AnimateLandOptions{
									Power:     4,
									Toughness: 5,
									SubTypes:  []string{"Treefolk"},
									Colors:    []core.Color{core.Green},
								})
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
							return nil
						}),
					false,
				).AddTarget(mage.TargetPermanent(mage.IsLand, mage.HasSubType("Forest")))),
			)
		})

		// Instant: target land you control becomes a 4/4 Elemental
		// creature with trample until end of turn (Elemental Uprising).
		reg("Test Elemental Uprising", func() mage.Card {
			return mage.NewInstant("Test Elemental Uprising", "{1}{G}",
				mage.NewTargetedSpell(
					mage.TargetLand(),
					mage.FuncEffect(
						"target land becomes a 4/4 Elemental creature with trample",
						mage.EffectProperties{},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							eff := mage.AnimateTargetLand(targets[0],
								mage.AnimateLandOptions{
									Power:     4,
									Toughness: 4,
									SubTypes:  []string{"Elemental"},
									Keywords:  []core.Attr{core.Trample},
								},
								core.EndOfTurn)
							eff.SetSourceID(sourceID)
							g.AddContinuousEffect(eff)
							return nil
						})),
			)
		})

		// Aura that puts {T}: Add {G} as an additional mana ability on
		// the enchanted land (mirrors New Horizons' mana-grant clause).
		reg("Test Mana Grant Aura", func() mage.Card {
			return mage.NewAura("Test Mana Grant Aura", "{G}",
				mage.WithCastTarget(mage.TargetLand()),
				mage.WithStaticAbility(
					mage.GrantManaAbilityToAttached(mage.ManaProduction{Color: core.Green, Amount: 1}),
				),
			)
		})
	})
}

// TestAnimateAttachedLand_BasicTransformation verifies that an aura that
// animates the enchanted land actually turns the land into a creature
// with the configured P/T, subtypes, color, and keyword — while keeping
// the land's land type and existing mana ability.
func TestAnimateAttachedLand_BasicTransformation(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	forestID := g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	auraID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Land Animator Aura")

	g.Attach(auraID, forestID)
	g.Effects.Apply(g.Game)

	g.StopAt(1, core.PrecombatMain)
	g.Execute()

	perm := g.FindPermanentByName("Forest", g.getPlayerID(PlayerA))
	if perm == nil {
		t.Fatal("Forest not found")
	}
	if !perm.HasType(core.TypeCreature) {
		t.Errorf("expected Forest to be a creature")
	}
	if !perm.HasType(core.TypeLand) {
		t.Errorf("expected Forest to remain a land")
	}
	if !perm.HasSubType("Elemental") {
		t.Errorf("expected Forest to have subtype Elemental")
	}
	if !perm.HasSubType("Forest") {
		t.Errorf("expected Forest to keep subtype Forest")
	}
	if !perm.HasKeyword(core.Trample) {
		t.Errorf("expected animated Forest to have trample")
	}
	g.AssertPowerToughness(PlayerA, "Forest", 6, 4)

	colors := perm.Colors()
	hasGreen := false
	for _, c := range colors {
		if c == core.Green {
			hasGreen = true
		}
	}
	if !hasGreen {
		t.Errorf("expected animated Forest to be green; got %v", colors)
	}
}

// TestAnimateAttachedLand_RevertsWhenAuraLeaves verifies that destroying
// the aura immediately reverts the land to non-creature status.
func TestAnimateAttachedLand_RevertsWhenAuraLeaves(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	forestID := g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	auraID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Land Animator Aura")

	g.Attach(auraID, forestID)
	g.Effects.Apply(g.Game)

	perm := g.FindPermanentByName("Forest", g.getPlayerID(PlayerA))
	if perm == nil || !perm.HasType(core.TypeCreature) {
		t.Fatal("setup: Forest should be a creature while aura is attached")
	}

	// Move the aura off the battlefield to simulate destruction.
	aura := g.FindPermanent(auraID)
	if aura == nil {
		t.Fatal("aura not found")
	}
	g.RemoveFromBattlefield(aura)
	g.Effects.Apply(g.Game)

	perm = g.FindPermanentByName("Forest", g.getPlayerID(PlayerA))
	if perm == nil {
		t.Fatal("Forest disappeared")
	}
	if perm.HasType(core.TypeCreature) {
		t.Errorf("Forest should revert to non-creature once aura leaves")
	}
	if !perm.HasType(core.TypeLand) {
		t.Errorf("Forest should still be a land")
	}
	if perm.HasKeyword(core.Trample) {
		t.Errorf("animated trample should be gone after aura leaves")
	}
}

// TestAnimateAttachedLand_CanAttack verifies the animated land can attack.
func TestAnimateAttachedLand_CanAttack(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	forestID := g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	auraID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Land Animator Aura")

	g.Attach(auraID, forestID)
	g.Effects.Apply(g.Game)

	g.Attack(1, PlayerA, "Forest")
	g.StopAt(1, core.EndCombat)
	g.Execute()

	g.AssertLife(PlayerB, 14) // 20 - 6 = 14
}

// TestAnimateLandWhileSourceOnBattlefield_AwakenerDruid verifies that
// Awakener Druid's ETB animates a target Forest, and that destroying
// the druid reverts the Forest.
func TestAnimateLandWhileSourceOnBattlefield_AwakenerDruid(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest")
	g.AddCard(core.ZoneHand, PlayerA, "Test Awakener")
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 4) // mana

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Test Awakener")
	g.ChooseTarget(PlayerA, "Forest")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	// Find any Forest, all should be on PlayerA's side
	perms := g.FilterBattlefield(mage.IsLand)
	var animated *mage.Permanent
	for _, p := range perms {
		if p.HasType(core.TypeCreature) {
			animated = p
			break
		}
	}
	if animated == nil {
		t.Fatal("no Forest was animated by Awakener Druid ETB")
	}
	if animated.CurrentPower(g.Game) != 4 || animated.CurrentToughness(g.Game) != 5 {
		t.Errorf("expected animated Forest to be 4/5; got %d/%d",
			animated.CurrentPower(g.Game), animated.CurrentToughness(g.Game))
	}
	if !animated.HasSubType("Treefolk") {
		t.Errorf("expected animated Forest to be a Treefolk")
	}

	// Destroy the druid; the Forest should revert.
	druid := g.FindPermanentByName("Test Awakener", g.getPlayerID(PlayerA))
	if druid == nil {
		t.Fatal("druid not found")
	}
	g.RemoveFromBattlefield(druid)
	g.Effects.Apply(g.Game)

	if animated.HasType(core.TypeCreature) {
		t.Errorf("Forest should revert to non-creature once Awakener Druid dies")
	}
}

// TestAnimateTargetLand_ElementalUprising verifies the EOT-duration
// pattern: an instant turns target land into a 4/4 Elemental with
// trample until end of turn, and the animation expires at cleanup.
func TestAnimateTargetLand_ElementalUprising(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 2)
	g.AddCard(core.ZoneHand, PlayerA, "Test Elemental Uprising")

	g.CastSpell(1, core.PrecombatMain, PlayerA, "Test Elemental Uprising", "Forest")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	perms := g.FilterBattlefield(mage.IsLand)
	var animated *mage.Permanent
	for _, p := range perms {
		if p.HasType(core.TypeCreature) {
			animated = p
			break
		}
	}
	if animated == nil {
		t.Fatal("Elemental Uprising did not animate a Forest")
	}
	if animated.CurrentPower(g.Game) != 4 || animated.CurrentToughness(g.Game) != 4 {
		t.Errorf("expected 4/4; got %d/%d",
			animated.CurrentPower(g.Game), animated.CurrentToughness(g.Game))
	}
	if !animated.HasKeyword(core.Trample) {
		t.Errorf("expected trample on animated land")
	}
	if !animated.HasType(core.TypeLand) {
		t.Errorf("animated land should still be a land")
	}

	// Run to end of turn — animation should expire.
	g2 := NewTestGame(t)
	g2.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 2)
	g2.AddCard(core.ZoneHand, PlayerA, "Test Elemental Uprising")
	g2.CastSpell(1, core.PrecombatMain, PlayerA, "Test Elemental Uprising", "Forest")
	g2.StopAt(2, core.Untap)
	g2.Execute()

	for _, p := range g2.FilterBattlefield(mage.IsLand) {
		if p.HasType(core.TypeCreature) {
			t.Errorf("animation should have expired by next turn; %s still a creature",
				p.Name())
		}
	}
}

// TestGrantManaAbilityToAttached verifies that an aura granting a
// mana ability adds it as an *additional* mana ability on the
// enchanted land — the land's existing mana ability is preserved.
func TestGrantManaAbilityToAttached(t *testing.T) {
	registerAnimateLandTestCards()

	g := NewTestGame(t)
	mountainID := g.AddCard(core.ZoneBattlefield, PlayerA, "Mountain")
	auraID := g.AddCard(core.ZoneBattlefield, PlayerA, "Test Mana Grant Aura")

	g.Attach(auraID, mountainID)
	g.Effects.Apply(g.Game)

	mountain := g.FindPermanent(mountainID)
	if mountain == nil {
		t.Fatal("mountain not found")
	}

	manaAbilityCount := 0
	colorsProduced := map[core.Color]bool{}
	for _, a := range mountain.RuntimeAbilities {
		if ma, ok := mage.UnwrapAbility(a).(*mage.ManaAbility); ok {
			manaAbilityCount++
			for _, prod := range ma.Productions {
				colorsProduced[prod.Color] = true
			}
		}
	}
	if manaAbilityCount < 2 {
		t.Errorf("expected at least 2 mana abilities on enchanted Mountain; got %d", manaAbilityCount)
	}
	if !colorsProduced[core.Red] {
		t.Errorf("expected Mountain to retain its base {R} ability")
	}
	if !colorsProduced[core.Green] {
		t.Errorf("expected granted {G} mana ability")
	}
}
