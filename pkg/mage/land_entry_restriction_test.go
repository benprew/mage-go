package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLandEntryRestrictionLifecycle(t *testing.T) {
	for _, state := range []string{"live", "tapped", "control", "phased", "cycle", "removed", "restored", "departed", "returned", "clone", "overlap", "face-down"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			card := NewEnchantment("Restriction", "", WithAbility(Static(PreventLandEntry())))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			blocked := true
			switch state {
			case "tapped":
				g.TapPermanent(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(source.ID())
				blocked = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "removed", "restored":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				g.ApplyContinuousEffects()
				blocked = false
				if state == "restored" {
					g.GetEffects().RemoveEndOfTurn()
					g.ApplyContinuousEffects()
					blocked = true
				}
			case "departed", "returned":
				g.BouncePermanentToHand(source)
				blocked = false
				if state == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, b.PlayerID())
					blocked = true
				}
			case "clone":
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
				if clone.PutOnBattlefield(NewLand("Clone land"), a.PlayerID()) == nil {
					t.Fatal("clone retained restriction")
				}
			case "overlap":
				g.PutOnBattlefield(NewEnchantment("Second", "", WithAbility(Static(PreventLandEntry()))), b.PlayerID())
				g.RemoveFromBattlefield(source)
			case "face-down":
				g.MutablePermanent(source.ID()).FaceDown = true
				g.ApplyContinuousEffects()
				blocked = false
			}
			for _, player := range []Player{a, b} {
				if (g.PutOnBattlefield(NewLand("Land"), player.PlayerID()) == nil) != blocked {
					t.Fatal("wrong entry restriction")
				}
				creature := NewCreature("Land creature", "", 1, 1)
				creature.AddType(core.TypeLand)
				if (g.PutOnBattlefield(creature, player.PlayerID()) == nil) != blocked {
					t.Fatal("land creature bypassed restriction")
				}
				if g.PutOnBattlefield(NewCreature("Allowed", "", 1, 1), player.PlayerID()) == nil {
					t.Fatal("nonland blocked")
				}
			}
		})
	}
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(Static(PreventLandEntry())))); err == nil {
		t.Fatal("nonpermanent declaration accepted")
	}
}

func TestLandEntryRestrictionZoneRetention(t *testing.T) {
	for _, path := range []string{"hand", "optional-hand", "graveyard", "source-graveyard", "library", "delayed-exile"} {
		t.Run(path, func(t *testing.T) {
			a := &handEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			land := NewLand("Saved land")
			land.SetOwner(a.PlayerID())
			filter := NewCardFilter("lands", func(c Card) bool { return c.HasType(core.TypeLand) })
			var effect Effect
			switch path {
			case "hand":
				a.AddToHand(land)
				effect = PutFromHandOntoBattlefield(filter)
			case "optional-hand":
				a.AddToHand(land)
				effect = MayPutFromHandOntoBattlefield(You, filter)
			case "graveyard":
				a.AddToGraveyard(land)
				effect = ReturnTargetFromAnyGraveyardToBattlefield()
			case "source-graveyard":
				a.AddToGraveyard(land)
				effect = ReturnSourceFromGraveyardToBattlefield()
			case "library":
				a.SetLibrary([]Card{land})
				effect = SearchLibraryToBattlefield(filter)
			case "delayed-exile":
				p := g.PutOnBattlefield(land, a.PlayerID())
				if err := ApplyEffect(g, ExileTargetReturnAtEndStep(nil), uuid.Nil, a.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			handVisit := a.handVisit(land.ID())
			g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), b.PlayerID())
			if path == "delayed-exile" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.ResolveStack()
			} else if err := ApplyEffect(g, effect, land.ID(), a.PlayerID(), []uuid.UUID{land.ID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(land.ID()) != nil {
				t.Fatal("prohibited land entered")
			}
			switch path {
			case "hand", "optional-hand":
				if len(a.Hand()) != 1 || a.handVisit(land.ID()) != handVisit {
					t.Fatal("hand visit lost")
				}
			case "graveyard", "source-graveyard":
				if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != land.ID() {
					t.Fatal("graveyard card lost")
				}
			case "library":
				if len(a.Library()) != 1 || a.Library()[0].ID() != land.ID() {
					t.Fatal("library card lost")
				}
			case "delayed-exile":
				if g.FindExiledCard(land.ID()) == nil {
					t.Fatal("exiled card lost")
				}
			}
		})
	}
}

func TestLandEntryRestrictionBatch(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), a.PlayerID())
	count := 0
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
		if e.ToZone == core.ZoneBattlefield {
			count++
		}
		return false
	}))), a.PlayerID())
	count = 0
	entered := g.putTokensOnBattlefieldTogether([]Card{NewLand("Blocked"), NewCreature("Allowed", "", 1, 1), NewLand("Blocked too")}, a.PlayerID())
	if len(entered) != 1 || count != 1 {
		t.Fatal("batch entry or event", len(entered), count)
	}
	if err := ApplyEffect(g, CreateTokens(2, "Land token", 0, 0, []core.CardType{core.TypeLand}, nil), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("prohibited tokens sent entry events")
	}
}

