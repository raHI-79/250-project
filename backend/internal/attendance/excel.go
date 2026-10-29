package attendance

import (
	"bytes"
	"fmt"
	"github.com/xuri/excelize/v2"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	SummarySheet  = "Summary"
	sheetNameMax  = 31
	dateHeader    = "02 Jan"
	noteAuthor    = "Attendance"
	columnNameW   = 26.0
	columnRegW    = 14.0
	columnDateW   = 9.0
	columnCountW  = 11.0
	presentLetter = "P"
	absentLetter  = "A"
)

type ExportRow struct {
	StudentID int64
	FullName  string
	RegNo     string
	SessionID int64
	StartedAt time.Time
	Status    string
	Note      string
}

type ExportClass struct {
	CourseCode  string
	CourseTitle string
	Batch       string
	Section     string
	Sessions    int
	Rows        []ExportRow
}

type studentLine struct {
	studentID int64
	name      string
	regNo     string
	marks     map[int64]ExportRow
	present   int
	held      int
}

func SheetName(class ExportClass, taken map[string]bool) string {
	base := fmt.Sprintf("%s %s-%s", class.CourseCode, class.Batch, class.Section)
	for _, bad := range []string{":", "\\", "/", "?", "*", "[", "]"} {
		base = strings.ReplaceAll(base, bad, " ")
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Class"
	}
	if len(base) > sheetNameMax {
		base = base[:sheetNameMax]
	}
	name := base
	for suffix := 2; taken[name]; suffix++ {
		tail := fmt.Sprintf(" %d", suffix)
		cut := sheetNameMax - len(tail)
		if cut > len(base) {
			cut = len(base)
		}
		name = base[:cut] + tail
	}
	taken[name] = true
	return name
}

func lines(class ExportClass) ([]*studentLine, []ExportRow) {
	order := make([]*studentLine, 0)
	byStudent := make(map[int64]*studentLine)
	sessions := make([]ExportRow, 0)
	seen := make(map[int64]bool)
	for _, row := range class.Rows {
		line, ok := byStudent[row.StudentID]
		if !ok {
			line = &studentLine{
				studentID: row.StudentID,
				name:      row.FullName,
				regNo:     row.RegNo,
				marks:     make(map[int64]ExportRow),
			}
			byStudent[row.StudentID] = line
			order = append(order, line)
		}
		line.marks[row.SessionID] = row
		line.held++
		if row.Status == Present {
			line.present++
		}
		if !seen[row.SessionID] {
			seen[row.SessionID] = true
			sessions = append(sessions, row)
		}
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.Before(sessions[j].StartedAt)
	})
	sort.SliceStable(order, func(i, j int) bool { return order[i].regNo < order[j].regNo })
	return order, sessions
}

func ClassAverage(class ExportClass) (float64, int) {
	order, _ := lines(class)
	if len(order) == 0 {
		return 0, 0
	}
	total, below := 0.0, 0
	for _, line := range order {
		share := Percentage(line.present, line.held)
		total += share
		if share < AlertPercentage {
			below++
		}
	}
	return total / float64(len(order)), below
}

func BuildWorkbook(classes []ExportClass, withSummary bool) (*bytes.Buffer, error) {
	book := excelize.NewFile()
	defer func() { _ = book.Close() }()
	header, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8EDF7"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return nil, fmt.Errorf("attendance: could not style the header: %w", err)
	}
	centered, err := book.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("attendance: could not style the marks: %w", err)
	}
	low, err := book.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "9B1C1C"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FDE8E8"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if err != nil {
		return nil, fmt.Errorf("attendance: could not style the low percentages: %w", err)
	}
	first := book.GetSheetName(0)
	taken := make(map[string]bool)
	names := make([]string, 0, len(classes))
	for index, class := range classes {
		name := SheetName(class, taken)
		names = append(names, name)
		if index == 0 {
			if err := book.SetSheetName(first, name); err != nil {
				return nil, fmt.Errorf("attendance: could not name the first sheet: %w", err)
			}
		} else if _, err := book.NewSheet(name); err != nil {
			return nil, fmt.Errorf("attendance: could not add the sheet %q: %w", name, err)
		}
		if err := writeClassSheet(book, name, class, header, centered, low); err != nil {
			return nil, err
		}
	}
	if len(classes) == 0 {
		if err := book.SetSheetName(first, SummarySheet); err != nil {
			return nil, fmt.Errorf("attendance: could not name the first sheet: %w", err)
		}
		withSummary = true
		names = append(names, SummarySheet)
	} else if withSummary {
		if _, err := book.NewSheet(SummarySheet); err != nil {
			return nil, fmt.Errorf("attendance: could not add the summary sheet: %w", err)
		}
	}
	if withSummary {
		if err := writeSummarySheet(book, classes, header, centered, low); err != nil {
			return nil, err
		}
	}
	if index, err := book.GetSheetIndex(names[0]); err == nil {
		book.SetActiveSheet(index)
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("attendance: could not write the workbook: %w", err)
	}
	return buffer, nil
}

