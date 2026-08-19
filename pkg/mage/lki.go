package mage

import (
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// Deferred follow-up steps for the LKI framework (steps A-C are done):
//
// Step D — Extend LKI to non-battlefield zone changes. Today only
// RemoveFromBattlefield captures a snapshot. CR 113.7a / 603.10 cover
// any zone change, so spells fizzling off the stack (stack -> graveyard
// / exile), discards (hand -> graveyard), milled cards (library ->
// graveyard), and exile-from-anywhere paths should snapshot too. Cards
// that need this: Goblin Welder reading swap targets when one is
// sacrificed mid-resolution; fizzled-spell triggers (Word of Command-
// like effects); discard-then-react patterns. Implementation: extend
// captureLKI to accept a non-battlefield Permanent-or-Card source, key
// snapshots by (id, fromZone) since the same UUID may move multiple
// times, and add corresponding fire sites in Stack.Resolve / Counter,
// PlayerDiscard, mill helpers.
//
// Step E — Refcounted / event-scoped retention. clearLKI fires at
// end-of-turn cleanup, which is wrong on both ends: too long for
// transient triggers (a finished trigger doesn't need its source
// snapshot anymore) and too short for durable references (Banishing
// Light / Astral Slide / Hazezon Tamar capture an LKI reference and
// need it across many turns). Implementation: add a scope/refcount
// model where the dispatcher opens a scope when a snapshot is taken,
// closes it after the spawned StackObjects resolve; continuous effects
// or delayed triggers that want durable LKI hold a strong reference
// to *PermanentLKI directly so it stays alive while the closure does.
// Drop the per-turn clearLKI in favor of refcount-driven cleanup.

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
	ViewAttachedTo() uuid.UUID
	ViewOwner() uuid.UUID
	ViewHasType(CardType) bool
	ViewHasSubType(string) bool
	ViewHasKeyword(Keyword) bool
	ViewHasColor(Color) bool
	ViewPower() int
	ViewToughness() int
	ViewIsToken() bool
	ViewCounters(CounterType) int
	ViewAbilities() []Ability
}

