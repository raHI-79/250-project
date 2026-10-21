package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

type RoomSlot struct {
	Room       string
	ItemDate   string
	StartTime  string
	EndTime    string
	Batch      string
	Section    string
	CourseCode string
	Status     string
}

type Publication struct {
	ID          int64
	Batch       string
	PublishedBy int64
	PublishedAt time.Time
}

type IncompleteItemError struct {
	Item ScheduleItem
}

func (e *IncompleteItemError) Error() string {
	return fmt.Sprintf("store: the item %s on %s still needs a date, a time and a room or a link",
		e.Item.CourseCode, e.Item.ItemDate)
}

const (
	listRoomAvailabilitySQL = `SELECT rooms.room, to_char(day::date, 'YYYY-MM-DD'),
COALESCE(to_char(i.start_time, 'HH24:MI'), ''), COALESCE(to_char(i.end_time, 'HH24:MI'), ''),
COALESCE(i.batch, ''), COALESCE(i.section, ''), COALESCE(c.code, ''), COALESCE(i.status, '')
FROM (SELECT DISTINCT room FROM routines) rooms
CROSS JOIN generate_series($1::date, $2::date, interval '1 day') AS day
LEFT JOIN schedule_items i ON i.room = rooms.room AND i.item_date = day::date
     AND i.mode = 'room' AND i.status <> 'cancelled' AND i.start_time IS NOT NULL
LEFT JOIN courses c ON c.id = i.course_id
ORDER BY rooms.room, day, i.start_time NULLS LAST, i.id`
	listBatchItemsSQL = selectItem + `
WHERE i.batch = $1 AND i.item_date >= $2::date
ORDER BY i.item_date, i.start_time NULLS LAST, i.section, i.id`
	listFinalItemsSQL = selectItem + `
WHERE i.batch = $1 AND (i.in_final OR pi.item_id IS NOT NULL)
ORDER BY i.item_date, i.start_time NULLS LAST, i.section, i.id`
	listItemsByIDsSQL = selectItem + `
WHERE i.batch = $1 AND i.id = ANY($2::bigint[])
ORDER BY i.item_date, i.start_time NULLS LAST, i.section, i.id`
	crAskRescheduleSQL = `UPDATE schedule_items
SET status = 'cr_reschedule_request',
    prior_status = $2,
    proposed_date = $3::date,
    proposed_start = $4::time,
    proposed_end = $5::time,
    proposed_mode = $6,
    proposed_room = $7,
    proposed_link = $8,
    updated_at = now()
WHERE id = $1`
	patchItemSQL = `UPDATE schedule_items
SET item_date = COALESCE($2::date, item_date),
    start_time = COALESCE($3::time, start_time),
    end_time = COALESCE($4::time, end_time),
    mode = COALESCE($5, mode),
    room = COALESCE($6, room),
    meeting_link = COALESCE($7, meeting_link),
    note = COALESCE($8, note),
    syllabus = COALESCE($9, syllabus),
    instructions = COALESCE($10, instructions),
    status = $11,
    teacher_message = CASE WHEN $11 = 'rescheduled' THEN '' ELSE teacher_message END,
    prior_status = '',
    updated_at = now()
WHERE id = $1`
	setItemFinalSQL = `UPDATE schedule_items
SET in_final = $2, updated_at = now()
WHERE id = $1 AND batch = $3`
	insertPublicationSQL = `INSERT INTO publications (batch, published_by)
VALUES ($1, NULLIF($2::bigint, 0))
RETURNING id, batch, COALESCE(published_by, 0), published_at`
	upsertPublishedItemSQL = `INSERT INTO published_items
(item_id, publication_id, batch, section, item_date, start_time, end_time, course_code, course_title,
teacher_name, kind, mode, room, meeting_link, note, syllabus, instructions, status, published_at)
VALUES ($1, $2, $3, $4, $5::date, NULLIF($6::text, '')::time, NULLIF($7::text, '')::time, $8, $9,
$10, $11, $12, $13, $14, $15, $16, $17, $18, now())
ON CONFLICT (item_id) DO UPDATE SET
    publication_id = EXCLUDED.publication_id,
    section = EXCLUDED.section,
    item_date = EXCLUDED.item_date,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    course_code = EXCLUDED.course_code,
    course_title = EXCLUDED.course_title,
    teacher_name = EXCLUDED.teacher_name,
    kind = EXCLUDED.kind,
    mode = EXCLUDED.mode,
    room = EXCLUDED.room,
    meeting_link = EXCLUDED.meeting_link,
    note = EXCLUDED.note,
    syllabus = EXCLUDED.syllabus,
    instructions = EXCLUDED.instructions,
    status = EXCLUDED.status,
    published_at = now()`
	syncPublishedStatusSQL = `UPDATE published_items SET status = $2, published_at = now() WHERE item_id = $1`
	syncPublishedLinkSQL   = `UPDATE published_items SET meeting_link = $2, published_at = now() WHERE item_id = $1`
)

