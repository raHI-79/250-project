package attendance

import (
	"bytes"
	"github.com/xuri/excelize/v2"
	"testing"
	"time"
)

func sampleClass() ExportClass {
	first := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	second := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	third := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	return ExportClass{
		CourseCode:  "CSE245",
		CourseTitle: "Databases",
		Batch:       "CSE-23",
		Section:     "A",
		Sessions:    3,
		Rows: []ExportRow{
			{StudentID: 1, FullName: "Ada Byron", RegNo: "2023331001", SessionID: 1,
				StartedAt: first, Status: Present},
			{StudentID: 1, FullName: "Ada Byron", RegNo: "2023331001", SessionID: 2,
				StartedAt: second, Status: Present},
			{StudentID: 1, FullName: "Ada Byron", RegNo: "2023331001", SessionID: 3,
				StartedAt: third, Status: Present},
			{StudentID: 2, FullName: "Alan Turing", RegNo: "2023331002", SessionID: 1,
				StartedAt: first, Status: Absent, Note: "Sick leave"},
			{StudentID: 2, FullName: "Alan Turing", RegNo: "2023331002", SessionID: 2,
				StartedAt: second, Status: Absent},
			{StudentID: 2, FullName: "Alan Turing", RegNo: "2023331002", SessionID: 3,
				StartedAt: third, Status: Present},
		},
	}
}

func openWorkbook(t *testing.T, buffer *bytes.Buffer) *excelize.File {
	t.Helper()
	book, err := excelize.OpenReader(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("the workbook could not be read back: %v", err)
	}
	t.Cleanup(func() { _ = book.Close() })
	return book
}

func cell(t *testing.T, book *excelize.File, sheet, reference string) string {
	t.Helper()
	value, err := book.GetCellValue(sheet, reference)
	if err != nil {
		t.Fatalf("could not read %s!%s: %v", sheet, reference, err)
	}
	return value
}

func TestWorkbookHoldsOneSheetPerClass(t *testing.T) {
	buffer, err := BuildWorkbook([]ExportClass{sampleClass()}, false)
	if err != nil {
		t.Fatalf("BuildWorkbook() returned an unexpected error: %v", err)
	}
	book := openWorkbook(t, buffer)
	sheets := book.GetSheetList()
	if len(sheets) != 1 || sheets[0] != "CSE245 CSE-23-A" {
		t.Fatalf("sheets = %v, want one sheet named CSE245 CSE-23-A", sheets)
	}
	sheet := sheets[0]
	tests := []struct {
		reference string
		want      string
	}{
		{reference: "A1", want: "Student"},
		{reference: "B1", want: "Reg no"},
		{reference: "C1", want: "05 Oct"},
		{reference: "D1", want: "12 Oct"},
		{reference: "E1", want: "19 Oct"},
		{reference: "F1", want: "Present"},
		{reference: "G1", want: "Total"},
		{reference: "H1", want: "Percentage"},
		{reference: "A2", want: "Ada Byron"},
		{reference: "B2", want: "2023331001"},
		{reference: "C2", want: "P"},
		{reference: "D2", want: "P"},
		{reference: "E2", want: "P"},
		{reference: "F2", want: "3"},
		{reference: "G2", want: "3"},
		{reference: "H2", want: "100 %"},
		{reference: "A3", want: "Alan Turing"},
		{reference: "C3", want: "A"},
		{reference: "D3", want: "A"},
		{reference: "E3", want: "P"},
		{reference: "F3", want: "1"},
		{reference: "G3", want: "3"},
		{reference: "H3", want: "33 %"},
	}
	for _, tc := range tests {
		if got := cell(t, book, sheet, tc.reference); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.reference, got, tc.want)
		}
	}
}

