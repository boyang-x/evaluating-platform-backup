package handler

import "github.com/gin-gonic/gin"

func newEnterpriseRoleRouter() *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_role", "enterprise")
		c.Next()
	})
	return router
}
