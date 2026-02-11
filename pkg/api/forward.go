// api/forward.go
package api

import (
	"Rshell/pkg/connection/tcp"
	"Rshell/pkg/connection/websocket"
	"Rshell/pkg/logger"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func ForwardConnect(c *gin.Context) {
	var forward struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		Proxy   string `json:"proxy"`
	}
	if err := c.BindJSON(&forward); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": err.Error()})
		return
	}

	switch forward.Type {
	case "websocket":
		// 验证地址格式
		if forward.Address == "" {
			c.JSON(http.StatusOK, gin.H{"status": 400, "data": "address is required"})
			return
		}

		// 确保地址有正确的协议前缀

		// 配置正向连接
		config := &websocket.ForwardConfig{
			ServerURL:   "ws://" + forward.Address + "/ws",
			Socks5Proxy: forward.Proxy, // 使用传入的代理地址
			Timeout:     30 * time.Second,
			MaxRetries:  5,
			RetryDelay:  10 * time.Second,
			Reconnect:   true,
			Headers:     map[string]string{
				//"User-Agent": "Rshell-Forward-Client/1.0",
			},
		}

		// 启动正向客户端
		client, err := websocket.StartForwardClient(config)
		if err != nil {
			logger.Error("Failed to start forward client:", err)
			c.JSON(http.StatusOK, gin.H{
				"status": 500,
				"data":   fmt.Sprintf("Failed to connect: %v", err),
			})
			return
		}

		// 返回连接信息
		c.JSON(http.StatusOK, gin.H{
			"status": 200,
			"data": gin.H{
				"temp_uid":   client.UID,
				"server":     config.ServerURL,
				"proxy_used": config.Socks5Proxy != "",
				"message":    "Forward connection established. Waiting for client registration...",
			},
		})

		// 异步监控连接状态
		go monitorForwardConnection(client.UID, "websocket")
	case "tcp":
		// TCP正向连接
		handleTCPForward(forward.Address, forward.Proxy, c)
	default:
		c.JSON(http.StatusOK, gin.H{"status": 400, "data": "unsupported connection type"})
	}
}

// handleTCPForward 处理TCP正向连接
func handleTCPForward(address, proxy string, c *gin.Context) {
	// 验证TCP地址格式（host:port）
	if !strings.Contains(address, ":") {
		// 如果没有端口，添加默认端口
		address = address + ":8080"
	}

	// 配置TCP正向连接
	config := &tcp.TCPForwardConfig{
		ServerAddress: address,
		Socks5Proxy:   proxy, // 使用传入的代理地址
		Timeout:       30 * time.Second,
		MaxRetries:    5,
		RetryDelay:    10 * time.Second,
		Reconnect:     true,
	}

	// 启动TCP正向客户端
	client, err := tcp.StartTCPForwardClient(config)
	if err != nil {
		logger.Error("Failed to start TCP forward client:", err)
		c.JSON(http.StatusOK, gin.H{
			"status": 500,
			"data":   fmt.Sprintf("Failed to connect: %v", err),
		})
		return
	}

	// 返回连接信息
	c.JSON(http.StatusOK, gin.H{
		"status": 200,
		"data": gin.H{
			"type":       "tcp",
			"temp_uid":   client.UID,
			"server":     config.ServerAddress,
			"proxy_used": config.Socks5Proxy != "",
			"message":    "TCP forward connection established. Waiting for client registration...",
		},
	})

	// 异步监控连接状态
	go monitorForwardConnection(client.UID, "tcp")
}

// monitorForwardConnection 监控正向连接状态
func monitorForwardConnection(uid, connType string) {
	// 等待一段时间检查连接状态
	time.Sleep(3 * time.Second)

	// 根据连接类型检查是否还是临时UID
	if connType == "websocket" {
		if strings.HasPrefix(uid, "temp_") {
			logger.Warn("WebSocket forward connection still using temporary UID after 3 seconds:", uid)
		} else {
			logger.Info("WebSocket forward connection established with client:", uid)
		}
	} else if connType == "tcp" {
		if strings.HasPrefix(uid, "tcp_temp_") {
			logger.Warn("TCP forward connection still using temporary UID after 3 seconds:", uid)
		} else {
			logger.Info("TCP forward connection established with client:", uid)
		}
	}
}
