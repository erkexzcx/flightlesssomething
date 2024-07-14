package flightlesssomething

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"html/template"
	"io/fs"
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
	store := gormsessions.NewStore(db, true, []byte(c.SessionSecret))
	db.AutoMigrate(&Benchmark{})

	// Setup gin //

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(sessions.Sessions("mysession", store))

	// Parse the embedded templates
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))
	r.SetHTMLTemplate(tmpl)

	// Serve static files
	r.GET("/static/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		file, err := staticFS.Open("static" + filepath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		defer file.Close()

		// Get file info
		fileInfo, err := file.Stat()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		// Read file content into a byte slice
		content, err := fs.ReadFile(staticFS, "static"+filepath)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		// Generate ETag based on file content
		hash := sha1.New()
		hash.Write(content)
		etag := hex.EncodeToString(hash.Sum(nil))

		// Set ETag and Cache-Control headers
		c.Header("ETag", etag)
		c.Header("Cache-Control", "public, max-age=3600")

		// Check if the ETag matches
		if match := c.GetHeader("If-None-Match"); match == etag {
			c.Status(http.StatusNotModified)
			return
		}

		// Serve the file with ETag and Last-Modified headers
		http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), bytes.NewReader(content))
	})

	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusTemporaryRedirect, "/benchmarks") })

	r.GET("/benchmarks", getBenchmarks)

	r.GET("/benchmark", getBenchmarkCreate)
	r.POST("/benchmark", postBenchmarkCreate)
	r.GET("/benchmark/:id", getBenchmark)
	r.DELETE("/benchmark/:id", deleteBenchmark)
	r.GET("/benchmark/:id/download", getBenchmarkDownload)

	r.GET("/user/:id", getUser)

	r.GET("/login", getLogin)
	r.GET("/login/callback", getLoginCallback)
	r.GET("/logout", getLogout)

	r.Run(c.Bind)
}
