// Example call GITEA_HOST="http://gitea.local.io" GITEA_TOKEN=a2e4fa854aa4b989ca6b46b6e589c8eba50492dc go run main.go

package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	limit "github.com/aviddiviner/gin-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

func createClient() *gitea.Client {
	client := gitea.NewClient(os.Getenv("GITEA_HOST"), os.Getenv("GITEA_TOKEN"))
	return client
}

func buildName(user *User) string {
	return user.Prefix + "-" + user.Name
}

func createUser(user *User) (*gitea.User, error) {
	client := createClient()
	idV4, _ := uuid.NewV4()
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
	rateLimit, _ := strconv.Atoi(os.Getenv("GIT_API_USER_RATE_LIMIT"))
	if rateLimit == 0 {
		rateLimit = 5 // some default
	}
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE"},
		AllowHeaders:     []string{"Cache-Control", "Accept", "Authorization", "Accept-Encoding", "Access-Control-Request-Headers", "User-Agent", "Access-Control-Request-Method", "Pragma", "Connection", "Host", "Content-Type"},
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
		// TODO probably it can be applied to ssh endpoints only
		userRoute.Use(limit.MaxAllowed(rateLimit))
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
		// userRoute.DELETE("/:prefix/:name", func(c *gin.Context) {
		// 	client := createClient()
		// 	username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
		// 	giteaUser, err := client.GetUserInfo(username)
		// 	userRepos, err := client.ListUserRepos(username)
		// 	for i:=0; i< len(userRepos); i++{
		// 		repo:= userRepos[i];
		// 		client.DeleteRepo(username, repo.repoName)
		// 	}
		// 	outputUser(c, giteaUser, err)
		// })

		userRoute.GET("/:prefix/:name/repos", func(c *gin.Context) {
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			client := createClient()
			client.SetSudo(username)
			userRepos, err := client.ListUserRepos(username)
			outputRepos(c, userRepos, err)
		})

		userRoute.PUT("/:prefix/:name/repos/:repoName", func(c *gin.Context) {
			client := createClient()
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			repo, err := client.AdminCreateRepo(username, gitea.CreateRepoOption{
				Name:    c.Param("repoName"),
				Private: true,
			})
			outputRepo(c, repo, err)
		})
		userRoute.GET("/:prefix/:name/repos/:repoName/*filepath", func(c *gin.Context) {
			client := createClient()
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			file, err := client.GetFile(username, c.Param("repoName"), "master", "/lunchbadger.json")
			if err != nil {
				fmt.Println(err)
			}
			c.JSON(200, gin.H{"data": file})
		})

		userRoute.PUT("/:prefix/:name/repos/:repoName/hook", func(c *gin.Context) {
			client := createClient()
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			// h, _ := client.GetRepoHook(username, "dev", 0)
			// fmt.Printf("%j", h)
			var createHookRx createHookRequest
			c.BindJSON(&createHookRx)

			if createHookRx.URL == "" {
				createHookRx.URL = "http://configstore.default/hook"
			}
			repo := c.Param("repoName")
			hooks, _ := client.ListRepoHooks(username, repo)
			registerHook := true
			for i := 0; i < len(hooks); i++ {
				hook := hooks[i]
				if hook.Config["url"] == createHookRx.URL {
					registerHook = false
				}
			}
			if registerHook {
				fmt.Printf("Registering hook for repo %s/%s to call %s", username, repo, createHookRx.URL)
				hookInfo, err := client.CreateRepoHook(username, repo, gitea.CreateHookOption{
					Type:   "gitea",
					Active: true,
					Events: []string{"push"},
					Config: map[string]string{
						"url":          createHookRx.URL,
						"content_type": "json",
					},
				})
				if err != nil {
					fmt.Println(err)
				}
				fmt.Println(hookInfo)
			}
			c.JSON(200, gin.H{"ok": true})
		})

		userRoute.POST("/:prefix/:name/ssh", func(c *gin.Context) {
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			client := createClient()
			var keyRx addKeyRequest
			bindErr := c.BindJSON(&keyRx)
			if bindErr == nil {
				fmt.Println(keyRx)
				var title string
				if keyRx.Type == "" {
					if keyRx.Title != "" {
						title = keyRx.Title
					} else {
						id, _ := uuid.NewV4()
						title = id.String()
					}
				} else {
					title = "lunchbadger-internal-" + keyRx.Type
					keys, listErr := client.ListPublicKeys(username)
					if listErr == nil {
						for i := 0; i < len(keys); i++ {
							// TODO: check for "LB gen" is for backwards compatibility, remove once not needed
							if strings.Contains(keys[i].Title, title) || strings.Contains(keys[i].Title, "LB gen") {
								fmt.Println("Removing KEY ", keys[i])
								go client.DeletePublicKey(keys[i].ID)
							}
						}
					} else {
						fmt.Println(listErr)
					}
				}

				pk, err := client.AdminCreateUserPublicKey(username, gitea.CreateKeyOption{
					Key:   keyRx.PublicKey,
					Title: title,
				})
				if err != nil {
					errorString := err.Error()
					// This is a string with different possible formats
					fmt.Println(errorString)

					// 422 has JSON in string like `422 Unprocessable Entity: {"message":"Key content has been used as non-deploy key","url":"https://godoc.org/github.com/go-gitea/go-sdk/gitea"}`
					// The code is to extract message
					if isValidationErr := strings.Contains(errorString, "422"); isValidationErr {
						res := regexp.MustCompile(":\\\"(.*)\\\",").FindStringSubmatch(errorString)
						if len(res) > 1 {
							errorString = res[1]
						}
						fmt.Println(errorString)
					}

					c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": errorString})
				} else {
					c.JSON(200, gin.H{"result": pk})
				}
			} else {
				c.JSON(400, gin.H{"err": bindErr})
			}

		})

		userRoute.GET("/:prefix/:name/ssh", func(c *gin.Context) {
			username := buildName(&User{Name: c.Param("name"), Prefix: c.Param("prefix")})
			fmt.Println(username)
			keys, err := createClient().ListPublicKeys(username)
			fmt.Println(err)
			filteredKeys := make([]*gitea.PublicKey, 0)
			for _, k := range keys {
				if !strings.Contains(k.Title, "lunchbadger-internal") {
					filteredKeys = append(filteredKeys, k)
				}
			}
			c.JSON(200, gin.H{"publicKeys": filteredKeys})
		})
		userRoute.DELETE("/:prefix/:name/ssh/:keyId", func(c *gin.Context) {
			keyID, err := strconv.ParseInt(c.Param("keyId"), 10, 64)
			fmt.Println(err)
			deleteResult := createClient().DeletePublicKey(keyID)
			if deleteResult == nil {
				c.JSON(204, nil)
			} else {
				c.JSON(400, gin.H{"result": "UNKNOWN_ID"})
			}

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

type addKeyRequest struct {
	PublicKey string `json:"publicKey"`
	Type      string `json:"type"`
	Title     string `json:"title"`
}

type createHookRequest struct {
	URL string `json:"url"`
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
