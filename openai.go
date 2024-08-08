package flightlesssomething

import (
	"context"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

const systemMessage = `
You are given PC benchmark data of several runs. All this data is visible in the website in a form of charts and your goal is to provide insights.

You MUST:
1. Write at max 3 sections (headers) - "Top runs", "Issues" (optional) and "Summary".
2. In Issues section, Figure out if any of the run is significantly worse then others in the same benchmark. You MUST use ONLY the data provided to explain the difference, and your points must be based only on the data provided. If there are no issues - do not write this section. Do not make any guesses. Additional requirements: (a) validate if the same hardware/software was used (by using provided text fields, NOT the data), (b) do not speculate, but use numbers to back up your claims, (c) only write if it's an actual issue with FPS (everything else is just additional information).
3. In Top runs section, provide which run has the (average) "Highest FPS", which has the "Smoothest FPS" (LOWEST std.dev. and variance of FPS value - LOWEST, NOT HIGHEST) and which is the best "Best overall" (preferrably lower std.dev./variance than higher FPS, but if slight decrease in stability gives significantly higher FPS - pick that one). NEVER consider runs that have significantly lower FPS or has other significant issues. Exclude runs from consideration if they are significantly worse than the rest (as it would be mentioned in issues section). Note that your goal is to pick winners and not do a comparison in this section. Include numbers to justify your claims.
4. In Summary section, provide an overview of all runs. Mention which runs are similar and which are different. Mention which runs are better in terms of FPS and which are better in terms of stability. Mention if there are any issues and what could be the reason for them. In short - summarize whole benchmark.
5. First 2 sections should be bullet points, no subpoints, only 1 bullet point per point, while summary should be a single paragraph.
6. NEVER use actual numbers. Instead, use percentage in comparison to other runs.
7. Use markdown, use code syntax for labels.
`

var (
	inProgressSummaries    = map[uint]struct{}{}
	inProgressSummariesMux = &sync.Mutex{}
)

func generateSummary(b *Benchmark, bds []*BenchmarkData) {
	// Check if OpenAI integration is not enabled
	if openaiClient == nil {
		return
	}

	// Lock mutex, as integration is enabled and might be already in progress
	inProgressSummariesMux.Lock()

	// Check if generation is already in progress
	if _, ok := inProgressSummaries[b.ID]; ok {
		inProgressSummariesMux.Unlock()
		return
	}
	inProgressSummaries[b.ID] = struct{}{}
	inProgressSummariesMux.Unlock()

	// Create user prompt
	userPrompt := writeAIPrompt(bds, b.Title, b.Description)

	// Retrieve AI response
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       openaiModel,
			Temperature: 0.0,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemMessage},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
		},
	)
	if err != nil {
		log.Println("Failed to generate AI summary:", err)
		return
	}

	db.Model(&Benchmark{}).Where("id = ?", b.ID).Update("AiSummary", resp.Choices[0].Message.Content)

	// Update status
	inProgressSummariesMux.Lock()
	delete(inProgressSummaries, b.ID)
	inProgressSummariesMux.Unlock()
}

