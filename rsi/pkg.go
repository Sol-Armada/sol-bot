package rsi

import (
	"errors"
	"fmt"

	"github.com/apex/log"
	"github.com/gocolly/colly/v2"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
)

var (
	UserNotFound error = errors.New("user was not found")
)

func GetOrgInfo(username string) (string, ranks.Rank, error) {
	c := colly.NewCollector()

	rank := ranks.Recruit
	var po string
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})

	c.OnXML(`//div[contains(@class, "main-org")]//div[@class="info"]//span[contains(text(), "rank")]/following-sibling::strong`, func(e *colly.XMLElement) {
		rank = ranks.GetRankByRSIRankName(e.Text)
	})

	c.OnXML(`//div[contains(@class, "main-org")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		po = e.Text
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", username)); err != nil {
		if err.Error() == "Not Found" {
			return po, ranks.Recruit, UserNotFound
		}

		return po, ranks.Recruit, err
	}

	if po != config.GetString("rsi_org_sid") {
		rank = ranks.Recruit
	}

	log.WithFields(log.Fields{
		"username":    username,
		"rank":        rank,
		"primary org": po,
	}).Debug("rsi info")

	return po, rank, err
}
