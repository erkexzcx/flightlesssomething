package flightlesssomething

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func getUser(c *gin.Context) {
	session := sessions.Default(c)

	// Get "page" value
	page := c.DefaultQuery("page", "1")
	pageInt, _ := strconv.Atoi(page)
	if pageInt < 1 {
		pageInt = 1
	}

	id := c.Param("id")

	// Get user details
	var user User
	result := db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while fetching user details",
		})
		return
	}

	// Get benchmarks of the user
	var benchmarks []Benchmark
	tx := db.
		Where("user_id = ?", id).
		Order("created_at DESC").
		Offset((pageInt - 1) * BENCHMARKS_PER_PAGE).
		Limit(BENCHMARKS_PER_PAGE)
	result = tx.Find(&benchmarks)
	if result.Error != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while fetching benchmarks",
		})
		return
	}

	// Get total number of benchmarks of the user
	var benchmarksTotal int64
	tx = db.Where("user_id = ?", id).Model(&Benchmark{})
	result = tx.Count(&benchmarksTotal)
	if result.Error != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while counting benchmarks",
		})
		return
	}

	// Calculate pagination values
	prevPage := pageInt - 1
	nextPage := pageInt + 1
	totalPages := (int(benchmarksTotal) + BENCHMARKS_PER_PAGE - 1) / BENCHMARKS_PER_PAGE

	c.HTML(http.StatusOK, "user.tmpl", gin.H{
		"activePage": "user",
		"username":   session.Get("Username"),
		"userID":     session.Get("ID"),

		"benchmarks":      benchmarks,
		"benchmarksTotal": benchmarksTotal,

		"user": user,

		// Query parameters
		"page": pageInt,

		// Pagination values
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
	})
}
