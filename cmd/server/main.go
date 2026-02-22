package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/mage/mage/internal/tui"
	"github.com/mage/mage/pkg/mage/interactive"
)

func main() {
	lobby := newLobby()

	port := os.Getenv("PORT")
	if port == "" {
		port = "2222"
	}

	if err := os.MkdirAll(".ssh", 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create .ssh dir: %v\n", err)
		os.Exit(1)
	}

	s, err := wish.NewServer(
		wish.WithAddress(":"+port),
		wish.WithHostKeyPath(".ssh/server_ed25519"),
		wish.WithMiddleware(gameMiddleware(lobby)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating server: %v\n", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("SSH server listening on :%s\n", port)

	go func() {
		if err := s.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	<-done
	fmt.Println("\nShutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = s.Shutdown(ctx)
}

func gameMiddleware(lobby *Lobby) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			username := sess.User()

			// Phase 1: Lobby
			lm := newLobbyModel(lobby, username)
			opts := append(bm.MakeOptions(sess), tea.WithAltScreen())

			result, _ := runProgram(sess, tea.NewProgram(lm, opts...))
			if result == nil {
				next(sess)
				return
			}

			finalLobby, ok := result.(lobbyModel)
			if !ok || finalLobby.session == nil {
				next(sess)
				return
			}

			playerSess := finalLobby.session

			// Phase 2: Game
			gameMdl := tui.NewModel(
				playerSess.ToGame,
				playerSess.FromGame,
				playerSess.ChoiceReqs,
				playerSess.ChoiceResps,
			)
			gameProgram := tea.NewProgram(gameMdl, opts...)
			runProgram(sess, gameProgram)

			// Signal the game loop that this player has disconnected by closing
			// the TUI→game channel. The game loop's readFrom detects ok=false.
			select {
			case playerSess.ToGame <- interactive.PriorityAction{Type: interactive.ActionPass}:
			default:
			}
			close(playerSess.ToGame)

			next(sess)
		}
	}
}

// runProgram runs a bubbletea program on an SSH session, forwarding PTY
// window resize events to the program.
func runProgram(sess ssh.Session, p *tea.Program) (tea.Model, error) {
	_, windowChanges, hasPty := sess.Pty()
	if hasPty {
		go func() {
			for w := range windowChanges {
				p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
			}
		}()
	}
	return p.Run()
}
