package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type graveyardTopPlayer struct {
	*BasePlayer
	onOrder func()
	order   []uuid.UUID
	targets [][]uuid.UUID
	seen    [][]uuid.UUID
	orders  int
}

func (p *graveyardTopPlayer) ChooseLibraryTopOrder(cards []Card, _ GameReader) []uuid.UUID {
	p.orders++
	if p.onOrder != nil {
		p.onOrder()
	}
	return slices.Clone(p.order)
}
func (p *graveyardTopPlayer) ChooseTargets(ids []uuid.UUID, minimum, maximum int, _ *Game) []uuid.UUID {
	p.seen = append(p.seen, slices.Clone(ids))
	if len(p.targets) == 0 {
		return nil
	}
	result := p.targets[0]
	p.targets = p.targets[1:]
	return result
}
func graveyardTopGame() (*Game, *graveyardTopPlayer, *graveyardTopPlayer) {
	a := &graveyardTopPlayer{BasePlayer: NewBasePlayer("A")}
	b := &graveyardTopPlayer{BasePlayer: NewBasePlayer("B")}
	g := NewGame(a, b)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.PrecombatMain)
	return g, a, b
}
func graveyardTopSpell() Card {
	player := DeclarePlayerTarget("player")
	cards := DeclareGraveyardCardTargets("cards", player, IsArtifactCard)
	return NewSorcery("Restore", "{U}", WithAbility(Spell(ReturnGraveyardCardsToLibraryTop(cards, You), WithTargets(player, cards))))
}
func TestDependentGraveyardTargets(t *testing.T) {
	for _, scenario := range []string{"ordered", "zero", "partial", "return", "clone", "copy", "invalid-order"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := graveyardTopGame()
			first, second, foreign := NewArtifact("First", ""), NewArtifact("Second", ""), NewArtifact("Foreign", "")
			first.SetOwner(b.PlayerID())
			second.SetOwner(b.PlayerID())
			foreign.SetOwner(a.PlayerID())
			b.AddToGraveyard(first)
			b.AddToGraveyard(second)
			a.AddToGraveyard(foreign)
			foreignSecond := NewArtifact("Foreign second", "")
			foreignSecond.SetOwner(a.PlayerID())
			a.AddToGraveyard(foreignSecond)
			bottom := NewLand("Bottom")
			bottom.SetOwner(b.PlayerID())
			b.AddToLibrary(bottom)
			spell := graveyardTopSpell()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
				t.Fatal(err)
			}
			a.ManaPool().Add(core.Blue, 1)
			for _, ids := range [][]uuid.UUID{{b.PlayerID(), foreign.ID()}, {b.PlayerID(), first.ID(), first.ID()}, {first.ID()}} {
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 0); err == nil || a.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("invalid selection spent resources")
				}
			}
			ids := []uuid.UUID{b.PlayerID(), first.ID(), second.ID()}
			if scenario == "zero" {
				ids = ids[:1]
			}
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), ids, 0); err != nil {
				t.Fatal(err)
			}
			a.order = []uuid.UUID{second.ID(), first.ID()}
			switch scenario {
			case "partial":
				b.RemoveFromGraveyard(first.ID())
				b.AddToHand(first)
			case "return":
				b.RemoveFromGraveyard(first.ID())
				b.AddToGraveyard(first)
			case "invalid-order":
				a.order = []uuid.UUID{first.ID(), first.ID()}
			case "copy":
				a.targets = [][]uuid.UUID{{a.PlayerID()}, {foreign.ID(), foreignSecond.ID()}}
				copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
				if copied == nil || !slices.Equal(copied.Targets, []uuid.UUID{a.PlayerID(), foreign.ID(), foreignSecond.ID()}) {
					t.Fatal("copy lost dependent groups")
				}
				if len(a.seen) != 2 || !slices.Equal(a.seen[1], []uuid.UUID{foreign.ID(), foreignSecond.ID()}) {
					t.Fatal("copy saw wrong graveyard")
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.GetPlayer(b.PlayerID()).RemoveFromGraveyard(first.ID())
			}
			g.ResolveStack()
			want := []uuid.UUID{second.ID(), first.ID(), bottom.ID()}
			if scenario == "zero" {
				want = []uuid.UUID{bottom.ID()}
			}
			if scenario == "partial" || scenario == "return" {
				want = []uuid.UUID{second.ID(), bottom.ID()}
			}
			if scenario == "invalid-order" {
				want = []uuid.UUID{first.ID(), second.ID(), bottom.ID()}
			}
			got := make([]uuid.UUID, 0, len(b.Library()))
			for _, card := range b.Library() {
				got = append(got, card.ID())
			}
			if !slices.Equal(got, want) {
				t.Fatalf("library=%v want=%v", got, want)
			}
			if b.orders != 0 {
				t.Fatal("library owner chose instead of controller")
			}
			if len(clone.GetPlayer(b.PlayerID()).Library()) != 1 {
				t.Fatal("clone shared library")
			}
			clone.ResolveStack()
			if scenario == "clone" && len(clone.GetPlayer(b.PlayerID()).Library()) != 2 {
				t.Fatal("clone lost exact visits")
			}
		})
	}
}
func TestDependentGraveyardTargetsValidation(t *testing.T) {
	for _, scenario := range []string{"nil", "unbound", "forward", "nil-chooser", "nil-set"} {
		player := DeclarePlayerTarget("player")
		reference := player
		if scenario == "nil" {
			reference = nil
		}
		if scenario == "unbound" {
			reference = DeclarePlayerTarget("other")
		}
		filter := IsArtifactCard
		cards := DeclareGraveyardCardTargets("cards", reference, filter)
		targets := []TargetBinding{player, cards}
		if scenario == "forward" {
			targets = []TargetBinding{cards, player}
		}
		var chooser PlayerRef = You
		if scenario == "nil-chooser" {
			chooser = nil
		}
		var subjects CardSet = cards
		if scenario == "nil-set" {
			subjects = nil
		}
		card := NewSorcery("Invalid", "", WithAbility(Spell(ReturnGraveyardCardsToLibraryTop(subjects, chooser), WithTargets(targets...))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted", scenario)
		}
	}
}

func TestDependentGraveyardReplacementsAndBatch(t *testing.T) {
	for _, scenario := range []string{"batch", "redirect", "cancel", "replacement-return", "replacement-return-other", "choice-return"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := graveyardTopGame()
			first, second := NewArtifact("First", ""), NewArtifact("Second", "")
			first.SetOwner(b.PlayerID())
			second.SetOwner(b.PlayerID())
			b.AddToGraveyard(first)
			b.AddToGraveyard(second)
			spell := graveyardTopSpell()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID(), first.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			calls := 0
			if scenario != "batch" && scenario != "choice-return" {
				g.effects.AddReplacement(&testPrioritizedReplacement{
					replacementBase: replacementBase{sourceID: uuid.New(), priority: ReplacementPriorityDefault},
					matchFunc: func(action Action) bool {
						move, ok := action.(*CardZoneMoveAction)
						return ok && move.CardID() == first.ID() && move.Destination() == core.ZoneLibrary
					},
					replaceFunc: func(action Action, game *Game) Action {
						calls++
						if len(b.Graveyard()) != 2 || len(b.Library()) != 0 {
							t.Error("movement before all replacements")
						}
						move := action.(*CardZoneMoveAction)
						if move.FromZone() != core.ZoneGraveyard || move.PlayerID() != b.PlayerID() {
							t.Error("replacement lost zone or owner")
						}
						if scenario == "cancel" {
							return nil
						}
						if scenario == "replacement-return-other" {
							b.RemoveFromGraveyard(second.ID())
							b.AddToGraveyard(second)
							return move
						}
						if scenario == "replacement-return" {
							b.RemoveFromGraveyard(first.ID())
							b.AddToGraveyard(first)
							return move
						}
						return move.WithExile(move.ActionSource())
					},
				})
			}
			a.order = []uuid.UUID{second.ID(), first.ID()}
			if scenario == "choice-return" {
				a.onOrder = func() { b.RemoveFromGraveyard(first.ID()); b.AddToGraveyard(first) }
			}
			observed, total := 0, 0
			observer := NewTriggered(core.EvtZoneChange, false).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if event.FromZone == core.ZoneGraveyard {
					observed++
					want := 2
					if scenario != "batch" {
						want = 1
					}
					if len(b.Library()) != want {
						t.Error("zone event preceded final placement")
					}
				}
				return false
			})
			countObserver := NewTriggered(core.EvtCardsLeftGraveyard, false).SetCondition(func(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
				if event.PlayerID == b.PlayerID() {
					total += event.Amount
				}
				return false
			})
			g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(observer), WithAbility(countObserver)), a.PlayerID())
			g.ResolveStack()
			expected := 1
			if scenario == "batch" || scenario == "redirect" {
				expected = 2
			}
			if observed != expected || total != expected {
				t.Fatalf("events=%d count=%d want=%d", observed, total, expected)
			}
			if scenario == "redirect" && g.FindExiledCard(first.ID()) == nil {
				t.Fatal("replacement ignored")
			}
			if scenario != "batch" && scenario != "choice-return" && calls != 1 {
				t.Fatal("replacement count", calls)
			}
		})
	}
}

