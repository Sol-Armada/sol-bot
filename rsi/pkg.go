package rsi

import (
	"errors"
	"fmt"

	"github.com/apex/log"
	"github.com/gocolly/colly/v2"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/users"
)

var (
	UserNotFound error = errors.New("user was not found")
)

func GetOrgInfo(username string) (string, users.Rank, error) {
	c := colly.NewCollector()

	rank := users.Recruit
	var po string
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})

	c.OnXML(`//div[contains(@class, "main-org")]//div[@class="info"]//span[contains(text(), "rank")]/following-sibling::strong`, func(e *colly.XMLElement) {
		switch e.Text {
		case "Director":
			rank = users.Admiral
		case "Commander":
			rank = users.Commander
		case "Lieutenant":
			rank = users.Lieutenant
		case "Chief":
			rank = users.Specialist
		case "Specialist":
			rank = users.Technician
		case "Initiate":
			rank = users.Member
		}
	})

	c.OnXML(`//div[contains(@class, "main-org")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		po = e.Text
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", username)); err != nil {
		if err.Error() == "Not Found" {
			return po, users.Recruit, UserNotFound
		}

		return po, users.Recruit, err
	}

	if po != config.GetString("rsi_org_sid") {
		rank = users.Recruit
	}

	log.WithFields(log.Fields{
		"username":    username,
		"rank":        rank,
		"primary org": po,
	}).Debug("rsi info")

	return po, rank, err
}