func TestWorkbookKeepsNotesAsCellComments(t *testing.T) {
	buffer, err := BuildWorkbook([]ExportClass{sampleClass()}, false)
	if err != nil {
		t.Fatalf("BuildWorkbook() returned an unexpected error: %v", err)
	}
	book := openWorkbook(t, buffer)
	comments, err := book.GetComments("CSE245 CSE-23-A")
	if err != nil {
		t.Fatalf("GetComments() returned an unexpected error: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("the sheet carries %d comments, want 1", len(comments))
	}
	if comments[0].Cell != "C3" {
		t.Errorf("the comment sits on %s, want C3", comments[0].Cell)
	}
	found := false
	for _, run := range comments[0].Paragraph {
		if run.Text == "Sick leave" {
			found = true
		}
	}
	if !found {
		t.Errorf("comment = %v, want the note Sick leave", comments[0].Paragraph)
	}
}

func TestWorkbookPaintsBelowFiftyRed(t *testing.T) {
	buffer, err := BuildWorkbook([]ExportClass{sampleClass()}, false)
	if err != nil {
		t.Fatalf("BuildWorkbook() returned an unexpected error: %v", err)
	}
	book := openWorkbook(t, buffer)
	sheet := "CSE245 CSE-23-A"
	low, err := book.GetCellStyle(sheet, "H3")
	if err != nil {
		t.Fatalf("GetCellStyle() returned an unexpected error: %v", err)
	}
	high, err := book.GetCellStyle(sheet, "H2")
	if err != nil {
		t.Fatalf("GetCellStyle() returned an unexpected error: %v", err)
	}
	if low == high {
		t.Fatal("the student below 50 % carries the same style as the one above it")
	}
	style, err := book.GetStyle(low)
	if err != nil {
		t.Fatalf("GetStyle() returned an unexpected error: %v", err)
	}
	if len(style.Fill.Color) == 0 || style.Fill.Color[0] != "FDE8E8" {
		t.Errorf("the low percentage fill = %v, want the red wash", style.Fill.Color)
	}
}

func TestWorkbookFreezesTheHeader(t *testing.T) {
	buffer, err := BuildWorkbook([]ExportClass{sampleClass()}, false)
	if err != nil {
		t.Fatalf("BuildWorkbook() returned an unexpected error: %v", err)
	}
	book := openWorkbook(t, buffer)
	panes, err := book.GetPanes("CSE245 CSE-23-A")
	if err != nil {
		t.Fatalf("GetPanes() returned an unexpected error: %v", err)
	}
	if !panes.Freeze {
		t.Error("the sheet header is not frozen")
	}
	if panes.YSplit != 1 || panes.XSplit != 2 {
		t.Errorf("panes split at x=%d y=%d, want x=2 y=1", panes.XSplit, panes.YSplit)
	}
}

func TestWorkbookOfEveryClassCarriesASummary(t *testing.T) {
	first := sampleClass()
	second := sampleClass()
	second.CourseCode = "CSE247"
	second.CourseTitle = "Operating Systems"
	second.Section = "B"
	buffer, err := BuildWorkbook([]ExportClass{first, second}, true)
	if err != nil {
		t.Fatalf("BuildWorkbook() returned an unexpected error: %v", err)
	}
	book := openWorkbook(t, buffer)
	sheets := book.GetSheetList()
	want := []string{"CSE245 CSE-23-A", "CSE247 CSE-23-B", SummarySheet}
	if len(sheets) != len(want) {
		t.Fatalf("sheets = %v, want %v", sheets, want)
	}
	for index, name := range want {
		if sheets[index] != name {
			t.Errorf("sheet %d = %q, want %q", index, sheets[index], name)
		}
	}
	tests := []struct {
		reference string
		want      string
	}{
		{reference: "A1", want: "Course"},
		{reference: "E1", want: "Sessions"},
		{reference: "F1", want: "Average"},
		{reference: "G1", want: "Below 50 %"},
		{reference: "A2", want: "CSE245"},
		{reference: "B2", want: "Databases"},
		{reference: "C2", want: "CSE-23"},
		{reference: "D2", want: "A"},
		{reference: "E2", want: "3"},
		{reference: "F2", want: "67 %"},
		{reference: "G2", want: "1"},
		{reference: "A3", want: "CSE247"},
		{reference: "D3", want: "B"},
	}
	for _, tc := range tests {
		if got := cell(t, book, SummarySheet, tc.reference); got != tc.want {
			t.Errorf("Summary!%s = %q, want %q", tc.reference, got, tc.want)
		}
	}
}

func TestSheetNamesStayShortAndUnique(t *testing.T) {
	taken := make(map[string]bool)
	tests := []struct {
		name  string
		class ExportClass
		want  string
	}{
		{
			name:  "plain class",
			class: ExportClass{CourseCode: "CSE245", Batch: "CSE-23", Section: "A"},
			want:  "CSE245 CSE-23-A",
		},
		{
			name:  "the same class twice",
			class: ExportClass{CourseCode: "CSE245", Batch: "CSE-23", Section: "A"},
			want:  "CSE245 CSE-23-A 2",
		},
		{
			name:  "a code Excel cannot take",
			class: ExportClass{CourseCode: "CSE/245*", Batch: "CSE-23", Section: "B"},
			want:  "CSE 245  CSE-23-B",
		},
		{
			name: "a very long code",
			class: ExportClass{
				CourseCode: "CSE245-ADVANCED-DATABASE-SYSTEMS-II",
				Batch:      "CSE-23",
				Section:    "A",
			},
			want: "CSE245-ADVANCED-DATABASE-SYSTEM",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SheetName(tc.class, taken)
			if got != tc.want {
				t.Errorf("SheetName() = %q, want %q", got, tc.want)
			}
			if len(got) > sheetNameMax {
				t.Errorf("SheetName() returned %d characters, want at most %d", len(got), sheetNameMax)
			}
		})
	}
}

func TestClassAverageCountsTheStudentsBelowFifty(t *testing.T) {
	average, below := ClassAverage(sampleClass())
	if int(average) != 66 {
		t.Errorf("average = %.2f, want about 66", average)
	}
	if below != 1 {
		t.Errorf("below = %d, want 1", below)
	}
	empty, none := ClassAverage(ExportClass{})
	if empty != 0 || none != 0 {
		t.Errorf("ClassAverage() on an empty class = %.2f, %d, want 0, 0", empty, none)
	}
}

func TestFileNameNamesOneClassOrEveryClass(t *testing.T) {
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	one := FileName([]ExportClass{sampleClass()}, from, to)
	if one != "attendance_CSE245_CSE-23-A_2026-09-01_2026-10-30.xlsx" {
		t.Errorf("FileName() for one class = %q", one)
	}
	all := FileName([]ExportClass{sampleClass(), sampleClass()}, from, to)
	if all != "attendance_all-classes_2026-09-01_2026-10-30.xlsx" {
		t.Errorf("FileName() for every class = %q", all)
	}
}
