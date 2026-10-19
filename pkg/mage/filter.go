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
	label     string
	fn        func(*Permanent, *Game) bool
	cardTypes *permanentCardTypeNode
	query     *permanentCollectionFilterNode
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
	label           string
	fn              func(Card) bool
	characteristics *cardCharacteristicNode
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
var AnyPermanent = permanentCardTypeFilter("permanent", &permanentCardTypeNode{operation: cardTypesAll})

// IsCreature matches creature permanents.
var IsCreature = permanentCardTypeFilter("creature", &permanentCardTypeNode{operation: cardTypesHas, kind: TypeCreature})

// IsArtifact matches artifact permanents.
var IsArtifact = permanentCardTypeFilter("artifact", &permanentCardTypeNode{operation: cardTypesHas, kind: TypeArtifact})

// IsEnchantment matches enchantment permanents.
var IsEnchantment = permanentCardTypeFilter("enchantment", &permanentCardTypeNode{operation: cardTypesHas, kind: TypeEnchantment})

// IsLand matches land permanents.
var IsLand = permanentCardTypeFilter("land", &permanentCardTypeNode{operation: cardTypesHas, kind: TypeLand})

// IsPlaneswalker matches planeswalker permanents.
var IsPlaneswalker = permanentCardTypeFilter("planeswalker", &permanentCardTypeNode{operation: cardTypesHas, kind: TypePlaneswalker})

// IsLegendary matches legendary permanents.
var IsLegendary = NewPermanentFilter("legendary", func(p *Permanent, _ *Game) bool {
	return p.Card.HasSuperType(SuperLegendary)
})

// IsCreatureCard matches creature cards.
var IsCreatureCard = NewCardFilter("creature card", func(c Card) bool {
	return slices.Contains(c.Types(), TypeCreature)
})

// IsLandCard matches land cards in any zone.
var IsLandCard = NewCardFilter("land card", func(c Card) bool {
	return slices.Contains(c.Types(), TypeLand)
})

// ControlledBy returns a filter matching permanents controlled by the given player.
func ControlledBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("you control", func(p *Permanent, _ *Game) bool {
		return p.ControllerID() == playerID
	})
}

// NotControlledBy returns a filter matching permanents not controlled by the given player.
func NotControlledBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("opponent controls", func(p *Permanent, _ *Game) bool {
		return p.ControllerID() != playerID
	})
}

// OwnedBy returns a filter matching permanents currently owned by playerID.
// Ownership and control are independent (CR 108.3, 110.2).
func OwnedBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("you own", func(p *Permanent, _ *Game) bool {
		return p.Card.Owner() == playerID
	})
}

// NotOwnedBy returns a filter matching permanents currently owned by another
// player, regardless of who controls them.
func NotOwnedBy(playerID uuid.UUID) PermanentFilter {
	return NewPermanentFilter("an opponent owns", func(p *Permanent, _ *Game) bool {
		return p.Card.Owner() != playerID
	})
}

// HasColorFilter returns a filter matching permanents whose current effective
// color set (honoring ColorOverride from continuous effects, CR 613 layer 5)
// includes the given color. Reading effective colors (not the raw mana cost)
// is what lets a later-layer effect like Crusade (+1/+1 to white creatures)
// pick up a creature that became white earlier in the same Apply() pass via
// a color-changing effect.
func HasColorFilter(c Color) PermanentFilter {
	filter := NewPermanentFilter(c.String(), func(p *Permanent, _ *Game) bool {
		return slices.Contains(p.Colors(), c)
	})
	filter.query = &permanentCollectionFilterNode{operation: collectionFilterColor, color: c}
	return filter
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
	result := NewPermanentFilter(label, func(p *Permanent, g *Game) bool {
		return !f.Match(p, g)
	})
	result.cardTypes = composePermanentCardTypes(cardTypesNot, []PermanentFilter{f})
	result.query = composePermanentCollectionFilters(collectionFilterNot, []PermanentFilter{f})
	return result
}

