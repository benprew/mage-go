package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatPairMatchingEngine(t *testing.T) {
	for _, attached := range []bool{false, true} {
		for _, kind := range []EventType{EvtDeclaredBlocker, EvtCreatureBlockedBy, EvtCombatPair} {
			for _, change := range []string{"none", "move", "source-leaves", "return", "phase", "phase-cycle", "color", "control", "copy", "regenerate"} {
				t.Run(kind.String()+"/"+boolLabel(attached)+"/"+change, func(t *testing.T) {
					a, b := NewBasePlayer("A"), NewBasePlayer("B")
					g := NewGame(a, b)
					binding := BindPermanent("opponent")
					var subject PermanentRef = This
					if attached {
						subject = AttachedPermanent
					}
					filter := Or(HasColorFilter(Green), HasColorFilter(White))
					pattern := BlocksOrBecomesBlockedByCreatureMatching(subject, filter)
					switch kind {
					case EvtDeclaredBlocker:
						pattern = BlocksCreatureMatching(subject, filter)
					case EvtCreatureBlockedBy:
						pattern = BecomesBlockedByCreatureMatching(subject, filter)
					}
					body := AtNext(EachEndOfCombat, Destroy(From(binding)))
					sourceCard := NewCreature("Source", "", 2, 6, WithAbility(WhenEvent(pattern.WithOpponent(binding), body)))
					if attached {
						sourceCard = NewAura("Source", "", WithAbility(WhenEvent(pattern.WithOpponent(binding), body)))
					}
					if err := ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
						t.Fatal(err)
					}
					p := g.PutOnBattlefield(sourceCard, a.PlayerID())
					host := p
					if attached {
						host = g.PutOnBattlefield(NewCreature("Host", "", 2, 6), b.PlayerID())
						g.Attach(p.ID(), host.ID())
					}
					otherCard := NewCreature("Other", "", 0, 8)
					otherCard.SetColorOverride([]Color{Green})
					q := g.PutOnBattlefield(otherCard, b.PlayerID())
					g.FireEvent(GameEvent{Type: kind, SourceID: host.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()})
					if change == "move" && attached {
						otherHost := g.PutOnBattlefield(NewCreature("New host", "", 1, 4), a.PlayerID())
						g.Attach(p.ID(), otherHost.ID())
					}
					if change == "source-leaves" {
						g.RemoveFromBattlefield(p)
					}
					g.PutTriggersOnStack()
					if g.StackSize() != 1 {
						t.Fatalf("stack=%d", g.StackSize())
					}
					if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					switch change {
					case "return":
						g.RemoveFromBattlefield(q)
						g.PutOnBattlefield(otherCard, b.PlayerID())
					case "phase":
						g.PhaseOut(q.ID())
					case "phase-cycle":
						g.PhaseIn(g.PhaseOut(q.ID()))
					case "color":
						if err := ApplyEffect(g, Until(Indefinite, SetColors(This, Blue)), q.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
						g.ApplyContinuousEffects()
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: p.ID(), TargetID: q.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
					}
					g.ResolveStack()
					if change == "regenerate" || change == "copy" {
						g.AddRegenerationShield(q.ID())
					}
					clone := g.Clone()
					clone.RunStepWithPriority(EndCombat)
					if change != "phase" && g.FindPermanent(q.ID()) == nil {
						t.Fatal("clone changed original")
					}
					g.RunStepWithPriority(EndCombat)
					for _, game := range []*Game{g, clone} {
						want := change == "return" || change == "regenerate"
						if (game.FindPermanent(q.ID()) != nil) != want {
							t.Fatal("wrong captured identity or regeneration")
						}
						if change == "phase" {
							game.PhaseIn([]uuid.UUID{q.ID()})
							if game.FindPermanent(q.ID()) == nil {
								t.Fatal("destroyed phased creature")
							}
						}
					}
				})
			}
		}
	}
}

func boolLabel(v bool) string {
	if v {
		return "host"
	}
	return "source"
}

