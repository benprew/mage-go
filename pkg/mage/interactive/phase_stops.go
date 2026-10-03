package interactive

import (
	"sync"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type phaseStopSettings struct {
	sync.RWMutex
	enabled bool
	stops   [2]map[core.PhaseStep]bool
}

// EnablePhaseStops sets the default stops for the player's turns.
func (p *HumanPlayer) EnablePhaseStops() {
	p.phaseStops.Lock()
	defer p.phaseStops.Unlock()
	p.phaseStops.enabled = true
	p.phaseStops.stops = [2]map[core.PhaseStep]bool{
		{core.EndStep: true},
		{core.PrecombatMain: true, core.PostcombatMain: true},
	}
}

// SetPhaseStop sets a stop for one step and turn owner.
func (p *HumanPlayer) SetPhaseStop(step core.PhaseStep, ownTurn, stop bool) {
	p.phaseStops.Lock()
	defer p.phaseStops.Unlock()
	idx := 0
	if ownTurn {
		idx = 1
	}
	if p.phaseStops.stops[idx] == nil {
		p.phaseStops.stops[idx] = make(map[core.PhaseStep]bool)
	}
	p.phaseStops.stops[idx][step] = stop
}

// PhaseStop returns whether the player has selected a stop.
func (p *HumanPlayer) PhaseStop(step core.PhaseStep, ownTurn bool) bool {
	p.phaseStops.RLock()
	defer p.phaseStops.RUnlock()
	idx := 0
	if ownTurn {
		idx = 1
	}
	return p.phaseStops.stops[idx][step]
}

func (p *HumanPlayer) autoPassPriority(g *mage.Game, playerIdx int, options []ActionOption) bool {
	p.phaseStops.RLock()
	enabled := p.phaseStops.enabled
	p.phaseStops.RUnlock()
	if !enabled {
		return shouldAutoPassPriority(options)
	}
	own := g.ActivePlayerIndex() == playerIdx
	if p.PhaseStop(g.GetStep(), own) || g.StackSize() > 0 {
		return false
	}
	if own {
		return g.GetStep() != core.BeginCombat || len(getEligibleAttackers(g, g.PlayerAt(playerIdx).PlayerID())) == 0
	}
	return len(g.CombatGroups()) == 0
}
