package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 全局配置
const (
	PORT         = "8080"
	STATIC_DIR   = "./"   // 前端文件所在目录（indexnew.html 需放在此目录）
	API_BASE_URL = "/api" // 接口基础路径
	WS_PATH      = "/ws"  // WebSocket 路径
)

// 错误码定义（与前端保持一致）
const (
	CODE_SUCCESS                 = 200
	CODE_PARAM_ERROR             = 1000
	CODE_NOT_LOGIN               = 1001
	CODE_ACCOUNT_ERROR           = 1002
	CODE_NO_PERMISSION           = 1003
	CODE_RESOURCE_NOT_FOUND      = 1004
	CODE_SERVER_BUSY             = 1005
	CODE_UNKNOWN_ERROR           = 1006
	CODE_ACCOUNT_NOT_EXIST       = 2000
	CODE_ACCOUNT_FROZEN          = 2001
	CODE_BALANCE_NOT_ENOUGH      = 2002
	CODE_TARGET_ACCOUNT_ABNORMAL = 2003
	CODE_ACCOUNT_LIMIT           = 2004
	CODE_RISK_CONTROL_REJECT     = 2005
)

// 响应结构体（统一返回格式）
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 账户信息结构体
type Account struct {
	AccountID string  `json:"accountId"`
	UserName  string  `json:"userName"`
	Balance   float64 `json:"balance"`
	Status    string  `json:"status"` // normal/frozen
	CreateAt  string  `json:"createAt"`
}

// 存款请求结构体
type DepositRequest struct {
	AccountID string  `json:"accountId"`
	Amount    float64 `json:"amount"`
}

// 转账请求结构体
type TransferRequest struct {
	FromAccount string  `json:"fromAccount"`
	ToAccount   string  `json:"toAccount"`
	Amount      float64 `json:"amount"`
}

// WebSocket 消息结构体
type WsMessage struct {
	Type       string  `json:"type"` // balanceUpdate/transactionAlert
	NewBalance float64 `json:"newBalance,omitempty"`
	Message    string  `json:"message,omitempty"`
}

// 全局变量
var (
	// 模拟数据库 - 存储账户信息（实际项目应使用真实数据库）
	accounts = map[string]Account{
		"8001234567": {
			AccountID: "8001234567",
			UserName:  "张三",
			Balance:   12580.00,
			Status:    "normal",
			CreateAt:  "2023-06-15",
		},
		// 可添加测试收款账户
		"8001234568": {
			AccountID: "8001234568",
			UserName:  "李四",
			Balance:   5000.00,
			Status:    "normal",
			CreateAt:  "2023-07-20",
		},
	}
	accountsMutex sync.RWMutex // 账户操作互斥锁

	// WebSocket 相关
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许跨域（开发环境）
		},
	}
	clients      = make(map[*websocket.Conn]bool) // 在线客户端
	clientsMutex sync.RWMutex
)

// 初始化函数
func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("服务初始化完成，监听端口: %s", PORT)
	log.Printf("静态文件目录: %s", STATIC_DIR)
	// 打印测试账户信息，方便测试人员查看
	printTestAccounts()
}

