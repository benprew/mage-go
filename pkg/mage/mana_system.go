package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ManaSystem manages mana-source discovery, production metadata, mana ability
// activation, solution planning, automatic payment support, and tapped-for-mana events.
type ManaSystem struct {
	scratch []manaSourceInfo
}

// NewManaSystem initializes a new ManaSystem.
func NewManaSystem() ManaSystem {
	return ManaSystem{}
}

// Clone creates an independent copy of ManaSystem for search. Scratch buffer
// is ephemeral to single planning operations and is reset.
func (ms *ManaSystem) Clone() ManaSystem {
	return ManaSystem{}
}

// EmptyManaPools empties every player's mana pool. Called at the end of every
// step and phase per CR 500.5.
func (ms *ManaSystem) EmptyManaPools(players []Player) {
	for _, p := range players {
		p.ManaPool().Clear()
	}
}

// ManaBonuses returns bonus mana colors for when a permanent is tapped for mana.
func (ms *ManaSystem) ManaBonuses(g *Game, permanentID uuid.UUID) []ManaBonusColor {
	tappedPerm := g.FindPermanent(permanentID)
	if tappedPerm == nil {
		return nil
	}
	var bonuses []ManaBonusColor
	for _, perm := range g.battlefield {
		for _, a := range perm.RuntimeAbilities {
			inner := UnwrapAbility(a)
			if mb, ok := inner.(*ManaBonusAbility); ok {
				if mb.AttachedOnly {
					if perm.AttachedTo == tappedPerm.ID() {
						bonuses = append(bonuses, ManaBonusColor(mb.BonusMana))
					}
				} else if mb.Filter.Match(tappedPerm, g) {
					if mb.MatchProduced {
						bonuses = append(bonuses, MatchProduced)
					} else {
						bonuses = append(bonuses, ManaBonusColor(mb.BonusMana))
					}
				}
			}
		}
	}
	return bonuses
}

// GetUntappedManaSources returns all untapped permanents with mana abilities for a player.
func (ms *ManaSystem) GetUntappedManaSources(g *Game, playerID uuid.UUID) []manaSourceInfo {
	var sources []manaSourceInfo
	return ms.AppendUntappedManaSources(g, playerID, sources)
}

// AppendUntappedManaSources appends untapped mana sources for a player to the target slice.
func (ms *ManaSystem) AppendUntappedManaSources(g *Game, playerID uuid.UUID, sources []manaSourceInfo) []manaSourceInfo {
	for _, perm := range g.battlefield {
		if perm.ControllerID() != playerID || perm.Tapped {
			continue
		}
		if perm.HasAttr(AttrCantActivate) {
			continue
		}
		// Skip summoning-sick creatures without haste
		if !perm.CanTapForEffect(g) {
			continue
		}
		var colors []Color
		seen := [AnyColor + 1]bool{}
		var abilities []manaSourceAbility
		for abilityIndex, ability := range perm.RuntimeAbilities {
			sourceAbility, ok := manaSourceAbilityForPlanning(ability, perm.ID(), abilityIndex, g)
			if !ok {
				continue
			}
			abilities = append(abilities, sourceAbility)
			for _, p := range sourceAbility.Productions {
				for _, c := range expandProductionColor(p.Color) {
					if !seen[c] {
						seen[c] = true
						colors = append(colors, c)
					}
				}
			}
		}
		if len(abilities) == 0 {
			continue
		}
		if len(colors) == 0 {
			colors = []Color{Colorless}
		}
		sources = append(sources, manaSourceInfo{
			PermanentID: perm.ID(),
			Colors:      colors,
			Abilities:   abilities,
			Bonuses:     ms.ManaBonuses(g, perm.ID()),
		})
	}
	return sources
}

