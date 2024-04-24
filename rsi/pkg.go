package rsi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/gocolly/colly/v2"
	"github.com/sol-armada/sol-bot/config"
	"github.com/sol-armada/sol-bot/ranks"
	"github.com/sol-armada/sol-bot/users"
	"github.com/sol-armada/sol-bot/utils"
)

var (
	UserNotFound error            = errors.New("user was not found")
	c            *colly.Collector = colly.NewCollector(colly.AllowURLRevisit())
)

func GetOrgInfo(u *users.User) (*users.User, error) {
	u.Rank = ranks.Guest
	u.PrimaryOrg = ""
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if e.Text == "" {
			e.Text = "None"
		}
		u.PrimaryOrg = e.Text
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[@class="info"]//span[contains(text(), "rank")]/following-sibling::strong`, func(e *colly.XMLElement) {
		if u.PrimaryOrg == config.GetString("rsi_org_sid") {
			u.Rank = ranks.GetRankByRSIRankName(e.Text)
		}
	})

	c.OnXML(`//div[contains(@class, "orgs-content")]`, func(e *colly.XMLElement) {
		u.Affilations = e.ChildTexts(`//div[contains(@class, "org affiliation")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`)
		if len(u.Affilations) == 0 {
			u.Affilations = append(u.Affilations, "None")
			return
		}
	})

	c.OnXML(`//div[contains(@class, "org main")]//div[contains(@class,"member-visibility-restriction")]`, func(e *colly.XMLElement) {
		u.PrimaryOrg = "REDACTED"
		u.Rank = ranks.Guest
	})

	url := fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s/organizations", strings.ReplaceAll(u.GetTrueNick(), ".", ""))
	if err := c.Visit(url); err != nil {
		t := err.Error()
		if t == "Not Found" {
			return u, UserNotFound
		}

		return u, err
	}

	log.WithFields(log.Fields{
		"user": u,
	}).Debug("rsi info")

	return u, err
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

func GetBio(nick string) (string, error) {
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})
	if err != nil {
		return "", err
	}

	bio := ""
	c.OnXML(`//div[@id="public-profile"]//div[contains(@class, "bio")]/div`, func(e *colly.XMLElement) {
		bio = e.Text
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", nick)); err != nil {
		if err.Error() == "Not Found" {
			return "", UserNotFound
		}

		return "", err
	}

	return bio, nil
}
