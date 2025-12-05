package main

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ResCode uint32

const (
	Success             ResCode = 200
	ErrInvalidParams    ResCode = 1001
	ErrBalanceNotEnough ResCode = 2003
)

type Response struct {
	Code ResCode     `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

var account = struct {
	Balance float64 `json:"balance"`
}{
	Balance: 1000.00,
}

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/api/account", func(c *gin.Context) {
		c.JSON(http.StatusOK, Response{
			Code: Success,
			Msg:  "success",
			Data: gin.H{"balance": account.Balance},
		})
	})

	r.POST("/api/deposit", func(c *gin.Context) {
		var req struct {
			Amount    float64 `json:"amount"`
			AccountId string  `json:"accountId"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			c.JSON(http.StatusOK, Response{
				Code: ErrInvalidParams,
				Msg:  "参数错误",
			})
			return
		}

		account.Balance += req.Amount
		c.JSON(http.StatusOK, Response{
			Code: Success,
			Msg:  "存款成功",
			Data: gin.H{"newBalance": account.Balance},
		})
	})

	r.POST("/api/transfer", func(c *gin.Context) {
		var req struct {
			FromAccount string  `json:"fromAccount"`
			ToAccount   string  `json:"toAccount"`
			Amount      float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 || req.ToAccount == "" {
			c.JSON(http.StatusOK, Response{
				Code: ErrInvalidParams,
				Msg:  "参数错误",
			})
			return
		}

		if req.Amount > account.Balance {
			c.JSON(http.StatusOK, Response{
				Code: ErrBalanceNotEnough,
				Msg:  "余额不足",
			})
			return
		}

		account.Balance -= req.Amount
		c.JSON(http.StatusOK, Response{
			Code: Success,
			Msg:  "转账成功",
			Data: gin.H{"newBalance": account.Balance},
		})
	})

	r.GET("/ws", func(c *gin.Context) {
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code: ErrInvalidParams,
				Msg:  "WebSocket升级失败",
			})
			return
		}
		defer ws.Close()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				msg := map[string]interface{}{
					"type":       "balanceUpdate",
					"newBalance": account.Balance,
				}
				if err := ws.WriteJSON(msg); err != nil {
					return
				}

				notice := map[string]interface{}{
					"type":    "transactionAlert",
					"message": "您的账户于" + time.Now().Format("2006-01-02 15:04:05") + "发生一笔系统测试交易",
				}
				if err := ws.WriteJSON(notice); err != nil {
					return
				}
			}
		}
	})
	r.Run(":8080")
}
