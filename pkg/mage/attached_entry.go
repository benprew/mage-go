package mage

// BattlefieldEntryOption changes how a saved card enters the battlefield.
type BattlefieldEntryOption interface {
	battlefieldEntryOption()
	applyBattlefieldEntry(*battlefieldEntryOptions)
}

type battlefieldEntryOptions struct {
	tapped                     bool
	host                       PermanentRef
	tappedCount, attachedCount int
	invalid                    bool
	withoutSubtype             *enterWithoutSubtypeOption
	withoutSubtypeCount        int
}

func newBattlefieldEntryOptions(options []BattlefieldEntryOption) battlefieldEntryOptions {
	var entry battlefieldEntryOptions
	for _, option := range options {
		if option == nil {
			entry.invalid = true
			continue
		}
		option.applyBattlefieldEntry(&entry)
	}
	return entry
}

func (e battlefieldEntryOptions) inspect(node *definitionNode) {
	if e.invalid {
		node.problem = "supply non-nil entry options"
	}
	if e.tappedCount > 1 || e.attachedCount > 1 || e.withoutSubtypeCount > 1 {
		node.problem = "supply one entry option of each kind"
	}
	if e.withoutSubtype != nil {
		if err := ValidateSubtypeFamily(e.withoutSubtype.family, e.withoutSubtype.subtype); err != nil || e.withoutSubtype.subtype == "" {
			node.problem = "supply one valid subtype to remove"
		}
	}
	if e.attachedCount > 0 {
		node.children = append(node.children, definitionEdge{"entry-host", e.host})
	}
}

type enterTappedOption struct{}

// EnterTapped makes the card enter tapped.
func EnterTapped() BattlefieldEntryOption         { return enterTappedOption{} }
func (enterTappedOption) battlefieldEntryOption() {}
func (enterTappedOption) applyBattlefieldEntry(e *battlefieldEntryOptions) {
	e.tapped = true
	e.tappedCount++
}

type enterAttachedOption struct{ host PermanentRef }

// EnterAttachedTo saves the exact host before entry replacements run.
// An Aura stays in its zone if it cannot enchant that host after replacements.
func EnterAttachedTo(host PermanentRef) BattlefieldEntryOption { return enterAttachedOption{host} }
func (enterAttachedOption) battlefieldEntryOption()            {}
func (o enterAttachedOption) applyBattlefieldEntry(e *battlefieldEntryOptions) {
	e.host = o.host
	e.attachedCount++
}

type entryAttachment struct{ host capturedPermanent }

func (e battlefieldEntryOptions) captureAttachment(ctx *EffectContext) *entryAttachment {
	if e.attachedCount == 0 {
		return nil
	}
	saved := &entryAttachment{}
	if host, ok := e.host.resolvePermanent(ctx); ok && !host.PhasedOut {
		saved.host = capturedPermanent{host.ID(), host.incarnationID}
	}
	return saved
}

func (a *entryAttachment) resolve(g *Game, entrant *Permanent) (capturedPermanent, bool) {
	host := g.FindPermanent(a.host.id)
	if host != nil && !host.PhasedOut && host.incarnationID == a.host.incarnation && g.canAttachEntering(entrant, host) {
		return a.host, true
	}
	return capturedPermanent{}, !entrant.HasSubType("Aura")
}

func (g *Game) canAttachEntering(entrant, host *Permanent) bool {
	previous := g.zones.EnteringPermanent()
	g.zones.SetEnteringPermanent(entrant)
	defer g.zones.SetEnteringPermanent(previous)
	return g.canAttach(entrant, host)
}
