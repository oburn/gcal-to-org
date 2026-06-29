package main

import (
	"context"
	"sort"
	"strings"
	"time"

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
