package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HandleDebugCalc computes statistics from raw FPS/Frametime data for verification.
// This allows the /debugcalc page to compare frontend and backend calculations.
func HandleDebugCalc() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			FPS       []float64 `json:"fps"`
			Frametime []float64 `json:"frametime"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		if len(req.FPS) == 0 && len(req.Frametime) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of fps or frametime arrays must be provided"})
			return
		}

		type methodResults struct {
			FPS       *MetricStats `json:"fps,omitempty"`
			Frametime *MetricStats `json:"frametime,omitempty"`
		}

		result := struct {
			Linear   methodResults `json:"linear"`
			MangoHud methodResults `json:"mangohud"`
		}{}

		// Calculate FPS stats
		if len(req.Frametime) > 0 {
			// Derive FPS from frametime (the correct way)
			result.Linear.FPS = computeFPSFromFrametimeForMethod(req.Frametime, req.FPS, "linear")
			result.MangoHud.FPS = computeFPSFromFrametimeForMethod(req.Frametime, req.FPS, "mangohud")

			// Frametime stats
			result.Linear.Frametime = computeMetricStatsForMethod(req.Frametime, "linear")
			result.MangoHud.Frametime = computeMetricStatsForMethod(req.Frametime, "mangohud")
		} else if len(req.FPS) > 0 {
			result.Linear.FPS = computeMetricStatsForMethod(req.FPS, "linear")
			result.MangoHud.FPS = computeMetricStatsForMethod(req.FPS, "mangohud")
		}

		c.JSON(http.StatusOK, result)
	}
}