// PreservationScore returns the score for a mana source under the smart-tap
// heuristic. Higher = prefer to keep untapped (tap last). See AutoTapHint.
func (ms *ManaSystem) PreservationScore(g *Game, src manaSourceInfo, hint AutoTapHint, handDemand [AnyColor + 1]int) int {
	score := 0
	if hint.ActivationSource != uuid.Nil && src.PermanentID == hint.ActivationSource {
		score += 1000
	}
	score += 10 * ms.UtilityAbilityCount(g, src.PermanentID)
	score += ms.ManaSourceDrawbackScore(g, src.PermanentID)
	for _, c := range src.Colors {
		if c == Colorless {
			continue
		}
		score++
		if c >= 0 && int(c) <= int(AnyColor) {
			score += handDemand[c]
		}
	}
	return score
}

// ManaSourceDrawbackScore calculates score penalties for tapping sources with detrimental triggers.
func (ms *ManaSystem) ManaSourceDrawbackScore(g *Game, permID uuid.UUID) int {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return 0
	}
	score := 0
	for _, ability := range perm.RuntimeAbilities {
		triggered, ok := UnwrapAbility(ability).(TriggeredAbility)
		if !ok || !triggered.CheckEventType(EvtTapped) {
			continue
		}
		for _, effect := range triggered.Effects() {
			props := effect.Properties()
			if props.Outcome != OutcomeDetriment {
				continue
			}
			severity := 1
			if props.DamageValue != nil {
				severity = max(props.DamageValue.Resolve(g, perm.ID(), perm.ControllerID(), nil), 1)
			}
			score += 25 * severity
		}
	}
	return score
}

// ComputeHandDemand sums colored pip demand across spells in the player's hand.
func (ms *ManaSystem) ComputeHandDemand(g *Game, playerID, exclude uuid.UUID) [AnyColor + 1]int {
	var demand [AnyColor + 1]int
	p := g.GetPlayer(playerID)
	if p == nil {
		return demand
	}
	for _, c := range p.Hand() {
		if c.ID() == exclude {
			continue
		}
		if c.HasType(TypeLand) {
			continue
		}
		mc := c.ManaCost()
		demand[White] += mc.White
		demand[Blue] += mc.Blue
		demand[Black] += mc.Black
		demand[Red] += mc.Red
		demand[Green] += mc.Green
		for _, h := range mc.Hybrid {
			demand[h.A]++
			demand[h.B]++
		}
	}
	return demand
}

// UtilityAbilityCount returns the number of non-mana activated abilities on a permanent.
func (ms *ManaSystem) UtilityAbilityCount(g *Game, permID uuid.UUID) int {
	perm := g.FindPermanent(permID)
	if perm == nil {
		return 0
	}
	count := 0
	for abilityIndex, a := range perm.RuntimeAbilities {
		inner := UnwrapAbility(a)
		if _, ok := inner.(ActivatedAbility); !ok {
			continue
		}
		if _, ok := manaSourceAbilityForPlanning(a, perm.ID(), abilityIndex, g); ok {
			continue
		}
		count++
	}
	return count
}

// HypotheticalMana returns the total hypothetical mana a player could produce right now.
func (ms *ManaSystem) HypotheticalMana(g *Game, playerID uuid.UUID) int {
	p := g.GetPlayer(playerID)
	if p == nil {
		return 0
	}
	total := p.ManaPool().TotalMana()
	sources := ms.AppendUntappedManaSources(g, playerID, ms.scratch[:0])
	ms.scratch = sources
	for _, src := range sources {
		total += maxManaSourceOutput(src)
	}
	return total
}

// CanAfford reports whether a player has enough mana to pay a cost.
func (ms *ManaSystem) CanAfford(g *Game, playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) bool {
	p := g.GetPlayer(playerID)
	if p == nil {
		return false
	}
	sources := ms.AppendUntappedManaSources(g, playerID, ms.scratch[:0])
	ms.scratch = sources
	return CanSolveMana(ManaSolverInputs{
		Pool:         p.ManaPool(),
		Cost:         mc,
		Sources:      sources,
		Conversions:  p.ManaPool().ManaConversions,
		SpellContext: spellCtx,
	})
}

