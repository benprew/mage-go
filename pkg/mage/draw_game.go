package mage

import "errors"

var errGameDrawn = errors.New("game ended in a draw")

// DrawGame ends the game without a winner or loser (CR 104.4c).
// Apply returns an internal stop signal through nested resolving bodies.
// ApplyEffect consumes that signal at the resolution boundary.
func DrawGame() ResolvingEffect { return &drawGameEffect{} }

type drawGameEffect struct{}

func (*drawGameEffect) Text() string                 { return "the game is a draw" }
func (*drawGameEffect) Properties() EffectProperties { return EffectProperties{} }
func (*drawGameEffect) Apply(ctx *EffectContext) error {
	if !ctx.Game.IsGameOver() {
		ctx.Game.drawn = true
	}
	return errGameDrawn
}
func (e *drawGameEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "draw-game", Text: e.Text()}
}
