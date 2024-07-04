package flightlesssomething

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/sessions"
	gormsessions "github.com/gin-contrib/sessions/gorm"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

var (
	// GORM database object
	db *gorm.DB

	// Discord conf object
	discordConf *oauth2.Config

	// Benchmarks directory
	benchmarksDir string
)

func Start(c *Config) {
	// Setup data dir //

	_, err := os.Stat(c.DataDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(c.DataDir, 0755)
		if err != nil {
			panic("Failed to create data dir: " + err.Error())
		}
	} else if err != nil {
		panic("Failed to check data dir: " + err.Error())
	}

	benchmarksDir = filepath.Join(c.DataDir, "benchmarks")
	_, err = os.Stat(benchmarksDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(benchmarksDir, 0755)
		if err != nil {
			panic("Failed to create benchmarks dir: " + err.Error())
		}
	} else if err != nil {
		panic("Failed to check benchmarks dir: " + err.Error())
	}

	// Setup Discord OAuth2 //

	discordConf = &oauth2.Config{
		Endpoint:     discord.Endpoint,
		Scopes:       []string{discord.ScopeIdentify},
		RedirectURL:  c.DiscordRedirectURL,
		ClientID:     c.DiscordClientID,
		ClientSecret: c.DiscordClientSecret,
	}

	// Setup gorm (database) //

	db, err = gorm.Open(sqlite.Open(filepath.Join(c.DataDir, "database.db")), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	store := gormsessions.NewStore(db, true, []byte("secret"))
	db.AutoMigrate(&Benchmark{})

	// Setup gin //

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(sessions.Sessions("mysession", store))

	// Parse the embedded templates
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))
	r.SetHTMLTemplate(tmpl)

	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/benchmarks") })

	r.GET("/benchmarks", getBenchmarks)

	r.GET("/benchmark", getBenchmarkCreate)
	r.POST("/benchmark", postBenchmarkCreate)
	r.GET("/benchmark/:id", getBenchmark)
	r.DELETE("/benchmark/:id", deleteBenchmark)

	r.GET("/user/:id", getUser)

	r.GET("/login", getLogin)
	r.GET("/login/callback", getLoginCallback)
	r.GET("/logout", getLogout)

	r.Run(c.Bind)
}