func writeClassSheet(book *excelize.File, sheet string, class ExportClass,
	header, centered, low int) error {
	order, sessions := lines(class)
	titles := []string{"Student", "Reg no"}
	for _, session := range sessions {
		titles = append(titles, session.StartedAt.Format(dateHeader))
	}
	titles = append(titles, "Present", "Total", "Percentage")
	for column, title := range titles {
		cell, err := excelize.CoordinatesToCellName(column+1, 1)
		if err != nil {
			return fmt.Errorf("attendance: could not place a header: %w", err)
		}
		if err := book.SetCellStr(sheet, cell, title); err != nil {
			return fmt.Errorf("attendance: could not write a header: %w", err)
		}
		if err := book.SetCellStyle(sheet, cell, cell, header); err != nil {
			return fmt.Errorf("attendance: could not style a header: %w", err)
		}
	}
	for index, line := range order {
		row := index + 2
		if err := book.SetCellStr(sheet, cellAt(1, row), line.name); err != nil {
			return fmt.Errorf("attendance: could not write a student name: %w", err)
		}
		if err := book.SetCellStr(sheet, cellAt(2, row), line.regNo); err != nil {
			return fmt.Errorf("attendance: could not write a registration number: %w", err)
		}
		for offset, session := range sessions {
			cell := cellAt(3+offset, row)
			mark, ok := line.marks[session.SessionID]
			letter := ""
			if ok {
				letter = absentLetter
				if mark.Status == Present {
					letter = presentLetter
				}
			}
			if err := book.SetCellStr(sheet, cell, letter); err != nil {
				return fmt.Errorf("attendance: could not write a mark: %w", err)
			}
			if err := book.SetCellStyle(sheet, cell, cell, centered); err != nil {
				return fmt.Errorf("attendance: could not style a mark: %w", err)
			}
			if ok && mark.Note != "" {
				comment := excelize.Comment{
					Cell:   cell,
					Author: noteAuthor,
					Paragraph: []excelize.RichTextRun{
						{Text: mark.Note},
					},
				}
				if err := book.AddComment(sheet, comment); err != nil {
					return fmt.Errorf("attendance: could not add a note: %w", err)
				}
			}
		}
		share := Percentage(line.present, line.held)
		presentCell := cellAt(3+len(sessions), row)
		totalCell := cellAt(4+len(sessions), row)
		shareCell := cellAt(5+len(sessions), row)
		if err := book.SetCellInt(sheet, presentCell, int64(line.present)); err != nil {
			return fmt.Errorf("attendance: could not write a present count: %w", err)
		}
		if err := book.SetCellInt(sheet, totalCell, int64(line.held)); err != nil {
			return fmt.Errorf("attendance: could not write a session count: %w", err)
		}
		if err := book.SetCellStr(sheet, shareCell, sharePercent(share)); err != nil {
			return fmt.Errorf("attendance: could not write a percentage: %w", err)
		}
		style := centered
		if share < AlertPercentage {
			style = low
		}
		if err := book.SetCellStyle(sheet, presentCell, shareCell, style); err != nil {
			return fmt.Errorf("attendance: could not style a percentage: %w", err)
		}
	}
	if err := book.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      2,
		YSplit:      1,
		TopLeftCell: "C2",
		ActivePane:  "bottomRight",
	}); err != nil {
		return fmt.Errorf("attendance: could not freeze the header: %w", err)
	}
	if err := book.SetColWidth(sheet, "A", "A", columnNameW); err != nil {
		return fmt.Errorf("attendance: could not size the name column: %w", err)
	}
	if err := book.SetColWidth(sheet, "B", "B", columnRegW); err != nil {
		return fmt.Errorf("attendance: could not size the registration column: %w", err)
	}
	if len(sessions) > 0 {
		from, _ := excelize.ColumnNumberToName(3)
		to, _ := excelize.ColumnNumberToName(2 + len(sessions))
		if err := book.SetColWidth(sheet, from, to, columnDateW); err != nil {
			return fmt.Errorf("attendance: could not size the date columns: %w", err)
		}
	}
	countFrom, _ := excelize.ColumnNumberToName(3 + len(sessions))
	countTo, _ := excelize.ColumnNumberToName(5 + len(sessions))
	if err := book.SetColWidth(sheet, countFrom, countTo, columnCountW); err != nil {
		return fmt.Errorf("attendance: could not size the total columns: %w", err)
	}
	return nil
}

