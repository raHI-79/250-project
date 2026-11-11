package handler

import (
	"slices"
	"strings"
	"testing"

	"github.com/raHI-79/250-project/backend/internal/store"
)

var everyStatus = []string{
	store.ItemAwaiting,
	store.ItemNoResponse,
	store.ItemConfirmed,
	store.ItemCancelled,
	store.ItemTeacherAsk,
	store.ItemCRAsk,
	store.ItemDeclined,
	store.ItemRescheduled,
	store.ItemScheduled,
}

var everyOrigin = []string{store.OriginFixed, store.OriginCR, store.OriginTeacherExam}

func joinedActions(role string, item store.ScheduleItem) string {
	got := AllowedActions(role, item, false)
	slices.Sort(got)
	return strings.Join(got, ",")
}

func TestTheFinalButtonsFollowInFinalForEveryStatus(t *testing.T) {
	for _, origin := range everyOrigin {
		for _, status := range everyStatus {
			for _, inFinal := range []bool{false, true} {
				for _, role := range []string{ViewerRoleCR, ViewerRoleTeacher} {
					item := store.ScheduleItem{Origin: origin, Status: status, InFinal: inFinal}
					got := AllowedActions(role, item, false)
					adds := slices.Contains(got, ActionAddToFinal)
					removes := slices.Contains(got, ActionRemoveFromFinal)
					name := role + "/" + origin + "/" + status
					switch {
					case status == store.ItemCancelled:
						if adds || removes {
							t.Errorf("%s cancelled, in_final=%v offers %v, want no final button",
								name, inFinal, got)
						}
					case role == ViewerRoleTeacher:
						if adds || removes {
							t.Errorf("%s offers %v to a teacher, want no final button", name, got)
						}
					case inFinal:
						if adds {
							t.Errorf("%s in final offers Add to final", name)
						}
						if !removes {
							t.Errorf("%s in final offers %v, want Remove from final", name, got)
						}
					default:
						if removes {
							t.Errorf("%s not in final offers Remove from final", name)
						}
						if !adds {
							t.Errorf("%s not in final offers %v, want Add to final", name, got)
						}
					}
				}
			}
		}
	}
}

func TestAllowedActionsForEveryStatusAndRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		item store.ScheduleItem
		want string
	}{
		{
			name: "the class representative on a fixed class nobody has answered",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemAwaiting},
			want: "add_to_final,request_reschedule",
		},
		{
			name: "the class representative on a fixed class already in the final routine",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemAwaiting, InFinal: true},
			want: "remove_from_final,request_reschedule",
		},
		{
			name: "the class representative on a fixed class the teacher never answered",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemNoResponse},
			want: "add_to_final,cancel,request_reschedule",
		},
		{
			name: "the class representative on a confirmed fixed class",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemConfirmed},
			want: "add_to_final,request_reschedule",
		},
		{
			name: "the class representative on a refused reschedule",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemDeclined},
			want: "add_to_final,cancel,request_reschedule",
		},
		{
			name: "the class representative on a slot the teacher asked to move",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemTeacherAsk},
			want: "add_to_final,cancel,set_slot",
		},
		{
			name: "the class representative on its own class",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemScheduled},
			want: "add_to_final,cancel,delete,edit",
		},
		{
			name: "the class representative on an exam a teacher created",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginTeacherExam, Kind: store.KindExam,
				Status: store.ItemScheduled},
			want: "add_to_final,set_time",
		},
		{
			name: "the class representative on a cancelled class",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCancelled},
			want: "",
		},
		{
			name: "the class representative on a cancelled class still in the final routine",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCancelled, InFinal: true},
			want: "",
		},
		{
			name: "the teacher on a fixed class nobody has answered",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemAwaiting},
			want: "cancel,confirm,edit,request_reschedule",
		},
		{
			name: "the teacher on a fixed class the deadline passed on",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemNoResponse},
			want: "cancel,confirm,edit,request_reschedule",
		},
		{
			name: "the teacher on its own reschedule request",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemTeacherAsk},
			want: "cancel,confirm,edit",
		},
		{
			name: "the teacher on a slot the class representative proposed",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCRAsk},
			want: "accept_reschedule,decline_reschedule",
		},
		{
			name: "the teacher on a confirmed fixed class",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemConfirmed},
			want: "cancel,edit,request_reschedule",
		},
		{
			name: "the teacher on a moved fixed class",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemRescheduled},
			want: "cancel,edit,request_reschedule",
		},
		{
			name: "the teacher on a class the class representative created",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemScheduled},
			want: "cancel,edit",
		},
		{
			name: "the teacher on its own exam",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginTeacherExam, Kind: store.KindExam,
				Status: store.ItemScheduled},
			want: "cancel,edit",
		},
		{
			name: "the teacher on a cancelled class",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemCancelled},
			want: "",
		},
		{
			name: "an online class offers the meeting link to both roles",
			role: ViewerRoleTeacher,
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemConfirmed,
				Mode: store.ModeOnline},
			want: "cancel,edit,edit_link,request_reschedule",
		},
		{
			name: "an online class the representative created",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemScheduled,
				Mode: store.ModeOnline},
			want: "add_to_final,cancel,delete,edit,edit_link",
		},
		{
			name: "a cancelled online class keeps the link it had",
			role: ViewerRoleCR,
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemCancelled,
				Mode: store.ModeOnline},
			want: "",
		},
		{
			name: "a student may change nothing",
			role: "student",
			item: store.ScheduleItem{Origin: store.OriginFixed, Status: store.ItemAwaiting},
			want: "",
		},
		{
			name: "an office admin may change nothing here",
			role: "admin",
			item: store.ScheduleItem{Origin: store.OriginCR, Status: store.ItemScheduled},
			want: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := joinedActions(test.role, test.item); got != test.want {
				t.Errorf("allowed actions = %q, want %q", got, test.want)
			}
		})
	}
}

func TestADayThatHasPassedOffersNothing(t *testing.T) {
	for _, role := range []string{ViewerRoleCR, ViewerRoleTeacher, "student"} {
		for _, status := range everyStatus {
			for _, inFinal := range []bool{false, true} {
				item := store.ScheduleItem{Origin: store.OriginFixed, Status: status, InFinal: inFinal}
				if got := AllowedActions(role, item, true); len(got) != 0 {
					t.Errorf("%s on a past %s item offers %v, want nothing", role, status, got)
				}
			}
		}
	}
}

func TestAllowedActionsNeverReturnsNil(t *testing.T) {
	item := store.ScheduleItem{Origin: "something new", Status: "something new"}
	for _, role := range []string{ViewerRoleCR, ViewerRoleTeacher} {
		if got := AllowedActions(role, item, false); got == nil {
			t.Errorf("%s on an unknown status returned nil", role)
		}
	}
}
