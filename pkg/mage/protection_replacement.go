package mage

type protectionDamageReplacement struct {
	replacementBase
	source DamageSource
}

func (r *protectionDamageReplacement) Matches(action Action, g GameReader) bool {
	damage, ok := action.(*DamageToCreatureAction)
	if !ok || damage.Amount() <= 0 {
		return false
	}
	p := g.FindPermanent(damage.PermanentID())
	return p != nil && !p.PhasedOut && protectedFromDamageSource(p, r.source)
}

func (*protectionDamageReplacement) Replace(Action, *Game) Action { return nil }
func (*protectionDamageReplacement) IsActive(GameReader) bool     { return true }
func (*protectionDamageReplacement) Text() string                 { return "Protection prevents damage" }
func (r *protectionDamageReplacement) Clone() ReplacementEffect   { cp := *r; return &cp }