// 主函数
func main() {
	// 路由注册
	mux := http.NewServeMux()

	// 1. 静态文件服务（解决 indexnew.html 404 问题）
	fileServer := http.FileServer(http.Dir(STATIC_DIR))
	mux.Handle("/", http.StripPrefix("/", fileServer))

	// 2. API 接口路由
	mux.HandleFunc(API_BASE_URL+"/account", getAccountInfo)  // 获取账户信息
	mux.HandleFunc(API_BASE_URL+"/deposit", handleDeposit)   // 存款接口
	mux.HandleFunc(API_BASE_URL+"/transfer", handleTransfer) // 转账接口

	// 3. WebSocket 路由
	mux.HandleFunc(WS_PATH, handleWebSocket)

	// 启动 HTTP 服务
	server := &http.Server{
		Addr:         ":" + PORT,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("服务启动成功，访问地址: http://localhost:%s", PORT)
	log.Println("=" + strings.Repeat("-", 50) + "=")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// -------------------------- API 接口实现 --------------------------

// 获取账户信息
func getAccountInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendResponse(w, CODE_PARAM_ERROR, "不支持的请求方法", nil)
		return
	}

	// 模拟获取当前登录用户的账户（实际项目应从 Token/Session 中获取）
	accountID := "8001234567" // 默认测试账户

	accountsMutex.RLock()
	account, exists := accounts[accountID]
	accountsMutex.RUnlock()

	if !exists {
		sendResponse(w, CODE_ACCOUNT_NOT_EXIST, "账户不存在", nil)
		return
	}

	// 终端提示：账户信息查询
	log.Println("\n[📋 账户查询]")
	log.Printf("查询时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("账户ID: %s", account.AccountID)
	log.Printf("用户名: %s", account.UserName)
	log.Printf("当前余额: %.2f 元", account.Balance)
	log.Printf("账户状态: %s", account.Status)
	log.Println("-" + strings.Repeat("-", 50) + "-")

	sendResponse(w, CODE_SUCCESS, "获取账户信息成功", account)
}

// 处理存款请求
func handleDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendResponse(w, CODE_PARAM_ERROR, "不支持的请求方法", nil)
		return
	}

	// 解析请求体
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, CODE_PARAM_ERROR, "请求参数格式错误", nil)
		return
	}

	// 参数校验
	if req.AccountID == "" || req.Amount <= 0 {
		sendResponse(w, CODE_PARAM_ERROR, "账户ID不能为空，存款金额必须大于0", nil)
		return
	}

	accountsMutex.Lock()
	defer accountsMutex.Unlock()

	// 检查账户是否存在
	account, exists := accounts[req.AccountID]
	if !exists {
		sendResponse(w, CODE_ACCOUNT_NOT_EXIST, "存款账户不存在", nil)
		return
	}

	// 检查账户状态
	if account.Status != "normal" {
		sendResponse(w, CODE_ACCOUNT_FROZEN, "账户已冻结，无法存款", nil)
		return
	}

	// 记录操作前余额
	oldBalance := account.Balance
	// 执行存款操作
	account.Balance += req.Amount
	accounts[req.AccountID] = account

	// 构造返回数据
	responseData := map[string]interface{}{
		"accountId":  req.AccountID,
		"amount":     req.Amount,
		"oldBalance": oldBalance,
		"newBalance": account.Balance,
		"time":       time.Now().Format("2006-01-02 15:04:05"),
	}

	// 发送 WebSocket 通知（实时更新余额）
	sendWsMessage(WsMessage{
		Type:       "balanceUpdate",
		NewBalance: account.Balance,
	})

	// 发送交易提醒
	sendWsMessage(WsMessage{
		Type:    "transactionAlert",
		Message: fmt.Sprintf("存款成功：+%.2f元，当前余额：%.2f元", req.Amount, account.Balance),
	})

	// 终端提示：存款操作详情（高亮显示金额）
	log.Println("\n[💰 存款操作]")
	log.Printf("操作时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("账户ID: %s", req.AccountID)
	log.Printf("用户名: %s", account.UserName)
	log.Printf("存款金额: \033[1;32m%.2f 元\033[0m", req.Amount) // 绿色高亮
	log.Printf("操作前余额: %.2f 元", oldBalance)
	log.Printf("操作后余额: \033[1;36m%.2f 元\033[0m", account.Balance) // 青色高亮
	log.Printf("操作状态: \033[1;32m成功\033[0m")                       // 绿色高亮
	log.Println("-" + strings.Repeat("-", 50) + "-")

	sendResponse(w, CODE_SUCCESS, "存款成功", responseData)
}