// MaxXValue returns the maximum X value a player can pay for a spell with cost mc.
func (ms *ManaSystem) MaxXValue(g *Game, playerID uuid.UUID, mc ManaCost, spellCtx *SpellPaymentContext) int {
	if !mc.HasX || mc.XCount == 0 {
		return 0
	}
	p := g.GetPlayer(playerID)
	if p == nil {
		return 0
	}
	sources := ms.AppendUntappedManaSources(g, playerID, ms.scratch[:0])
	ms.scratch = sources
	scores := make([]int, len(sources))
	conv := p.ManaPool().ManaConversions
	pool := p.ManaPool()

	upperMana := pool.TotalMana()
	for _, src := range sources {
		upperMana += maxManaSourceOutput(src)
	}
	tryX := func(x int) bool {
		cost := ManaCost{
			Generic: mc.Generic + x*mc.XCount,
			White:   mc.White,
			Blue:    mc.Blue,
			Black:   mc.Black,
			Red:     mc.Red,
			Green:   mc.Green,
			Hybrid:  mc.Hybrid,
		}
		_, err := SolveMana(ManaSolverInputs{
			Pool:         pool,
			Cost:         cost,
			Sources:      sources,
			Scores:       scores,
			Conversions:  conv,
			SpellContext: spellCtx,
		})
		return err == nil
	}
	if !tryX(0) {
		return 0
	}
	hi := upperMana / mc.XCount
	if hi == 0 {
		return 0
	}
	lo := 0
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if tryX(mid) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// PlanManaForCost computes an optimal activation plan for a mana cost.
func (ms *ManaSystem) PlanManaForCost(g *Game, playerID uuid.UUID, mc ManaCost, hint AutoTapHint, spellCtx *SpellPaymentContext) (*ManaSolution, error) {
	p := g.GetPlayer(playerID)
	if p == nil {
		return nil, ErrPlayerNotFound
	}
	if mc.IsZero() {
		return &ManaSolution{}, nil
	}
	sources := ms.GetUntappedManaSources(g, playerID)

	if len(hint.ReservedSources) > 0 || hint.ActivationTapsSource && hint.ActivationSource != uuid.Nil {
		filtered := sources[:0]
		for _, src := range sources {
			reserved := hint.ActivationTapsSource && src.PermanentID == hint.ActivationSource
			if !reserved {
				if slices.Contains(hint.ReservedSources, src.PermanentID) {
					reserved = true
				}
			}
			if !reserved {
				filtered = append(filtered, src)
			}
		}
		sources = filtered
	}

	handDemand := ms.ComputeHandDemand(g, playerID, hint.CastingCard)
	scores := make([]int, len(sources))
	for i, src := range sources {
		scores[i] = ms.PreservationScore(g, src, hint, handDemand)
	}

	solverInputs := ManaSolverInputs{
		Pool:         p.ManaPool(),
		Cost:         mc,
		Sources:      sources,
		Scores:       scores,
		Conversions:  p.ManaPool().ManaConversions,
		SpellContext: spellCtx,
	}
	return SolveMana(solverInputs)
}

// AutoTapForCost plans and activates mana sources to pay a mana cost.
func (ms *ManaSystem) AutoTapForCost(g *Game, playerID uuid.UUID, mc ManaCost, hint AutoTapHint, spellCtx *SpellPaymentContext) error {
	solution, err := ms.PlanManaForCost(g, playerID, mc, hint, spellCtx)
	if err != nil {
		return err
	}
	return ms.ApplyManaSolution(g, playerID, solution)
}

// ApplyManaSolution executes a planned mana solution.
func (ms *ManaSystem) ApplyManaSolution(g *Game, playerID uuid.UUID, solution *ManaSolution) error {
	if solution == nil {
		return nil
	}
	for _, action := range solution.SourcesToTap {
		if err := ms.ActivatePlannedManaSource(g, playerID, action); err != nil {
			return err
		}
	}
	return nil
}

// ActivatePlannedManaSource activates a specific planned mana ability on a permanent.
func (ms *ManaSystem) ActivatePlannedManaSource(g *Game, playerID uuid.UUID, action ManaTap) error {
	perm := g.FindPermanent(action.PermanentID)
	if perm == nil || action.AbilityIndex < 0 || action.AbilityIndex >= len(perm.RuntimeAbilities) {
		return ErrPermanentNotFound
	}
	if perm.ControllerID() != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}
	if perm.HasAttr(AttrCantActivate) || !perm.CanTapForEffect(g) {
		return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
	}

	ability, ok := manaSourceAbilityForPlanning(perm.RuntimeAbilities[action.AbilityIndex], perm.ID(), action.AbilityIndex, g)
	if !ok {
		return fmt.Errorf("ability is not a supported mana ability")
	}
	if !plannedManaChoiceAvailable(ability.Productions, ms.ManaBonuses(g, perm.ID()), action) {
		return fmt.Errorf("planned mana production is no longer available")
	}
	player := g.GetPlayer(playerID)
	if player == nil {
		return ErrPlayerNotFound
	}
	if !player.ManaPool().CanPay(ability.ManaCost, nil) {
		return fmt.Errorf("cannot pay mana ability activation cost %s", ability.ManaCost)
	}
	if err := player.ManaPool().Pay(ability.ManaCost, nil); err != nil {
		return err
	}

	g.TapPermanent(perm)
	addConcreteMana(player.ManaPool(), action.Productions, action.BonusColors)

	inner := UnwrapAbility(perm.RuntimeAbilities[action.AbilityIndex])
	switch activated := inner.(type) {
	case *ManaAbility:
		if err := ms.RunManaPostProduction(g, activated, player, perm); err != nil {
			return err
		}
	case *SimpleActivatedAbility:
		activated.MarkActivated()
		g.FireEvent(GameEvent{Type: EvtAbilityActivated, SourceID: perm.ID(), PlayerID: playerID})
	}
	ms.FireTappedForMana(g, perm.ID(), playerID, productionsTotalAmount(action.Productions))
	return nil
}

