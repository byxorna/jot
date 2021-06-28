package model

import (
	"fmt"
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
func TitleFromTime(t time.Time) string {
	title := t.Format("2006-01-02 Monday")

	c := cal.NewBusinessCalendar()
	c.AddHoliday(
		us.NewYear,
		us.MemorialDay,
		us.IndependenceDay,
		us.Juneteenth,
		us.DayAfterThanksgivingDay,
		us.LaborDay,
		us.ThanksgivingDay,
		us.ChristmasDay,
	)
	c.SetWorkHours(9*time.Hour, 18*time.Hour+30*time.Minute)

	actual, observed, holiday := c.IsHoliday(t)
	if actual || observed {
		title += fmt.Sprintf(" (%s)", holiday.Name)
	}

	return title
}
