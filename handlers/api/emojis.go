package api

import (
	"net/http"

	"github.com/apex/log"
	"github.com/bwmarrin/discordgo"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/bot"
)

type GetEmojisResponse struct {
	Emojis []*discordgo.Emoji `json:"emojis"`
}

func GetEmojisHandler(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetEmojis",
	})
	logger.Debug("getting emojis")

	b, err := bot.GetBot()
	if err != nil {
		logger.WithError(err).Error("getting bot")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	emojis, err := b.GetEmojis()
	if err != nil {
		logger.WithError(err).Error("getting emojis")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, &GetEmojisResponse{
		Emojis: emojis,
	})
}