func TestCombatPairMatchingDeclarations(t *testing.T) {
	binding := BindPermanent("opponent")
	for _, build := range []func(PermanentRef, PermanentFilter) filteredCombatPairPattern{
		BlocksCreatureMatching, BecomesBlockedByCreatureMatching, BlocksOrBecomesBlockedByCreatureMatching,
	} {
		RunDefinitionValidationTable(t, []DefinitionValidationCase{
			{Name: "source", Card: NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithOpponent(binding), Destroy(From(binding)))))},
			{Name: "host", Card: NewAura("Valid", "", WithAbility(WhenEvent(build(AttachedPermanent, IsCreature).WithOpponent(binding), Destroy(From(binding)))))},
			{Name: "nil subject", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(build(nil, IsCreature), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "subject"},
			{Name: "saved subject", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(build(From(binding), IsCreature), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "subject"},
			{Name: "nil binding", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithOpponent(nil), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "binding"},
			{Name: "empty binding", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithOpponent(BindPermanent("")), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "binding"},
			{Name: "duplicate capture", Card: NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(build(This, IsCreature).WithOpponent(binding).WithOpponent(binding), GainLifeFor(You, Fixed(1))))), WantErrSubstr: "one opponent"},
		})
	}
}

func TestCombatPairMatchingDetection(t *testing.T) {
	for _, scenario := range []string{"green", "white", "both", "blue", "wall", "noncreature", "opponent-phase", "host-phase", "source-phase", "source-removed", "wrong-host", "stale-host", "stale-opponent", "wrong-direction", "ability-removed"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			binding := BindPermanent("opponent")
			filter := And(Not(HasSubType("Wall")), Or(HasColorFilter(Green), HasColorFilter(White)))
			aura := g.PutOnBattlefield(NewAura("Watcher", "", WithAbility(WhenEvent(
				BlocksCreatureMatching(AttachedPermanent, filter).WithOpponent(binding),
				Sequence(AddCountersTo(From(binding), Charge, Fixed(1)), GainLifeFor(You, Fixed(1)))))), a.PlayerID())
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 6), b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			other := NewCreature("Other", "", 0, 8)
			other.SetColorOverride([]Color{Green})
			if scenario == "white" {
				other.SetColorOverride([]Color{White})
			}
			if scenario == "both" {
				other.SetColorOverride([]Color{Green, White})
			}
			if scenario == "blue" {
				other.SetColorOverride([]Color{Blue})
			}
			if scenario == "wall" {
				other = NewCreature("Wall", "", 0, 8, WithSubTypes("Wall"))
				other.SetColorOverride([]Color{Green})
			}
			if scenario == "noncreature" {
				other = NewEnchantment("Other", "")
				other.SetColorOverride([]Color{Green})
			}
			q := g.PutOnBattlefield(other, b.PlayerID())
			event := GameEvent{Type: EvtDeclaredBlocker, SourceID: host.ID(), TargetID: q.ID(), PlayerID: b.PlayerID()}
			switch scenario {
			case "opponent-phase":
				g.PhaseOut(q.ID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "source-phase":
				g.PhaseOut(aura.ID())
			case "source-removed":
				g.RemoveFromBattlefield(aura)
			case "wrong-host":
				event.SourceID = q.ID()
			case "stale-host":
				event.SourceIncarnation = uuid.New()
			case "stale-opponent":
				event.TargetIncarnation = uuid.New()
			case "wrong-direction":
				event.Type = EvtCreatureBlockedBy
			case "ability-removed":
				g.AddContinuousEffect(RemoveAllAbilities(aura.ID(), Indefinite))
				g.ApplyContinuousEffects()
			}
			g.FireEvent(event)
			g.PutTriggersOnStack()
			want := 0
			if scenario == "green" || scenario == "white" || scenario == "both" {
				want = 1
			}
			if g.StackSize() != want {
				t.Fatalf("stack=%d want=%d", g.StackSize(), want)
			}
			g.ResolveStack()
			if a.Life() != 20+want || b.Life() != 20 {
				t.Fatal("wrong trigger controller")
			}
			if want == 1 && g.FindPermanent(q.ID()).Counters[Charge] != 1 {
				t.Fatal("opponent was not captured")
			}
		})
	}
}
