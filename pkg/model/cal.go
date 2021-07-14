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
	embeddedcal = cal.NewBusinessCalendar()
)

func init() {
	embeddedcal.AddHoliday(
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

func TitleFromTime(t time.Time, startWorkHours, endWorkHours time.Duration) string {
	embeddedcal.SetWorkHours(startWorkHours, endWorkHours)
	title := t.Format("2006-01-02 Monday")
	actual, observed, holiday := embeddedcal.IsHoliday(t)
	if actual || observed {
		title += fmt.Sprintf(" (%s)", holiday.Name)
	}
	return title
}

func DefaultTagsForTime(t time.Time, holidayTags, workdayTags, weekendTags []string) []string {
	var tags []string
	actual, observed, _ := embeddedcal.IsHoliday(t)
	if actual || observed {
		tags = append(tags, holidayTags...)
	}
	if embeddedcal.IsWorkday(t) {
		tags = append(tags, workdayTags...)
	} else {
		tags = append(tags, weekendTags...)
	}

	sort.Strings(tags)
	return tags
}
