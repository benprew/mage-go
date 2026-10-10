package mage

// SacrificeChosenBy makes a required choice at resolution for a sacrificing player.
// The chooser can differ from that player. The shared sacrifice service moves it.
func SacrificeChosenBy(player, chooser PlayerRef, subjects PermanentSet, options ...SacrificeResultOption) ResolvingEffect {
	binding := BindPermanent("permanent to sacrifice")
	return &chosenSacrificeEffect{player: player, chooser: chooser, subjects: subjects, binding: binding,
		sacrifice: Sacrifice(From(binding), options...).(*sacrificeEffect)}
}

type chosenSacrificeEffect struct {
	player    PlayerRef
	chooser   PlayerRef
	subjects  PermanentSet
	binding   *PermanentBinding
	sacrifice *sacrificeEffect
}

func (*chosenSacrificeEffect) Text() string { return "sacrifice a chosen permanent" }
func (*chosenSacrificeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *chosenSacrificeEffect) inspectDefinition() definitionNode {
	n := e.sacrifice.inspectDefinition()
	n.Text = e.Text()
	n.children = append([]definitionEdge{{"player", e.player}, {"chooser", e.chooser}, {"subjects", e.subjects}}, n.children[1:]...)
	return n
}
func (e *chosenSacrificeEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || playerRefReadsSourcePermanent(e.chooser) || permanentSetReadsSourcePermanent(e.subjects)
}
func (e *chosenSacrificeEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	sacrificeCtx := *ctx
	if !ok {
		delete(ctx.Game.resolution.permanentBindings, e.binding)
		return e.sacrifice.Apply(&sacrificeCtx)
	}
	sacrificeCtx.Controller = id
	subjects := FilterPermanentSet(e.subjects, ControlledBy(id))
	if err := ChoosePermanent(e.chooser, subjects, e.binding).Apply(ctx); err != nil {
		return err
	}
	var err error
	ctx.Game.withZoneChangeBatch(func() {
		err = e.sacrifice.Apply(&sacrificeCtx)
	})
	return err
}
