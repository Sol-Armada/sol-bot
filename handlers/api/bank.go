package api

import (
	"net/http"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/sol-armada/admin/stores"
	"github.com/sol-armada/admin/transactions"
	"go.mongodb.org/mongo-driver/bson"
)

type GetBankBalanceResponse struct {
	Balance int32 `json:"balance"`
}

func GetBankBalance(c echo.Context) error {
	logger := log.WithFields(log.Fields{
		"endpoint": "GetBackBalance",
	})
	logger.Debug("getting bank balance")

	cur, err := stores.Transactions.List(bson.D{})
	if err != nil {
		logger.WithError(err).Error("getting transactions cursor")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	transactions := []*transactions.Transaction{}
	if err := cur.All(c.Request().Context(), &transactions); err != nil {
		logger.WithError(err).Error("getting transactions from collection")
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	balance := int32(0)
	for _, transaction := range transactions {
		balance += transaction.Amount
	}

	return c.JSON(http.StatusOK, GetBankBalanceResponse{
		Balance: balance,
	})
}
