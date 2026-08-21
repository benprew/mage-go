package mage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type costedManaSource struct {
	PermanentID  uuid.UUID
	AbilityIndex int
	Productions  []ManaProduction
	ManaCost     ManaCost
}

type manaPlanSource struct {
	PermanentID  uuid.UUID
	AbilityIndex int
	Productions  []ManaProduction
	Colors       []Color
	Amount       int
	PerTapOutput map[Color]int
	ManaCost     ManaCost
}

type manaProductionChoice struct {
	Color       Color
	Productions []ManaProduction
}

func costedActivatedManaSource(a Ability, sourceID uuid.UUID, abilityIndex int, g *Game) (costedManaSource, bool) {
	aa, ok := UnwrapAbility(a).(*SimpleActivatedAbility)
	if !ok || len(aa.targets) != 0 || len(aa.effects) == 0 {
		return costedManaSource{}, false
	}

	var productions []ManaProduction
	for _, effect := range aa.effects {
		switch e := effect.(type) {
		case *addManaEffect:
			productions = append(productions, ManaProduction{Color: e.color, Amount: e.amount})
		case *addAnyManaEffect:
			productions = append(productions, ManaProduction{Color: AnyColor, Amount: e.amount})
		default:
			return costedManaSource{}, false
		}
	}

	var manaCost ManaCost
	hasManaCost := false
	hasTapCost := false
	for _, cost := range aa.costs {
		switch c := cost.(type) {
		case *ManaCostPayment:
			if hasManaCost {
				return costedManaSource{}, false
			}
			manaCost = c.reducedCost(sourceID, g)
			hasManaCost = !manaCost.IsZero()
		case *tap:
			hasTapCost = true
		default:
			return costedManaSource{}, false
		}
	}
	if !hasManaCost || !hasTapCost {
		return costedManaSource{}, false
	}

	return costedManaSource{
		PermanentID:  sourceID,
		AbilityIndex: abilityIndex,
		Productions:  productions,
		ManaCost:     manaCost,
	}, true
}

func (g *Game) getCostedManaSources(playerID uuid.UUID) []costedManaSource {
	var result []costedManaSource
	for _, perm := range g.battlefield {
		if perm.ControllerID() != playerID || perm.Tapped || perm.HasAttr(AttrCantActivate) || !perm.CanTapForEffect(g) {
			continue
		}
		for i, ability := range perm.RuntimeAbilities {
			source, ok := costedActivatedManaSource(ability, perm.ID(), i, g)
			if ok {
				result = append(result, source)
			}
		}
	}
	return result
}

func planSourcesForMana(direct []manaSourceInfo, costed []costedManaSource) []manaPlanSource {
	result := make([]manaPlanSource, 0, len(direct)+len(costed))
	for _, source := range direct {
		result = append(result, manaPlanSource{
			PermanentID:  source.PermanentID,
			AbilityIndex: -1,
			Colors:       source.Colors,
			Amount:       source.Amount,
			PerTapOutput: source.PerTapOutput,
		})
	}
	for _, source := range costed {
		result = append(result, manaPlanSource{
			PermanentID:  source.PermanentID,
			AbilityIndex: source.AbilityIndex,
			Productions:  source.Productions,
			ManaCost:     source.ManaCost,
		})
	}
	return result
}

func manaProductionChoices(source manaPlanSource) []manaProductionChoice {
	if source.AbilityIndex < 0 {
		if len(source.PerTapOutput) > 0 {
			productions := make([]ManaProduction, 0, len(source.PerTapOutput))
			for color, amount := range source.PerTapOutput {
				productions = append(productions, ManaProduction{Color: color, Amount: amount})
			}
			return []manaProductionChoice{{Color: Colorless, Productions: productions}}
		}
		choices := make([]manaProductionChoice, 0, len(source.Colors))
		for _, color := range source.Colors {
			choices = append(choices, manaProductionChoice{
				Color:       color,
				Productions: []ManaProduction{{Color: color, Amount: source.Amount}},
			})
		}
		return choices
	}

	for _, production := range source.Productions {
		if production.Color == AnyColor {
			return []manaProductionChoice{
				{Color: White, Productions: source.Productions},
				{Color: Blue, Productions: source.Productions},
				{Color: Black, Productions: source.Productions},
				{Color: Red, Productions: source.Productions},
				{Color: Green, Productions: source.Productions},
			}
		}
	}
	return []manaProductionChoice{{Color: Colorless, Productions: source.Productions}}
}