func TestDependentGraveyardQueries(t *testing.T) {
	g, a, b := graveyardTopGame()
	spell := graveyardTopSpell()
	for range 125 {
		card := NewArtifact("Artifact", "")
		card.SetOwner(b.PlayerID())
		b.AddToGraveyard(card)
	}
	other := NewCreature("Creature", "", 1, 1)
	other.SetOwner(b.PlayerID())
	b.AddToGraveyard(other)
	g.resolution.SetX(9)
	specs := spell.CastTargets()
	for range 2 {
		if got := g.PossibleTargetGroup(a.PlayerID(), spell, specs, 0, nil, 0); len(got) != 2 {
			t.Fatal("empty graveyard prevented player selection")
		}
		if got := g.PossibleTargetGroup(a.PlayerID(), spell, specs, 1, []uuid.UUID{b.PlayerID()}, 0); len(got) != 125 {
			t.Fatal("dependent query", len(got))
		}
		if g.resolution.X() != 9 || g.resolution.targetQueryBindings != nil || len(a.seen) != 0 || len(a.Hand()) != 0 {
			t.Fatal("query changed game")
		}
	}
}

func TestDependentGraveyardOwnersAndAPNAP(t *testing.T) {
	g, a, b := graveyardTopGame()
	g.SetActivePlayerIndex(1)
	cards := make([]Card, 0, 4)
	for _, player := range []*graveyardTopPlayer{a, b, a, b} {
		card := NewArtifact("Saved", "")
		card.SetOwner(player.PlayerID())
		player.AddToGraveyard(card)
		cards = append(cards, card)
	}
	saved := BindCards("saved")
	captured := make([]capturedCard, 0, len(cards))
	for _, card := range cards {
		captured = append(captured, capturedCard{id: card.ID(), visit: g.graveyardVisit(card.ID()), zone: core.ZoneGraveyard})
	}
	g.resolution.cardsBindings = map[*CardsBinding][]capturedCard{saved: captured}
	a.order = []uuid.UUID{cards[3].ID(), cards[2].ID(), cards[1].ID(), cards[0].ID()}
	var owners []uuid.UUID
	g.effects.AddReplacement(&testPrioritizedReplacement{
		replacementBase: replacementBase{sourceID: uuid.New(), priority: ReplacementPriorityDefault},
		matchFunc: func(action Action) bool {
			move, ok := action.(*CardZoneMoveAction)
			return ok && move.Destination() == core.ZoneLibrary
		},
		replaceFunc: func(action Action, _ *Game) Action {
			owners = append(owners, action.(*CardZoneMoveAction).PlayerID())
			if len(a.Library()) != 0 || len(b.Library()) != 0 {
				t.Error("movement during replacement choices")
			}
			return action
		},
	})
	if err := ReturnGraveyardCardsToLibraryTop(FromCards(saved), You).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(owners, []uuid.UUID{b.PlayerID(), b.PlayerID(), a.PlayerID(), a.PlayerID()}) {
		t.Fatal("replacement order", owners)
	}
	if len(a.Library()) != 2 || len(b.Library()) != 2 || a.Library()[0].ID() != cards[2].ID() || b.Library()[0].ID() != cards[3].ID() || a.orders != 1 || b.orders != 0 {
		t.Fatal("owner zones or per-library order")
	}
}
