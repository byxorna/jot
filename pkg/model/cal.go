package model

import (
	"fmt"
	"sort"
	"time"

	cal "github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

/*
Juneteenth - June 19th (observed in 2021, Monday, August 2nd)
Labor Day - Monday, September 6th
Indigenous Peoples' Day - Monday, October 11th
Thanksgiving Day - Thursday, November 25th
Day after Thanksgiving - Friday, November 26th
Christmas Day - December 25th (observed, Friday, December 24th)
*/

var (
	calendar = cal.NewBusinessCalendar()
)

func init() {
	calendar.AddHoliday(
		us.NewYear,
		us.MemorialDay,
		us.IndependenceDay,
		us.Juneteenth,
		us.DayAfterThanksgivingDay,
		us.LaborDay,
		us.ThanksgivingDay,
		us.ChristmasDay,
	)
}

func (m *Model) TitleFromTime(t time.Time) string {
	calendar.SetWorkHours(m.Config.StartWorkHours, m.Config.EndWorkHours)
	title := t.Format("2006-01-02 Monday")
	actual, observed, holiday := calendar.IsHoliday(t)
	if actual || observed {
		title += fmt.Sprintf(" (%s)", holiday.Name)
	}
	return title
}

func (m *Model) DefaultTagsForTime(t time.Time) []string {
	var tags []string
	actual, observed, _ := calendar.IsHoliday(t)
	if actual || observed {
		tags = append(tags, m.Config.HolidayTags...)
	}
	if calendar.IsWorkday(t) {
		tags = append(tags, m.Config.WorkdayTags...)
	} else {
		tags = append(tags, m.Config.WeekendTags...)
	}

	sort.Strings(tags)
	return tags
}