func writeSummarySheet(book *excelize.File, classes []ExportClass, header, centered, low int) error {
	titles := []string{"Course", "Title", "Batch", "Section", "Sessions", "Average", "Below 50 %"}
	for column, title := range titles {
		cell := cellAt(column+1, 1)
		if err := book.SetCellStr(SummarySheet, cell, title); err != nil {
			return fmt.Errorf("attendance: could not write a summary header: %w", err)
		}
		if err := book.SetCellStyle(SummarySheet, cell, cell, header); err != nil {
			return fmt.Errorf("attendance: could not style a summary header: %w", err)
		}
	}
	for index, class := range classes {
		row := index + 2
		average, below := ClassAverage(class)
		values := []string{class.CourseCode, class.CourseTitle, class.Batch, class.Section}
		for column, value := range values {
			if err := book.SetCellStr(SummarySheet, cellAt(column+1, row), value); err != nil {
				return fmt.Errorf("attendance: could not write a summary cell: %w", err)
			}
		}
		if err := book.SetCellInt(SummarySheet, cellAt(5, row), int64(class.Sessions)); err != nil {
			return fmt.Errorf("attendance: could not write a summary session count: %w", err)
		}
		if err := book.SetCellStr(SummarySheet, cellAt(6, row), sharePercent(average)); err != nil {
			return fmt.Errorf("attendance: could not write a summary average: %w", err)
		}
		if err := book.SetCellInt(SummarySheet, cellAt(7, row), int64(below)); err != nil {
			return fmt.Errorf("attendance: could not write a summary count: %w", err)
		}
		style := centered
		if average < AlertPercentage {
			style = low
		}
		if err := book.SetCellStyle(SummarySheet, cellAt(5, row), cellAt(7, row), style); err != nil {
			return fmt.Errorf("attendance: could not style a summary row: %w", err)
		}
	}
	if err := book.SetPanes(SummarySheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("attendance: could not freeze the summary header: %w", err)
	}
	if err := book.SetColWidth(SummarySheet, "A", "G", columnCountW+4); err != nil {
		return fmt.Errorf("attendance: could not size the summary columns: %w", err)
	}
	return nil
}

func cellAt(column, row int) string {
	name, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		return "A1"
	}
	return name
}

func sharePercent(share float64) string {
	return fmt.Sprintf("%d %%", int(math.Round(share)))
}

func FileName(classes []ExportClass, from, to time.Time) string {
	stamp := from.Format("2006-01-02") + "_" + to.Format("2006-01-02")
	if len(classes) == 1 {
		clean := strings.ReplaceAll(classes[0].CourseCode, " ", "")
		return fmt.Sprintf("attendance_%s_%s-%s_%s.xlsx", clean,
			classes[0].Batch, classes[0].Section, stamp)
	}
	return fmt.Sprintf("attendance_all-classes_%s.xlsx", stamp)
}