func writeAIPrompt(bds []*BenchmarkData, bdTitle, bdDescription string) string {
	sb := strings.Builder{}
	sb.WriteString("Benchmark title: ")
	sb.WriteString(bdTitle)
	sb.WriteString("\n")
	sb.WriteString("Benchmark description: \n")
	sb.WriteString(bdDescription)
	sb.WriteString("\n\n")
	sb.WriteString("Benchmark contains ")
	sb.WriteString(strconv.Itoa(len(bds)))
	sb.WriteString(" runs:\n")

	for _, benchmarkRun := range bds {
		sb.WriteString("\nLabel: ")
		sb.WriteString(benchmarkRun.Label)
		sb.WriteString("\n")

		sb.WriteString("OS: ")
		sb.WriteString(benchmarkRun.SpecOS)
		sb.WriteString("\n")
		sb.WriteString("GPU: ")
		sb.WriteString(benchmarkRun.SpecGPU)
		sb.WriteString("\n")
		sb.WriteString("CPU: ")
		sb.WriteString(benchmarkRun.SpecCPU)
		sb.WriteString("\n")
		sb.WriteString("RAM: ")
		sb.WriteString(benchmarkRun.SpecRAM)
		sb.WriteString("\n")
		sb.WriteString("Linux kernel: ")
		sb.WriteString(benchmarkRun.SpecLinuxKernel)
		sb.WriteString("\n")
		sb.WriteString("Linux scheduler: ")
		sb.WriteString(benchmarkRun.SpecLinuxScheduler)
		sb.WriteString("\n")

		// FPS
		stats := calculateAIPromptArrayStats(benchmarkRun.DataFPS)
		sb.WriteString("FPS: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// Frame time
		stats = calculateAIPromptArrayStats(benchmarkRun.DataFrameTime)
		sb.WriteString("Frame time: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// CPU load
		stats = calculateAIPromptArrayStats(benchmarkRun.DataCPULoad)
		sb.WriteString("CPU load: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU load
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPULoad)
		sb.WriteString("GPU load: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// CPU temp
		stats = calculateAIPromptArrayStats(benchmarkRun.DataCPUTemp)
		sb.WriteString("CPU temp: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU temp
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPUTemp)
		sb.WriteString("GPU temp: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU core clock
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPUCoreClock)
		sb.WriteString("GPU core clock: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU mem clock
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPUMemClock)
		sb.WriteString("GPU mem clock: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU VRAM used
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPUVRAMUsed)
		sb.WriteString("GPU VRAM used: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// GPU power
		stats = calculateAIPromptArrayStats(benchmarkRun.DataGPUPower)
		sb.WriteString("GPU power: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// RAM used
		stats = calculateAIPromptArrayStats(benchmarkRun.DataRAMUsed)
		sb.WriteString("RAM used: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")

		// Swap used
		stats = calculateAIPromptArrayStats(benchmarkRun.DataSwapUsed)
		sb.WriteString("Swap used: ")
		sb.WriteString(stats.String())
		sb.WriteString("\n")
	}

	return sb.String()
}

type AIPromptArrayStats struct {
	Count        int
	Lowest       float64
	Low1Percent  float64
	Mean         float64
	Median       float64
	Top97Percent float64
	Highest      float64
	StdDev       float64
	Variance     float64
}

func calculateAIPromptArrayStats(data []float64) AIPromptArrayStats {
	if len(data) == 0 {
		return AIPromptArrayStats{}
	}

	sort.Float64s(data)
	count := len(data)
	lowest := data[0]
	highest := data[count-1]

	low1PercentIndex := int(math.Ceil(0.01*float64(count))) - 1
	if low1PercentIndex < 0 {
		low1PercentIndex = 0
	}
	low1Percent := data[low1PercentIndex]

	top97PercentIndex := int(math.Ceil(0.97*float64(count))) - 1
	if top97PercentIndex < 0 {
		top97PercentIndex = 0
	}
	top97Percent := data[top97PercentIndex]

	mean := 0.0
	for _, value := range data {
		mean += value
	}
	mean /= float64(count)

	median := 0.0
	if count%2 == 0 {
		median = (data[count/2-1] + data[count/2]) / 2
	} else {
		median = data[count/2]
	}

	variance := 0.0
	for _, value := range data {
		variance += (value - mean) * (value - mean)
	}
	variance /= float64(count)

	stdDev := math.Sqrt(variance)

	return AIPromptArrayStats{
		Count:        count,
		Lowest:       lowest,
		Low1Percent:  low1Percent,
		Mean:         mean,
		Median:       median,
		Top97Percent: top97Percent,
		Highest:      highest,
		StdDev:       stdDev,
		Variance:     variance,
	}
}

func (as AIPromptArrayStats) String() string {
	return strings.Join([]string{
		"Count: " + strconv.Itoa(as.Count),
		"Lowest: " + strconv.FormatFloat(as.Lowest, 'f', -1, 64),
		"Low1Percent: " + strconv.FormatFloat(as.Low1Percent, 'f', -1, 64),
		"Mean: " + strconv.FormatFloat(as.Mean, 'f', -1, 64),
		"Median: " + strconv.FormatFloat(as.Median, 'f', -1, 64),
		"Top97Percent: " + strconv.FormatFloat(as.Top97Percent, 'f', -1, 64),
		"Highest: " + strconv.FormatFloat(as.Highest, 'f', -1, 64),
		"StdDev: " + strconv.FormatFloat(as.StdDev, 'f', -1, 64),
		"Variance: " + strconv.FormatFloat(as.Variance, 'f', -1, 64),
	}, ", ")
}