func (s *Store) ListRoomAvailability(ctx context.Context, from, to string) ([]RoomSlot, error) {
	rows, err := s.pool.Query(ctx, listRoomAvailabilitySQL, from, to)
	if err != nil {
		return nil, fmt.Errorf("store: could not list the room availability from %s: %w", from, err)
	}
	defer rows.Close()
	slots := make([]RoomSlot, 0)
	for rows.Next() {
		var slot RoomSlot
		err := rows.Scan(
			&slot.Room,
			&slot.ItemDate,
			&slot.StartTime,
			&slot.EndTime,
			&slot.Batch,
			&slot.Section,
			&slot.CourseCode,
			&slot.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("store: could not scan a room slot: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: could not read the room availability: %w", err)
	}
	return slots, nil
}

func (s *Store) CreateCRItem(ctx context.Context, item NewScheduleItem) (ScheduleItem, error) {
	item.Origin = OriginCR
	item.Status = ItemScheduled
	return s.insertItem(ctx, item)
}

func (s *Store) ListBatchItems(ctx context.Context, batch, from string) ([]ScheduleItem, error) {
	return s.items(ctx, listBatchItemsSQL, batch, from)
}

func (s *Store) ListFinalItems(ctx context.Context, batch string) ([]ScheduleItem, error) {
	return s.items(ctx, listFinalItemsSQL, batch)
}

func (s *Store) ListItemsByIDs(ctx context.Context, batch string, ids []int64) ([]ScheduleItem, error) {
	return s.items(ctx, listItemsByIDsSQL, batch, ids)
}

func ownedByBatch(item ScheduleItem, batch string) error {
	if item.Batch != batch {
		return ErrItemNotFound
	}
	return nil
}

func (s *Store) CRRequestReschedule(ctx context.Context, itemID int64, batch string,
	proposal Proposal) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByBatch(item, batch); err != nil {
			return err
		}
		if item.Origin != OriginFixed {
			return ErrItemState
		}
		switch item.Status {
		case ItemAwaiting, ItemNoResponse, ItemConfirmed, ItemDeclined:
		default:
			return ErrItemState
		}
		err := checkSlot(ctx, tx, slotRequest{
			ExcludeID: item.ID,
			Batch:     item.Batch,
			Section:   item.Section,
			ItemDate:  proposal.ItemDate,
			StartTime: proposal.StartTime,
			EndTime:   proposal.EndTime,
			Mode:      proposal.Mode,
			Room:      proposal.Room,
		})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, crAskRescheduleSQL, item.ID, item.Status, proposal.ItemDate,
			proposal.StartTime, proposal.EndTime, proposal.Mode, proposal.Room, proposal.MeetingLink)
		if err != nil {
			return fmt.Errorf("store: could not ask for a reschedule of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

func patchAllowed(item ScheduleItem, patch ItemPatch) error {
	slotChange := patch.ItemDate != nil || patch.StartTime != nil || patch.EndTime != nil ||
		patch.Mode != nil || patch.Room != nil
	switch {
	case item.Origin == OriginCR:
		return nil
	case item.Status == ItemTeacherAsk:
		if patch.Syllabus != nil || patch.Instructions != nil {
			return ErrItemState
		}
		return nil
	case item.Origin == OriginTeacherExam:
		if patch.ItemDate != nil || patch.Syllabus != nil || patch.Instructions != nil {
			return ErrItemState
		}
		return nil
	case !slotChange && patch.Syllabus == nil && patch.Instructions == nil:
		return nil
	default:
		return ErrItemState
	}
}

func (s *Store) UpdateItemAsCR(ctx context.Context, itemID int64, batch string,
	patch ItemPatch) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByBatch(item, batch); err != nil {
			return err
		}
		if item.Status == ItemCancelled {
			return ErrItemState
		}
		if err := patchAllowed(item, patch); err != nil {
			return err
		}
		merged := item
		if patch.ItemDate != nil {
			merged.ItemDate = *patch.ItemDate
		}
		if patch.StartTime != nil {
			merged.StartTime = *patch.StartTime
		}
		if patch.EndTime != nil {
			merged.EndTime = *patch.EndTime
		}
		if patch.Mode != nil {
			merged.Mode = *patch.Mode
		}
		if patch.Room != nil {
			merged.Room = *patch.Room
		}
		err := checkSlot(ctx, tx, slotRequest{
			ExcludeID: item.ID,
			Batch:     merged.Batch,
			Section:   merged.Section,
			ItemDate:  merged.ItemDate,
			StartTime: merged.StartTime,
			EndTime:   merged.EndTime,
			Mode:      merged.Mode,
			Room:      merged.Room,
		})
		if err != nil {
			return err
		}
		status := item.Status
		if item.Status == ItemTeacherAsk {
			status = ItemRescheduled
		}
		_, err = tx.Exec(ctx, patchItemSQL, item.ID, patch.ItemDate, patch.StartTime, patch.EndTime,
			patch.Mode, patch.Room, patch.MeetingLink, patch.Note, patch.Syllabus, patch.Instructions, status)
		if err != nil {
			return fmt.Errorf("store: could not update the schedule item %d: %w", item.ID, err)
		}
		if item.Published() && patch.MeetingLink != nil {
			if _, err := tx.Exec(ctx, syncPublishedLinkSQL, item.ID, *patch.MeetingLink); err != nil {
				return fmt.Errorf("store: could not update the published link of the item %d: %w", item.ID, err)
			}
		}
		return nil
	})
}

