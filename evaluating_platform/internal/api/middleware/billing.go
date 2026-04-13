package middleware

import (
	"github.com/gin-gonic/gin"
)

// BillingMeter 计费计量中间件（记录 API 调用）
func BillingMeter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// TODO: 记录 API 调用，写入计费流水
		// 工具调用的计费在 executor 层处理
	}
}
