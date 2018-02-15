// Example call GITEA_HOST="http://gitea.local.io" GITEA_TOKEN=a2e4fa854aa4b989ca6b46b6e589c8eba50492dc go run main.go

package main

import (
	"encoding/hex"
	"os"

	"code.gitea.io/sdk/gitea"
	"github.com/LunchBadger/git-api/sshGen"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

var db = make(map[string]*sshGen.SSHKey)

func createClient() *gitea.Client {
	client := gitea.NewClient(os.Getenv("GITEA_HOST"), os.Getenv("GITEA_TOKEN"))
	return client
}

func buildName(user *User) string {
	return user.Prefix + "-" + user.Name
}

func createUser(user *User) (*gitea.User, error) {
	client := createClient()
	idV4 := uuid.NewV4()
	id := hex.EncodeToString(idV4[:])
	return client.AdminCreateUser(gitea.CreateUserOption{
		Username: buildName(user),
		Email:    user.Name + "@" + user.Prefix + ".com",
		Password: id, // nobody will know password
	})
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	// Ping test

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE"},
		AllowHeaders:     []string{"Cache-Control", "Accept", "Authorization", "Accept-Encoding", "Access-Control-Request-Headers", "User-Agent", "Access-Control-Request-Method", "Pragma", "Connection", "Host"},
		AllowCredentials: true,
	}))
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	searchRoute := r.Group("/search")
	{
		searchRoute.GET("/users", func(c *gin.Context) {
			client := createClient()
			users, err := client.SearchUsers(c.Query("q"), 100)
			outputUsers(c, users, err)
		})
	}

	userRoute := r.Group("/users")
	{
		userRoute.POST("/", func(c *gin.Context) {
			var user User
			if c.BindJSON(&user) == nil {
				giteaUser, err := createUser(&user)
				outputUser(c, giteaUser, err)
			}
		})
		userRoute.GET("/:prefix/:name", func(c *gin.Context) {
			client := createClient()
			giteaUser, err := client.GetUserInfo(buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")}))
			outputUser(c, giteaUser, err)
		})

		userRoute.GET("/:prefix/:name/repos", func(c *gin.Context) {
			client := createClient()
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			userRepos, err := client.ListUserRepos(username)
			outputRepos(c, userRepos, err)
		})

		userRoute.PUT("/:prefix/:name/repos/:repoName", func(c *gin.Context) {
			client := createClient()
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			repo, err := client.AdminCreateRepo(username, gitea.CreateRepoOption{
				Name: c.Param("repoName"),
			})
			outputRepo(c, repo, err)
		})

		userRoute.POST("/:prefix/:name/ssh", func(c *gin.Context) {
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			keys, _ := sshGen.Gen()
			client := createClient()

			pk, _ := client.AdminCreateUserPublicKey(username, gitea.CreateKeyOption{
				Key:   keys.PublicKey,
				Title: "LB gen " + uuid.NewV4().String(),
			})

			db[username] = keys
			c.JSON(200, gin.H{"keys": keys, "hash": pk})
		})

		// If key exists no need to generate new one

		userRoute.GET("/:prefix/:name/ssh", func(c *gin.Context) {
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			keys := db[username]
			c.JSON(200, gin.H{"keys": keys})
		})
	}
	return r
}

func main() {
	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}

// User Create request
type User struct {
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
}

func outputUser(c *gin.Context, user *gitea.User, err error) {
	if err == nil {
		c.JSON(200, gin.H{"user": user})
	} else {
		c.JSON(500, gin.H{"err": err})
	}
}

func outputUsers(c *gin.Context, users []*gitea.User, err error) {
	if err == nil {
		c.JSON(200, gin.H{"users": users})
	} else {
		c.JSON(500, gin.H{"err": err})
	}
}

func outputRepo(c *gin.Context, repo *gitea.Repository, err error) {
	if err == nil {
		c.JSON(200, gin.H{"repo": repo})
	} else {
		c.JSON(500, gin.H{"err": err})
	}
}

func outputRepos(c *gin.Context, repos []*gitea.Repository, err error) {
	if err == nil {
		c.JSON(200, gin.H{"repos": repos})
	} else {
		c.JSON(500, gin.H{"err": err})
	}
}