func (s *Store) CancelItemAsCR(ctx context.Context, itemID int64, batch, reason string) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByBatch(item, batch); err != nil {
			return err
		}
		if item.Status == ItemCancelled {
			return ErrItemState
		}
		if item.Origin != OriginCR {
			switch item.Status {
			case ItemTeacherAsk, ItemNoResponse, ItemDeclined:
			default:
				return ErrItemState
			}
		}
		return cancelAndSync(ctx, tx, item, reason)
	})
}

func (s *Store) DeleteCRItem(ctx context.Context, itemID int64, batch string) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByBatch(item, batch); err != nil {
			return err
		}
		if item.Origin != OriginCR {
			return ErrItemState
		}
		if _, err := tx.Exec(ctx, deleteItemSQL, item.ID); err != nil {
			return fmt.Errorf("store: could not delete the schedule item %d: %w", item.ID, err)
		}
		return nil
	})
}

func cancelAndSync(ctx context.Context, tx pgx.Tx, item ScheduleItem, reason string) error {
	if _, err := tx.Exec(ctx, cancelItemSQL, item.ID, reason); err != nil {
		return fmt.Errorf("store: could not cancel the schedule item %d: %w", item.ID, err)
	}
	if !item.Published() {
		return nil
	}
	if _, err := tx.Exec(ctx, syncPublishedStatusSQL, item.ID, ItemCancelled); err != nil {
		return fmt.Errorf("store: could not cancel the published copy of the item %d: %w", item.ID, err)
	}
	return nil
}