// 处理转账请求
func handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendResponse(w, CODE_PARAM_ERROR, "不支持的请求方法", nil)
		return
	}

	// 解析请求体
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, CODE_PARAM_ERROR, "请求参数格式错误", nil)
		return
	}

	// 参数校验
	if req.FromAccount == "" || req.ToAccount == "" || req.Amount <= 0 {
		sendResponse(w, CODE_PARAM_ERROR, "转出账户、收款账户不能为空，转账金额必须大于0", nil)
		return
	}

	if req.FromAccount == req.ToAccount {
		sendResponse(w, CODE_PARAM_ERROR, "不能向自己转账", nil)
		return
	}

	accountsMutex.Lock()
	defer accountsMutex.Unlock()

	// 检查转出账户
	fromAccount, fromExists := accounts[req.FromAccount]
	if !fromExists {
		sendResponse(w, CODE_ACCOUNT_NOT_EXIST, "转出账户不存在", nil)
		return
	}

	// 检查转出账户状态
	if fromAccount.Status != "normal" {
		sendResponse(w, CODE_ACCOUNT_FROZEN, "转出账户已冻结，无法转账", nil)
		return
	}

	// 检查余额是否充足
	if fromAccount.Balance < req.Amount {
		// 终端提示：转账失败（余额不足）
		log.Println("\n[❌ 转账操作 - 失败]")
		log.Printf("操作时间: %s", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("转出账户ID: %s", req.FromAccount)
		log.Printf("转出用户名: %s", fromAccount.UserName)
		log.Printf("收款账户ID: %s", req.ToAccount)
		log.Printf("转账金额: %.2f 元", req.Amount)
		log.Printf("当前余额: %.2f 元", fromAccount.Balance)
		log.Printf("失败原因: 余额不足")
		log.Println("-" + strings.Repeat("-", 50) + "-")

		sendResponse(w, CODE_BALANCE_NOT_ENOUGH, "余额不足，无法完成转账", nil)
		return
	}

	// 检查收款账户
	toAccount, toExists := accounts[req.ToAccount]
	if !toExists {
		// 终端提示：转账失败（收款账户不存在）
		log.Println("\n[❌ 转账操作 - 失败]")
		log.Printf("操作时间: %s", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("转出账户ID: %s", req.FromAccount)
		log.Printf("转出用户名: %s", fromAccount.UserName)
		log.Printf("收款账户ID: %s", req.ToAccount)
		log.Printf("转账金额: %.2f 元", req.Amount)
		log.Printf("失败原因: 收款账户不存在")
		log.Println("-" + strings.Repeat("-", 50) + "-")

		sendResponse(w, CODE_TARGET_ACCOUNT_ABNORMAL, "收款账户不存在", nil)
		return
	}

	// 检查收款账户状态
	if toAccount.Status != "normal" {
		// 终端提示：转账失败（收款账户异常）
		log.Println("\n[❌ 转账操作 - 失败]")
		log.Printf("操作时间: %s", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("转出账户ID: %s", req.FromAccount)
		log.Printf("转出用户名: %s", fromAccount.UserName)
		log.Printf("收款账户ID: %s", req.ToAccount)
		log.Printf("收款用户名: %s", toAccount.UserName)
		log.Printf("转账金额: %.2f 元", req.Amount)
		log.Printf("失败原因: 收款账户状态异常（%s）", toAccount.Status)
		log.Println("-" + strings.Repeat("-", 50) + "-")

		sendResponse(w, CODE_TARGET_ACCOUNT_ABNORMAL, "收款账户状态异常", nil)
		return
	}

	// 记录操作前余额
	fromOldBalance := fromAccount.Balance
	toOldBalance := toAccount.Balance

	// 执行转账操作
	fromAccount.Balance -= req.Amount
	toAccount.Balance += req.Amount
	accounts[req.FromAccount] = fromAccount
	accounts[req.ToAccount] = toAccount

	// 构造返回数据
	responseData := map[string]interface{}{
		"fromAccount": req.FromAccount,
		"toAccount":   req.ToAccount,
		"amount":      req.Amount,
		"newBalance":  fromAccount.Balance,
		"time":        time.Now().Format("2006-01-02 15:04:05"),
	}

	// 发送 WebSocket 通知（更新转出账户余额）
	sendWsMessage(WsMessage{
		Type:       "balanceUpdate",
		NewBalance: fromAccount.Balance,
	})

	// 发送交易提醒
	sendWsMessage(WsMessage{
		Type:    "transactionAlert",
		Message: fmt.Sprintf("转账成功：-%.2f元，当前余额：%.2f元", req.Amount, fromAccount.Balance),
	})

	// 终端提示：转账操作详情（高亮显示关键信息）
	log.Println("\n[🔄 转账操作]")
	log.Printf("操作时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("转出账户ID: %s", req.FromAccount)
	log.Printf("转出用户名: %s", fromAccount.UserName)
	log.Printf("收款账户ID: %s", req.ToAccount)
	log.Printf("收款用户名: %s", toAccount.UserName)
	log.Printf("转账金额: \033[1;31m%.2f 元\033[0m", req.Amount) // 红色高亮
	log.Printf("转出账户 - 操作前: %.2f 元 → 操作后: \033[1;36m%.2f 元\033[0m", fromOldBalance, fromAccount.Balance)
	log.Printf("收款账户 - 操作前: %.2f 元 → 操作后: \033[1;36m%.2f 元\033[0m", toOldBalance, toAccount.Balance)
	log.Printf("操作状态: \033[1;32m成功\033[0m") // 绿色高亮
	log.Println("-" + strings.Repeat("-", 50) + "-")

	sendResponse(w, CODE_SUCCESS, "转账成功", responseData)
}

// -------------------------- WebSocket 实现 --------------------------

// 处理 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 连接为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	// 终端提示：WebSocket 连接状态
	log.Println("\n[📡 WebSocket 连接]")
	log.Printf("连接时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("客户端地址: %s", conn.RemoteAddr())
	log.Printf("连接状态: 成功建立")
	log.Println("-" + strings.Repeat("-", 50) + "-")

	// 添加客户端到连接池
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	// 延迟关闭连接
	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		// 终端提示：WebSocket 断开连接
		log.Println("\n[📡 WebSocket 连接]")
		log.Printf("断开时间: %s", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("客户端地址: %s", conn.RemoteAddr())
		log.Printf("连接状态: 已断开")
		log.Println("-" + strings.Repeat("-", 50) + "-")
		conn.Close()
	}()

	// 循环读取客户端消息（保持连接）
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 读取错误: %v", err)
			}
			break
		}
	}
}

