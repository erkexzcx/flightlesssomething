package flightlesssomething

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const BENCHMARKS_PER_PAGE = 10

func getBenchmarks(c *gin.Context) {
	session := sessions.Default(c)

	// Get "query" value
	query := c.Query("query")

	// Get "page" value
	page := c.DefaultQuery("page", "1")
	pageInt, _ := strconv.Atoi(page)
	if pageInt < 1 {
		pageInt = 1
	}

	// Get benchmarks according to query
	var benchmarks []Benchmark
	tx := db.
		Preload("User").
		Order("created_at DESC").
		Offset((pageInt - 1) * BENCHMARKS_PER_PAGE).
		Limit(BENCHMARKS_PER_PAGE)
	if query != "" {
		tx = tx.Where("title LIKE ?", "%"+query+"%").Or("description LIKE ?", "%"+query+"%")
	}
	result := tx.Find(&benchmarks)
	if result.Error != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while fetching benchmarks",
		})
		return
	}

	// Get total number of benchmarks matching the query
	var benchmarksTotal int64
	tx = db.Model(&Benchmark{})
	if query != "" {
		tx = tx.Where("title LIKE ?", "%"+query+"%").Or("description LIKE ?", "%"+query+"%")
	}
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

	c.HTML(http.StatusOK, "benchmarks.tmpl", gin.H{
		"activePage": "benchmarks",
		"username":   session.Get("Username"),
		"userID":     session.Get("ID"),

		"benchmarks":      benchmarks,
		"benchmarksTotal": benchmarksTotal,

		// Query parameters
		"query": query,
		"page":  pageInt,

		// Pagination values
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"totalPages": totalPages,
	})
}

func getBenchmarkCreate(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("Username") == "" {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Please authenticate to create a benchmark",
		})
	}

	c.HTML(http.StatusOK, "benchmark_create.tmpl", gin.H{
		"activePage": "benchmark",
		"username":   session.Get("Username"),
		"userID":     session.Get("ID"),
	})
}

func postBenchmarkCreate(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("Username") == "" {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Please authenticate to create a benchmark",
		})
		return
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if len(title) > 100 || title == "" {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Title must not be empty or exceed 100 characters",
		})
		return
	}

	description := strings.TrimSpace(c.PostForm("description"))
	if len(description) > 500 || description == "" {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Description must not be empty or exceed 500 characters",
		})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while parsing form data",
		})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "No files uploaded",
		})
		return
	}
	if len(files) > 30 {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Too many files uploaded (max 30)",
		})
		return
	}

	// Read CSV files
	// Store to disk only when DB record is created successfully
	csvFiles, csvSpecs, err := readCSVFiles(files)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while reading CSV files: " + err.Error(),
		})
		return
	}

	benchmark := Benchmark{
		UserID:      session.Get("ID").(uint),
		Title:       title,
		Description: description,

		SpecDistro:    csvSpecs.Distro,
		SpecCPU:       csvSpecs.Distro,
		SpecGPU:       csvSpecs.GPU,
		SpecRAM:       csvSpecs.RAM,
		SpecKernel:    csvSpecs.Kernel,
		SpecScheduler: csvSpecs.Scheduler,
	}

	result := db.Create(&benchmark)
	if result.Error != nil {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while creating benchmark: " + result.Error.Error(),
		})
		return
	}

	// Store CSV files to disk
	err = storeBenchmarkData(csvFiles, benchmark.ID)
	if err != nil {
		db.Unscoped().Delete(&benchmark) // Hard delete from DB
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Error occurred while storing benchmark data: " + err.Error(),
		})
		return
	}

	// Redirect to the newly created benchmark using GET request
	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/benchmark/%d", benchmark.ID))
}

func deleteBenchmark(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("Username") == "" {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Please authenticate to create a benchmark",
		})
		return
	}

	// Get benchmark ID from the path
	id := c.Param("id")

	// Check if user owns the benchmark
	var benchmark Benchmark
	result := db.First(&benchmark, id)
	if result.Error != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Internal server error occurred: " + result.Error.Error(),
		})
		return
	}
	if benchmark.UserID != session.Get("ID") {
		c.HTML(http.StatusUnauthorized, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "You do not own this benchmark",
		})
		return
	}

	// Delete benchmark from DB
	result = db.Delete(&benchmark)
	if result.Error != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Internal server error occurred: " + result.Error.Error(),
		})
		return
	}

	// Delete benchmark data from disk
	err := deleteBenchmarkData(benchmark.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Internal server error occurred: " + err.Error(),
		})
		return
	}

	// Redirect to the benchmarks page
	c.Header("HX-Redirect", "/benchmarks")
	c.JSON(http.StatusOK, gin.H{
		"message": "Benchmark deleted successfully",
	})
}

func getBenchmark(c *gin.Context) {
	session := sessions.Default(c)

	// Get benchmark ID from the path
	id := c.Param("id")

	// Get benchmark details
	intID, err := strconv.Atoi(id)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
			"activePage": "error",
			"username":   session.Get("Username"),
			"userID":     session.Get("ID"),

			"errorMessage": "Internal server error occurred: " + err.Error(),
		})
		return
	}

	var benchmark Benchmark
	benchmark.ID = uint(intID)

	var csvFiles []*CSVFile
	var errCSV, errDB error
	errHTTPStatus := http.StatusInternalServerError

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		csvFiles, errCSV = retrieveBenchmarkData(benchmark.ID)
	}()

	go func() {
		defer wg.Done()
		result := db.Preload("User").First(&benchmark, id)
		if result.Error != nil {
			errDB = result.Error
			return
		}
		if result.RowsAffected == 0 {
			errDB = fmt.Errorf("Benchmark not found")
			errHTTPStatus = http.StatusNotFound
			return
		}
	}()

	wg.Wait()

	err = errDB
	if err == nil {
		err = errCSV
	}
	if err != nil {
		c.HTML(errHTTPStatus, "error.tmpl", gin.H{
			"activePage":   "error",
			"username":     session.Get("Username"),
			"userID":       session.Get("ID"),
			"errorMessage": "Error occurred: " + errDB.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "benchmark.tmpl", gin.H{
		"activePage": "benchmark",
		"username":   session.Get("Username"),
		"userID":     session.Get("ID"),

		"benchmark":     benchmark,
		"benchmarkData": csvFiles,
	})
}
