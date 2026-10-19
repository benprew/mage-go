package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func faceUpReplacementGame(t *testing.T, install bool) (*Game, *BasePlayer, *BasePlayer, *Permanent) {
	t.Helper()
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Hidden", "{U}", 5, 6, WithKeyword(core.Lifelink))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
	if err != nil {
		t.Fatal(err)
	}
	binding := BindDamageSource("cast")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: damageSourceFromSpellObject(obj)}
	if install {
		if err := TurnResolvedPermanentFaceUpBeforeAction(FromDamageSource(binding)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	return g, a, b, g.FindPermanent(card.ID())
}

func TestFaceUpReplacement(t *testing.T) {
	for _, installed := range []bool{false, true} {
		for _, action := range []string{"tap", "damage", "zero", "assignment"} {
			t.Run(action+map[bool]string{false: "/ordinary", true: "/installed"}[installed], func(t *testing.T) {
				g, a, b, p := faceUpReplacementGame(t, installed)
				switch action {
				case "tap":
					g.TapPermanent(p)
				case "damage":
					g.DealDamageToPlayer(b, 2, p.ID())
				case "zero":
					g.DealDamageToPlayer(b, 0, p.ID())
				case "assignment":
					g.combat.AddAttacker(p.ID(), b.PlayerID())
					g.ExecuteCombatDamage()
				}
				want := installed && action != "zero"
				if g.FindPermanent(p.ID()).FaceDown == want {
					t.Fatal("wrong face-up status")
				}
				if action == "damage" && a.Life() != 20+map[bool]int{false: 0, true: 2}[installed] {
					t.Fatal("source snapshot preceded replacement")
				}
				if action == "assignment" && b.Life() != 20-map[bool]int{false: 2, true: 5}[installed] {
					t.Fatal("assignment used wrong power")
				}
			})
		}
	}
}

func TestFaceUpReplacementDefinition(t *testing.T) {
	card := NewArtifact("Invalid", "", WithAbility(NewActivated(GenericCost(0), TurnResolvedPermanentFaceUpBeforeAction(nil))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("nil spell accepted")
	}
}

func TestFaceUpReplacementLifecycle(t *testing.T) {
	for _, scenario := range []string{"pending-clone", "control", "counter", "copy", "manual-up", "simultaneous", "prevention", "protection", "tap-group", "unrelated"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b, p := faceUpReplacementGame(t, true)
			switch scenario {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.DealDamageToPlayer(a, 2, p.ID())
				if b.Life() != 22 {
					t.Fatal("face-up damage lost new controller")
				}
			case "manual-up":
				g.turnFaceUp(p)
				g.MutablePermanent(p.ID()).FaceDown = true
				g.TapPermanent(g.FindPermanent(p.ID()))
				if !g.FindPermanent(p.ID()).FaceDown {
					t.Fatal("ended duration restarted")
				}
				return
			case "simultaneous":
				g.DealSimultaneousDamage(PlayerDamageAssignment(b, 2, g.CaptureDamageSource(p.ID())), PlayerDamageAssignment(b, 3, g.CaptureDamageSource(p.ID())))
				if a.Life() != 25 || b.Life() != 15 {
					t.Fatal("batch did not refresh all source snapshots")
				}
			case "prevention":
				g.SetPlayerAt(1, &faceUpOrderPlayer{BasePlayer: b, choice: 1})
				g.AddPreventionShield(b.PlayerID(), 5)
				g.DealDamageToPlayer(b, 2, p.ID())
				if !g.FindPermanent(p.ID()).FaceDown || b.Life() != 20 {
					t.Fatal("prevented event turned subject up")
				}
				return
			case "protection":
				enemy := g.PutOnBattlefield(NewCreature("Protected", "", 1, 6, WithAbility(ProtectionFromColors(core.Blue))), b.PlayerID())
				g.DealDamageToPermanent(enemy, 2, p.ID())
				if g.FindPermanent(enemy.ID()).Damage != 0 {
					t.Fatal("protection did not recheck the revealed source color")
				}
			case "tap-group":
				second := g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
				g.tapPermanentsTogether([]*Permanent{p, second})
				if !g.FindPermanent(second.ID()).Tapped {
					t.Fatal("tap batch lost member")
				}
			case "unrelated":
				other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
				other.FaceDown = true
				g.TapPermanent(other)
				if !g.FindPermanent(other.ID()).FaceDown || !g.FindPermanent(p.ID()).FaceDown {
					t.Fatal("unrelated object changed")
				}
				return
			case "pending-clone", "counter", "copy":
				// A second cast checks the replacement while its spell is pending.
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(p.ID())
				a.AddToHand(NewCreature("Second", "", 4, 4))
				card := a.Hand()[0]
				obj, err := g.castHandCardFaceDownWithoutPaying(a.PlayerID(), card.ID())
				if err != nil {
					t.Fatal(err)
				}
				binding := BindDamageSource("second")
				g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: damageSourceFromSpellObject(obj)}
				if err := TurnResolvedPermanentFaceUpBeforeAction(FromDamageSource(binding)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if scenario == "counter" {
					g.CounterStackObject(obj.ID)
					g.ResolveStack()
					if g.FindPermanent(card.ID()) != nil {
						t.Fatal("counter produced subject")
					}
					return
				}
				if scenario == "copy" {
					copied := g.CopyStackObjectDirect(obj, b.PlayerID(), false)
					g.ResolveTopOfStack()
					copyPermanent := g.FindPermanent(copied.Card.ID())
					g.MutablePermanent(copyPermanent.ID()).FaceDown = true
					g.TapPermanent(g.FindPermanent(copyPermanent.ID()))
					if !g.FindPermanent(copyPermanent.ID()).FaceDown {
						t.Fatal("copy inherited replacement")
					}
				}
				clone := g.Clone()
				clone.ResolveStack()
				clone.TapPermanent(clone.FindPermanent(card.ID()))
				if clone.FindPermanent(card.ID()).FaceDown || g.FindPermanent(card.ID()) != nil {
					t.Fatal("pending clone changed original")
				}
				g.ResolveStack()
				p = g.FindPermanent(card.ID())
				g.TapPermanent(p)
			}
			if g.FindPermanent(p.ID()).FaceDown {
				t.Fatal("subject did not turn face up")
			}
		})
	}
}

