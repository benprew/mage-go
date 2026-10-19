package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func legendaryTokenDeclaration() TokenDefinition {
	return DefineToken("Stangg Twin", TokenLegendary(), TokenTypes(core.TypeCreature), TokenSubtypes("Human", "Warrior"), TokenColors(core.Red, core.Green), TokenPT(3, 4))
}

func TestLegendaryTokenCreation(t *testing.T) {
	for _, planned := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "planned"}[planned], func(t *testing.T) {
			declaration := legendaryTokenDeclaration()
			if planned {
				factory, err := NewTokenDefinitionFactory(legendaryTokenDeclaration)
				if err != nil {
					t.Fatal(err)
				}
				declaration = factory()
			}
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(2)), declaration), a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			tokens := g.FilterBattlefield(IsToken)
			if len(tokens) != 2 {
				t.Fatal("state actions ran inside the creation instruction")
			}
			for _, token := range tokens {
				if !token.Card.HasSuperType(core.SuperLegendary) || token.Name() != "Stangg Twin" || token.Card.Owner() != a.PlayerID() || token.CurrentPower(g) != 3 || token.CurrentToughness(g) != 4 {
					t.Fatal("token characteristics")
				}
			}
			if planned {
				values, err := g.CaptureCopiableValues(tokens[0])
				if err != nil {
					t.Fatal(err)
				}
				copyCard, err := values.Build()
				if err != nil || !copyCard.HasSuperType(core.SuperLegendary) || values.HasManaCost() || copyCard.IsToken() {
					t.Fatal("copiable legendary characteristic", err)
				}
				g.PutOnBattlefield(copyCard, b.PlayerID())
			} else {
				if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(1)), declaration), b.PlayerID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			wantTokens := 1
			if !planned {
				wantTokens = 2
			}
			clone := g.Clone()
			g.CheckStateBasedActions()
			if len(g.FilterBattlefield(IsLegendary)) != 2 || len(g.FilterBattlefield(IsToken)) != wantTokens {
				t.Fatal("legend rule must group by controller")
			}
			if len(clone.FilterBattlefield(IsToken)) != wantTokens+1 {
				t.Fatal("live state actions changed clone")
			}
			clone.CheckStateBasedActions()
			if len(clone.FilterBattlefield(IsLegendary)) != 2 {
				t.Fatal("clone lost legendary values")
			}
		})
	}
}

func TestLegendaryTokenPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	effect := CreateTokenCopies(You, ValueAtResolution(Fixed(1)), legendaryTokenDeclaration())
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	token := g.FilterBattlefield(IsToken)[0]
	phased := g.PhaseOut(token.ID())
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.CheckStateBasedActions()
	if len(g.zones.battlefield) != 2 {
		t.Fatal("phased token participated in legend rule")
	}
	g.PhaseIn(phased)
	g.CheckStateBasedActions()
	if len(g.FilterBattlefield(IsToken)) != 1 {
		t.Fatal("phase return did not restore legend rule")
	}
}

func TestLegendaryTokenInvalid(t *testing.T) {
	declaration := DefineToken("Invalid", TokenTypes(core.TypeCreature), TokenPT(1, 1), TokenLegendary(), TokenLegendary())
	card := NewInstant("Invalid", "", CreateTokenCopies(You, ValueAtResolution(Fixed(1)), declaration))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("duplicate supertype declaration accepted")
	}
}

func TestLegendaryTokenBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	deaths, sacrifices := 0, 0
	g.PutOnBattlefield(NewArtifact("Observer", "",
		WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(e *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
			if e.FromZone == core.ZoneBattlefield && e.ToZone == core.ZoneGraveyard {
				deaths++
				if len(g.FilterBattlefield(IsToken)) != 1 || len(reader.FilterBattlefield(IsToken)) != 3 {
					t.Error("legend departures were not simultaneous")
				}
			}
			return false
		})),
		WithAbility(NewTriggered(core.EvtSacrifice, false).SetCondition(func(_ *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
			sacrifices++
			return false
		}))), a.PlayerID())
	declaration := DefineToken("Stangg Twin", TokenLegendary(), TokenTypes(core.TypeCreature), TokenPT(3, 4), TokenKeywords(core.Indestructible))
	if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(3)), declaration), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.CheckStateBasedActions()
	if deaths != 2 || sacrifices != 0 || len(g.FilterBattlefield(IsToken)) != 1 {
		t.Fatalf("legend rule: deaths=%d sacrifices=%d", deaths, sacrifices)
	}
}

type legendaryTokenChooser struct {
	Player
	choose func([]*Permanent) *Permanent
}

func (p *legendaryTokenChooser) ChoosePermanent(permanents []*Permanent, _ string, _ GameReader) *Permanent {
	return p.choose(permanents)
}

func TestLegendaryTokenChoices(t *testing.T) {
	var order []string
	a := &legendaryTokenChooser{Player: NewBasePlayer("A")}
	b := &legendaryTokenChooser{Player: NewBasePlayer("B")}
	g := NewGame(a, b)
	for _, player := range []*legendaryTokenChooser{a, b} {
		player.choose = func(permanents []*Permanent) *Permanent {
			order = append(order, player.Name())
			if len(g.FilterBattlefield(IsToken)) != 6 {
				t.Error("permanents moved before all legend choices finished")
			}
			if player == a {
				return nil
			}
			return permanents[len(permanents)-1]
		}
		if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(3)), legendaryTokenDeclaration()), player.PlayerID(), player.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	bLast := g.FilterBattlefield(IsToken)[5].ID()
	g.CheckStateBasedActions()
	if len(order) != 2 || order[0] != "A" || order[1] != "B" || len(g.FilterBattlefield(IsToken)) != 2 || g.FindPermanent(bLast) == nil {
		t.Fatal("turn order, fallback or selected survivor", order)
	}
}

func TestLegendaryTokenFactoryDrift(t *testing.T) {
	legendary := true
	factory, err := NewTokenDefinitionFactory(func() TokenDefinition {
		options := []TokenOption{TokenTypes(core.TypeCreature), TokenPT(1, 1)}
		if legendary {
			options = append(options, TokenLegendary())
		}
		return DefineToken("Probe", options...)
	})
	if err != nil {
		t.Fatal(err)
	}
	declaration := factory()
	legendary = false
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(2)), declaration), a.PlayerID(), a.PlayerID(), nil); err == nil || len(g.FilterBattlefield(IsToken)) != 0 {
		t.Fatal("changed supertype must fail before the batch enters")
	}
}

func TestLegendaryTokenReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, CreateTokenCopies(You, ValueAtResolution(Fixed(2)), legendaryTokenDeclaration()), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	tokens := g.FilterBattlefield(IsToken)
	g.AddExileIfWouldGoToGraveyardThisTurn(tokens[1].ID(), source.ID())
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.CheckStateBasedActions()
		if len(game.FilterBattlefield(IsToken)) != 1 || game.FindPermanent(tokens[1].ID()) != nil || game.trackers.Turn.CreatureDeaths() != 0 {
			t.Fatal("legend move bypassed exile replacement")
		}
	}
}
