package model

import (
	"fmt"
	"time"

	cal "github.com/rickar/cal/v2"
	"github.com/rickar/cal/v2/us"
)

func TitleFromTime(t time.Time) string {
	title := t.Format("2006-01-02 Monday")

	c := cal.NewBusinessCalendar()
	c.AddHoliday(us.Holidays...)
	c.SetWorkHours(9*time.Hour, 18*time.Hour+30*time.Minute)

	actual, observed, holiday := c.IsHoliday(t)
	if actual || observed {
		title += fmt.Sprintf(" (%s)", holiday.Name)
	}

	return title
}
