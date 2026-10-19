package mage

import (
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDestructionDestination(t *testing.T) {
	for _, state := range []string{"plain", "exile", "other exile", "old exile", "cycle exile", "source left", "cleanup", "hand", "wrapped hand", "face down", "removed", "shield", "static", "cancel", "cancel consumed", "no regen", "prohibited", "indestructible", "phase", "phase cycle", "return", "token", "competing", "competing no regen", "two exiles"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("Owner"), NewBasePlayer("Controller")
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 3)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			visit := PermanentVisit{p.ID(), p.incarnationID}
			want, noRegen := DestructionToGraveyard, false
			exile := func(id, incarnation uuid.UUID) {
				g.effects.AddReplacement(&temporaryDeathExileReplacement{replacementBase: replacementBase{sourceID: uuid.New(), duration: core.EndOfTurn}, subject: id, incarnation: incarnation})
			}
			switch state {
			case "exile", "cycle exile", "source left", "cleanup", "two exiles":
				exile(p.ID(), p.incarnationID)
				want = DestructionToExile
				if state == "two exiles" {
					exile(p.ID(), p.incarnationID)
				}
				if state == "cycle exile" {
					g.effects.cycleReplacements = g.effects.replacements
					g.effects.replacements = nil
				}
				if state == "cleanup" {
					g.effects.ClearReplacementsEndOfTurn()
					want = DestructionToGraveyard
				}
			case "other exile":
				exile(uuid.New(), uuid.New())
			case "old exile":
				exile(p.ID(), uuid.New())
			case "hand", "wrapped hand", "face down", "removed", "token":
				ability := ReturnToHandInsteadOfDyingWithPlayLock()
				if state == "wrapped hand" {
					ability = WrapGrantedAbility(ability)
				}
				p.RuntimeAbilities = append(p.RuntimeAbilities, ability)
				want = DestructionToHand
				if state == "face down" {
					p.FaceDown = true
					want = DestructionToGraveyard
				}
				if state == "removed" {
					p.RuntimeAbilities = nil
					want = DestructionToGraveyard
				}
				if state == "token" {
					p.IsToken = true
				}
			case "shield", "no regen", "prohibited":
				g.AddRegenerationShield(p.ID())
				want = DestructionNoMove
				if state == "no regen" {
					noRegen = true
					want = DestructionToGraveyard
				}
				if state == "prohibited" {
					p.GrantBaseAttr(core.CantRegenerate)
					want = DestructionToGraveyard
				}
			case "static":
				p.RuntimeAbilities = append(p.RuntimeAbilities, RegenerateInsteadOfDestroy())
				want = DestructionNoMove
			case "cancel", "cancel consumed":
				g.effects.AddReplacement(&nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID, consumed: state == "cancel consumed"})
				if state == "cancel" {
					want = DestructionNoMove
				}
			case "indestructible":
				p.GrantBaseAttr(core.Indestructible)
				want = DestructionNoMove
			case "phase":
				g.PhaseOut(p.ID())
				want = DestructionNoMove
			case "phase cycle":
				exile(p.ID(), p.incarnationID)
				g.PhaseIn(g.PhaseOut(p.ID()))
				want = DestructionToExile
			case "return":
				exile(p.ID(), p.incarnationID)
				g.RemoveFromBattlefield(p)
				p = g.PutOnBattlefield(card, b.PlayerID())
				want = DestructionNoMove
			case "competing", "competing no regen":
				p.RuntimeAbilities = append(p.RuntimeAbilities, ReturnToHandInsteadOfDyingWithPlayLock())
				exile(p.ID(), p.incarnationID)
				g.AddRegenerationShield(p.ID())
				want = DestructionToHand | DestructionToExile | DestructionNoMove
				if state == "competing no regen" {
					noRegen = true
					want = DestructionToHand | DestructionToExile
				}
			}
			p.Damage = 1
			before, random := g.Clone(), g.random.Clone()
			for _, game := range []*Game{g, g.Clone()} {
				for range 2 {
					got, known := game.InspectDestructionDestinations(visit, noRegen)
					if !known || got != want {
						t.Fatalf("destinations %v known %v, want %v", got, known, want)
					}
				}
			}
			if !reflect.DeepEqual(before.effects.replacements, g.Clone().effects.replacements) || !reflect.DeepEqual(random, g.random) || p.Damage != 1 || p.Tapped || len(a.Hand()) != 0 || len(a.Graveyard()) != 0 {
				t.Fatal("inspection changed state")
			}
			if state == "return" {
				if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != DestructionToGraveyard {
					t.Fatal("returned visit inherited exile")
				}
				return
			}
			if want != DestructionToGraveyard && want != DestructionToExile && want != DestructionToHand && want != DestructionNoMove {
				return
			}
			for _, game := range []*Game{g.Clone(), g} {
				game.destroyPermanents([]*Permanent{game.FindPermanent(p.ID())}, noRegen)
				switch want {
				case DestructionNoMove:
					if game.FindPermanentIncludingPhased(p.ID()) == nil {
						t.Fatal("native move disagrees")
					}
				case DestructionToGraveyard:
					if len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
						t.Fatal("native graveyard disagrees")
					}
				case DestructionToExile:
					if game.FindExiledCard(card.ID()) == nil {
						t.Fatal("native exile disagrees")
					}
				case DestructionToHand:
					if !p.IsToken && !game.IsHandCardRevealed(a.PlayerID(), card.ID()) {
						t.Fatal("native hand return disagrees")
					}
				}
			}
		})
	}
}

