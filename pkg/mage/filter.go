package mage

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/catalog"
	. "github.com/benprew/mage-go/pkg/mage/core"
)

// PermanentFilter is a labeled predicate on permanents.
// The zero value matches all permanents.
type PermanentFilter struct {
	label string
	fn    func(*Permanent, *Game) bool
}

// Match reports whether p satisfies this filter.
// A zero-value filter (nil fn) matches all permanents.
func (f PermanentFilter) Match(p *Permanent, g *Game) bool {
	if f.fn == nil {
		return true
	}
	return f.fn(p, g)
}

// Text returns a human-readable description of what this filter matches.
func (f PermanentFilter) Text() string { return f.label }

// IsZero reports whether this is the zero-value filter (matches all permanents).
func (f PermanentFilter) IsZero() bool { return f.fn == nil }

// NewPermanentFilter creates a PermanentFilter with the given label and predicate.
func NewPermanentFilter(label string, fn func(*Permanent, *Game) bool) PermanentFilter {
	return PermanentFilter{label: label, fn: fn}
}

// CardFilter is a labeled predicate on cards.
// The zero value matches all cards.
type CardFilter struct {
	label string
	fn    func(Card) bool
}

// Match reports whether c satisfies this filter.
// A zero-value filter (nil fn) matches all cards.
func (f CardFilter) Match(c Card) bool {
	if f.fn == nil {
		return true
	}
	return f.fn(c)
}

// Text returns a human-readable description of what this filter matches.
func (f CardFilter) Text() string { return f.label }

// IsZero reports whether this is the zero-value filter (matches all cards).
func (f CardFilter) IsZero() bool { return f.fn == nil }

// NewCardFilter creates a CardFilter with the given label and predicate.
func NewCardFilter(label string, fn func(Card) bool) CardFilter {
	return CardFilter{label: label, fn: fn}
}

// AnyPermanent matches all permanents without restriction.
var AnyPermanent = NewPermanentFilter("permanent", func(_ *Permanent, _ *Game) bool { return true })

// IsCreature matches creature permanents.
var IsCreature = NewPermanentFilter("creature", func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeCreature)
})

// IsArtifact matches artifact permanents.
var IsArtifact = NewPermanentFilter("artifact", func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeArtifact)
})

// IsEnchantment matches enchantment permanents.
var IsEnchantment = NewPermanentFilter("enchantment", func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeEnchantment)
})

// IsLand matches land permanents.
var IsLand = NewPermanentFilter("land", func(p *Permanent, _ *Game) bool {
	return p.HasType(TypeLand)
})

// IsPlaneswalker matches planeswalker permanents.
var IsPlaneswalker = NewPermanentFilter("planeswalker", func(p *Permanent, _ *Game) bool {
	return p.HasType(TypePlaneswalker)
})

// IsLegendary matches legendary permanents.
var IsLegendary = NewPermanentFilter("legendary", func(p *Permanent, _ *Game) bool {
	return p.Card.HasSuperType(SuperLegendary)
})

// IsCreatureCard matches creature cards.
var IsCreatureCard = NewCardFilter("creature card", func(c Card) bool {
	return slices.Contains(c.Types(), TypeCreature)
})

// ControlledBy returns a filter matching permanents controlled by the given player.
func ControlledBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("you control", func(p *Permanent, _ *Game) bool {
		return p.Controller == playerID
	})
}

// NotControlledBy returns a filter matching permanents not controlled by the given player.
func NotControlledBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("opponent controls", func(p *Permanent, _ *Game) bool {
		return p.Controller != playerID
	})
}

// HasColorFilter returns a filter matching permanents whose current effective
// color set (honoring ColorOverride from continuous effects, CR 613 layer 5)
// includes the given color. Reading effective colors (not the raw mana cost)
// is what lets a later-layer effect like Crusade (+1/+1 to white creatures)
// pick up a creature that became white earlier in the same Apply() pass via
// a color-changing effect.
func HasColorFilter(c Color) PermanentFilter {
	return NewPermanentFilter(c.String(), func(p *Permanent, _ *Game) bool {
		return slices.Contains(p.Colors(), c)
	})
}

// NotID returns a filter that excludes a specific permanent.
func NotID(id uuid.UUID) PermanentFilter {
	return NewPermanentFilter("", func(p *Permanent, _ *Game) bool {
		return p.ID() != id
	})
}

// Not negates a filter.
func Not(f PermanentFilter) PermanentFilter {
	label := "non-" + f.label
	if f.label == "" {
		label = ""
	}
	return NewPermanentFilter(label, func(p *Permanent, g *Game) bool {
		return !f.Match(p, g)
	})
}

// And combines filters with logical AND.
func And(fs ...PermanentFilter) PermanentFilter {
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.label != "" {
			parts = append(parts, f.label)
		}
	}
	return NewPermanentFilter(strings.Join(parts, " "), func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if !f.Match(p, g) {
				return false
			}
		}
		return true
	})
}

// Or combines filters with logical OR.
func Or(fs ...PermanentFilter) PermanentFilter {
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.label != "" {
			parts = append(parts, f.label)
		}
	}
	return NewPermanentFilter(strings.Join(parts, " or "), func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if f.Match(p, g) {
				return true
			}
		}
		return false
	})
}

// HasSubType returns a filter matching permanents with the given subtype.
func HasSubType(subType string) PermanentFilter {
	return NewPermanentFilter(subType, func(p *Permanent, _ *Game) bool {
		return p.HasSubType(subType)
	})
}

