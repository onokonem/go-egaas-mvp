package controllers

import (
	"github.com/DayLightProject/go-daylight/packages/utils"
)


type dashboardAnonymPage struct {
	Lang                  map[string]string
	Title                 string
	CountSign             int
	CountSignArr          []int
	SignData              string
	ShowSignData          bool
}

func (c *Controller) DashboardAnonym() (string, error) {

	TemplateStr, err := makeTemplate("dashboard_anonym", "dashboardAnonym", &dashboardAnonymPage{
		CountSignArr:          c.CountSignArr,
		CountSign:             c.CountSign,
		Lang:                  c.Lang,
		Title:                 "Home",
		ShowSignData:          c.ShowSignData,
		SignData:              ""})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}