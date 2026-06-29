package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/calendar/v3"
)

const defaultRoomForwardDays = 14

func roomCommand() *cli.Command {
	return &cli.Command{
		Name:  "room",
		Usage: "find upcoming events without rooms and suggest available rooms on L28/L30",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "forwardDays",
				Value: defaultRoomForwardDays,
				Usage: "How many days forward to look for events without rooms",
			},
			&cli.IntFlag{
				Name:  "backDays",
				Value: 90,
				Usage: "How many days back to mine for known room resources",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			svc, err := newCalendarService(ctx, c.String("store"))
			if err != nil {
				return err
			}
			return runRoomTUI(ctx, svc, c.Int("backDays"), c.Int("forwardDays"))
		},
	}
}

type roomResource struct {
	email       string
	displayName string
}

func mineRoomResources(ctx context.Context, svc *calendar.Service, now time.Time, backDays int) ([]roomResource, error) {
	seen := make(map[string]string) // email -> displayName

	err := svc.Events.List(primaryCalendarID).
		SingleEvents(true).
		TimeMin(now.AddDate(0, 0, -backDays).Format(time.RFC3339)).
		TimeMax(now.Format(time.RFC3339)).
		Pages(ctx, func(events *calendar.Events) error {
			for _, event := range events.Items {
				for _, attendee := range event.Attendees {
					if !attendee.Resource {
						continue
					}
					if !strings.HasPrefix(attendee.DisplayName, roomDisplayPrefix) {
						continue
					}
					if !isLevel28or30(attendee.DisplayName) {
						continue
					}
					seen[attendee.Email] = attendee.DisplayName
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	rooms := make([]roomResource, 0, len(seen))
	for email, name := range seen {
		rooms = append(rooms, roomResource{email: email, displayName: name})
	}
	return rooms, nil
}

func isLevel28or30(displayName string) bool {
	// Room names: "SYD 363 George St-28-28.11 Name (cap) [type]"
	after, _ := strings.CutPrefix(displayName, roomDisplayPrefix)
	return strings.HasPrefix(after, "28-") || strings.HasPrefix(after, "30-")
}

func findEventsWithoutRooms(ctx context.Context, svc *calendar.Service, now time.Time, forwardDays int) ([]*calendar.Event, error) {
	var result []*calendar.Event

	err := svc.Events.List(primaryCalendarID).
		SingleEvents(true).
		OrderBy("startTime").
		TimeMin(now.Format(time.RFC3339)).
		TimeMax(now.AddDate(0, 0, forwardDays).Format(time.RFC3339)).
		Pages(ctx, func(events *calendar.Events) error {
			for _, event := range events.Items {
				if !relevantEvent(event) {
					continue
				}
				if roomEvent(event) == "💻" {
					result = append(result, event)
				}
			}
			return nil
		})

	return result, err
}

func availableRooms(ctx context.Context, svc *calendar.Service, timeMin, timeMax string, rooms []roomResource) ([]roomResource, error) {
	items := make([]*calendar.FreeBusyRequestItem, len(rooms))
	for i, r := range rooms {
		items[i] = &calendar.FreeBusyRequestItem{Id: r.email}
	}

	fb, err := svc.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: timeMin,
		TimeMax: timeMax,
		Items:   items,
	}).Do()
	if err != nil {
		return nil, err
	}

	var available []roomResource
	for _, r := range rooms {
		cal, ok := fb.Calendars[r.email]
		if !ok || len(cal.Busy) == 0 {
			available = append(available, r)
		}
	}
	return available, nil
}

func sortRooms(rooms []roomResource) {
	sort.Slice(rooms, func(i, j int) bool {
		zi := strings.Contains(rooms[i].displayName, "Zoom")
		zj := strings.Contains(rooms[j].displayName, "Zoom")
		if zi != zj {
			return zi
		}
		return formatRoomName(rooms[i].displayName) < formatRoomName(rooms[j].displayName)
	})
}

func bookRoom(ctx context.Context, svc *calendar.Service, event *calendar.Event, room roomResource) error {
	patch := &calendar.Event{
		Attendees: make([]*calendar.EventAttendee, 0, len(event.Attendees)+1),
	}
	for _, a := range event.Attendees {
		patch.Attendees = append(patch.Attendees, a)
	}
	patch.Attendees = append(patch.Attendees, &calendar.EventAttendee{
		Email:       room.email,
		DisplayName: room.displayName,
		Resource:    true,
	})
	_, err := svc.Events.Patch(primaryCalendarID, event.Id, patch).
		SendUpdates("none").
		Context(ctx).Do()
	return err
}

func formatRoomName(displayName string) string {
	output := roomPrefixRE.ReplaceAllString(displayName, "")
	output = roomSuffixRE.ReplaceAllString(output, "")
	output = collabRE.ReplaceAllString(output, "")
	return output
}

// TUI

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
