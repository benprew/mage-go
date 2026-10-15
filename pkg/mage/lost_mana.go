package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// LostManaBinding names the mana removed by one resolving instruction.
type LostManaBinding struct{ name string }

// BindLostMana declares a result that retains each removed mana unit.
func BindLostMana(name string) *LostManaBinding { return &LostManaBinding{name: name} }
func (b *LostManaBinding) bindingName() string  { return b.name }
func (b *LostManaBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "lost-mana-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

func cloneLostManaBindings(src map[*LostManaBinding][]Mana) map[*LostManaBinding][]Mana {
	if src == nil {
		return nil
	}
	dst := make(map[*LostManaBinding][]Mana, len(src))
	for binding, mana := range src {
		dst[binding] = slices.Clone(mana)
	}
	return dst
}

type lostManaEffect struct {
	player  PlayerRef
	result  *LostManaBinding
	restore bool
}

// LoseUnspentManaInto removes all unspent mana and saves its full values.
func LoseUnspentManaInto(player PlayerRef, result *LostManaBinding) ResolvingEffect {
	return &lostManaEffect{player: player, result: result}
}

// AddLostManaFor transfers saved mana without new mana production (CR 106.13).
func AddLostManaFor(player PlayerRef, result *LostManaBinding) ResolvingEffect {
	return &lostManaEffect{player: player, result: result, restore: true}
}
func (e *lostManaEffect) Text() string {
	if e.restore {
		return "add the mana lost this way"
	}
	return "lose all unspent mana and save it"
}
func (*lostManaEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *lostManaEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *lostManaEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "lost-mana-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"result", e.result}}}
	if e.result == nil {
		n.problem = "supply a lost mana binding"
	}
	if e.restore {
		n.bindingRead = e.result
	} else {
		n.bindingWrite = e.result
	}
	return n
}
func (e *lostManaEffect) Apply(ctx *EffectContext) error {
	state := &ctx.Game.resolution
	if !e.restore {
		if state.lostManaBindings == nil {
			state.lostManaBindings = make(map[*LostManaBinding][]Mana)
		}
		state.lostManaBindings[e.result] = nil
	}
	id, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(id) == nil {
		return nil
	}
	pool := ctx.Game.GetPlayer(id).ManaPool()
	if e.restore {
		pool.RestorePool(append(pool.SnapshotPool(), state.lostManaBindings[e.result]...))
	} else {
		state.lostManaBindings[e.result] = pool.SnapshotPool()
		pool.Clear()
	}
	return nil
}
