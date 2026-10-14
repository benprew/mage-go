package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellExileResolution(t *testing.T) {
	for _, scenario := range []string{"normal", "zero", "wide", "copy", "clone", "other-controller", "return", "exile-on-leave"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			count := 2
			if scenario == "zero" {
				count = 0
			}
			if scenario == "wide" {
				count = 300
			}
			observed := 0
			probe := FuncEffect("observe exile", EffectProperties{}, func(game *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
				ec := game.FindExiledCard(source)
				if ec == nil || ec.FaceDown || ec.Counters[core.Scream] != count || ec.Counters[core.Charge] != 0 {
					t.Fatal("exile was not initialized before the next instruction")
				}
				observed++
				if scenario == "return" {
					card, _ := game.RemoveFromExile(source)
					game.GetPlayer(card.Owner()).AddToHand(card)
				}
				return nil
			})
			card := NewSorcery("Exile source", "", WithAbility(Spell(Sequence(ExileSpellWithCounters(ThisSpell, core.Scream, count), probe, ExileSpellWithCounters(ThisSpell, core.Scream, 9), GainLifeFor(You, Fixed(1))))))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "exile-on-leave" {
				g.StackPeek().ExileOnLeaveStack = true
			}
			if scenario == "copy" {
				cp := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
				copyID := cp.SourceID
				g.ResolveTopOfStack()
				if g.FindExiledCard(copyID) != nil || g.FindExiledCard(card.ID()) != nil || b.Life() != 21 {
					t.Fatal("copy cleanup changed original or retained copy")
				}
			}
			if scenario == "other-controller" {
				g.StackPeek().Controller = b.PlayerID()
			}
			if scenario == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if g.FindExiledCard(card.ID()) != nil || g.StackSize() != 1 || clone.FindExiledCard(card.ID()).Counters[core.Scream] != count {
					t.Fatal("clone shared mutation")
				}
			}
			g.ResolveStack()
			if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 {
				t.Fatal("cleanup moved an exiled spell again")
			}
			if scenario == "return" {
				if len(a.Hand()) != 1 || g.FindExiledCard(card.ID()) != nil {
					t.Fatal("repeat exile moved a new visit")
				}
			} else if ec := g.FindExiledCard(card.ID()); ec == nil || ec.Card.Owner() != a.PlayerID() || ec.Counters[core.Scream] != count {
				t.Fatal("exile visit or counters lost")
			}
			if observed == 0 {
				t.Fatal("later instruction did not run")
			}
		})
	}
}

func TestSpellExileDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{ExileSpell(nil), ExileSpell(FromSpell(BindSpell("unwritten"))), ExileSpellWithCounters(ThisSpell, core.Scream, -1), ExileSpellWithCounters(ThisSpell, core.NumCounters, 1)} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	g, a, _, _ := spellSourceFixture(t)
	if err := ExileSpell(ThisSpell).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(g.GetExile()) != 0 || g.StackSize() != 1 {
		t.Fatal("unbound self reference moved a live spell")
	}
}

func TestSpellExileSavedVisitAndObservation(t *testing.T) {
	g, a, b, obj := spellSourceFixture(t)
	binding := BindSpell("saved")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
	ctx := &EffectContext{Game: g, Controller: b.PlayerID(), SourceID: uuid.New()}
	observations := 0
	observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false, GainLifeFor(You, Fixed(1))).SetCondition(func(evt *core.GameEvent, reader GameReader, source, controller uuid.UUID) bool {
		if evt.FromZone != core.ZoneStack || evt.ToZone != core.ZoneExile {
			return false
		}
		ec := g.FindExiledCard(evt.SourceID)
		if ec == nil || ec.Counters[core.Scream] != 2 || evt.PlayerID != a.PlayerID() {
			t.Fatal("zone observation preceded counter initialization")
		}
		observations++
		return true
	})))
	g.PutOnBattlefield(observer, b.PlayerID())
	effect := ExileSpellWithCounters(FromSpell(binding), core.Scream, 2)
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if observations != 1 || g.StackSize() != 0 || g.FindExiledCard(obj.SourceID).ExiledBy != ctx.SourceID {
		t.Fatal("saved exile did not depart or notify")
	}
	g.cleanupSpell(obj)
	if len(a.Graveyard()) != 0 {
		t.Fatal("cleanup duplicated departure")
	}
	card, _ := g.RemoveFromExile(obj.SourceID)
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := effect.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || g.FindExiledCard(card.ID()) != nil || observations != 1 {
		t.Fatal("saved reference followed a later cast")
	}
	g.ResolveStack()
	if b.Life() != 21 {
		t.Fatal("exile observation did not queue its body")
	}
}

func TestSpellExileResolvingClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	probe := FuncEffect("clone the resolving frame", EffectProperties{}, func(game *Game, source, controller uuid.UUID, _ []uuid.UUID) error {
		clone := game.Clone()
		visit := clone.FindExiledCard(source).Visit
		if err := ExileSpellWithCounters(ThisSpell, core.Scream, 9).Apply(&EffectContext{Game: clone, SourceID: source, Controller: controller}); err != nil {
			t.Fatal(err)
		}
		if ec := clone.FindExiledCard(source); ec.Visit != visit || ec.Counters[core.Scream] != 2 {
			t.Fatal("resolving clone forgot that its spell had departed")
		}
		clone.FindExiledCard(source).Counters[core.Scream] = 7
		if game.FindExiledCard(source).Counters[core.Scream] != 2 {
			t.Fatal("resolving clone shared exile counters")
		}
		return nil
	})
	card := NewSorcery("Resolving clone", "", WithAbility(Spell(Sequence(ExileSpellWithCounters(ThisSpell, core.Scream, 2), probe))))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
}