// TapForMana taps a permanent for mana, picking an ability matching preferredColor if available.
func (ms *ManaSystem) TapForMana(g *Game, playerID, permanentID uuid.UUID, preferredColor Color) error {
	perm := g.FindPermanent(permanentID)
	if perm == nil {
		return ErrPermanentNotFound
	}
	if perm.ControllerID() != playerID {
		return fmt.Errorf("you don't control that permanent")
	}
	if perm.Tapped {
		return fmt.Errorf("permanent is already tapped")
	}
	if perm.HasAttr(AttrCantActivate) {
		return fmt.Errorf("cannot activate mana ability of %s", perm.Name())
	}

	var chosen []ManaProduction
	var chosenAbility *ManaAbility
	var firstProd []ManaProduction
	var firstAbility *ManaAbility
	for _, a := range perm.RuntimeAbilities {
		productions := abilityManaProductions(a, g, perm.ID())
		if productions == nil {
			continue
		}
		if firstProd == nil {
			firstProd = productions
			firstAbility, _ = UnwrapAbility(a).(*ManaAbility)
		}
		if preferredColor != Colorless && productionsMatchColor(productions, preferredColor) {
			chosen = productions
			chosenAbility, _ = UnwrapAbility(a).(*ManaAbility)
			break
		}
	}
	if chosen == nil {
		chosen = firstProd
		chosenAbility = firstAbility
	}
	if chosen == nil {
		return fmt.Errorf("permanent has no mana ability")
	}
	if !perm.CanTapForEffect(g) {
		return fmt.Errorf("creature has summoning sickness")
	}
	g.TapPermanent(perm)
	if p := g.GetPlayer(playerID); p != nil {
		produced := ms.AddManaProductions(g, chosen, p, perm, preferredColor)
		if err := ms.RunManaPostProduction(g, chosenAbility, p, perm); err != nil {
			return err
		}
		ms.FireTappedForMana(g, perm.ID(), playerID, produced)
	}
	return nil
}

// AddManaFromAbility resolves a mana ability's productions and post-production effects.
func (ms *ManaSystem) AddManaFromAbility(g *Game, ma *ManaAbility, p Player, perm *Permanent) (int, error) {
	productions := ma.currentProductions(g, perm.ID())
	produced := ms.AddManaProductions(g, productions, p, perm, Colorless)
	return produced, ms.RunManaPostProduction(g, ma, p, perm)
}

