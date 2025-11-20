package flightlesssomething

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func getLogin(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("ID") != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	discordState, err := getRandomString()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to generate random string",
		})
		return
	}

	session.Set("DiscordState", discordState)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, discordConf.AuthCodeURL(discordState))
}

func getLoginCallback(c *gin.Context) {
	session := sessions.Default(c)

	discordState := session.Get("DiscordState")
	if c.Query("state") != discordState {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Invalid Discord state",
		})
		return
	}

	token, err := discordConf.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to exchange code for token",
		})
		return
	}

	res, err := discordConf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to get user details from Discord",
		})
		return
	}
	if res.StatusCode != 200 {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to get user details from Discord",
		})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to read response body from Discord",
		})
		return
	}

	var discordUser DiscordUser
	err = json.Unmarshal([]byte(body), &discordUser)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to unmarshal response body from Discord",
		})
		return
	}

	user := User{
		DiscordID: discordUser.ID,
		Username:  discordUser.Username,
	}
	result := db.Model(&user).Where("discord_id = ?", discordUser.ID).Updates(&user)
	if result.Error != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Error occurred while updating user details",
		})
		return
	}
	if result.RowsAffected == 0 {
		result = db.Create(&user)
		if result.Error != nil {
			c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
				"errorMessage": "Error occurred while creating user",
			})
			return
		}
	}

	// Retrieve the updated user details from the database
	result = db.Model(&user).Where("discord_id = ?", discordUser.ID).First(&user)
	if result.Error != nil {
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Error occurred while retrieving user details",
		})
		return
	}

	session.Set("ID", user.ID)
	session.Set("Username", user.Username)
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func getLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func getRandomString() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func getAdminLogin(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("ID") != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	c.HTML(http.StatusOK, "admin_login.tmpl", gin.H{
		"activePage": "login",
		"error":      c.Query("error"),
	})
}

func postAdminLogin(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("ID") != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	// Check if admin credentials are configured
	if adminUsername == "" || adminPassword == "" {
		c.Redirect(http.StatusSeeOther, "/login/admin?error=Admin+login+not+configured")
		return
	}

	// Verify credentials
	if username != adminUsername || password != adminPassword {
		c.Redirect(http.StatusSeeOther, "/login/admin?error=Invalid+credentials")
		return
	}

	// Check if admin user exists, create if not
	var user User
	result := db.Where("username = ? AND is_admin = ?", adminUsername, true).First(&user)
	if result.Error != nil {
		// Create admin user
		user = User{
			Username: adminUsername,
			IsAdmin:  true,
		}
		result = db.Create(&user)
		if result.Error != nil {
			c.Redirect(http.StatusSeeOther, "/login/admin?error=Failed+to+create+admin+user")
			return
		}
	}

	// Set session
	session.Set("ID", user.ID)
	session.Set("Username", user.Username)
	session.Save()

	c.Redirect(http.StatusSeeOther, "/")
}