func (s *Store) SetItemFinal(ctx context.Context, itemID int64, batch string, inFinal bool) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		if err := ownedByBatch(item, batch); err != nil {
			return err
		}
		if item.Status == ItemCancelled {
			return ErrItemState
		}
		if _, err := tx.Exec(ctx, setItemFinalSQL, item.ID, inFinal, batch); err != nil {
			return fmt.Errorf("store: could not change the final routine of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

func (s *Store) PublishFinal(ctx context.Context, batch string, publishedBy int64,
	ids []int64) (Publication, []ScheduleItem, error) {
	if len(ids) == 0 {
		return Publication{}, nil, ErrNothingToPublish
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Publication{}, nil, fmt.Errorf("store: could not start the publishing transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, listItemsByIDsSQL, batch, ids)
	if err != nil {
		return Publication{}, nil, fmt.Errorf("store: could not load the items to publish: %w", err)
	}
	items := make([]ScheduleItem, 0, len(ids))
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			rows.Close()
			return Publication{}, nil, fmt.Errorf("store: could not scan an item to publish: %w", err)
		}
		items = append(items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Publication{}, nil, fmt.Errorf("store: could not read the items to publish: %w", err)
	}
	if len(items) != len(ids) {
		return Publication{}, nil, ErrItemNotFound
	}
	for _, item := range items {
		if !item.InFinal {
			return Publication{}, nil, ErrItemState
		}
		if !item.Complete() {
			return Publication{}, nil, &IncompleteItemError{Item: item}
		}
	}
	var publication Publication
	err = tx.QueryRow(ctx, insertPublicationSQL, batch, publishedBy).Scan(
		&publication.ID,
		&publication.Batch,
		&publication.PublishedBy,
		&publication.PublishedAt,
	)
	if err != nil {
		return Publication{}, nil, fmt.Errorf("store: could not create the publication: %w", err)
	}
	changed := make([]ScheduleItem, 0, len(items))
	for _, item := range items {
		if item.Published() && item.ChangedSincePublish() {
			changed = append(changed, item)
		}
		_, err = tx.Exec(ctx, upsertPublishedItemSQL,
			item.ID, publication.ID, item.Batch, item.Section, item.ItemDate, item.StartTime, item.EndTime,
			item.CourseCode, item.CourseTitle, item.TeacherName, item.Kind, item.Mode, item.Room,
			item.MeetingLink, item.Note, item.Syllabus, item.Instructions, item.Status)
		if err != nil {
			return Publication{}, nil, fmt.Errorf("store: could not publish the item %d: %w", item.ID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Publication{}, nil, fmt.Errorf("store: could not commit the publication: %w", err)
	}
	return publication, changed, nil
}

func (s *Store) UpdateItemLink(ctx context.Context, itemID, userID int64, link string) (ScheduleItem, error) {
	return s.withItem(ctx, itemID, func(ctx context.Context, tx pgx.Tx, item ScheduleItem) error {
		allowed, err := mayEditLink(ctx, tx, item, userID)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrItemNotFound
		}
		if _, err := tx.Exec(ctx, setItemLinkSQL, item.ID, link); err != nil {
			return fmt.Errorf("store: could not update the link of the item %d: %w", item.ID, err)
		}
		if !item.Published() {
			return nil
		}
		if _, err := tx.Exec(ctx, syncPublishedLinkSQL, item.ID, link); err != nil {
			return fmt.Errorf("store: could not update the published link of the item %d: %w", item.ID, err)
		}
		return nil
	})
}

const (
	setItemLinkSQL = `UPDATE schedule_items
SET meeting_link = $2, updated_at = now()
WHERE id = $1`
	isBatchCRSQL = `SELECT EXISTS (
    SELECT 1 FROM students s
    JOIN users u ON u.id = s.user_id
    WHERE s.user_id = $1 AND s.is_cr AND u.status = 'active' AND s.batch = $2
)`
)

func mayEditLink(ctx context.Context, tx pgx.Tx, item ScheduleItem, userID int64) (bool, error) {
	if item.TeacherUserID == userID && userID != 0 {
		return true, nil
	}
	var isCR bool
	if err := tx.QueryRow(ctx, isBatchCRSQL, userID, item.Batch).Scan(&isCR); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("store: could not check the class representative of %s: %w", item.Batch, err)
	}
	return isCR, nil
}
