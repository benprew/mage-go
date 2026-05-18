package interactive

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

func buildRulesText(c mage.Card) string {
	var parts []string
	formatActivated := func(ab mage.ActivatedAbility) string {
		var costParts []string
		for _, cost := range ab.Costs() {
			costParts = append(costParts, cost.Text())
		}
		var effParts []string
		for _, eff := range ab.Effects() {
			effParts = append(effParts, eff.Text())
		}
		return fmt.Sprintf("%s: %s", joinStrings(costParts), joinStrings(effParts))
	}
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
			if ab.Kind() != mage.ActionSpell {
				parts = append(parts, formatActivated(ab))
				break
			}
			for _, eff := range ab.Effects() {
				parts = append(parts, eff.Text())
			}
		case mage.ActivatedAbility:
			parts = append(parts, formatActivated(ab))
		}
	}
	return joinStrings(parts)
}

func joinStrings(ss []string) string {
	var result strings.Builder
	for i, s := range ss {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(s)
	}
	return result.String()
}

// SnapshotGameState creates a read-only snapshot of the game for the TUI.
func SnapshotGameState(g *mage.Game, humanIndex int) *GameState {
	human := g.PlayerAt(humanIndex)
	aiIndex := (humanIndex + 1) % 2
	ai := g.PlayerAt(aiIndex)

	viewerID := human.PlayerID()
	return &GameState{
		Turn:         g.CurrentTurn(),
		Step:         g.GetStep().String(),
		ActivePlayer: g.ActivePlayerObj().Name(),
		You:          snapshotPlayer(g, human, viewerID, true),
		Opponent:     snapshotPlayer(g, ai, viewerID, false),
		StackItems:   snapshotStack(g),
	}
}

