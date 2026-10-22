package validate

import (
	"fmt"
	"sort"
	"strings"
)

const MaxPeriodLabel = 40

type Period struct {
	Label   string
	Start   string
	End     string
	IsBreak bool
}

func PeriodLabel(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required.", label)
	}
	if len([]rune(trimmed)) > MaxPeriodLabel {
		return "", fmt.Errorf("%s must be at most %d characters long.", label, MaxPeriodLabel)
	}
	return trimmed, nil
}

func PeriodWindow(start, end string) error {
	from, to := Minutes(start), Minutes(end)
	if from < 0 || to < 0 {
		return fmt.Errorf("Start time and end time must look like 09:30.")
	}
	if from >= to {
		return fmt.Errorf("Start time must be earlier than end time.")
	}
	return nil
}

func PeriodFits(periods []Period, candidate Period) error {
	if err := PeriodWindow(candidate.Start, candidate.End); err != nil {
		return err
	}
	from, to := Minutes(candidate.Start), Minutes(candidate.End)
	for _, period := range periods {
		if from < Minutes(period.End) && Minutes(period.Start) < to {
			return fmt.Errorf("This period overlaps %s (%s–%s).", period.Label, period.Start, period.End)
		}
	}
	return nil
}

func DayWindow(periods []Period) (int, int) {
	if len(periods) == 0 {
		return -1, -1
	}
	opening, closing := Minutes(periods[0].Start), Minutes(periods[0].End)
	for _, period := range periods {
		if start := Minutes(period.Start); start < opening {
			opening = start
		}
		if end := Minutes(period.End); end > closing {
			closing = end
		}
	}
	return opening, closing
}

func ClassWindow(periods []Period, start, end string) error {
	from, to := Minutes(start), Minutes(end)
	if from < 0 || to < 0 {
		return fmt.Errorf("Start time and end time must look like 09:30.")
	}
	if from >= to {
		return fmt.Errorf("Start time must be earlier than end time.")
	}
	opening, closing := DayWindow(periods)
	if opening < 0 {
		return fmt.Errorf("The class periods are not set up yet. Ask the department office to add them.")
	}
	if from < opening || to > closing {
		return fmt.Errorf("Classes must be between %s and %s.", Clock(opening), Clock(closing))
	}
	for _, period := range periods {
		if !period.IsBreak {
			continue
		}
		if from < Minutes(period.End) && Minutes(period.Start) < to {
			return fmt.Errorf("Classes cannot run during %s (%s–%s).", period.Label, period.Start, period.End)
		}
	}
	return nil
}

func SortPeriods(periods []Period) []Period {
	sorted := append([]Period(nil), periods...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return Minutes(sorted[i].Start) < Minutes(sorted[j].Start)
	})
	return sorted
}
