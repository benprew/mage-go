package mage

// SacrificeChosenBy makes a required choice at resolution for a sacrificing player.
// The chooser can differ from that player. The shared sacrifice service moves it.
func SacrificeChosenBy(player, chooser PlayerRef, subjects PermanentSet) ResolvingEffect {
	return &chosenSacrificeEffect{player: player, chooser: chooser, subjects: subjects, binding: BindPermanent("permanent to sacrifice")}
}

type chosenSacrificeEffect struct {
	player   PlayerRef
	chooser  PlayerRef
	subjects PermanentSet
	binding  *PermanentBinding
}

func (*chosenSacrificeEffect) Text() string { return "sacrifice a chosen permanent" }
func (*chosenSacrificeEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *chosenSacrificeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"chooser", e.chooser}, {"subjects", e.subjects}}}
}
func (e *chosenSacrificeEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || playerRefReadsSourcePermanent(e.chooser) || permanentSetReadsSourcePermanent(e.subjects)
}
func (e *chosenSacrificeEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	subjects := FilterPermanentSet(e.subjects, ControlledBy(id))
	if err := ChoosePermanent(e.chooser, subjects, e.binding).Apply(ctx); err != nil {
		return err
	}
	p, ok := From(e.binding).resolvePermanent(ctx)
	if ok && p.ControllerID() == id {
		ctx.Game.controllersSacrificePermanents([]*Permanent{p})
	}
	return nil
}
