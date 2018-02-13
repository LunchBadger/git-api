package main

import (
	"fmt"

	"encoding/hex"

	"code.gitea.io/sdk/gitea"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// Ping test
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	r.GET("/create", func(c *gin.Context) {
		idV4 := uuid.NewV4()
		id := hex.EncodeToString(idV4[:])
		client := gitea.NewClient("http://gitea.local.io", "a2e4fa854aa4b989ca6b46b6e589c8eba50492dc")
		user, err := client.AdminCreateUser(gitea.CreateUserOption{
			Username: id,
			Email:    id + "@xx.com",
			Password: "test",
		})
		if err != nil {
			fmt.Print(err)
			c.JSON(500, gin.H{"err": err})
		} else {
			c.JSON(200, gin.H{"user": user})
		}
	})

	// // Get user value
	// r.GET("/user/:name", func(c *gin.Context) {
	// 	user := c.Params.ByName("name")
	// 	value, ok := DB[user]
	// 	if ok {
	// 		c.JSON(200, gin.H{"user": user, "value": value})
	// 	} else {
	// 		c.JSON(200, gin.H{"user": user, "status": "no value"})
	// 	}
	// })
	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
