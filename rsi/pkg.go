package rsi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/gocolly/colly/v2"
	"github.com/sol-armada/sol-bot/members"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
)

var (
	RsiUserNotFound error            = errors.New("rsi user was not found")
	c               *colly.Collector = colly.NewCollector(colly.AllowURLRevisit())
)

func UpdateRsiInfo(member *members.Member) (*members.Member, error) {
	member.RSIMember = false
	member.IsAlly = false
	member.IsGuest = true
	member.Rank = ranks.None
	member.PrimaryOrg = ""
	member.Affilations = []string{}

	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = RsiUserNotFound
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if e.Text == "" {
			e.Text = "None"
		}
		member.PrimaryOrg = e.Text
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "rank")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if member.PrimaryOrg == settings.GetString("rsi_org_sid") {
			member.Rank = ranks.GetRankByRSIRankName(e.Text)
			member.IsGuest = false
		}
	})

	c.OnXML(`//div[contains(@class, "orgs-content")]`, func(e *colly.XMLElement) {
		member.Affilations = e.ChildTexts(`//div[contains(@class, "org affiliation")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`)
		if utils.StringSliceContains(member.Affilations, settings.GetString("rsi_org_sid")) {
			member.IsAffiliate = true
			member.Rank = ranks.Member
			member.IsGuest = false
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[contains(@class,"member-visibility-restriction")]`, func(e *colly.XMLElement) {
		member.PrimaryOrg = "REDACTED"
		member.IsGuest = true
	})

	url := fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", strings.ReplaceAll(member.Name, ".", ""))
	if err := c.Visit(url); err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return member, RsiUserNotFound
		}

		return member, err
	}

	member.RSIMember = true

	if isAllyOrg(member.PrimaryOrg) {
		member.IsAlly = true
	}

	log.WithFields(log.Fields{
		"member": member,
	}).Debug("rsi info")

	return member, err
}

func isAllyOrg(org string) bool {
	whiteListOrgs := settings.GetStringSlice("ALLIES")
	return utils.StringSliceContains(whiteListOrgs, org)
}

func ValidHandle(handle string) bool {
	c := colly.NewCollector()

	exists := true
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != 200 {
			exists = false
		}
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", handle)); err != nil {
		if err.Error() == "Not Found" {
			exists = false
		}
	}

	return exists
}

func IsMemberOfOrg(handle string, org string) (bool, error) {
	c := colly.NewCollector()

	var orgs []string
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = RsiUserNotFound
		}
	})

	c.OnXML(`//div[@id="public-profile"]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		orgs = []string{}
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", handle)); err != nil {
		if err.Error() == "Not Found" {
			return false, RsiUserNotFound
		}

		return false, err
	}

	log.WithFields(log.Fields{
		"handle": handle,
		"orgs":   orgs,
	}).Debug("rsi info")

	for _, o := range orgs {
		if strings.EqualFold(o, org) {
			return true, nil
		}
	}

	return false, err
}

func GetBio(handle string) (string, error) {
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = RsiUserNotFound
		}
	})
	if err != nil {
		return "", err
	}

	bio := ""
	c.OnXML(`//div[@id="public-profile"]//div[contains(@class, "bio")]/div`, func(e *colly.XMLElement) {
		bio = e.Text
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", handle)); err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return "", RsiUserNotFound
		}

		return "", err
	}

	return bio, nil
}
