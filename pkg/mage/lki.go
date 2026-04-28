package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// LKIView is the read-only query interface satisfied by both a live
// *Permanent (via livePermanentView) and an LKI snapshot (*PermanentLKI).
// Predicates and effects that need to inspect an object whose liveness is
// uncertain — leave-triggers consulting CR 603.6c LKI, dies-triggers
// reading the dead creature's pre-removal state, "deal damage equal to its
// toughness" effects resolving after the source is gone — should query
// through this interface so the live/snapshot branch happens once at
// lookup time rather than at every call site.
//
// CR alignment: 113.7a / 603.10 say the game looks at the most recent
// state of an object before the relevant event. LKIView is that "most
// recent state" view, regardless of whether the object is still live.
type LKIView interface {
	ViewID() uuid.UUID
	ViewName() string
	ViewController() uuid.UUID
	ViewOwner() uuid.UUID
	ViewHasType(CardType) bool
	ViewHasSubType(string) bool
	ViewHasKeyword(Keyword) bool
	ViewPower() int
	ViewToughness() int
	ViewIsToken() bool
	ViewCounters(CounterType) int
	ViewAbilities() []Ability
}

// LookupObject returns an LKIView for the given object ID, preferring the
// live permanent on the battlefield and falling back to the LKI snapshot
// recorded at leave time. Returns nil if no live or snapshot view exists
// (the object never existed in a tracked zone, or its snapshot has been
// cleared).
//
// This is the canonical entry point for predicates and effects that need
// to read state about an object across the live/dead boundary. Use it
// instead of branching on FindPermanent + LKI manually.
func (g *Game) LookupObject(id uuid.UUID) LKIView {
	if p := g.FindPermanent(id); p != nil {
		return livePermanentView{p: p, g: g}
	}
	if lki := g.LKI(id); lki != nil {
		return lki
	}
	return nil
}

// livePermanentView adapts a live *Permanent to the LKIView interface so
// the same predicates work whether the object is live or snapshotted. The
// wrapper is value-typed and cheap to construct on each LookupObject call.
type livePermanentView struct {
	p *Permanent
	g GameReader
}

func (v livePermanentView) ViewID() uuid.UUID         { return v.p.ID() }
func (v livePermanentView) ViewName() string          { return v.p.Name() }
func (v livePermanentView) ViewController() uuid.UUID { return v.p.Controller }
func (v livePermanentView) ViewOwner() uuid.UUID {
	if o := v.p.Card.Owner(); o != uuid.Nil {
		return o
	}
	return v.p.Controller
}
func (v livePermanentView) ViewHasType(t CardType) bool   { return v.p.HasType(t) }
func (v livePermanentView) ViewHasSubType(s string) bool  { return v.p.HasSubType(s) }
func (v livePermanentView) ViewHasKeyword(k Keyword) bool { return v.p.HasKeyword(k) }
func (v livePermanentView) ViewPower() int                { return v.p.CurrentPower(v.g) }
func (v livePermanentView) ViewToughness() int            { return v.p.CurrentToughness(v.g) }
func (v livePermanentView) ViewIsToken() bool {
	if bc, ok := v.p.Card.(*BaseCard); ok {
		return bc.IsToken()
	}
	return false
}
func (v livePermanentView) ViewCounters(ct CounterType) int { return int(v.p.Counters[ct]) }
func (v livePermanentView) ViewAbilities() []Ability        { return v.p.RuntimeAbilities }

// PermanentLKI captures Last-Known-Information about a permanent at the
// moment it left the battlefield. Triggered abilities that reference the
// dying/leaving permanent (e.g. "whenever a creature an opponent controls
// dies") need its controller, types, and P/T after it has been moved to
// the graveyard. Per CR 603.10 and CR 113.7a, such triggers see the
// permanent's last existence on the battlefield.
//
// LKI is recorded by RemoveFromBattlefield just before the permanent leaves
// g.battlefield, and is read via Game.LKI(permID). Entries persist for the
// remainder of the current step so death-trigger conditions and effects can
// use them; they are cleared at end-of-turn cleanup.
//
// Step A note: the snapshot's field set is still the pre-framework subset
// (controller, owner, power, toughness, types, subtypes, isToken). Step B
// will replace this with a deep clone of the live *Permanent so abilities,
// counters, and attachments are preserved. Until then ViewCounters and
// ViewAbilities return zero / nil for snapshot views.
type PermanentLKI struct {
	ID         uuid.UUID
	Name       string
	Controller uuid.UUID
	Owner      uuid.UUID
	Power      int
	Toughness  int
	Types      []CardType
	SubTypes   []string
	IsToken    bool
	Keywords   []Keyword // captured for ViewHasKeyword
}

