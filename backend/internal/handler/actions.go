package handler

import (
	"github.com/raHI-79/250-project/backend/internal/auth"
	"github.com/raHI-79/250-project/backend/internal/store"
)

const (
	ViewerRoleCR      = auth.RoleCR
	ViewerRoleTeacher = auth.RoleTeacher
)

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

func finalActions(item store.ScheduleItem, actions []string) []string {
	if item.Status == store.ItemCancelled {
		return actions
	}
	if item.InFinal {
		return append(actions, ActionRemoveFromFinal)
	}
	return append(actions, ActionAddToFinal)
}

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
