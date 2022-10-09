package rsi

import (
	"errors"
	"fmt"

	"github.com/gocolly/colly/v2"
)

var (
	UserNotFound error = errors.New("user was not found")
)

func GetPrimaryOrg(username string) (string, error) {
	c := colly.NewCollector()

	var r string
	var err error
	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = UserNotFound
		}
	})

	c.OnXML(`//div[contains(@class, "main-org")]//div[@class="info"]//span[contains(text(), "SID")]/following-sibling::strong`, func(e *colly.XMLElement) {
		r = e.Text
	})

	if err := c.Visit(fmt.Sprintf("https://robertsspaceindustries.com/citizens/%s", username)); err != nil {
		if err.Error() == "Not Found" {
			return "", UserNotFound
		}

		return "", err
	}

	return r, err
}