// HasType reports whether the LKI snapshot had the given card type.
func (l *PermanentLKI) HasType(t CardType) bool {
	for _, x := range l.Types {
		if x == t {
			return true
		}
	}
	return false
}

// HasSubType reports whether the LKI snapshot had the given subtype.
func (l *PermanentLKI) HasSubType(s string) bool {
	for _, x := range l.SubTypes {
		if x == s {
			return true
		}
	}
	return false
}

// LKIView interface implementation for *PermanentLKI. View* methods read
// from the snapshot fields directly.

func (l *PermanentLKI) ViewID() uuid.UUID         { return l.ID }
func (l *PermanentLKI) ViewName() string          { return l.Name }
func (l *PermanentLKI) ViewController() uuid.UUID { return l.Controller }
func (l *PermanentLKI) ViewOwner() uuid.UUID      { return l.Owner }
func (l *PermanentLKI) ViewHasType(t CardType) bool {
	return l.HasType(t)
}
func (l *PermanentLKI) ViewHasSubType(s string) bool {
	return l.HasSubType(s)
}
func (l *PermanentLKI) ViewHasKeyword(k Keyword) bool {
	for _, kw := range l.Keywords {
		if kw == k {
			return true
		}
	}
	return false
}
func (l *PermanentLKI) ViewPower() int                  { return l.Power }
func (l *PermanentLKI) ViewToughness() int              { return l.Toughness }
func (l *PermanentLKI) ViewIsToken() bool               { return l.IsToken }
func (l *PermanentLKI) ViewCounters(_ CounterType) int  { return 0 }      // step B: real
func (l *PermanentLKI) ViewAbilities() []Ability        { return nil }    // step B: real

// captureLKI records an LKI snapshot for a permanent that is about to leave
// the battlefield. Idempotent — re-snapshotting overwrites.
func (g *Game) captureLKI(p *Permanent) {
	if p == nil {
		return
	}
	if g.lki == nil {
		g.lki = make(map[uuid.UUID]*PermanentLKI)
	}
	types := make([]CardType, 0, 4)
	for _, t := range []CardType{TypeCreature, TypeArtifact, TypeEnchantment, TypeLand, TypeInstant, TypeSorcery} {
		if p.HasType(t) {
			types = append(types, t)
		}
	}
	subTypes := make([]string, len(p.Card.SubTypes()))
	copy(subTypes, p.Card.SubTypes())
	isToken := false
	if bc, ok := p.Card.(*BaseCard); ok {
		isToken = bc.IsToken()
	}
	owner := p.Card.Owner()
	keywords := captureLKIKeywords(p)
	g.lki[p.ID()] = &PermanentLKI{
		ID:         p.ID(),
		Name:       p.Name(),
		Controller: p.Controller,
		Owner:      owner,
		Power:      p.CurrentPower(g),
		Toughness:  p.CurrentToughness(g),
		Types:      types,
		SubTypes:   subTypes,
		IsToken:    isToken,
		Keywords:   keywords,
	}
}

// captureLKIKeywords extracts the keywords currently on a permanent so the
// LKI snapshot can answer ViewHasKeyword. Walks the keyword-range Attrs
// (Flying and above per IsKeywordAttr / attr.go), which is the universe
// of keywords a permanent can carry — abilities-block keywords like
// Indestructible, Hexproof, Trample, etc.
func captureLKIKeywords(p *Permanent) []Keyword {
	if p == nil {
		return nil
	}
	out := make([]Keyword, 0, 4)
	for a := Flying; a < Attr(NumAttrs); a++ {
		if !IsKeywordAttr(a) {
			continue
		}
		if p.HasKeyword(a) {
			out = append(out, a)
		}
	}
	return out
}

// LKI returns the last-known-information snapshot for a permanent that has
// left the battlefield, or nil if none is recorded. Triggered abilities that
// reference the dying permanent's controller, types, or P/T (e.g. "whenever
// a creature an opponent controls dies") should consult LKI when
// FindPermanent returns nil.
func (g *Game) LKI(id uuid.UUID) *PermanentLKI {
	if g.lki == nil {
		return nil
	}
	return g.lki[id]
}

// clearLKI wipes LKI entries. Called at the end of each turn's cleanup.
func (g *Game) clearLKI() {
	g.lki = nil
}
