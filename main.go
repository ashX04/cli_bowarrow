package main

import (
	"fmt"
	"math/rand"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Game states
const (
	playing = iota
	gameOver
)

// Balloon represents a target
type Balloon struct {
	x, y   int
	popped bool
	symbol []string // Changed to string slice for multi-line art
	color  lipgloss.Color
	width  int
	height int
}

// Arrow represents the player's projectile
type Arrow struct {
	x, y   int
	active bool
	symbol string
}

// Model represents the game state
type Model struct {
	width, height int
	archer        int // archer's vertical position
	arrows        []Arrow
	balloons      []Balloon
	score         int
	state         int
	timer         int
	minBalloonX   int // Add this field
	maxBalloonX   int // Add this field
}

// Initialize the game
func initialModel() Model {
	width := 80
	return Model{
		width:       width - 2, // Account for padding
		height:      20,
		archer:      10,
		arrows:      make([]Arrow, 0),
		balloons:    make([]Balloon, 0),
		state:       playing,
		timer:       0,
		minBalloonX: (width - 2) / 2, // Account for padding
		maxBalloonX: width - 7,       // Account for padding and balloon width
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tick(), spawnBalloon())
}

// Update handles game logic
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.archer > 0 {
				m.archer--
			}
		case "down":
			if m.archer < m.height-1 {
				m.archer++
			}
		case " ": // Space to shoot
			if len(m.arrows) < 3 { // Limit arrows
				m.arrows = append(m.arrows, Arrow{
					x:      2,
					y:      m.archer,
					active: true,
					symbol: "═>", // Longer arrow symbol
				})
			}
		}

	case spawnMsg:
		balloon := Balloon(msg)
		m.balloons = append(m.balloons, balloon)
		return m, nil

	case tickMsg:
		// Update arrows
		for i := range m.arrows {
			if m.arrows[i].active {
				m.arrows[i].x += 2
				if m.arrows[i].x >= m.width {
					m.arrows[i].active = false
				}
			}
		}

		// Update balloons
		for i := range m.balloons {
			if !m.balloons[i].popped {
				// Move upward with slight horizontal wobble
				m.balloons[i].y--
				m.balloons[i].x += rand.Intn(3) - 1

				// Keep within bounds
				if m.balloons[i].x < m.minBalloonX {
					m.balloons[i].x = m.minBalloonX
				}
				if m.balloons[i].x > m.maxBalloonX {
					m.balloons[i].x = m.maxBalloonX
				}

				// Remove if it reaches the top
				if m.balloons[i].y < 0 {
					m.balloons[i].popped = true
				}
			}
		}

		// Check collisions
		for i := range m.arrows {
			if m.arrows[i].active {
				for j := range m.balloons {
					if !m.balloons[j].popped &&
						m.arrows[i].x+4 >= m.balloons[j].x &&
						m.arrows[i].x <= m.balloons[j].x+m.balloons[j].width &&
						m.arrows[i].y >= m.balloons[j].y &&
						m.arrows[i].y <= m.balloons[j].y+m.balloons[j].height {
						m.balloons[j].popped = true
						m.arrows[i].active = false
						m.score++
						// Replace balloon with explosion
						m.balloons[j].symbol = []string{
							"  \\|/  ",
							"  /|\\  ",
							"   *   ",
						}
						m.balloons[j].height = 3
						m.balloons[j].width = 7
					}
				}
			}
		}

		// Clean up inactive elements
		m.arrows = filterActiveArrows(m.arrows)
		m.balloons = filterActiveBalloons(m.balloons)

		return m, tea.Batch(tick(), spawnBalloon())
	}

	return m, nil
}

// View renders the game
func (m Model) View() string {
	// Create game board
	board := make([][]string, m.height)
	for i := range board {
		board[i] = make([]string, m.width)
		for j := range board[i] {
			board[i][j] = " "
		}
	}

	// Draw archer
	archerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	bowSymbol := "|)"
	board[m.archer][0] = archerStyle.Render(bowSymbol)

	// Draw arrows
	for _, arrow := range m.arrows {
		if arrow.active && arrow.x < m.width {
			board[arrow.y][arrow.x] = arrow.symbol
		}
	}

	// Draw balloons
	for _, balloon := range m.balloons {
		if !balloon.popped {
			balloonStyle := lipgloss.NewStyle().Foreground(balloon.color)
			// Draw each line of the balloon
			for i, line := range balloon.symbol {
				if balloon.y+i >= 0 && balloon.y+i < m.height {
					for j, char := range line {
						if balloon.x+j < m.width {
							board[balloon.y+i][balloon.x+j] = balloonStyle.Render(string(char))
						}
					}
				}
			}
		}
	}

	// Render board with border
	var gameArea string
	for i := range board {
		row := ""
		for j := range board[i] {
			row += board[i][j]
		}
		gameArea += row + "\n"
	}

	// Create border styles
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Light blue border
		Padding(0, 1).                          // Add some padding
		Width(m.width + 2).                     // Account for padding
		Align(lipgloss.Center)

	// Create title style
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")). // Pink color
		Bold(true).
		MarginBottom(1)

	// Create score style
	scoreStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		MarginTop(1)

	// Create controls style
	controlsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")). // Subtle gray
		MarginTop(1)

	// Combine all elements
	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🎯 Balloon Archer 🎈"),
		borderStyle.Render(gameArea),
		scoreStyle.Render(fmt.Sprintf("Score: %d", m.score)),
		controlsStyle.Render("Controls: ↑/↓ to move, SPACE to shoot, q to quit"),
	)
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type spawnMsg Balloon

func spawnBalloon() tea.Cmd {
	return func() tea.Msg {
		if rand.Float64() < 0.1 {
			balloonArts := [][]string{
				{
					"  .-^^-.",
					" /      \\",
					"|        |",
					" \\      /",
					"  `----´",
					"    ||   ",
				},
				{
					"  .===.",
					" (     )",
					"|       |",
					" (     )",
					"  `---´",
					"   ||  ",
				},
				{
					"  _____",
					" /     \\",
					"|   ○   |",
					" \\     /",
					"  ‾‾‾‾‾",
					"   ||   ",
				},
				{
					"  .===.",
					" /     \\",
					"|   •   |",
					" \\     /",
					"  `---´",
					"   ||   ",
				},
			}

			balloonColors := []lipgloss.Color{
				"213", // Pink
				"204", // Red
				"39",  // Blue
				"48",  // Green
			}

			symbolIndex := rand.Intn(len(balloonArts))
			selectedBalloon := balloonArts[symbolIndex]

			// Calculate balloon dimensions
			width := len(selectedBalloon[0])
			height := len(selectedBalloon)

			screenWidth := 80
			minX := screenWidth / 2
			maxX := screenWidth - width - 2
			spawnX := minX + rand.Intn(maxX-minX)

			return spawnMsg(Balloon{
				x:      spawnX,
				y:      19,
				popped: false,
				symbol: selectedBalloon,
				color:  balloonColors[symbolIndex],
				width:  width,
				height: height,
			})
		}
		return nil
	}
}

func filterActiveArrows(arrows []Arrow) []Arrow {
	active := make([]Arrow, 0)
	for _, arrow := range arrows {
		if arrow.active {
			active = append(active, arrow)
		}
	}
	return active
}

func filterActiveBalloons(balloons []Balloon) []Balloon {
	active := make([]Balloon, 0)
	for _, balloon := range balloons {
		if !balloon.popped {
			active = append(active, balloon)
		}
	}
	return active
}

func main() {
	rand.Seed(time.Now().UnixNano())

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Error running program: %v", err)
		return
	}
}