// LKIAbilities returns the captured runtime-abilities of an object that has
// just left the battlefield. Used by zone-change paths to dispatch
// leave-triggers via checkAbilitiesForEvent without each path needing its
// own pre-removal selfAbilities capture. Returns nil if no LKI snapshot
// exists for id.
func (g *Game) LKIAbilities(id uuid.UUID) []Ability {
	if lki := g.LKI(id); lki != nil {
		return lki.ViewAbilities()
	}
	return nil
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
func (v livePermanentView) ViewController() uuid.UUID { return v.p.ControllerID() }
func (v livePermanentView) ViewAttachedTo() uuid.UUID { return v.p.AttachedTo }
func (v livePermanentView) ViewOwner() uuid.UUID {
	if o := v.p.Card.Owner(); o != uuid.Nil {
		return o
	}
	return v.p.ControllerID()
}
func (v livePermanentView) ViewHasType(t CardType) bool   { return v.p.HasType(t) }
func (v livePermanentView) ViewHasSubType(s string) bool  { return v.p.HasSubType(s) }
func (v livePermanentView) ViewHasKeyword(k Keyword) bool { return v.p.HasKeyword(k) }
func (v livePermanentView) ViewHasColor(c Color) bool {
	return slices.Contains(v.p.Colors(), c)
}
func (v livePermanentView) ViewPower() int     { return v.p.CurrentPower(v.g) }
func (v livePermanentView) ViewToughness() int { return v.p.CurrentToughness(v.g) }
func (v livePermanentView) ViewIsToken() bool {
	return v.p.IsToken
}
func (v livePermanentView) ViewCounters(ct CounterType) int { return int(v.p.Counters[ct]) }
func (v livePermanentView) ViewAbilities() []Ability        { return v.p.RuntimeAbilities }

// PermanentLKI captures Last-Known-Information about a permanent at the
// moment it left the battlefield. Triggered abilities that reference the
// dying/leaving permanent need its controller, types, P/T, counters,
// runtime abilities, and attachments after the move (CR 603.6c, 603.10,
// 113.7a).
//
// Snapshot is a deep clone of the live *Permanent at capture time. Power
// and Toughness are also recorded as plain ints because CurrentPower /
// CurrentToughness on the cloned permanent would consult continuous
// effects sourced from the now-removed permanent and return zero.
//
// LKI is recorded by RemoveFromBattlefield just before the permanent
// leaves g.battlefield, read via Game.LKI(id) or Game.LookupObject(id),
// and cleared at end-of-turn cleanup.
type PermanentLKI struct {
	ID        uuid.UUID
	Name      string
	Owner     uuid.UUID
	Power     int
	Toughness int
	IsToken   bool

	// Snapshot is a frozen *Permanent capturing controller, types,
	// subtypes, keywords, counters, runtime abilities, attachments, and
	// the various override fields. Read-only — never mutate.
	Snapshot *Permanent
}

// HasType reports whether the LKI snapshot had the given card type.
func (l *PermanentLKI) HasType(t CardType) bool {
	if l.Snapshot == nil {
		return false
	}
	return l.Snapshot.HasType(t)
}

// HasSubType reports whether the LKI snapshot had the given subtype.
func (l *PermanentLKI) HasSubType(s string) bool {
	if l.Snapshot == nil {
		return false
	}
	return l.Snapshot.HasSubType(s)
}

// LKIView interface implementation for *PermanentLKI. Read-only,
// snapshot-backed.

func (l *PermanentLKI) ViewID() uuid.UUID { return l.ID }
func (l *PermanentLKI) ViewName() string  { return l.Name }
func (l *PermanentLKI) ViewController() uuid.UUID {
	if l.Snapshot == nil {
		return uuid.Nil
	}
	return l.Snapshot.ControllerID()
}
func (l *PermanentLKI) ViewAttachedTo() uuid.UUID {
	if l.Snapshot == nil {
		return uuid.Nil
	}
	return l.Snapshot.AttachedTo
}
func (l *PermanentLKI) ViewOwner() uuid.UUID { return l.Owner }
func (l *PermanentLKI) ViewHasType(t CardType) bool {
	return l.HasType(t)
}
func (l *PermanentLKI) ViewHasSubType(s string) bool {
	return l.HasSubType(s)
}
func (l *PermanentLKI) ViewHasKeyword(k Keyword) bool {
	if l.Snapshot == nil {
		return false
	}
	return l.Snapshot.HasKeyword(k)
}
func (l *PermanentLKI) ViewHasColor(c Color) bool {
	return l.Snapshot != nil && slices.Contains(l.Snapshot.Colors(), c)
}
func (l *PermanentLKI) ViewPower() int     { return l.Power }
func (l *PermanentLKI) ViewToughness() int { return l.Toughness }
func (l *PermanentLKI) ViewIsToken() bool  { return l.IsToken }
func (l *PermanentLKI) ViewCounters(ct CounterType) int {
	if l.Snapshot == nil {
		return 0
	}
	return int(l.Snapshot.Counters[ct])
}
func (l *PermanentLKI) ViewAbilities() []Ability {
	if l.Snapshot == nil {
		return nil
	}
	return l.Snapshot.RuntimeAbilities
}

// captureLKI records an LKI snapshot for a permanent that is about to leave
// the battlefield. Deep-clones the live *Permanent so the snapshot is a
// faithful frozen copy of all its state — counters, runtime abilities,
// attachments, P/T overrides — not a hand-picked subset. Idempotent;
// re-snapshotting overwrites.
//
// CR alignment: 113.7a / 603.10 say a triggered ability looks at the most
// recent existence of the object before the event fired. The snapshot is
// that "most recent existence." Power and Toughness are recorded outside
// the snapshot because CurrentPower(g) on the clone would consult
// continuous effects sourced from the (now-removed) permanent and return
// the wrong value.
func (g *Game) captureLKI(p *Permanent) {
	if p == nil {
		return
	}
	if g.lki == nil {
		g.lki = make(map[uuid.UUID]*PermanentLKI)
	}
	isToken := p.IsToken
	owner := p.Card.Owner()
	if owner == uuid.Nil {
		owner = p.ControllerID()
	}
	snap := &Permanent{}
	clonePermanentInto(snap, p)
	g.lki[p.ID()] = &PermanentLKI{
		ID:        p.ID(),
		Name:      p.Name(),
		Owner:     owner,
		Power:     p.CurrentPower(g),
		Toughness: p.CurrentToughness(g),
		IsToken:   isToken,
		Snapshot:  snap,
	}
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
