package rsi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/gocolly/colly/v2"
	"github.com/sol-armada/admin/config"
	"github.com/sol-armada/admin/ranks"
	"github.com/sol-armada/admin/utils"
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

	c.OnXML(`//div[contains(@class, "main-org")]//div[contains(@class,"member-visibility-restriction")]`, func(e *colly.XMLElement) {
		po = "REDACTED"
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", username)); err != nil {
		if err.Error() == "Not Found" {
			return po, ranks.Recruit, UserNotFound
		}

		return po, ranks.Recruit, err
	}

	if po != config.GetString("rsi_org_sid") || po == "REDACTED" {
		rank = ranks.Recruit
	}

	log.WithFields(log.Fields{
		"username":    username,
		"rank":        rank,
		"primary org": po,
	}).Debug("rsi info")

	return po, rank, err
}

func IsAllyOrg(org string) bool {
	whiteListOrgs := config.GetStringSlice("ALLIES")
	return utils.StringSliceContains(whiteListOrgs, org)
}

func ValidHandle(username string) bool {
	c := colly.NewCollector()

	exists := true
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			exists = false
		}
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", username)); err != nil {
		if err.Error() == "Not Found" {
			exists = false
		}
	}

	return exists
}

func IsMemberOfOrg(username string, org string) (bool, error) {
	c := colly.NewCollector()

	var orgs []string
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})

	c.OnXML(`//div[@id="public-profile"]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		orgs = []string{}
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", username)); err != nil {
		if err.Error() == "Not Found" {
			return false, UserNotFound
		}

		return false, err
	}

	log.WithFields(log.Fields{
		"username": username,
		"orgs":     orgs,
	}).Debug("rsi info")

	for _, o := range orgs {
		if strings.EqualFold(o, org) {
			return true, nil
		}
	}

	return false, err
}
