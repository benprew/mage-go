package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// ---------------------------------------------------------------------------
// 1. Regeneration: replaces destruction with tap + remove damage + remove from combat
// ---------------------------------------------------------------------------

type regenerationReplacement struct {
	replacementBase
	permanentID uuid.UUID
	shields     int
}

func (r *regenerationReplacement) Matches(a Action, g GameReader) bool {
	da, ok := a.(*DestroyPermanentAction)
	if !ok {
		return false
	}
	if da.PermanentID() != r.permanentID {
		return false
	}
	perm := g.FindPermanent(r.permanentID)
	if perm == nil {
		return false
	}
	return !perm.HasKeyword(CantRegenerate)
}

func (r *regenerationReplacement) Replace(a Action, g *Game) Action {
	r.shields--
	perm := g.MutablePermanent(r.permanentID)
	if perm != nil {
		g.TapPermanent(perm)
		perm.Damage = 0
		g.RemoveFromCombat(perm.ID())
	}
	return nil // destruction is fully replaced
}

func (r *regenerationReplacement) IsActive(_ GameReader) bool {
	return r.shields > 0
}

func (r *regenerationReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 2. Prevention shield: absorbs damage up to remaining amount
// ---------------------------------------------------------------------------

type preventionShieldReplacement struct {
	replacementBase
	targetID  uuid.UUID
	remaining int
}

func (r *preventionShieldReplacement) Matches(a Action, g GameReader) bool {
	switch act := a.(type) {
	case *DamageToPlayerAction:
		return act.PlayerID() == r.targetID
	case *DamageToCreatureAction:
		return act.PermanentID() == r.targetID
	}
	return false
}

func (r *preventionShieldReplacement) Replace(a Action, g *Game) Action {
	switch act := a.(type) {
	case *DamageToPlayerAction:
		prevented := min(act.Amount(), r.remaining)
		r.remaining -= prevented
		newAmount := act.Amount() - prevented
		if newAmount <= 0 {
			return nil
		}
		return act.WithAmount(newAmount)
	case *DamageToCreatureAction:
		prevented := min(act.Amount(), r.remaining)
		r.remaining -= prevented
		newAmount := act.Amount() - prevented
		if newAmount <= 0 {
			return nil
		}
		return act.WithAmount(newAmount)
	}
	return a
}

func (r *preventionShieldReplacement) IsActive(_ GameReader) bool {
	return r.remaining > 0
}

func (r *preventionShieldReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

type sourcePreventionShieldReplacement struct {
	replacementBase
	playerID  uuid.UUID
	dmgSource uuid.UUID
	remaining int
}

func (r *sourcePreventionShieldReplacement) Matches(action Action, _ GameReader) bool {
	damage, ok := action.(*DamageToPlayerAction)
	return ok && damage.PlayerID() == r.playerID && damage.ActionSource() == r.dmgSource
}

func (r *sourcePreventionShieldReplacement) Replace(action Action, _ *Game) Action {
	damage := action.(*DamageToPlayerAction)
	prevented := min(damage.Amount(), r.remaining)
	r.remaining -= prevented
	if remaining := damage.Amount() - prevented; remaining > 0 {
		return damage.WithAmount(remaining)
	}
	return nil
}

func (r *sourcePreventionShieldReplacement) IsActive(_ GameReader) bool { return r.remaining > 0 }
func (r *sourcePreventionShieldReplacement) Clone() ReplacementEffect {
	cp := *r
	return &cp
}

// ---------------------------------------------------------------------------
// 3. Fog: prevent all combat damage
// ---------------------------------------------------------------------------

type fogReplacement struct {
	replacementBase
}

func (r *fogReplacement) Matches(a Action, _ GameReader) bool {
	switch act := a.(type) {
	case *DamageToPlayerAction:
		return act.IsCombatDamage()
	case *DamageToCreatureAction:
		return act.IsCombatDamage()
	}
	return false
}

func (r *fogReplacement) Replace(a Action, g *Game) Action {
	return nil
}

func (r *fogReplacement) IsActive(_ GameReader) bool {
	return true
}

func (r *fogReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 4. Forcefield: reduce unblocked combat damage to player to 1
// ---------------------------------------------------------------------------

type forcefieldReplacement struct {
	replacementBase
	playerID   uuid.UUID
	attackerID uuid.UUID
	consumed   bool
}

func (r *forcefieldReplacement) Matches(a Action, _ GameReader) bool {
	if r.consumed {
		return false
	}
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	return act.IsCombatDamage() && act.PlayerID() == r.playerID && act.ActionSource() == r.attackerID && act.Amount() > 1
}

func (r *forcefieldReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	r.consumed = true
	return act.WithAmount(1)
}

func (r *forcefieldReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *forcefieldReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 5. Color prevention: prevent all damage from one source of a matching color
// ---------------------------------------------------------------------------

type colorPreventionReplacement struct {
	replacementBase
	playerID uuid.UUID
	color    Color
	consumed bool
}

func (r *colorPreventionReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.playerID {
		return false
	}
	sourceCard := g.FindCardAnywhere(act.ActionSource())
	if sourceCard == nil {
		return false
	}

	if slices.Contains(sourceCard.ManaCost().Colors(), r.color) {
		return true
	}
	return false
}

func (r *colorPreventionReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	return nil
}

func (r *colorPreventionReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *colorPreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// 5b. Source prevention: prevent next damage from a specific source to a player
// ---------------------------------------------------------------------------

type sourcePreventionReplacement struct {
	replacementBase
	playerID  uuid.UUID
	dmgSource uuid.UUID
	consumed  bool
}

func (r *sourcePreventionReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.playerID {
		return false
	}
	return act.ActionSource() == r.dmgSource
}

func (r *sourcePreventionReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	return nil
}

func (r *sourcePreventionReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *sourcePreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// 6. Type prevention: prevent all damage from one source of a matching card type
// ---------------------------------------------------------------------------

type typePreventionReplacement struct {
	replacementBase
	playerID uuid.UUID
	cardType CardType
	consumed bool
}

func (r *typePreventionReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.playerID {
		return false
	}
	sourceCard := g.FindCardAnywhere(act.ActionSource())
	if sourceCard == nil {
		return false
	}
	return sourceCard.HasType(r.cardType)
}

func (r *typePreventionReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	return nil
}

func (r *typePreventionReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *typePreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 7. Reverse Damage: gain life equal to prevented damage
// ---------------------------------------------------------------------------

type reverseDamageReplacement struct {
	replacementBase
	playerID uuid.UUID
	consumed bool
}

func (r *reverseDamageReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	return act.PlayerID() == r.playerID
}

func (r *reverseDamageReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	p := g.GetPlayer(r.playerID)
	if p != nil {
		p.GainLife(act.Amount())
	}
	r.consumed = true
	return nil
}

func (r *reverseDamageReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *reverseDamageReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 8. Bodyguard: redirect unblocked combat damage to a creature (Veteran Bodyguard)
// ---------------------------------------------------------------------------

type bodyguardReplacement struct {
	replacementBase
	controllerID    uuid.UUID
	bodyguardPermID uuid.UUID
}

func (r *bodyguardReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if !act.IsCombatDamage() || act.PlayerID() != r.controllerID {
		return false
	}
	bg := g.FindPermanent(r.bodyguardPermID)
	return bg != nil && !bg.Tapped
}

func (r *bodyguardReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(act.ActionSource(), r.bodyguardPermID, act.Amount(), act.IsCombatDamage())
}

func (r *bodyguardReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.bodyguardPermID) != nil
}

func (r *bodyguardReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 9. Player damage redirect: redirect all damage to a creature (Personal Incarnation)
// ---------------------------------------------------------------------------

type playerDamageRedirectReplacement struct {
	replacementBase
	controllerID   uuid.UUID
	redirectPermID uuid.UUID
}

func (r *playerDamageRedirectReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.controllerID {
		return false
	}
	return g.FindPermanent(r.redirectPermID) != nil
}

func (r *playerDamageRedirectReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(act.ActionSource(), r.redirectPermID, act.Amount(), act.IsCombatDamage())
}

func (r *playerDamageRedirectReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.redirectPermID) != nil
}

func (r *playerDamageRedirectReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 10. Artifact damage redirect: redirect artifact damage to a creature (Martyrs of Korlis)
// ---------------------------------------------------------------------------

type artifactDamageRedirectReplacement struct {
	replacementBase
	controllerID   uuid.UUID
	redirectPermID uuid.UUID
}

func (r *artifactDamageRedirectReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.controllerID {
		return false
	}
	sourceCard := g.FindCardAnywhere(act.ActionSource())
	if sourceCard == nil {
		return false
	}
	return sourceCard.HasType(TypeArtifact)
}

func (r *artifactDamageRedirectReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(act.ActionSource(), r.redirectPermID, act.Amount(), act.IsCombatDamage())
}

func (r *artifactDamageRedirectReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.redirectPermID) != nil
}

func (r *artifactDamageRedirectReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 11. Creature damage redirect: redirect creature damage to a player (Jade Monolith)
// ---------------------------------------------------------------------------

type creatureDamageRedirectReplacement struct {
	replacementBase
	creatureID     uuid.UUID
	targetPlayerID uuid.UUID
	consumed       bool
}

func (r *creatureDamageRedirectReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DamageToCreatureAction)
	if !ok {
		return false
	}
	return act.PermanentID() == r.creatureID
}

func (r *creatureDamageRedirectReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToCreatureAction)
	r.consumed = true
	return NewDamageToPlayerAction(act.ActionSource(), r.targetPlayerID, act.Amount(), act.IsCombatDamage())
}

func (r *creatureDamageRedirectReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *creatureDamageRedirectReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 12. Attacker damage redirect: redirect damage from specific attacker to absorber (Shimian Night Stalker)
// ---------------------------------------------------------------------------

type attackerDamageRedirectReplacement struct {
	replacementBase
	attackerID     uuid.UUID
	absorberPermID uuid.UUID
}

func (r *attackerDamageRedirectReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	return act.ActionSource() == r.attackerID
}

func (r *attackerDamageRedirectReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	return NewDamageToCreatureAction(act.ActionSource(), r.absorberPermID, act.Amount(), act.IsCombatDamage())
}

func (r *attackerDamageRedirectReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.absorberPermID) != nil
}

func (r *attackerDamageRedirectReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 13. Lich life gain: draw cards instead of gaining life
// ---------------------------------------------------------------------------

type lichLifeGainReplacement struct {
	replacementBase
	playerID uuid.UUID
}

func (r *lichLifeGainReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*LifeGainAction)
	if !ok {
		return false
	}
	return act.PlayerID() == r.playerID
}

func (r *lichLifeGainReplacement) Replace(a Action, g *Game) Action {
	act := a.(*LifeGainAction)
	p := g.GetPlayer(r.playerID)
	if p != nil {
		for i := 0; i < act.Amount(); i++ {
			g.PlayerDrawCard(p)
		}
	}
	return nil
}

func (r *lichLifeGainReplacement) IsActive(g GameReader) bool {
	// Active while source (the Lich permanent) is on the battlefield
	return g.FindPermanent(r.sourceID) != nil
}

func (r *lichLifeGainReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 14. Minimum life: cap damage so life doesn't go below 1 (Ali from Cairo)
// ---------------------------------------------------------------------------

type minimumLifeReplacement struct {
	replacementBase
	playerID uuid.UUID
}

func (r *minimumLifeReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DamageToPlayerAction)
	if !ok {
		return false
	}
	return act.PlayerID() == r.playerID
}

func (r *minimumLifeReplacement) Replace(a Action, g *Game) Action {
	act := a.(*DamageToPlayerAction)
	p := g.GetPlayer(r.playerID)
	if p == nil {
		return a
	}
	maxDamage := max(p.Life()-1, 0)
	if act.Amount() <= maxDamage {
		return a
	}
	if maxDamage <= 0 {
		return nil
	}
	return act.WithAmount(maxDamage)
}

func (r *minimumLifeReplacement) IsActive(_ GameReader) bool {
	return true // per-cycle, re-registered by continuous effect
}

func (r *minimumLifeReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 15. Skip draw: skip the next normal draw (Island Sanctuary)
// ---------------------------------------------------------------------------

type skipDrawReplacement struct {
	replacementBase
	playerID uuid.UUID
	consumed bool
}

func (r *skipDrawReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DrawCardAction)
	if !ok {
		return false
	}
	return act.IsNormalDraw() && act.PlayerID() == r.playerID
}

func (r *skipDrawReplacement) Replace(a Action, g *Game) Action {
	r.consumed = true
	return nil
}

func (r *skipDrawReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *skipDrawReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 16. Island Sanctuary: optional skip-draw with sanctuary protection
// ---------------------------------------------------------------------------

type islandSanctuaryReplacement struct {
	replacementBase
	playerID uuid.UUID
}

func (r *islandSanctuaryReplacement) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DrawCardAction)
	if !ok {
		return false
	}
	// IsNormalDraw is set only for the active player's draw-step draw, so
	// matching on it + playerID covers Oracle's "during your draw step" gate.
	return act.IsNormalDraw() && act.PlayerID() == r.playerID
}

func (r *islandSanctuaryReplacement) Replace(a Action, g *Game) Action {
	p := g.GetPlayer(r.playerID)
	if p == nil {
		return a
	}
	if !p.ChooseMayAbility("skip your draw to activate Island Sanctuary") {
		return a
	}
	g.SetSanctuaryActive(r.playerID)
	return nil
}

func (r *islandSanctuaryReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *islandSanctuaryReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 17. Draw replacement: Aladdin's Lamp draw replacement
// ---------------------------------------------------------------------------

type drawReplacementEffect struct {
	replacementBase
	playerID uuid.UUID
	count    int
	consumed bool
}

func (r *drawReplacementEffect) Matches(a Action, _ GameReader) bool {
	act, ok := a.(*DrawCardAction)
	if !ok {
		return false
	}
	return act.IsNormalDraw() && act.PlayerID() == r.playerID
}

func (r *drawReplacementEffect) Replace(a Action, g *Game) Action {
	r.consumed = true
	p := g.GetPlayer(r.playerID)
	if p != nil {
		g.applyDrawReplacement(p, r.count)
	}
	return nil
}

func (r *drawReplacementEffect) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *drawReplacementEffect) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 16b. Empty-library draw replacement: "if you would draw a card while your
//      library has no cards in it, instead <callback>" (Ormos, Archive Keeper).
//      Stays active while the source permanent is on the battlefield.
// ---------------------------------------------------------------------------

type emptyLibraryDrawReplacement struct {
	replacementBase
	playerID uuid.UUID
	callback func(g *Game, sourceID uuid.UUID)
}

func (r *emptyLibraryDrawReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*DrawCardAction)
	if !ok {
		return false
	}
	if act.PlayerID() != r.playerID {
		return false
	}
	p := g.GetPlayer(r.playerID)
	if p == nil {
		return false
	}
	return len(p.Library()) == 0
}

func (r *emptyLibraryDrawReplacement) Replace(_ Action, g *Game) Action {
	if r.callback != nil {
		r.callback(g, r.sourceID)
	}
	return nil
}

func (r *emptyLibraryDrawReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *emptyLibraryDrawReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 17. Damage prevention rule: from/to filter-based prevention
// ---------------------------------------------------------------------------

type damagePreventionRuleReplacement struct {
	replacementBase
	from          PermanentFilter
	to            PermanentFilter
	oneShot       bool
	consumed      bool
	combatOnly    bool
	noncombatOnly bool
	playerOnly    bool
	// toPlayerID, when non-zero, makes this rule also match damage dealt to
	// that player (DamageToPlayerAction). Used for prevention rules that
	// protect a specific player (e.g. Blessed Sanctuary's "dealt to you").
	toPlayerID uuid.UUID
}

func (r *damagePreventionRuleReplacement) Matches(a Action, g GameReader) bool {
	game := g.(*Game)
	switch act := a.(type) {
	case *DamageToPlayerAction:
		// Player-targeted rules: a rule with toPlayerID set protects that
		// player; otherwise (legacy) the rule must have a "from" filter and
		// no "to" filter (source-only prevention like Lady Evangela).
		if r.toPlayerID != uuid.Nil {
			if act.PlayerID() != r.toPlayerID {
				return false
			}
			if !r.from.IsZero() {
				source := g.FindPermanent(act.ActionSource())
				if source == nil || !r.from.Match(source, game) {
					return false
				}
			}
			if r.combatOnly && !act.IsCombatDamage() {
				return false
			}
			if r.noncombatOnly && act.IsCombatDamage() {
				return false
			}
			return true
		}
		if !r.to.IsZero() {
			return false
		}
		if r.from.IsZero() {
			return false
		}
		source := g.FindPermanent(act.ActionSource())
		if source == nil || !r.from.Match(source, game) {
			return false
		}
		if r.combatOnly && !act.IsCombatDamage() {
			return false
		}
		if r.noncombatOnly && act.IsCombatDamage() {
			return false
		}
		return true
	case *DamageToCreatureAction:
		if r.playerOnly {
			return false
		}
		// A rule scoped exclusively to a target player (toPlayerID set, no
		// from/to filters) protects only that player, not creatures.
		if r.from.IsZero() && r.to.IsZero() {
			return false
		}
		source := g.FindPermanent(act.ActionSource())
		target := g.FindPermanent(act.PermanentID())
		if target == nil {
			return false
		}
		fromMatch := r.from.IsZero() || (source != nil && r.from.Match(source, game))
		toMatch := r.to.IsZero() || r.to.Match(target, game)
		if r.combatOnly && !act.IsCombatDamage() {
			return false
		}
		if r.noncombatOnly && act.IsCombatDamage() {
			return false
		}
		return fromMatch && toMatch
	}
	return false
}

func (r *damagePreventionRuleReplacement) Replace(a Action, g *Game) Action {
	if r.oneShot {
		r.consumed = true
	}
	return nil
}

func (r *damagePreventionRuleReplacement) IsActive(_ GameReader) bool {
	return !r.consumed
}

func (r *damagePreventionRuleReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// counterDamagePreventionReplacement prevents damage to its source by
// removing one counter of the configured type for each damage prevented.
type counterDamagePreventionReplacement struct {
	replacementBase
	counterType CounterType
}

func (r *counterDamagePreventionReplacement) Matches(a Action, g GameReader) bool {
	damage, ok := a.(*DamageToCreatureAction)
	if !ok || damage.PermanentID() != r.sourceID {
		return false
	}
	permanent := g.FindPermanent(r.sourceID)
	return permanent != nil && permanent.Counters[r.counterType] > 0
}

func (r *counterDamagePreventionReplacement) Replace(a Action, g *Game) Action {
	damage := a.(*DamageToCreatureAction)
	permanent := g.MutablePermanent(r.sourceID)
	if permanent == nil {
		return a
	}
	prevented := min(damage.Amount(), int(permanent.Counters[r.counterType]))
	permanent.RemoveCounter(r.counterType, prevented)
	remaining := damage.Amount() - prevented
	if remaining == 0 {
		return nil
	}
	return damage.WithAmount(remaining)
}

func (r *counterDamagePreventionReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *counterDamagePreventionReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

func (*counterDamagePreventionReplacement) IsPreventionEffect() bool { return true }

// ---------------------------------------------------------------------------
// Prevention effect classification (CR 616.1 ordering helper)
// ---------------------------------------------------------------------------

// PreventionEffect is an optional marker interface. Replacement effects that
// are conceptually "prevention" or "reduction" effects (damage prevention
// shields, Fog, Forcefield, color/source/type-scoped damage prevention, etc.)
// implement this so the replacement pipeline can order them after modifying
// replacements when both are applicable to the same event.
type PreventionEffect interface {
	IsPreventionEffect() bool
}

// isPreventionReplacement reports whether a replacement effect is a prevention
// effect under the PreventionEffect marker.
func isPreventionReplacement(r ReplacementEffect) bool {
	pe, ok := r.(PreventionEffect)
	return ok && pe.IsPreventionEffect()
}

func (*preventionShieldReplacement) IsPreventionEffect() bool       { return true }
func (*fogReplacement) IsPreventionEffect() bool                    { return true }
func (*forcefieldReplacement) IsPreventionEffect() bool             { return true }
func (*colorPreventionReplacement) IsPreventionEffect() bool        { return true }
func (*sourcePreventionReplacement) IsPreventionEffect() bool       { return true }
func (*sourcePreventionShieldReplacement) IsPreventionEffect() bool { return true }
func (*typePreventionReplacement) IsPreventionEffect() bool         { return true }
func (*damagePreventionRuleReplacement) IsPreventionEffect() bool   { return true }

// ---------------------------------------------------------------------------
// 18. Counter doubler: doubles +1/+1 counter placements on matching permanents
// (Doubling Season, Branching Evolution-style — CR 614.1c)
// ---------------------------------------------------------------------------

type counterDoublerReplacement struct {
	replacementBase
	counterType CounterType
	filter      PermanentFilter
}

func (r *counterDoublerReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*AddCountersAction)
	if !ok {
		return false
	}
	if act.CounterType() != r.counterType {
		return false
	}
	if act.Amount() <= 0 {
		return false
	}
	game, _ := g.(*Game)
	perm := g.FindPermanent(act.PermanentID())
	if perm == nil {
		return false
	}
	if r.filter.IsZero() {
		return true
	}
	return r.filter.Match(perm, game)
}

func (r *counterDoublerReplacement) Replace(a Action, _ *Game) Action {
	act := a.(*AddCountersAction)
	return act.WithAmount(act.Amount() * 2)
}

func (r *counterDoublerReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *counterDoublerReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// ---------------------------------------------------------------------------
// 19. ETB-additional counters: adds N more counters when matching permanents
// enter the battlefield (Oona's Blackguard, Winding Constrictor — CR 614.1c).
// ---------------------------------------------------------------------------

type etbAdditionalCountersReplacement struct {
	replacementBase
	counterType CounterType
	extra       int
	filter      PermanentFilter
	// excludeSelf, if true, prevents the source permanent from buffing itself.
	excludeSelf bool
}

func (r *etbAdditionalCountersReplacement) Matches(a Action, g GameReader) bool {
	act, ok := a.(*AddCountersAction)
	if !ok || !act.OnEntry() {
		return false
	}
	if act.CounterType() != r.counterType {
		return false
	}
	if r.excludeSelf && act.PermanentID() == r.sourceID {
		return false
	}
	game, _ := g.(*Game)
	perm := g.FindPermanent(act.PermanentID())
	if perm == nil {
		return false
	}
	if r.filter.IsZero() {
		return true
	}
	return r.filter.Match(perm, game)
}

func (r *etbAdditionalCountersReplacement) Replace(a Action, _ *Game) Action {
	act := a.(*AddCountersAction)
	return act.WithAmount(act.Amount() + r.extra)
}

func (r *etbAdditionalCountersReplacement) IsActive(g GameReader) bool {
	return g.FindPermanent(r.sourceID) != nil
}

func (r *etbAdditionalCountersReplacement) Clone() ReplacementEffect {
	c := *r
	return &c
}

// etbAdditionalCounterTypesFor returns the unique set of counter types from
// active etbAdditionalCountersReplacement effects whose filter matches the
// entering permanent. Used by PutOnBattlefield (CR 614.1c) to synthesize
// AddCountersActions for permanents with no native "enters with" clause so
// effects like Oona's Blackguard can still place counters.
func (g *Game) etbAdditionalCounterTypesFor(perm *Permanent) []CounterType {
	seen := map[CounterType]bool{}
	var out []CounterType
	for _, r := range g.effects.replacements {
		etb, ok := r.(*etbAdditionalCountersReplacement)
		if !ok {
			continue
		}
		if !etb.IsActive(g) {
			continue
		}
		if etb.excludeSelf && perm.ID() == etb.sourceID {
			continue
		}
		if !etb.filter.IsZero() && !etb.filter.Match(perm, g) {
			continue
		}
		if seen[etb.counterType] {
			continue
		}
		seen[etb.counterType] = true
		out = append(out, etb.counterType)
	}
	return out
}