func addPlannedProductions(pool *ManaPool, productions []ManaProduction, preferred Color, bonuses []ManaBonusColor) {
	for _, production := range productions {
		amount := production.Amount
		if amount <= 0 {
			amount = 1
		}
		color := production.Color
		if color == AnyColor {
			color = preferred
		}
		pool.Add(color, amount)
	}
	for _, bonus := range bonuses {
		pool.Add(bonus.Resolve(preferred), 1)
	}
}

func planCostedManaPayment(in ManaSolverInputs) ([]ManaTap, bool) {
	if in.Pool == nil || len(in.CostedSources) == 0 {
		return nil, false
	}

	pool := NewManaPool()
	pool.RestorePool(in.Pool.SnapshotPool())
	pool.ManaConversions = in.Conversions
	sources := planSourcesForMana(in.Sources, in.CostedSources)
	used := make([]bool, len(sources))
	path := make([]ManaTap, 0, len(sources))
	memo := make(map[string]bool)
	bonusesFor := in.BonusesFor
	if bonusesFor == nil {
		bonusesFor = func(uuid.UUID) []ManaBonusColor { return nil }
	}

	var search func() bool
	search = func() bool {
		if pool.CanPay(in.Cost, in.SpellContext) {
			return true
		}
		key := manaPlanStateKey(pool, sources, used)
		if memo[key] {
			return false
		}
		memo[key] = true

		for i, source := range sources {
			if used[i] || !pool.CanPay(source.ManaCost, nil) {
				continue
			}
			for _, choice := range manaProductionChoices(source) {
				snapshot := pool.SnapshotPool()
				if err := pool.Pay(source.ManaCost, nil); err != nil {
					pool.RestorePool(snapshot)
					continue
				}
				addPlannedProductions(pool, choice.Productions, choice.Color, bonusesFor(source.PermanentID))
				for j := range sources {
					if sources[j].PermanentID == source.PermanentID {
						used[j] = true
					}
				}
				path = append(path, ManaTap{
					PermanentID:  source.PermanentID,
					AbilityIndex: source.AbilityIndex,
					Color:        choice.Color,
				})
				if search() {
					return true
				}
				path = path[:len(path)-1]
				for j := range sources {
					if sources[j].PermanentID == source.PermanentID {
						used[j] = false
					}
				}
				pool.RestorePool(snapshot)
			}
		}
		return false
	}

	if !search() {
		return nil, false
	}
	return append([]ManaTap(nil), path...), true
}

func manaPlanStateKey(pool *ManaPool, sources []manaPlanSource, used []bool) string {
	var b strings.Builder
	for _, color := range manaPoolColors {
		b.WriteString(strconv.Itoa(pool.Count(color)))
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for i, source := range sources {
		if used[i] {
			b.WriteString(source.PermanentID.String())
			b.WriteByte(',')
		}
	}
	return b.String()
}

func (g *Game) applyManaSolution(playerID uuid.UUID, solution *ManaSolution) error {
	if solution == nil {
		return nil
	}
	plan := solution.SourcesToTap
	for _, action := range plan {
		if action.AbilityIndex < 0 {
			if err := g.TapForManaWithColor(playerID, action.PermanentID, action.Color); err != nil {
				return err
			}
			continue
		}
		if err := g.activateCostedManaSource(playerID, action); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) activateCostedManaSource(playerID uuid.UUID, action ManaTap) error {
	perm := g.FindPermanent(action.PermanentID)
	if perm == nil || action.AbilityIndex < 0 || action.AbilityIndex >= len(perm.RuntimeAbilities) {
		return ErrPermanentNotFound
	}
	source, ok := costedActivatedManaSource(perm.RuntimeAbilities[action.AbilityIndex], perm.ID(), action.AbilityIndex, g)
	if !ok {
		return fmt.Errorf("ability is not a costed mana ability")
	}
	aa := UnwrapAbility(perm.RuntimeAbilities[action.AbilityIndex]).(*SimpleActivatedAbility)
	if err := g.payActionCosts(playerID, perm.ID(), aa.Costs()); err != nil {
		return err
	}
	if player := g.GetPlayer(playerID); player != nil {
		g.addManaProductionsForColor(source.Productions, player, perm, action.Color)
	}
	aa.MarkActivated()
	g.FireEvent(GameEvent{Type: EvtAbilityActivated, SourceID: perm.ID(), PlayerID: playerID})
	return nil
}
