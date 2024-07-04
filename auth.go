package flightlesssomething

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
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
		log.Println(err)
		c.HTML(http.StatusInternalServerError, "error_raw.tmpl", gin.H{
			"errorMessage": "Failed to exchange code for token",
		})
		return
	}

	res, err := discordConf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")
	if err != nil {
		log.Println(err)
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