type faceUpOrderPlayer struct {
	*BasePlayer
	choice int
}

func (p *faceUpOrderPlayer) ChooseMode([]string, string) int { return p.choice }

func TestFaceUpReplacementAssignmentEligibility(t *testing.T) {
	for _, scenario := range []string{"blocker", "zero-power", "first-strike-step", "blocked-no-blockers", "suppressed", "already-tapped"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b, p := faceUpReplacementGame(t, true)
			switch scenario {
			case "blocker":
				attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 1, 8), b.PlayerID())
				g.combat.AddAttacker(attacker.ID(), a.PlayerID())
				g.combat.AddBlocker(p.ID(), attacker.ID())
				g.ExecuteCombatDamage()
				if g.FindPermanent(attacker.ID()).Damage != 5 {
					t.Fatal("blocker assigned hidden power")
				}
			case "zero-power":
				p.BasePTOverride = &[2]int{0, 2}
				g.combat.AddAttacker(p.ID(), b.PlayerID())
				g.ExecuteCombatDamage()
			case "first-strike-step":
				g.combat.AddAttacker(p.ID(), b.PlayerID())
				g.combat.ResolveDamage(g, true)
			case "blocked-no-blockers":
				g.combat.AddAttacker(p.ID(), b.PlayerID())
				g.combat.Groups[0].Blocked = true
				g.ExecuteCombatDamage()
			case "suppressed":
				g.effects.noCombatAssignmentRules = map[uuid.UUID]uuid.UUID{p.ID(): p.incarnationID}
				g.combat.AddAttacker(p.ID(), b.PlayerID())
				g.ExecuteCombatDamage()
			case "already-tapped":
				p.Tapped = true
				g.TapPermanent(p)
			}
			if g.FindPermanent(p.ID()).FaceDown != (scenario != "blocker") {
				t.Fatal("wrong assignment eligibility")
			}
		})
	}
}

func TestFaceUpReplacementCostRollback(t *testing.T) {
	for _, affordable := range []bool{false, true} {
		g, a, _, p := faceUpReplacementGame(t, true)
		source := g.PutOnBattlefield(NewArtifact("Cost source", "", WithAbility(Activate(Costs(&exactPermanentTapCost{subject: capturedPermanent{p.ID(), p.incarnationID}}, GenericCost(1)), GainLifeFor(You, Fixed(1))))), a.PlayerID())
		if affordable {
			a.ManaPool().Add(core.Blue, 1)
		}
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
		if (err == nil) != affordable {
			t.Fatal("unexpected payment result", err)
		}
		live := g.FindPermanent(p.ID())
		if live.Tapped != affordable || live.FaceDown == affordable {
			t.Fatal("failed payment leaked face-up or tap state")
		}
	}
}

func TestFaceUpReplacementLateInstallation(t *testing.T) {
	g, a, _, p := faceUpReplacementGame(t, false)
	spell := DamageSource{id: p.ID(), stackID: p.resolvedSpellID}
	g.turnFaceUp(p)
	g.MutablePermanent(p.ID()).FaceDown = true
	binding := BindDamageSource("late saved spell")
	g.resolution.damageSourceBindings = map[*DamageSourceBinding]DamageSource{binding: spell}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		if err := TurnResolvedPermanentFaceUpBeforeAction(FromDamageSource(binding)).Apply(&EffectContext{Game: game, Controller: a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		game.TapPermanent(game.FindPermanent(p.ID()))
		if !game.FindPermanent(p.ID()).FaceDown {
			t.Fatal("late installation forgot earlier face-up status")
		}
	}
}

func TestFaceUpReplacementKeepsStrikeEligibility(t *testing.T) {
	g, a, b, p := faceUpReplacementGame(t, true)
	p.GrantBaseAttr(core.FirstStrike)
	g.combat.AddAttacker(p.ID(), b.PlayerID())
	g.combat.ResolveDamage(g, true)
	if g.combat.assignmentEligibility != nil {
		t.Fatal("assignment scope escaped the damage step")
	}
	if b.Life() != 15 || a.Life() != 25 {
		t.Fatalf("face-up assignment lost strike eligibility: %d/%d", a.Life(), b.Life())
	}
	g.combat.ResolveDamage(g, false)
	if b.Life() != 15 {
		t.Fatal("revealed creature assigned damage twice")
	}
}