// snapshotPlayer builds a PlayerState for player p, redacting any private
// information that viewerID is not permitted to see (most importantly,
// face-down exiled cards owned by p that viewerID is not in the
// RevealedTo set for, per CR 707.2).
func snapshotPlayer(g *mage.Game, p mage.Player, viewerID uuid.UUID, showHand bool) PlayerState {
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

	exile := g.GetExile()
	for i := range exile {
		ec := &exile[i]
		if ec.Card.Owner() != p.PlayerID() {
			continue
		}
		cs := CardState{
			ID:       ec.Card.ID(),
			FaceDown: ec.FaceDown,
		}
		if ec.VisibleTo(viewerID) {
			c := ec.Card
			cs.Name = c.Name()
			cs.ManaCost = c.ManaCost().String()
			cs.IsLand = c.HasType(core.TypeLand)
			cs.Power = c.Power()
			cs.Toughness = c.Toughness()
			cs.RulesText = buildRulesText(c)
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
		}
		ps.Exile = append(ps.Exile, cs)
	}

	for _, perm := range g.AllBattlefield() {
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
			FaceDown:   perm.FaceDown,
			PhasedOut:  perm.PhasedOut,
			IsCreature: perm.HasType(core.TypeCreature),
			IsLand:     perm.HasType(core.TypeLand),
			IsArtifact: perm.HasType(core.TypeArtifact),
			Attacking:  g.GetCombat().IsAttacking(perm.ID()),
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
		for ct := range core.NumCounters {
			n := perm.Counters[ct]
			if n == 0 {
				continue
			}
			permState.RawCounters[ct] = n
			if permState.Counters == nil {
				permState.Counters = make(map[string]int)
			}
			permState.Counters[ct.String()] = int(n)
		}
		permState.Keywords = perm.KeywordNames()
		permState.AttachedTo = perm.AttachedTo
	outer:
		for _, group := range g.CombatGroups() {
			for _, bid := range group.BlockerIDs {
				if bid == perm.ID() {
					permState.Blocking = group.AttackerID
					break outer
				}
			}
		}
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
	for _, obj := range g.StackObjects() {
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
		var targetIDs []uuid.UUID
		for _, tid := range obj.Targets {
			targetNames = append(targetNames, resolveTargetName(g, tid))
			targetIDs = append(targetIDs, tid)
		}
		items = append(items, StackItemState{
			ID:          obj.SourceID.String(),
			Name:        name,
			Controller:  controller,
			IsAbility:   obj.IsAbility,
			Targets:     targetNames,
			TargetIDs:   targetIDs,
			XValue:      obj.XValue,
			EventAmount: obj.EventAmount,
		})
	}
	return items
}

// GetAvailableActions returns the actions available to a player right now.
func GetAvailableActions(g *mage.Game, playerID uuid.UUID) []ActionOption {
	return getAvailableActions(g, playerID, true)
}

// GetAvailableActionsWithoutLabels returns legal actions without UI-only label
// strings. It is intended for non-interactive callers that only need action
// metadata.
func GetAvailableActionsWithoutLabels(g *mage.Game, playerID uuid.UUID) []ActionOption {
	return getAvailableActions(g, playerID, false)
}

func getAvailableActions(g *mage.Game, playerID uuid.UUID, includeLabels bool) []ActionOption {
	var options []ActionOption

	for _, c := range g.GetPlayableLands(playerID) {
		opt := ActionOption{
			Type:     ActionPlayLand,
			CardID:   c.ID(),
			CardName: c.Name(),
		}
		if includeLabels {
			opt.Label = fmt.Sprintf("Play %s", c.Name())
		}
		options = append(options, opt)
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
			if len(validTargets) == 0 {
				// MTG 601.2c: can't begin to cast a spell with no legal targets.
				continue
			}
			if includeLabels {
				validLabels = buildTargetLabels(g, validTargets)
			}
		}
		mc := card.ManaCost()
		opt := ActionOption{
			Type:              ActionCastSpell,
			CardID:            card.ID(),
			CardName:          card.Name(),
			NeedsTarget:       needsTarget,
			TargetType:        targetType,
			ValidTargets:      validTargets,
			ValidTargetLabels: validLabels,
		}
		if includeLabels {
			opt.Label = fmt.Sprintf("Cast %s %s", card.Name(), mc)
			opt.ManaCost = mc.String()
		}
		if mc.HasX {
			opt.NeedsX = true
			opt.MaxXValue = g.MaxXValue(playerID, mc, mage.SpellContextForCard(card))
		}
		options = append(options, opt)
	}

	for _, info := range g.GetActivatableAbilities(playerID) {
		opt := ActionOption{
			Type:         ActionActivateAbility,
			PermanentID:  info.PermanentID,
			AbilityIndex: info.AbilityIndex,
		}
		if includeLabels {
			opt.Label = fmt.Sprintf("Activate %s: %s", info.PermanentName, info.Description)
		}
		perm := g.FindPermanent(info.PermanentID)
		if perm != nil {
			aa, ok := perm.RuntimeAbilities[info.AbilityIndex].(mage.ActivatedAbility)
			if ok {
				if targets := aa.Targets(); len(targets) > 0 {
					opt.NeedsTarget = true
					opt.TargetType = targets[0]
					opt.ValidTargets = targets[0].Possible(perm.Controller, perm.Card, g)
					if len(opt.ValidTargets) == 0 {
						// MTG 602.5b: can't begin to activate an ability with no legal targets.
						continue
					}
					if includeLabels {
						opt.ValidTargetLabels = buildTargetLabels(g, opt.ValidTargets)
					}
				}
			}
		}
		options = append(options, opt)
	}

	options = append(options, ActionOption{
		Type: ActionPass,
	})
	if includeLabels {
		options[len(options)-1].Label = "Pass"
	}

	return options
}

func buildTargetLabels(g *mage.Game, ids []uuid.UUID) []string {
	labels := make([]string, len(ids))
	for i, id := range ids {
		for _, p := range g.AllPlayers() {
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
