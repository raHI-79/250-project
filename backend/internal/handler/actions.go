package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
)

// ViewerRoleCR and ViewerRoleTeacher name the two roles that may change a schedule item.
// A student only reads, so AllowedActions answers nothing for every other role.
const (
	ViewerRoleCR      = auth.RoleCR
	ViewerRoleTeacher = auth.RoleTeacher
)

// AllowedActions is the one place that decides what a role may do with a schedule item.
// It takes the role, the item and whether the day has already passed, and the handlers
// put the answer in allowed_actions so that the website and the app only ever draw the
// buttons the server would accept.
//
// Two rules hold for every row, whatever the status:
//   - an item that is not in a final routine never offers "Remove from final", and an
//     item that is in one never offers "Add to final";
//   - a cancelled item offers neither, because its place in the routine is settled.
func AllowedActions(role string, item store.ScheduleItem, past bool) []string {
	if past {
		return []string{}
	}
	switch role {
	case ViewerRoleCR:
		return crActions(item)
	case ViewerRoleTeacher:
		return teacherActions(item)
	default:
		return []string{}
	}
}

// finalActions appends the one final-routine button the item deserves. A cancelled item
// gets none: it stays where it is until the class representative publishes again.
func finalActions(item store.ScheduleItem, actions []string) []string {
	if item.Status == store.ItemCancelled {
		return actions
	}
	if item.InFinal {
		return append(actions, ActionRemoveFromFinal)
	}
	return append(actions, ActionAddToFinal)
}

// linkActions adds the meeting link button. Only an online item has one to change, and a
// cancelled item keeps the link it had.
func linkActions(item store.ScheduleItem, actions []string) []string {
	if item.Status == store.ItemCancelled || item.Mode != store.ModeOnline {
		return actions
	}
	return append(actions, ActionEditLink)
}

func teacherActions(item store.ScheduleItem) []string {
	if item.Status == store.ItemCancelled {
		return []string{}
	}
	if item.Status == store.ItemCRAsk {
		return []string{ActionAccept, ActionDecline}
	}
	// A teacher may change the slot of anything it owns that has not started yet. The
	// store refuses one that has already begun, and the views hide the button once the
	// start time has passed.
	actions := make([]string, 0, 5)
	switch {
	case item.Kind == store.KindExam || item.Origin != store.OriginFixed:
		actions = append(actions, ActionCancel)
	default:
		switch item.Status {
		case store.ItemAwaiting, store.ItemNoResponse:
			actions = append(actions, ActionConfirm, ActionCancel, ActionReschedule)
		case store.ItemTeacherAsk:
			actions = append(actions, ActionConfirm, ActionCancel)
		case store.ItemConfirmed, store.ItemRescheduled, store.ItemDeclined:
			actions = append(actions, ActionCancel, ActionReschedule)
		default:
			actions = append(actions, ActionCancel)
		}
	}
	actions = append(actions, ActionEdit)
	return linkActions(item, actions)
}

func crActions(item store.ScheduleItem) []string {
	if item.Status == store.ItemCancelled {
		return []string{}
	}
	actions := make([]string, 0, 5)
	switch {
	case item.Origin == store.OriginCR:
		actions = append(actions, ActionEdit, ActionCancel, ActionDelete)
	case item.Status == store.ItemTeacherAsk:
		actions = append(actions, ActionSetSlot, ActionCancel)
	case item.Origin == store.OriginTeacherExam:
		actions = append(actions, ActionSetTime)
	}
	if item.Origin == store.OriginFixed {
		switch item.Status {
		case store.ItemAwaiting, store.ItemNoResponse, store.ItemConfirmed, store.ItemDeclined:
			actions = append(actions, ActionReschedule)
		}
		switch item.Status {
		case store.ItemNoResponse, store.ItemDeclined:
			actions = append(actions, ActionCancel)
		}
	}
	return linkActions(item, finalActions(item, actions))
}
