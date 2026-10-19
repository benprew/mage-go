package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPlayerDamageHistoryQueryValues(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("history")
	source := g.PutOnBattlefield(NewCreature("artifact", "", 1, 20, WithCardType(core.TypeArtifact)), b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.DealDamageToPlayer(a, 4, source.ID())
	g.DealDamageToPlayer(b, 2, source.ID())
	a.LoseLife(2)
	g.RemoveFromBattlefield(source)
	for _, ref := range []PlayerRef{You, ActivePlayer, target} {
		for _, artifacts := range []bool{false, true} {
			value := DamageTakenThisTurn(ref)
			if artifacts {
				value = ArtifactDamageTakenThisTurn(ref)
			}
			for _, amount := range []ResolutionValue{value, MultiplyValues(Fixed(2), value), MultiplyValues(value, Fixed(2)), MultiplyValues(Fixed(2), MultiplyValues(Fixed(2), value))} {
				predicate, ok := InspectPlayerDamageHistory(amount)
				if !ok {
					t.Fatal("native history rejected")
				}
				for _, controller := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
					for active := range 2 {
						g.SetActivePlayerIndex(active)
						ctx := &EffectContext{Game: g, Controller: controller, targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: b.PlayerID()}}}}
						want := amount.resolveValue(ctx)
						rng := g.random.Clone()
						for range 2 {
							if got, known := predicate.Read(ctx); !known || got != want {
								t.Fatalf("read=%d/%v want=%d", got, known, want)
							}
						}
						clone := g.Clone()
						child := *ctx
						child.Game = clone
						if got, known := predicate.Read(&child); !known || got != want {
							t.Fatal("clone lost history")
						}
						if !reflect.DeepEqual(rng, g.random) || a.Life() != 15 || b.Life() != 18 {
							t.Fatal("query changed state")
						}
					}
				}
			}
		}
	}
	g.RunStepWithPriority(core.Cleanup)
	predicate, _ := InspectPlayerDamageHistory(DamageTakenThisTurn(You))
	if got, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID()}); !known || got != 0 {
		t.Fatal("cleanup retained history")
	}
}

func TestPlayerDamageHistoryQueryBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target, foreign := DeclarePlayerTarget("same"), DeclarePlayerTarget("same")
	predicate, _ := InspectPlayerDamageHistory(DamageTakenThisTurn(target))
	for _, frame := range []map[*targetDeclaration][]capturedPermanent{nil, {foreign.targetDeclaration: {{id: a.PlayerID()}}}, {target.targetDeclaration: {{id: a.PlayerID()}, {id: b.PlayerID()}}}, {target.targetDeclaration: {{id: uuid.Nil}}}, {target.targetDeclaration: {{id: a.PlayerID(), incarnation: uuid.New()}}}} {
		if _, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: frame}); known {
			t.Fatal("invalid frame accepted")
		}
	}
	for _, values := range [][]capturedPermanent{nil, {{id: uuid.New()}}} {
		if got, known := predicate.Read(&EffectContext{Game: g, Controller: a.PlayerID(), targetBindings: map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: values}}); !known || got != 0 {
			t.Fatal("absent player supplied history")
		}
	}
	for _, ctx := range []*EffectContext{nil, {}, {Game: g, Controller: uuid.New()}} {
		if _, known := predicate.Read(ctx); known {
			t.Fatal("invalid context accepted")
		}
	}
	if _, known := (PlayerDamageHistoryPredicate{}).Read(&EffectContext{Game: g, Controller: a.PlayerID()}); known {
		t.Fatal("zero predicate accepted")
	}
	var missing *PlayerTargetBinding
	for _, value := range []ResolutionValue{nil, Fixed(3), damageInspectionValueTrap{}, DamageTakenThisTurn(nil), DamageTakenThisTurn(missing), DamageTakenThisTurn(lifeInspectionPlayerTrap{}), DamageTakenThisTurn(FromPlayer(BindPlayer("saved"))), MultiplyValues(Fixed(3), DamageTakenThisTurn(You)), MultiplyValues(Fixed(2), damageInspectionValueTrap{}), AddValues(DamageTakenThisTurn(You), Fixed(1))} {
		if _, ok := InspectPlayerDamageHistory(value); ok {
			t.Fatalf("opaque value accepted: %T", value)
		}
	}
}