// AddManaProductions adds mana to the player's pool from productions.
func (ms *ManaSystem) AddManaProductions(g *Game, productions []ManaProduction, p Player, perm *Permanent, preferredColor Color) int {
	var producedColors []Color
	producedAmount := 0
	for _, prod := range productions {
		amt := prod.Amount
		if amt <= 0 {
			amt = 1
		}
		producedAmount += amt
		if prod.Color == AnyColor && prod.AnyCombination && amt > 1 {
			for i := 0; i < amt; i++ {
				color := p.ChooseManaColor("add mana")
				addManaProductionToPool(p.ManaPool(), color, 1, prod.Restriction)
				producedColors = appendProducedColor(producedColors, color)
			}
			continue
		}
		color := prod.Color
		if color == AnyColor {
			if preferredColor != Colorless {
				color = preferredColor
			} else {
				color = p.ChooseManaColor("add mana")
			}
		}
		addManaProductionToPool(p.ManaPool(), color, amt, prod.Restriction)
		producedColors = appendProducedColor(producedColors, color)
	}
	ms.ApplyManaBonuses(g, perm, producedColors, p)
	return producedAmount
}

// RunManaPostProduction runs follow-up effects from mana production.
func (ms *ManaSystem) RunManaPostProduction(g *Game, ma *ManaAbility, p Player, perm *Permanent) error {
	if ma == nil || len(ma.postProduction) == 0 {
		return nil
	}
	perm = g.MutablePermanent(perm.ID())
	if perm == nil {
		return nil
	}
	ctx := &EffectContext{
		Game:       g,
		SourceID:   perm.ID(),
		Controller: p.PlayerID(),
		Vars:       make(map[string]any),
	}
	for _, effect := range ma.postProduction {
		if err := effect.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}

// FireTappedForMana emits EvtTappedForMana event.
func (ms *ManaSystem) FireTappedForMana(g *Game, sourceID, playerID uuid.UUID, amount int) {
	if amount <= 0 {
		return
	}
	g.FireEvent(GameEvent{
		Type:     EvtTappedForMana,
		SourceID: sourceID,
		PlayerID: playerID,
		Amount:   amount,
	})
}

// ApplyManaBonuses applies any mana bonus abilities when a permanent is tapped for mana.
func (ms *ManaSystem) ApplyManaBonuses(g *Game, tappedPerm *Permanent, producedColors []Color, p Player) {
	for _, bonus := range ms.ManaBonuses(g, tappedPerm.ID()) {
		bonusColor := Color(bonus)
		if bonus == MatchProduced {
			if len(producedColors) == 0 {
				continue
			}
			bonusColor = producedColors[0]
			if len(producedColors) > 1 {
				chosen := p.ChooseManaColor("add bonus mana")
				if slices.Contains(producedColors, chosen) {
					bonusColor = chosen
				}
			}
		}
		p.ManaPool().Add(bonusColor, 1)
	}
}

func addManaProductionToPool(pool *ManaPool, color Color, amount int, restriction ManaRestriction) {
	if restriction != nil {
		pool.AddRestricted(color, amount, restriction)
		return
	}
	pool.Add(color, amount)
}

func expandProductionColor(c Color) []Color {
	if c == AnyColor {
		return []Color{White, Blue, Black, Red, Green}
	}
	return []Color{c}
}

func productionsTotalAmount(productions []ManaProduction) int {
	total := 0
	for _, p := range productions {
		if p.Amount <= 0 {
			total++
		} else {
			total += p.Amount
		}
	}
	return total
}

func maxManaSourceOutput(source manaSourceInfo) int {
	maximum := 0
	for _, ability := range source.Abilities {
		maximum = max(maximum, productionsTotalAmount(ability.Productions)+len(source.Bonuses))
	}
	return maximum
}

func appendProducedColor(colors []Color, color Color) []Color {
	if slices.Contains(colors, color) {
		return colors
	}
	return append(colors, color)
}
