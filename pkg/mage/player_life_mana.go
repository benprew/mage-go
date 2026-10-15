package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PlayerLifeManaThisTurn permits repeated life payments for one fixed mana.
// The selected player keeps the permission through source departure.
func PlayerLifeManaThisTurn(player PlayerRef, life int, color core.Color) ResolvingEffect {
	return &playerLifeManaInstallation{player: player, life: life, color: color}
}

type playerLifeManaInstallation struct {
	player PlayerRef
	life   int
	color  core.Color
}

func (*playerLifeManaInstallation) Text() string                 { return "pay life to add mana this turn" }
func (*playerLifeManaInstallation) Properties() EffectProperties { return EffectProperties{} }
func (e *playerLifeManaInstallation) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *playerLifeManaInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "player-life-mana", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
	if nilDefinition(e.player) || e.life <= 0 || e.color < core.Colorless || e.color > core.Green {
		n.problem = "supply a player, positive life cost and fixed mana type"
	}
	return n
}
func (e *playerLifeManaInstallation) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	ctx.Game.playerManaAbilities = append(ctx.Game.playerManaAbilities, installedPlayerManaAbility{
		id: uuid.New(), player: player, source: ctx.SourceID, life: e.life, color: e.color,
	})
	return nil
}

type installedPlayerManaAbility struct {
	id, player, source uuid.UUID
	life               int
	color              core.Color
}

// PlayerManaAbilityInfo identifies one life-paid mana permission.
type PlayerManaAbilityInfo struct {
	ID, SourceID uuid.UUID
	LifeCost     int
	Color        core.Color
}

// GetPlayerManaAbilities lists affordable permissions without spending resources.
func (g *Game) GetPlayerManaAbilities(player uuid.UUID) []PlayerManaAbilityInfo {
	p := g.GetPlayer(player)
	if p == nil || !g.CanLoseLife(player) || g.resolution.resolvingController != uuid.Nil {
		return nil
	}
	var result []PlayerManaAbilityInfo
	for _, a := range g.playerManaAbilities {
		if a.player == player && p.Life() >= a.life {
			result = append(result, PlayerManaAbilityInfo{a.id, a.source, a.life, a.color})
		}
	}
	return result
}

// ActivatePlayerManaAbility pays life and immediately adds mana without a stack object.
// Direct callers must supply priority. Shared payment services supply mana windows.
func (g *Game) ActivatePlayerManaAbility(player, id uuid.UUID) error {
	if g.resolution.resolvingController != uuid.Nil {
		return fmt.Errorf("player mana action requires priority or a mana payment")
	}
	return g.activatePlayerManaAbility(player, id)
}

func (g *Game) activatePlayerManaAbility(player, id uuid.UUID) error {
	for _, a := range g.playerManaAbilities {
		if a.id != id {
			continue
		}
		if a.player != player {
			return fmt.Errorf("player mana permission belongs to another player")
		}
		cost := LifePayCost(a.life)
		if !cost.CanPay(a.source, player, g) {
			return fmt.Errorf("cannot pay %d life", a.life)
		}
		if err := cost.Pay(a.source, player, g); err != nil {
			return err
		}
		productions := g.replaceManaProduction(&manaProductionAction{source: a.source, player: player, productions: []ManaProduction{{Color: a.color, Amount: 1}}}, nil)
		addConcreteMana(g.GetPlayer(player).ManaPool(), productions, nil)
		return nil
	}
	return fmt.Errorf("player mana permission is absent or expired")
}

func (g *Game) appendPlayerManaSources(player uuid.UUID, sources []manaSourceInfo) []manaSourceInfo {
	p := g.GetPlayer(player)
	if p == nil || !g.CanLoseLife(player) {
		return sources
	}
	for _, a := range g.playerManaAbilities {
		if a.player != player || p.Life() < a.life {
			continue
		}
		sources = append(sources, manaSourceInfo{PlayerManaAbilityID: a.id, Colors: []core.Color{a.color}, Abilities: []manaSourceAbility{{LifeCost: a.life, Repeatable: true, Productions: []ManaProduction{{Color: a.color, Amount: 1}}}}})
	}
	return sources
}

func (g *Game) playerLifeManaCapacity(player uuid.UUID) int {
	p := g.GetPlayer(player)
	if p == nil || !g.CanLoseLife(player) {
		return 0
	}
	capacity := 0
	for _, a := range g.playerManaAbilities {
		if a.player == player {
			capacity = max(capacity, max(0, p.Life())/a.life)
		}
	}
	return capacity
}
