package flightlesssomething

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const systemMessage = `
You are given a summary of PC benchmark data. Your task is to provide conclusion and overview of the given data:

0. Your summary must consist of max 3 segments - "Highest and Smoothest FPS", "Anomalies" and "Summary".
1. Provide which run has the highest (average) fps and which has the smoothest fps (based on fps/frametime std.dev. and variance). Do not hesitate to mention multiple runs if they are incredibly similar. Also provide overall the best run. Try to understand which one has the best sweet "average" in terms of being smoothest and highest FPS.
2. Anomalies in the data (if any). For example, if all benchmarks uses the same hardware/software? Or of certain run has lower/higher FPS that correlates to higher/lower VRAM usage, core clock, mem clock, etc. Try to figure out why is it so, by looking ONLY at the provided data. Do NOT mention anything if it's not an anomaly.
3. If certain run had much worse FPS/Frametime than others, then exclude it from consideration in point 1. In point 2, try to figure out why it is so (first consider GPU VRAM, core clock, mem clock, then RAM/SWAP and other factors, while lastly CPU and GPU usage). If you can't figure out why, then just say so.
4. Point 3 must be your TOP priority. Do NOT provide any other information than requested.
5. You can mention labels in a natural way. E.g. you can call "lavd-defaults" just "LAVD" (if this makes sense).
6. Use bullet points for point 1 and 2. Use paragraph for point 3.
7. NEVER provide actual number or "higher/lower than". Instead, ALWAYS provide exact/approximate percentage in comparison to others.
8. NEVER guess the issue outside of the provided data. If you can't figure out why, then just say so.
9. ALWAYS mention in "anomalies" if certain run has correlation of higher/lower FPS with certain metrics (e.g. VRAM usage, core clock, mem clock, ram, swap, cpu, gpu). Only mention if there is significant correlation, at least 5 percent.
10. Provide an extended summary overview of all runs, but avoid repeating yourself of what you mentioned in point 1 and 2.

Do not provide numbers or visualize anything - user can already see charts.
`

func getAISummary(bds []*BenchmarkData, bdTitle, bdDescription, openaiApiKey, openaiModel string) (string, error) {
	userPrompt := writeAIPrompt(bds, bdTitle, bdDescription)
	fmt.Println(userPrompt)

	return "", nil

	client := openai.NewClient(openaiApiKey)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openaiModel,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemMessage},
				{Role: openai.ChatMessageRoleUser, Content: userPrompt},
			},
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
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
