package handler

import (
	"github.com/raHI-79/250-project/backend/internal/store"
	"github.com/raHI-79/250-project/backend/internal/validate"
)

type courseTeacher struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type courseSlot struct {
	DayOfWeek int    `json:"day_of_week"`
	Day       string `json:"day"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Room      string `json:"room"`
	Type      string `json:"type"`
}

type crCourseItem struct {
	ID      int64         `json:"id"`
	Code    string        `json:"code"`
	Title   string        `json:"title"`
	Teacher courseTeacher `json:"teacher"`
	Slots   []courseSlot  `json:"slots"`
}

type crCoursesResponse struct {
	Items []crCourseItem `json:"items"`
}

func groupCourseSlots(slots []store.CourseSlotRow) []crCourseItem {
	items := make([]crCourseItem, 0)
	index := make(map[int64]int)
	for _, slot := range slots {
		position, seen := index[slot.CourseID]
		if !seen {
			position = len(items)
			index[slot.CourseID] = position
			items = append(items, crCourseItem{
				ID:    slot.CourseID,
				Code:  slot.CourseCode,
				Title: slot.CourseTitle,
				Teacher: courseTeacher{
					ID:     slot.TeacherID,
					UserID: slot.TeacherUserID,
					Name:   slot.TeacherName,
				},
				Slots: make([]courseSlot, 0, 1),
			})
		}
		items[position].Slots = append(items[position].Slots, courseSlot{
			DayOfWeek: slot.DayOfWeek,
			Day:       validate.DayName(slot.DayOfWeek),
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Room:      slot.Room,
			Type:      slot.Type,
		})
	}
	return items
}