// 发送 WebSocket 消息给所有在线客户端
func sendWsMessage(msg WsMessage) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket 消息序列化失败: %v", err)
		return
	}

	// 终端提示：WebSocket 消息推送
	log.Println("\n[📤 WebSocket 消息推送]")
	log.Printf("推送时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("消息类型: %s", msg.Type)
	if msg.Type == "balanceUpdate" {
		log.Printf("更新余额: %.2f 元", msg.NewBalance)
	} else {
		log.Printf("消息内容: %s", msg.Message)
	}
	log.Printf("在线客户端数: %d", len(clients))
	log.Println("-" + strings.Repeat("-", 50) + "-")

	// 发送给所有客户端
	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket 消息发送失败（客户端: %s）: %v", client.RemoteAddr(), err)
			client.Close()
			delete(clients, client)
		}
	}
}

// -------------------------- 工具函数 --------------------------

// 发送统一格式响应
func sendResponse(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK) // 所有响应都返回 200，业务错误通过 code 区分

	response := Response{
		Code:    code,
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("响应发送失败: %v", err)
	}
}

// 检查文件是否存在（用于调试静态文件服务）
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// 打印测试账户信息
func printTestAccounts() {
	log.Println("\n[📋 测试账户信息]")
	accountsMutex.RLock()
	defer accountsMutex.RUnlock()
	for _, acc := range accounts {
		log.Printf("账户ID: %s | 用户名: %s | 初始余额: %.2f 元 | 状态: %s",
			acc.AccountID, acc.UserName, acc.Balance, acc.Status)
	}
	log.Println("-" + strings.Repeat("-", 50) + "-")
}
