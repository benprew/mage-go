package mage

import "github.com/google/uuid"

// PermanentFilter is a predicate on permanents.
type PermanentFilter func(*Permanent, *Game) bool

// CardFilter is a predicate on cards.
type CardFilter func(Card) bool

// IsCreature matches creature permanents.
var IsCreature PermanentFilter = func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeCreature)
}

// IsArtifact matches artifact permanents.
var IsArtifact PermanentFilter = func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeArtifact)
}

// IsEnchantment matches enchantment permanents.
var IsEnchantment PermanentFilter = func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeEnchantment)
}

// IsLand matches land permanents.
var IsLand PermanentFilter = func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeLand)
}

// IsCreatureCard matches creature cards.
var IsCreatureCard CardFilter = func(c Card) bool {
	for _, t := range c.Types() {
		if t == TypeCreature {
			return true
		}
	}
	return false
}

// ControlledBy returns a filter matching permanents controlled by the given player.
func ControlledBy(playerID uuid.UUID) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.Controller == playerID
	}
}

// NotControlledBy returns a filter matching permanents not controlled by the given player.
func NotControlledBy(playerID uuid.UUID) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.Controller != playerID
	}
}

// HasColor returns a filter matching permanents whose card has the given color.
func HasColorFilter(c Color) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		for _, col := range p.Card.ManaCost().Colors() {
			if col == c {
				return true
			}
		}
		return false
	}
}

// NotID returns a filter that excludes a specific permanent.
func NotID(id uuid.UUID) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.ID() != id
	}
}

// Not negates a filter.
func Not(f PermanentFilter) PermanentFilter {
	return func(p *Permanent, g *Game) bool {
		return !f(p, g)
	}
}

// And combines filters with logical AND.
func And(fs ...PermanentFilter) PermanentFilter {
	return func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if !f(p, g) {
				return false
			}
		}
		return true
	}
}

// Or combines filters with logical OR.
func Or(fs ...PermanentFilter) PermanentFilter {
	return func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if f(p, g) {
				return true
			}
		}
		return false
	}
}

// HasSubType returns a filter matching permanents with the given subtype.
func HasSubType(subType string) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.HasSubType(subType)
	}
}

// Named returns a filter matching permanents with the given name.
func Named(name string) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.Name() == name
	}
}

// IsTapped matches tapped permanents.
var IsTapped PermanentFilter = func(p *Permanent, _ *Game) bool {
	return p.Tapped
}

// IsUntapped matches untapped permanents.
var IsUntapped PermanentFilter = func(p *Permanent, _ *Game) bool {
	return !p.Tapped
}

// IsAttacking matches creatures currently declared as attackers.
var IsAttacking PermanentFilter = func(p *Permanent, g *Game) bool {
	return g.Combat.IsAttacking(p.ID())
}

// HasKeywordFilter returns a filter matching permanents with the given keyword.
func HasKeywordFilter(kw Keyword) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.HasKeyword(kw)
	}
}

// NotHasKeywordFilter returns a filter matching permanents without the given keyword.
func NotHasKeywordFilter(kw Keyword) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return !p.HasKeyword(kw)
	}
}

// IsID returns a filter matching a specific permanent by ID.
func IsID(id uuid.UUID) PermanentFilter {
	return func(p *Permanent, _ *Game) bool {
		return p.ID() == id
	}
}

// IsBandedWith returns a filter matching permanents that are banded with the
// given permanent in the current combat.
func IsBandedWith(id uuid.UUID) PermanentFilter {
	return func(p *Permanent, g *Game) bool {
		if g.Combat == nil {
			return false
		}
		return g.Combat.IsBandedWith(p.ID(), id)
	}
}

// HasPowerGTE returns a filter matching creatures with power >= n.
func HasPowerGTE(n int) PermanentFilter {
	return func(p *Permanent, g *Game) bool {
		return p.CurrentPower(g) >= n
	}
}