func TestPlayerDamageHistoryQuerySpell(t *testing.T) {
	for _, damage := range []bool{false, true} {
		for _, state := range []string{"only", "prefix", "suffix", "zero", "missing", "foreign", "duplicate", "illegal", "all illegal", "unknown suffix", "unknown all illegal", "missing all illegal", "wrong source", "unrelated history"} {
			t.Run(state+map[bool]string{false: " life", true: " damage"}[damage], func(t *testing.T) {
				g, p, obj, land := targetQueryFixture(t)
				player := DeclarePlayerTarget("history")
				obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
				obj.Targets = append(obj.Targets, obj.Controller)
				amount := DamageTakenThisTurn(player)
				if state == "unrelated history" {
					amount = MultiplyValues(Fixed(2), ArtifactDamageTakenThisTurn(ActivePlayer))
				}
				known, want := true, false
				switch state {
				case "missing", "missing all illegal":
					obj.TargetSpecs = obj.TargetSpecs[:1]
					obj.Targets = obj.Targets[:1]
					known = false
				case "foreign":
					amount = DamageTakenThisTurn(DeclarePlayerTarget("history"))
					known = false
				case "duplicate":
					obj.TargetSpecs = append(obj.TargetSpecs, player.targetSpec())
					obj.Targets = append(obj.Targets, obj.Controller)
					known = false
				case "illegal", "all illegal":
					obj.Targets[1] = uuid.New()
				case "unknown all illegal":
					amount = MultiplyValues(Fixed(2), damageInspectionValueTrap{})
					known = false
				}
				if state == "all illegal" || state == "unknown all illegal" || state == "missing all illegal" {
					p.GrantBaseAttr(core.Shroud)
					if len(obj.Targets) > 1 {
						obj.Targets[1] = uuid.New()
					}
				}
				effect := GainLifeFor(You, amount)
				if damage {
					effect = DealDamageFrom(This, land, amount)
				}
				if state == "wrong source" {
					if damage {
						effect = DealDamageFrom(damageSourceQueryTrap{}, You, amount)
						known = false
					}
				}
				obj.Effects = []Effect{effect}
				switch state {
				case "prefix", "zero":
					obj.Effects = []Effect{Sequence(effect, Destroy(land))}
					known = false
				case "suffix":
					obj.Effects = []Effect{Sequence(Destroy(land), effect)}
					want = true
				case "unknown suffix":
					obj.Effects = []Effect{Sequence(Destroy(land), GainLifeFor(You, MultiplyValues(Fixed(2), damageInspectionValueTrap{})))}
					known = false
				}
				for _, game := range []*Game{g, g.Clone()} {
					rng := game.random.Clone()
					for range 2 {
						got, k := game.InspectSpellDestruction(game.StackPeek(), []PermanentVisit{{p.ID(), p.incarnationID}})
						if got != want || k != known {
							t.Fatalf("proof=%v/%v want=%v/%v", got, k, want, known)
						}
					}
					if !reflect.DeepEqual(rng, game.random) || game.GetPlayer(obj.Controller).Life() != 20 || game.FindPermanent(p.ID()) == nil {
						t.Fatal("query changed state")
					}
				}
			})
		}
	}
}

type damageSourceQueryTrap struct{}

func (damageSourceQueryTrap) damageSourceRef() {}
func (damageSourceQueryTrap) resolveDamageSource(*EffectContext) DamageSource {
	panic("source callback")
}

func TestPlayerDamageHistoryQueryPermanentRecipient(t *testing.T) {
	for _, state := range []string{"normal", "phase", "phase cycle", "return", "shroud", "missing visit", "missing zone", "foreign", "duplicate"} {
		t.Run(state, func(t *testing.T) {
			g, p, obj, target := targetQueryFixture(t)
			predicate, ok := InspectDamageRecipient(target)
			if !ok {
				t.Fatal("native permanent recipient rejected")
			}
			wantKnown, wantVisit := true, true
			switch state {
			case "phase":
				g.PhaseOut(p.ID())
				wantVisit = false
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, p.ControllerID())
				wantVisit = false
			case "shroud":
				p.GrantBaseAttr(core.Shroud)
				wantVisit = false
			case "missing visit":
				delete(obj.targetIncarnations, p.ID())
				wantKnown = false
			case "missing zone":
				delete(obj.TargetZones, p.ID())
				wantKnown = false
			case "foreign":
				predicate, _ = InspectDamageRecipient(DeclarePermanentTarget("subject", IsLand))
				wantKnown = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, p.ID())
				wantKnown = false
			}
			for _, game := range []*Game{g, g.Clone()} {
				selected, known := predicate.Select(game, game.StackPeek())
				if known != wantKnown || known && !selected.Empty() != wantVisit {
					t.Fatalf("selection=%v/%v", selected, known)
				}
				if known && wantVisit && selected.Permanent != (PermanentVisit{p.ID(), p.incarnationID}) {
					t.Fatal("recipient lost visit")
				}
			}
		})
	}
	var missing *PermanentTargetBinding
	for _, ref := range []DamageRecipientRef{missing, DeclarePermanentTarget("opaque", NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") }))} {
		if _, ok := InspectDamageRecipient(ref); ok {
			t.Fatal("incomplete permanent declaration accepted")
		}
	}
}
