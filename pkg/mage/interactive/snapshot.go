package interactive

import (
	"fmt"

	"github.com/google/uuid"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func buildRulesText(c mage.Card) string {
	var parts []string
	// Keywords come from the card's attr seeds (canonical storage).
	for a, count := range c.AttrSeeds() {
		if count > 0 && core.IsKeywordAttr(a) {
			parts = append(parts, a.String())
		}
	}
	for _, a := range c.Abilities() {
		switch ab := a.(type) {
		case *mage.ProtectionAbility:
			var colors []string
			for _, col := range ab.FromColors {
				colors = append(colors, col.String())
			}
			parts = append(parts, fmt.Sprintf("Protection from %s", joinStrings(colors)))
		case *mage.SpellAbility:
			for _, eff := range ab.Effects() {
				parts = append(parts, eff.Text())
			}
		case mage.ActivatedAbility:
			var costParts []string
			for _, cost := range ab.Costs() {
				costParts = append(costParts, cost.Text())
			}
			var effParts []string
			for _, eff := range ab.Effects() {
				effParts = append(effParts, eff.Text())
			}
			parts = append(parts, fmt.Sprintf("%s: %s", joinStrings(costParts), joinStrings(effParts)))
		}
	}
	return joinStrings(parts)
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// SnapshotGameState creates a read-only snapshot of the game for the TUI.
func SnapshotGameState(g *mage.Game, humanIndex int) *GameState {
	human := g.Players[humanIndex]
	aiIndex := (humanIndex + 1) % 2
	ai := g.Players[aiIndex]

	return &GameState{
		Turn:         g.Turn,
		Step:         g.Step.String(),
		ActivePlayer: g.ActivePlayerObj().Name(),
		You:          snapshotPlayer(g, human, true),
		Opponent:     snapshotPlayer(g, ai, false),
		StackItems:   snapshotStack(g),
	}
}

func snapshotPlayer(g *mage.Game, p mage.Player, showHand bool) PlayerState {
	ps := PlayerState{
		ID:             p.PlayerID(),
		Name:           p.Name(),
		Life:           p.Life(),
		HandCount:      len(p.Hand()),
		GraveyardCount: len(p.Graveyard()),
		LibraryCount:   len(p.Library()),
		ManaPool:       snapshotManaPool(p.ManaPool()),
	}

	if showHand {
		for _, c := range p.Hand() {
			cs := CardState{
				ID:        c.ID(),
				Name:      c.Name(),
				ManaCost:  c.ManaCost().String(),
				IsLand:    c.HasType(core.TypeLand),
				Power:     c.Power(),
				Toughness: c.Toughness(),
				RulesText: buildRulesText(c),
			}
			for _, t := range c.Types() {
				if cs.Types != "" {
					cs.Types += " "
				}
				cs.Types += t.String()
			}
			for _, st := range c.SubTypes() {
				if cs.SubTypes != "" {
					cs.SubTypes += " "
				}
				cs.SubTypes += st
			}
			ps.Hand = append(ps.Hand, cs)
		}
	}

	for _, c := range p.Graveyard() {
		cs := CardState{
			ID:        c.ID(),
			Name:      c.Name(),
			ManaCost:  c.ManaCost().String(),
			IsLand:    c.HasType(core.TypeLand),
			Power:     c.Power(),
			Toughness: c.Toughness(),
			RulesText: buildRulesText(c),
		}
		for _, t := range c.Types() {
			if cs.Types != "" {
				cs.Types += " "
			}
			cs.Types += t.String()
		}
		for _, st := range c.SubTypes() {
			if cs.SubTypes != "" {
				cs.SubTypes += " "
			}
			cs.SubTypes += st
		}
		ps.Graveyard = append(ps.Graveyard, cs)
	}

	for _, perm := range g.Battlefield {
		if perm.Controller != p.PlayerID() {
			continue
		}
		permState := PermanentState{
			ID:         perm.ID(),
			Name:       perm.Name(),
			Power:      perm.CurrentPower(g),
			Toughness:  perm.CurrentToughness(g),
			Tapped:     perm.Tapped,
			SummonSick: perm.HasAttr(core.AttrSummonSick),
			IsCreature: perm.HasType(core.TypeCreature),
			IsLand:     perm.HasType(core.TypeLand),
			IsArtifact: perm.HasType(core.TypeArtifact),
			Attacking:  g.Combat.IsAttacking(perm.ID()),
			ManaCost:   perm.Card.ManaCost().String(),
			RulesText:  buildRulesText(perm.Card),
		}
		for _, t := range perm.Card.Types() {
			if permState.Types != "" {
				permState.Types += " "
			}
			permState.Types += t.String()
		}
		for _, st := range perm.Card.SubTypes() {
			if permState.SubTypes != "" {
				permState.SubTypes += " "
			}
			permState.SubTypes += st
		}
		if len(perm.Counters) > 0 {
			permState.Counters = make(map[string]int)
			for ct, n := range perm.Counters {
				permState.Counters[ct.String()] = n
			}
		}
		permState.Keywords = perm.KeywordNames()
		ps.Battlefield = append(ps.Battlefield, permState)
	}

	return ps
}

func snapshotManaPool(mp *mage.ManaPool) ManaPoolState {
	return ManaPoolState{
		White:     mp.Count(core.White),
		Blue:      mp.Count(core.Blue),
		Black:     mp.Count(core.Black),
		Red:       mp.Count(core.Red),
		Green:     mp.Count(core.Green),
		Colorless: mp.Count(core.Colorless),
	}
}

func snapshotStack(g *mage.Game) []StackItemState {
	var items []StackItemState
	for _, obj := range g.Stack.Objects() {
		name := "Ability"
		if obj.Card != nil {
			name = obj.Card.Name()
		}
		controller := ""
		p := g.GetPlayer(obj.Controller)
		if p != nil {
			controller = p.Name()
		}
		var targetNames []string
		for _, tid := range obj.Targets {
			targetNames = append(targetNames, resolveTargetName(g, tid))
		}
		items = append(items, StackItemState{
			Name:       name,
			Controller: controller,
			IsAbility:  obj.IsAbility,
			Targets:    targetNames,
		})
	}
	return items
}

// GetAvailableActions returns the actions available to a player right now.
func GetAvailableActions(g *mage.Game, playerID uuid.UUID) []ActionOption {
	var options []ActionOption

	for _, c := range g.GetPlayableLands(playerID) {
		options = append(options, ActionOption{
			Type:     ActionPlayLand,
			Label:    fmt.Sprintf("Play %s", c.Name()),
			CardID:   c.ID(),
			CardName: c.Name(),
		})
	}

	for _, card := range g.GetCastableSpells(playerID) {
		needsTarget := false
		var targetType mage.Target
		var validTargets []uuid.UUID
		var validLabels []string
		if ct := card.CastTargets(); len(ct) > 0 {
			needsTarget = true
			targetType = ct[0]
			validTargets = targetType.Possible(playerID, card, g)
			validLabels = buildTargetLabels(g, validTargets)
		}
		options = append(options, ActionOption{
			Type:              ActionCastSpell,
			Label:             fmt.Sprintf("Cast %s %s", card.Name(), card.ManaCost()),
			CardID:            card.ID(),
			CardName:          card.Name(),
			NeedsTarget:       needsTarget,
			TargetType:        targetType,
			ManaCost:          card.ManaCost().String(),
			ValidTargets:      validTargets,
			ValidTargetLabels: validLabels,
		})
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		opt := ActionOption{
			Type:         ActionActivateAbility,
			Label:        fmt.Sprintf("Activate %s: %s", info.PermanentName, info.Description),
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
		}
		perm := g.FindPermanent(info.PermanentID)
		if perm != nil {
			aa, ok := perm.RuntimeAbilities[info.AbilityIndex].(mage.ActivatedAbility)
			if ok {
				if targets := aa.Targets(); len(targets) > 0 {
					opt.NeedsTarget = true
					opt.TargetType = targets[0]
					opt.ValidTargets = targets[0].Possible(perm.Controller, perm.Card, g)
					opt.ValidTargetLabels = buildTargetLabels(g, opt.ValidTargets)
				}
			}
		}
		options = append(options, opt)
	}

	options = append(options, ActionOption{
		Type:  ActionPass,
		Label: "Pass",
	})

	return options
}

func buildTargetLabels(g *mage.Game, ids []uuid.UUID) []string {
	labels := make([]string, len(ids))
	for i, id := range ids {
		for _, p := range g.Players {
			if p.PlayerID() == id {
				labels[i] = fmt.Sprintf("%s (player)", p.Name())
				break
			}
		}
		if labels[i] != "" {
			continue
		}
		if perm := g.FindPermanent(id); perm != nil {
			if perm.HasType(core.TypeCreature) {
				labels[i] = fmt.Sprintf("%s %d/%d", perm.Name(), perm.CurrentPower(g), perm.CurrentToughness(g))
			} else {
				labels[i] = perm.Name()
			}
		}
	}
	return labels
}
