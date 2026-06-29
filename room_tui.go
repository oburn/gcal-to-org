package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/api/calendar/v3"
)

type tuiState int

const (
	stateLoading tuiState = iota
	stateEventList
	stateLoadingRooms
	stateRoomList
	stateBooking
	stateError
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	zoomStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
)

type dataLoadedMsg struct {
	knownRooms []roomResource
	events     []*calendar.Event
	err        error
}

type roomsLoadedMsg struct {
	rooms []roomResource
	err   error
}

type bookedMsg struct {
	err error
}

type roomModel struct {
	ctx         context.Context
	svc         *calendar.Service
	backDays    int
	forwardDays int
	knownRooms  []roomResource

	state        tuiState
	spinnerFrame int

	events      []*calendar.Event
	eventCursor int

	selectedEvent *calendar.Event
	availRooms    []roomResource
	roomCursor    int

	err error
}

func (m roomModel) Init() tea.Cmd {
	return tea.Batch(spinnerTick(), m.loadData())
}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return t })
}

func (m roomModel) loadData() tea.Cmd {
	ctx := m.ctx
	svc := m.svc
	backDays := m.backDays
	forwardDays := m.forwardDays
	return func() tea.Msg {
		now := time.Now()
		known, err := mineRoomResources(ctx, svc, now, backDays)
		if err != nil {
			return dataLoadedMsg{err: err}
		}
		events, err := findEventsWithoutRooms(ctx, svc, now, forwardDays)
		return dataLoadedMsg{knownRooms: known, events: events, err: err}
	}
}

func (m roomModel) loadRooms() tea.Cmd {
	ctx := m.ctx
	svc := m.svc
	event := m.selectedEvent
	known := m.knownRooms
	return func() tea.Msg {
		rooms, err := availableRooms(ctx, svc, event.Start.DateTime, event.End.DateTime, known)
		if err != nil {
			return roomsLoadedMsg{err: err}
		}
		sortRooms(rooms)
		return roomsLoadedMsg{rooms: rooms}
	}
}

func (m roomModel) book(room roomResource) tea.Cmd {
	ctx := m.ctx
	svc := m.svc
	event := m.selectedEvent
	return func() tea.Msg {
		err := bookRoom(ctx, svc, event, room)
		return bookedMsg{err: err}
	}
}

func (m roomModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case time.Time:
		if m.state == stateLoading || m.state == stateLoadingRooms || m.state == stateBooking {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, spinnerTick()
		}
		return m, nil

	case dataLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.knownRooms = msg.knownRooms
		m.events = msg.events
		m.state = stateEventList
		m.eventCursor = 0
		return m, nil

	case roomsLoadedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.availRooms = msg.rooms
		m.state = stateRoomList
		m.roomCursor = 0
		return m, nil

	case bookedMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		remaining := make([]*calendar.Event, 0, len(m.events)-1)
		for _, e := range m.events {
			if e.Id != m.selectedEvent.Id {
				remaining = append(remaining, e)
			}
		}
		m.events = remaining
		m.selectedEvent = nil
		m.state = stateEventList
		if m.eventCursor >= len(m.events) && m.eventCursor > 0 {
			m.eventCursor--
		}
		return m, nil
	}

	return m, nil
}

func (m roomModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateEventList:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.eventCursor < len(m.events)-1 {
				m.eventCursor++
			}
		case "k", "up":
			if m.eventCursor > 0 {
				m.eventCursor--
			}
		case "enter":
			if len(m.events) > 0 {
				m.selectedEvent = m.events[m.eventCursor]
				m.state = stateLoadingRooms
				m.spinnerFrame = 0
				return m, tea.Batch(spinnerTick(), m.loadRooms())
			}
		}

	case stateRoomList:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.state = stateEventList
			m.selectedEvent = nil
		case "j", "down":
			if m.roomCursor < len(m.availRooms)-1 {
				m.roomCursor++
			}
		case "k", "up":
			if m.roomCursor > 0 {
				m.roomCursor--
			}
		case "enter":
			if len(m.availRooms) > 0 {
				room := m.availRooms[m.roomCursor]
				m.state = stateBooking
				m.spinnerFrame = 0
				return m, tea.Batch(spinnerTick(), m.book(room))
			}
		}

	case stateError:
		return m, tea.Quit
	}

	return m, nil
}

func (m roomModel) View() string {
	var b strings.Builder
	switch m.state {
	case stateLoading:
		fmt.Fprintf(&b, "\n  %s Loading events…\n", spinnerFrames[m.spinnerFrame])

	case stateEventList:
		b.WriteString(titleStyle.Render("Events without rooms") + "\n\n")
		if len(m.events) == 0 {
			b.WriteString(dimStyle.Render("  All events have rooms booked.") + "\n")
		}
		for i, e := range m.events {
			start, _ := time.Parse(time.RFC3339, e.Start.DateTime)
			end, _ := time.Parse(time.RFC3339, e.End.DateTime)
			label := fmt.Sprintf("%s %s-%s  %s",
				start.Format(dateLayout),
				start.Format(timeLayout),
				end.Format(timeLayout),
				e.Summary,
			)
			if i == m.eventCursor {
				b.WriteString(selectedStyle.Render("> "+label) + "\n")
			} else {
				b.WriteString("  " + label + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("j/k: move  enter: pick  q: quit") + "\n")

	case stateLoadingRooms:
		start, _ := time.Parse(time.RFC3339, m.selectedEvent.Start.DateTime)
		fmt.Fprintf(&b, "\n  %s Loading rooms for %s at %s…\n",
			spinnerFrames[m.spinnerFrame],
			m.selectedEvent.Summary,
			start.Format(timeLayout),
		)

	case stateRoomList:
		start, _ := time.Parse(time.RFC3339, m.selectedEvent.Start.DateTime)
		end, _ := time.Parse(time.RFC3339, m.selectedEvent.End.DateTime)
		header := fmt.Sprintf("Available rooms — %s %s-%s  %s",
			start.Format(dateLayout),
			start.Format(timeLayout),
			end.Format(timeLayout),
			m.selectedEvent.Summary,
		)
		b.WriteString(titleStyle.Render(header) + "\n\n")
		if len(m.availRooms) == 0 {
			b.WriteString(dimStyle.Render("  No rooms available on L28/L30.") + "\n")
		}
		for i, r := range m.availRooms {
			name := formatRoomName(r.displayName)
			suffix := ""
			if strings.Contains(r.displayName, "Zoom") {
				suffix = "  " + zoomStyle.Render("[Zoom]")
			}
			if i == m.roomCursor {
				b.WriteString(selectedStyle.Render("> "+name) + suffix + "\n")
			} else {
				b.WriteString("  " + name + suffix + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("j/k: move  enter: book  esc: back  q: quit") + "\n")

	case stateBooking:
		name := ""
		if len(m.availRooms) > m.roomCursor {
			name = formatRoomName(m.availRooms[m.roomCursor].displayName)
		}
		fmt.Fprintf(&b, "\n  %s Booking %s…\n", spinnerFrames[m.spinnerFrame], name)

	case stateError:
		b.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
		b.WriteString(dimStyle.Render("Press any key to quit.") + "\n")
	}
	return b.String()
}

func runRoomTUI(ctx context.Context, svc *calendar.Service, backDays, forwardDays int) error {
	m := roomModel{
		ctx:         ctx,
		svc:         svc,
		backDays:    backDays,
		forwardDays: forwardDays,
		state:       stateLoading,
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
