package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestDeathResultReturn(t *testing.T) {
	for _, noRegen := range []bool{false, true} {
		t.Run(map[bool]string{false: "regeneration", true: "no-regeneration"}[noRegen], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			makeCreature := func(name string) *Permanent {
				c := NewCreature(name, "", 2, 3)
				c.SetOwner(b.PlayerID())
				return g.PutOnBattlefield(c, b.PlayerID())
			}
			first := makeCreature("First")
			second := makeCreature("Second")
			shield := makeCreature("Shield")
			g.AddRegenerationShield(shield.ID())
			indestructible := makeCreature("Indestructible")
			indestructible.GrantBaseAttr(core.Indestructible)
			phased := makeCreature("Phased")
			g.PhaseOut(phased.ID())
			token := g.PutOnBattlefield(NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID())
			result, member := BindPermanentSet("deaths"), BindPermanent("member")
			var destroy = DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result))
			if noRegen {
				destroy = DestroyEachWithoutRegeneration(MatchingPermanents(IsCreature), CreatureDeathsInto(result))
			}
			body := Sequence(destroy, ReturnChosenGraveyardCardsForEachPermanent(result, member, You, IsCreatureCard))
			card := NewInstant("Valid", "", WithAbility(Spell(body)))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			for _, game := range []*Game{clone, g} {
				if err := body.Apply(&EffectContext{Game: game, Controller: b.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				want := 3
				if noRegen {
					want++
				}
				if len(game.resolution.createdTokenBindings[result]) != want {
					t.Fatal("incorrect death visits")
				}
				for _, p := range []*Permanent{first, second} {
					if game.FindPermanent(p.ID()) == nil {
						t.Fatal("missing returned creature")
					}
				}
				if game.FindPermanent(token.ID()) != nil {
					t.Fatal("returned a token")
				}
				if len(game.resolution.permanentBindings) != 0 {
					t.Fatal("member escaped its scope")
				}
			}
			if g.FindPermanent(indestructible.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
				t.Fatal("destroyed excluded subjects")
			}
		})
	}
}

func TestDeathResultReturnDeclarations(t *testing.T) {
	result, member := BindPermanentSet("deaths"), BindPermanent("member")
	for _, body := range []ResolvingEffect{
		DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(nil)),
		DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(BindPermanentSet(""))),
		DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result), CreatureDeathsInto(result)),
		ReturnChosenGraveyardCardsForEachPermanent(result, member, You, IsCreatureCard),
		Sequence(DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result)), ReturnChosenGraveyardCardsForEachPermanent(result, nil, You, IsCreatureCard)),
		Sequence(DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result)), ReturnChosenGraveyardCardsForEachPermanent(result, member, nil, IsCreatureCard)),
		Sequence(DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result)), ReturnChosenGraveyardCardsForEachPermanent(result, member, You, CardFilter{})),
	} {
		if ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithAbility(Spell(body)))) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestDeathResultReturnBatchAndVisits(t *testing.T) {
	for _, scenario := range []string{"batch", "returned-choice", "failed-entry", "insufficient", "replaced-death"} {
		t.Run(scenario, func(t *testing.T) {
			a := &requiredGraveyardPlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			victims := make([]*Permanent, 0, 2)
			for range 2 {
				c := NewCreature("Victim", "", 1, 1)
				c.SetOwner(b.PlayerID())
				victims = append(victims, g.PutOnBattlefield(c, b.PlayerID()))
			}
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 3, 3)
			first.SetOwner(a.PlayerID())
			second.SetOwner(a.PlayerID())
			a.AddToGraveyard(first)
			if scenario != "insufficient" {
				a.AddToGraveyard(second)
			}
			if scenario == "failed-entry" {
				second.AddAbility(&graveyardRejectEntry{})
			}
			if scenario == "returned-choice" {
				first.AddAbility(&opponentEntryProbe{check: func(_ *Game, _ *Permanent) {
					c, _, ok := g.MoveFromAnyGraveyard(second.ID(), core.ZoneHand)
					if !ok {
						t.Fatal("missing selected visit")
					}
					a.AddToGraveyard(c)
				}})
			}
			a.mutate = func() {
				if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || g.FindPermanent(victims[0].ID()) != nil || g.FindPermanent(victims[1].ID()) != nil {
					t.Fatal("movement preceded all choices or deaths")
				}
			}
			if scenario == "replaced-death" {
				for _, p := range victims {
					g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
				}
			}
			observed := 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if e.FromZone == core.ZoneGraveyard && e.ToZone == core.ZoneBattlefield {
					observed++
					if scenario == "batch" && (reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil) {
						t.Error("partial batch observation")
					}
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer)), a.PlayerID())
			result, member := BindPermanentSet("deaths"), BindPermanent("member")
			sentinel := capturedPermanent{uuid.New(), uuid.New()}
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{member: sentinel}
			if err := DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			savedClone := g.Clone()
			if err := ReturnChosenGraveyardCardsForEachPermanent(result, member, You, IsCreatureCard).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			want := 2
			if scenario == "insufficient" || scenario == "returned-choice" || scenario == "failed-entry" {
				want = 1
			}
			if scenario == "replaced-death" {
				want = 0
			}
			if observed != want || g.resolution.permanentBindings[member] != sentinel {
				t.Fatal("wrong entry batch or outer frame")
			}
			graveyardCount := 2
			if scenario == "insufficient" {
				graveyardCount = 1
			}
			if savedClone.FindPermanent(first.ID()) != nil || len(savedClone.GetPlayer(a.PlayerID()).Graveyard()) != graveyardCount {
				t.Fatal("saved result clone was changed")
			}
			if err := DestroyEach(MatchingPermanents(IsLand), CreatureDeathsInto(result)).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.resolution.createdTokenBindings[result]) != 0 || len(savedClone.resolution.createdTokenBindings[result]) != map[bool]int{true: 0, false: 2}[scenario == "replaced-death"] {
				t.Fatal("result reset or clone isolation")
			}
		})
	}
}

func TestDeathResultReturnHistoricalMembers(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	wall := g.PutOnBattlefield(NewArtifact("Blocker", ""), a.PlayerID())
	first, second := g.PutOnBattlefield(NewCreature("First", "", 2, 3), a.PlayerID()), g.PutOnBattlefield(NewCreature("Second", "", 2, 3), b.PlayerID())
	first.Card.SetOwner(a.PlayerID())
	second.Card.SetOwner(b.PlayerID())
	g.recordBlockHistory(wall, first)
	g.recordBlockHistory(wall, second)
	savedWall, result, member := BindPermanent("wall"), BindPermanentSet("deaths"), BindPermanent("member")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{savedWall: {wall.ID(), wall.incarnationID}}
	if err := DestroyEach(MatchingPermanents(IsCreature), CreatureDeathsInto(result)).Apply(&EffectContext{Game: g}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(wall)

	body := ReturnChosenGraveyardCardsForEachPermanent(result, member, ControllerWhenLastBlockedThisTurn(From(member), From(savedWall)), IsCreatureCard)
	if err := body.Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 0 || len(g.AllBattlefield()) != 2 {
		t.Fatal("dead identities lost distinct historical controllers")
	}
	if !effectReadsSourcePermanent(ReturnChosenGraveyardCardsForEachPermanent(result, member, ControllerWhenLastBlockedThisTurn(From(member), This), IsCreatureCard)) {
		t.Fatal("waiting query lost its source read")
	}
}