// And combines filters with logical AND.
func And(fs ...PermanentFilter) PermanentFilter {
	fs = slices.Clone(fs)
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.label != "" {
			parts = append(parts, f.label)
		}
	}
	result := NewPermanentFilter(strings.Join(parts, " "), func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if !f.Match(p, g) {
				return false
			}
		}
		return true
	})
	result.cardTypes = composePermanentCardTypes(cardTypesAll, fs)
	result.query = composePermanentCollectionFilters(collectionFilterAll, fs)
	return result
}

// Or combines filters with logical OR.
func Or(fs ...PermanentFilter) PermanentFilter {
	fs = slices.Clone(fs)
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.label != "" {
			parts = append(parts, f.label)
		}
	}
	result := NewPermanentFilter(strings.Join(parts, " or "), func(p *Permanent, g *Game) bool {
		for _, f := range fs {
			if f.Match(p, g) {
				return true
			}
		}
		return false
	})
	result.cardTypes = composePermanentCardTypes(cardTypesAny, fs)
	result.query = composePermanentCollectionFilters(collectionFilterAny, fs)
	return result
}

// HasSubType returns a filter matching permanents with the given subtype.
func HasSubType(subType string) PermanentFilter {
	filter := NewPermanentFilter(subType, func(p *Permanent, _ *Game) bool {
		return p.HasSubType(subType)
	})
	filter.query = &permanentCollectionFilterNode{operation: collectionFilterSubtype, subtype: subType}
	return filter
}

// Named returns a filter matching permanents with the given name.
func Named(name string) PermanentFilter {
	return NewPermanentFilter(name, func(p *Permanent, _ *Game) bool {
		return p.Name() == name
	})
}

// IsTapped matches tapped permanents.
var IsTapped = PermanentFilter{
	label: "tapped",
	fn:    func(p *Permanent, _ *Game) bool { return p.Tapped },
	query: &permanentCollectionFilterNode{operation: collectionFilterTapped},
}

// IsUntapped matches untapped permanents.
var IsUntapped = PermanentFilter{
	label: "untapped",
	fn:    func(p *Permanent, _ *Game) bool { return !p.Tapped },
	query: &permanentCollectionFilterNode{operation: collectionFilterNot, children: []*permanentCollectionFilterNode{{operation: collectionFilterTapped}}},
}

// IsEnchanted matches permanents with an Aura attached to them.
var IsEnchanted = NewPermanentFilter("enchanted", func(p *Permanent, g *Game) bool {
	return permanentIsEnchanted(p, g)
})

func permanentIsEnchanted(p *Permanent, g GameReader) bool {
	for _, attachmentID := range p.Attachments {
		attachment := g.FindPermanent(attachmentID)
		if attachment != nil && attachment.HasType(TypeEnchantment) && attachment.HasSubType("Aura") {
			return true
		}
	}
	return false
}

// IsAttacking matches creatures currently declared as attackers.
var IsAttacking = PermanentFilter{
	label: "attacking",
	fn: func(p *Permanent, g *Game) bool {
		return g.combat != nil && g.combat.IsAttacking(p.ID())
	},
	query: &permanentCollectionFilterNode{operation: collectionFilterAttacking, contextual: true},
}

// IsBlocking matches creatures currently declared as blockers.
var IsBlocking = PermanentFilter{
	label: "blocking",
	fn: func(p *Permanent, g *Game) bool {
		return g.combat != nil && g.combat.IsBlocking(p.ID())
	},
	query: &permanentCollectionFilterNode{operation: collectionFilterBlocking, contextual: true},
}

// HasKeywordFilter returns a filter matching permanents with the given keyword.
func HasKeywordFilter(kw Keyword) PermanentFilter {
	filter := NewPermanentFilter("with "+kw.String(), func(p *Permanent, _ *Game) bool {
		return p.HasKeyword(kw)
	})
	filter.query = &permanentCollectionFilterNode{operation: collectionFilterKeyword, keyword: kw}
	return filter
}