func TestDestructionDestinationUnknown(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
		visit := PermanentVisit{p.ID(), p.incarnationID}
		var absent *temporaryDeathExileReplacement
		for _, r := range []ReplacementEffect{nil, absent, &proposalTrap{}, &domainImpostor{&fogReplacement{}}, &faceUpDomainImpostor{&faceUpActionReplacement{}}, &temporaryDeathExileReplacement{historySource: damageSourceFromPermanent(p), historyFilter: NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter") })}} {
			g.effects.replacements = nil
			g.effects.cycleReplacements = nil
			if cycle {
				g.effects.cycleReplacements = []ReplacementEffect{r}
			} else {
				g.effects.replacements = []ReplacementEffect{r}
			}
			g.AddRegenerationShield(p.ID())
			if got, known := g.InspectDestructionDestinations(visit, false); known || got != 0 {
				t.Fatalf("%T accepted: %v", r, got)
			}
		}
		p.GrantBaseAttr(core.Indestructible)
		if got, known := g.InspectDestructionDestinations(visit, false); !known || got != DestructionNoMove {
			t.Fatal("impossible destruction reached callbacks")
		}
	}
	var absent *Game
	if _, known := absent.InspectDestructionDestinations(PermanentVisit{}, false); known {
		t.Fatal("nil game accepted")
	}
}

type destinationOrderPlayer struct {
	*BasePlayer
	index, calls int
}

func (p *destinationOrderPlayer) ChooseMode(options []string, _ string) int {
	p.calls++
	if len(options) != 3 {
		panic("unexpected replacement candidates")
	}
	return p.index
}

func TestDestructionDestinationOrders(t *testing.T) {
	for _, choice := range []string{"return to hand", "regeneration", "exile"} {
		for _, reverse := range []bool{false, true} {
			a := NewBasePlayer("Owner")
			index := 2
			if choice == "regeneration" {
				index = 0
			}
			if choice == "exile" {
				index = 1
			}
			if reverse && index != 2 {
				index = 1 - index
			}
			b := &destinationOrderPlayer{BasePlayer: NewBasePlayer("Controller"), index: index}
			g := NewGame(a, b)
			card := NewCreature("Subject", "", 3, 3, WithAbility(ReturnToHandInsteadOfDyingWithPlayLock()))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			g.AddRegenerationShield(p.ID())
			if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if reverse {
				g.effects.replacements[0], g.effects.replacements[1] = g.effects.replacements[1], g.effects.replacements[0]
			}
			want := DestructionNoMove | DestructionToExile | DestructionToHand
			if got, known := g.InspectDestructionDestinations(PermanentVisit{p.ID(), p.incarnationID}, false); !known || got != want || b.calls != 0 {
				t.Fatal("query chose or omitted an order")
			}
			g.DestroyPermanent(p)
			if (g.FindPermanent(p.ID()) != nil) != (choice == "regeneration") || (g.FindExiledCard(card.ID()) != nil) != (choice == "exile") || (len(a.Hand()) == 1) != (choice == "return to hand") {
				t.Fatal("native order disagrees")
			}
		}
	}
}

func TestDestructionDestinationNoCallbacks(t *testing.T) {
	a := &proposalChoiceTrap{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	p := g.PutOnBattlefield(NewCreature("Subject", "", 3, 3), a.PlayerID())
	visit := PermanentVisit{p.ID(), p.incarnationID}
	trap := NewPermanentFilter("trap", func(*Permanent, *Game) bool { panic("filter callback") })
	g.effects.AddReplacement(&damagePreventionRuleReplacement{from: trap, to: trap})
	g.AddRegenerationShield(p.ID())
	g.effects.AddReplacement(&nextDestructionReplacement{subject: p.ID(), incarnation: p.incarnationID})
	if err := ApplyEffect(g, ExileIfWouldDieThisTurn(This), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got, known := g.InspectDestructionDestinations(visit, false); !known || got != DestructionNoMove|DestructionToExile {
		t.Fatal("damage filter or choice reached")
	}
	g.effects.effects = append(g.effects.effects, &liveDamageRegenerationRule{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { panic("apply callback") }, func(*Game, uuid.UUID) bool { panic("active callback") }).(*funcContinuousEffect),
		filter:               trap,
	})
	if got, known := g.InspectDestructionDestinations(visit, false); known || got != 0 {
		t.Fatal("live prohibition accepted")
	}
	if got, known := g.InspectDestructionDestinations(visit, true); !known || got != DestructionNoMove|DestructionToExile {
		t.Fatal("unneeded regeneration filter reached")
	}
	if _, known := g.InspectDestructionDestinations(PermanentVisit{ID: p.ID()}, false); known {
		t.Fatal("incomplete visit accepted")
	}
	if got, known := g.InspectDestructionDestinations(PermanentVisit{ID: uuid.New(), Incarnation: uuid.New()}, false); !known || got != DestructionNoMove {
		t.Fatal("absent visit unknown")
	}
}