type landEntryObserverReplacement struct {
	replacementBase
	calls *int
}

func (*landEntryObserverReplacement) IsActive(GameReader) bool { return true }
func (*landEntryObserverReplacement) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*battlefieldEntryAction)
	return ok
}
func (r *landEntryObserverReplacement) Clone() ReplacementEffect         { clone := *r; return &clone }
func (r *landEntryObserverReplacement) Replace(a Action, _ *Game) Action { *r.calls++; return a }

func TestLandEntryRestrictionCopyRecheck(t *testing.T) {
	for _, path := range []string{"direct", "hand", "graveyard", "delayed-exile", "library"} {
		t.Run(path, func(t *testing.T) {
			a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			plan, err := NewCardDefinitionPlan(func() Card { return NewLand("Land donor") })
			if err != nil {
				t.Fatal(err)
			}
			donor, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			g.PutOnBattlefield(donor, a.PlayerID())
			blocker := g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), a.PlayerID())
			calls := 0
			g.GetEffects().AddReplacement(&landEntryObserverReplacement{replacementBase: replacementBase{priority: ReplacementPriorityDefault}, calls: &calls})
			plan, err = NewCardDefinitionPlan(func() Card { return NewCreature("Copy", "", 1, 1, WithAbility(MayCopyAsEnters(IsLand))) })
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			switch path {
			case "direct":
				if g.PutOnBattlefield(card, a.PlayerID()) != nil {
					t.Fatal("copy entered as land")
				}
			case "graveyard":
				a.AddToGraveyard(card)
				visit := g.graveyardVisit(card.ID())
				returnGraveyardCardToBattlefield(g, card.ID(), a.PlayerID())
				if g.graveyardVisit(card.ID()) != visit || len(a.Graveyard()) != 1 {
					t.Fatal("copy prohibition lost graveyard visit")
				}
			case "library":
				a.SetLibrary([]Card{card})
				if err := ApplyEffect(g, SearchLibraryToBattlefield(IsPermanentCard), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if len(a.Library()) != 1 || a.Library()[0].ID() != card.ID() {
					t.Fatal("copy prohibition lost library card")
				}
			case "delayed-exile":
				g.PhaseOut(blocker.ID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				if err := ApplyEffect(g, ExileTargetReturnAtEndStep(nil), uuid.Nil, a.PlayerID(), []uuid.UUID{p.ID()}); err != nil {
					t.Fatal(err)
				}
				visit := g.FindExiledCard(card.ID()).Visit
				g.PhaseIn([]uuid.UUID{blocker.ID()})
				calls = 0
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				g.ResolveStack()
				if exiled := g.FindExiledCard(card.ID()); exiled == nil || exiled.Visit != visit {
					t.Fatal("copy prohibition lost exile visit")
				}
			default:
				a.AddToHand(card)
				visit := a.handVisit(card.ID())
				if err := ApplyEffect(g, MayPutFromHandOntoBattlefield(You, IsPermanentCard), uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if len(a.Hand()) != 1 || a.handVisit(card.ID()) != visit {
					t.Fatal("copy prohibition lost hand visit")
				}
			}

			if calls != 0 || (len(a.Graveyard()) != 0 && path != "graveyard") {
				t.Fatal("prohibited copy offered later replacement or moved to graveyard", calls)
			}
		})
	}
}

func TestLandEntryRestrictionPlay(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	g.SetStep(core.PrecombatMain)
	g.PutOnBattlefield(NewEnchantment("Block", "", WithAbility(Static(PreventLandEntry()))), a.PlayerID())
	land := NewLand("Land")
	a.AddToHand(land)
	visit := a.handVisit(land.ID())
	if err := g.PlayLand(a.PlayerID(), land.ID()); err == nil {
		t.Fatal("played prohibited entry")
	}
	if a.handVisit(land.ID()) != visit || g.GetLandsPlayedThisTurn() != 0 || len(g.GetPlayableLands(a.PlayerID())) != 0 {
		t.Fatal("prohibited play changed zone or history")
	}
}