// Named returns a filter matching permanents with the given name.
func Named(name string) PermanentFilter {
	return NewPermanentFilter(name, func(p *Permanent, _ *Game) bool {
		return p.Name() == name
	})
}

// IsTapped matches tapped permanents.
var IsTapped = NewPermanentFilter("tapped", func(p *Permanent, _ *Game) bool {
	return p.Tapped
})

// IsUntapped matches untapped permanents.
var IsUntapped = NewPermanentFilter("untapped", func(p *Permanent, _ *Game) bool {
	return !p.Tapped
})

// IsAttacking matches creatures currently declared as attackers.
var IsAttacking = NewPermanentFilter("attacking", func(p *Permanent, g *Game) bool {
	return g.combat.IsAttacking(p.ID())
})

// IsBlocking matches creatures currently declared as blockers.
var IsBlocking = NewPermanentFilter("blocking", func(p *Permanent, g *Game) bool {
	return g.combat != nil && g.combat.IsBlocking(p.ID())
})

// HasKeywordFilter returns a filter matching permanents with the given keyword.
func HasKeywordFilter(kw Keyword) PermanentFilter {
	return NewPermanentFilter("with "+kw.String(), func(p *Permanent, _ *Game) bool {
		return p.HasKeyword(kw)
	})
}

// NotHasKeywordFilter returns a filter matching permanents without the given keyword.
func NotHasKeywordFilter(kw Keyword) PermanentFilter {
	return NewPermanentFilter("without "+kw.String(), func(p *Permanent, _ *Game) bool {
		return !p.HasKeyword(kw)
	})
}

// IsID returns a filter matching a specific permanent by ID.
func IsID(id uuid.UUID) PermanentFilter {
	return NewPermanentFilter("", func(p *Permanent, _ *Game) bool {
		return p.ID() == id
	})
}

// IsBandedWith returns a filter matching permanents that are banded with the
// given permanent in the current combat.
func IsBandedWith(id uuid.UUID) PermanentFilter {
	return NewPermanentFilter("", func(p *Permanent, g *Game) bool {
		if g.combat == nil {
			return false
		}
		return g.combat.IsBandedWith(p.ID(), id)
	})
}

// IsArtifactCard matches artifact cards (for graveyard/stack filtering).
var IsArtifactCard = NewCardFilter("artifact card", func(c Card) bool {
	return c.HasType(TypeArtifact)
})

// IsEnchantmentCard matches enchantment cards.
var IsEnchantmentCard = NewCardFilter("enchantment card", func(c Card) bool {
	return c.HasType(TypeEnchantment)
})

// IsInstantCard matches instant cards.
var IsInstantCard = NewCardFilter("instant card", func(c Card) bool {
	return c.HasType(TypeInstant)
})

// IsSorceryCard matches sorcery cards.
var IsSorceryCard = NewCardFilter("sorcery card", func(c Card) bool {
	return c.HasType(TypeSorcery)
})

// IsInstantOrSorceryCard matches cards that are instants or sorceries.
// Used by triggers like Opus ("Whenever you cast an instant or sorcery
// spell, …") and by cost-reduction value sources.
var IsInstantOrSorceryCard = NewCardFilter("instant or sorcery card", func(c Card) bool {
	return c.HasType(TypeInstant) || c.HasType(TypeSorcery)
})

// HasColorCardFilter returns a CardFilter matching cards with the given color.
func HasColorCardFilter(color Color) CardFilter {
	return NewCardFilter(color.String()+" card", func(c Card) bool {
		return slices.Contains(c.ManaCost().Colors(), color)
	})
}

// PrintedInSet returns a filter matching permanents whose card name was
// originally printed in the given set (by set code, e.g. "ARN").
func PrintedInSet(setCode string) PermanentFilter {
	return NewPermanentFilter(fmt.Sprintf("printed in %s", setCode), func(p *Permanent, g *Game) bool {
		return catalog.Global().CardInSet(setCode, p.Name())
	})
}

// HasPowerLTE returns a filter matching creatures with power <= n.
func HasPowerLTE(n int) PermanentFilter {
	return NewPermanentFilter(fmt.Sprintf("with power %d or less", n), func(p *Permanent, g *Game) bool {
		return p.CurrentPower(g) <= n
	})
}

// HasPowerGTE returns a filter matching creatures with power >= n.
func HasPowerGTE(n int) PermanentFilter {
	return NewPermanentFilter(fmt.Sprintf("with power %d or greater", n), func(p *Permanent, g *Game) bool {
		return p.CurrentPower(g) >= n
	})
}

// IsAuraOnLand matches enchantments (auras) attached to land permanents.
var IsAuraOnLand = NewPermanentFilter("Aura attached to a land", func(p *Permanent, g *Game) bool {
	if !p.HasType(TypeEnchantment) || !p.IsAttached() {
		return false
	}
	host := g.FindPermanent(p.AttachedTo)
	return host != nil && host.HasType(TypeLand)
})

// IsToken matches token permanents.
var IsToken = NewPermanentFilter("token", func(p *Permanent, g *Game) bool {
	return p.IsToken
})

// CreatedByFilter matches tokens created by a specific permanent.
func CreatedByFilter(id uuid.UUID) PermanentFilter {
	return NewPermanentFilter("created by source", func(p *Permanent, g *Game) bool {
		return p.CreatedBy == id
	})
}