// NotHasKeywordFilter returns a filter matching permanents without the given keyword.
func NotHasKeywordFilter(kw Keyword) PermanentFilter {
	filter := NewPermanentFilter("without "+kw.String(), func(p *Permanent, _ *Game) bool {
		return !p.HasKeyword(kw)
	})
	filter.query = composePermanentCollectionFilters(collectionFilterNot, []PermanentFilter{HasKeywordFilter(kw)})
	return filter
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
	return typedCardFilter(color.String()+" card", &cardCharacteristicNode{operation: cardCharacteristicColor, color: color})
}

// HasSubTypeCardFilter returns a CardFilter matching cards with the given subtype.
func HasSubTypeCardFilter(subType string) CardFilter {
	return typedCardFilter(subType+" card", &cardCharacteristicNode{operation: cardCharacteristicSubtype, subtype: subType})
}

// IsPermanentCard matches cards with permanent types (Artifact, Creature, Enchantment, Land, Planeswalker).
var IsPermanentCard = NewCardFilter("permanent card", func(c Card) bool {
	return c.HasType(TypeArtifact) || c.HasType(TypeCreature) || c.HasType(TypeEnchantment) || c.HasType(TypeLand) || c.HasType(TypePlaneswalker)
})

// AndCardFilter returns a CardFilter that matches only if all provided filters match.
func AndCardFilter(filters ...CardFilter) CardFilter {
	filters = slices.Clone(filters)
	return CardFilter{label: "matching card", characteristics: composeCardCharacteristics(cardCharacteristicAll, filters), fn: func(c Card) bool {
		for _, f := range filters {
			if !f.Match(c) {
				return false
			}
		}
		return true
	}}
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

// HasPowerEqual matches permanents with current power equal to n.
func HasPowerEqual(n int) PermanentFilter {
	return fixedNumericPermanentFilter(fmt.Sprintf("with power %d", n), collectionFilterPowerEqual, n)
}

// HasToughnessEqual matches permanents with current toughness equal to n.
func HasToughnessEqual(n int) PermanentFilter {
	return fixedNumericPermanentFilter(fmt.Sprintf("with toughness %d", n), collectionFilterToughnessEqual, n)
}

// HasToughnessGTE matches permanents with current toughness at least n.
func HasToughnessGTE(n int) PermanentFilter {
	return fixedNumericPermanentFilter(fmt.Sprintf("with toughness %d or greater", n), collectionFilterToughnessGTE, n)
}

func fixedNumericPermanentFilter(label string, operation permanentCollectionFilterOperation, limit int) PermanentFilter {
	node := &permanentCollectionFilterNode{operation: operation, limit: limit}
	return PermanentFilter{label: label, query: node, fn: node.matches}
}

// IsAuraOnLand matches Aura enchantments attached to land permanents.
var IsAuraOnLand = NewPermanentFilter("Aura attached to a land", func(p *Permanent, g *Game) bool {
	if !p.HasType(TypeEnchantment) || !p.HasSubType("Aura") || !p.IsAttached() {
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

// IsAttachedTo matches permanents attached to a host that matches the filter.
func IsAttachedTo(filter PermanentFilter) PermanentFilter {
	result := NewPermanentFilter("attached to "+filter.Text(), func(p *Permanent, g *Game) bool {
		host := currentAttachmentHost(p, g)
		return host != nil && filter.Match(host, g)
	})
	if child, ok := inspectContextualPermanentCollectionFilter(filter); ok {
		result.query = &permanentCollectionFilterNode{operation: collectionFilterAttachment, children: []*permanentCollectionFilterNode{child}, contextual: true}
	}
	return result
}

func currentAttachmentHost(p *Permanent, g *Game) *Permanent {
	if p == nil || p.PhasedOut || g == nil {
		return nil
	}
	host := g.FindPermanent(p.AttachedTo)
	if host == nil || host.PhasedOut || p.attachedIncarnation != host.incarnationID {
		return nil
	}
	return host
}
