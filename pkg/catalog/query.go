package catalog

import "strings"

// CardByName returns all printings of the named card (case-insensitive).
// For split cards, searching by either face name returns the full card.
func (c *Catalog) CardByName(name string) []CardEntry {
	indices := c.byName[strings.ToLower(name)]
	return c.gather(indices)
}

// CardBySetAndNumber returns the card with the given set code and collector number.
func (c *Catalog) CardBySetAndNumber(set, number string) (CardEntry, bool) {
	key := strings.ToLower(set) + ":" + strings.ToLower(number)
	idx, ok := c.bySetNumber[key]
	if !ok {
		return CardEntry{}, false
	}
	return c.cards[idx], true
}

// CardBySetAndName returns the card with the given set code and name.
func (c *Catalog) CardBySetAndName(set, name string) (CardEntry, bool) {
	key := strings.ToLower(set) + ":" + strings.ToLower(name)
	idx, ok := c.bySetName[key]
	if !ok {
		return CardEntry{}, false
	}
	return c.cards[idx], true
}

// CardsBySet returns all cards in the given set.
func (c *Catalog) CardsBySet(setCode string) []CardEntry {
	indices := c.bySet[strings.ToLower(setCode)]
	return c.gather(indices)
}

// CardsByArtist returns all cards illustrated by the given artist (case-insensitive).
func (c *Catalog) CardsByArtist(artist string) []CardEntry {
	indices := c.byArtist[strings.ToLower(artist)]
	return c.gather(indices)
}

// AllPrintings returns all printings of a card identified by its Oracle ID.
func (c *Catalog) AllPrintings(oracleID string) []CardEntry {
	indices := c.byOracleID[oracleID]
	return c.gather(indices)
}

// SetInfo returns metadata for the given set code.
func (c *Catalog) GetSetInfo(code string) (SetInfo, bool) {
	si, ok := c.sets[strings.ToLower(code)]
	return si, ok
}

// AllSets returns metadata for all loaded sets.
func (c *Catalog) AllSets() []SetInfo {
	result := make([]SetInfo, 0, len(c.sets))
	for _, si := range c.sets {
		result = append(result, si)
	}
	return result
}

// IsReservedList returns true if any printing of the named card is on the reserved list.
func (c *Catalog) IsReservedList(name string) bool {
	for _, idx := range c.byName[strings.ToLower(name)] {
		if c.cards[idx].Reserved {
			return true
		}
	}
	return false
}

// CardInSet returns true if a card with the given name exists in the given set.
func (c *Catalog) CardInSet(setCode, name string) bool {
	_, ok := c.bySetName[strings.ToLower(setCode)+":"+strings.ToLower(name)]
	return ok
}

// CardCount returns the total number of card entries in the catalog.
func (c *Catalog) CardCount() int {
	return len(c.cards)
}

func (c *Catalog) gather(indices []int) []CardEntry {
	if len(indices) == 0 {
		return nil
	}
	result := make([]CardEntry, len(indices))
	for i, idx := range indices {
		result[i] = c.cards[idx]
	}
	return result
}
