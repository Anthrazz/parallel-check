package main

import (
	"fmt"
	"time"

	"github.com/Anthrazz/parallel-check/plugins"
	"github.com/fatih/color"
)

/*
 * Server
 */

// Server represents a single Server that should be tested
type Server struct {
	SuccessQueries int                     // amount of successful Queries
	ErrorQueries   int                     // amount of queries with errors
	TestPlugin     plugins.PluginInterface // TestPlugin is the interface to the test plugin which is used for this Server
	LastDelay      time.Duration           // last answer delay
	BestDelay      time.Duration           // lowest answer delay
	WorstDelay     time.Duration           // highest answer delay
	DelaySum       time.Duration           // a sum of all answer delays for calculation of the averageDelay
	AverageDelay   time.Duration           // the average answer delays
	Answers        []TestResult            // slice with all TestResult's for this DNS resolver
}

// newServer creates a new Server
func newServer(testPluginName string) (Server, error) {
	s := Server{
		Answers: make([]TestResult, 0),
	}

	// Load the wanted test plugin for this server
	var err error
	s.TestPlugin, err = Plugins.GetNewPlugin(testPluginName)
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *Server) GetQuerySum() int {
	return s.SuccessQueries + s.ErrorQueries
}
func (s *Server) GetErrorPercentage() float64 {
	return float64(s.ErrorQueries) / float64(s.GetQuerySum()) * 100
}

// ExecuteQuery do query the resolver and parses the result
func (s *Server) ExecuteQuery() {
	dataPoint, err := s.TestPlugin.ExecuteTest()
	if err != nil {
		return
	}

	s.AppendAnswer(dataPoint.GetDelay(), dataPoint.GetResult())
	if dataPoint.GetResult() {
		s.SuccessQueries++
		// set the last, best, worst and average answer delay for this resolver
		s.LastDelay = dataPoint.GetDelay()
		s.SetBestDelay(dataPoint.GetDelay())
		s.SetWorstDelay(dataPoint.GetDelay())
		s.SetAverageDelay(dataPoint.GetDelay())

		// Set overall worst response delay
		GlobalState.SetWorstResponseDelay(dataPoint.GetDelay())
	} else {
		s.ErrorQueries++
	}

	// delete oldest dns answer to free up not needed memory
	s.DeleteOldestTest()

	return
}

func (s *Server) SetBestDelay(d time.Duration) {
	// Default value for s.BestDelay is 0 - so set it explicit at the first query
	if GlobalState.TestCounter == 1 {
		s.BestDelay = d
	} else if s.BestDelay > d {
		s.BestDelay = d
	}
}
func (s *Server) SetWorstDelay(d time.Duration) {
	if s.WorstDelay < d {
		s.WorstDelay = d
	}
}
func (s *Server) SetAverageDelay(d time.Duration) {
	s.DelaySum += d
	s.AverageDelay = time.Duration(
		int64(s.DelaySum) / int64(s.GetQuerySum()),
	)
}

func (s *Server) AppendAnswer(delay time.Duration, result bool) {
	s.Answers = append(s.Answers, TestResult{
		Delay:  delay,
		Result: result,
	})
}

// DeleteOldestTest deletes the oldest Server.Answer entry when it
// would exceed the query history length to be displayed
func (s *Server) DeleteOldestTest() {
	toRemove := len(s.Answers) - GlobalState.MaximumHistoryLength
	if toRemove >= 1 {
		s.Answers = s.Answers[toRemove:]
	}
}

// GetQueryHistory returns a pretty history of the last DNS queries
func (s *Server) GetQueryHistory() string {
	history := ""

	for _, answer := range s.Answers {
		history += answer.GetColoredHistoryEntry()
	}

	return history
}

// Reset does clear all Test History but not the Collector Configuration
func (s *Server) Reset() {
	s.SuccessQueries = 0
	s.ErrorQueries = 0
	s.LastDelay = 0
	s.BestDelay = 0
	s.WorstDelay = 0
	s.DelaySum = 0
	s.AverageDelay = 0
	s.Answers = make([]TestResult, 0)
}

/*
 * TestResult
 */

// TestResult represents a single Test result of a tested instance
type TestResult struct {
	Delay  time.Duration // delay between question and answer
	Result bool          // true if the request was ok, false if not
}

func (a *TestResult) GetColoredHistoryEntry() string {
	if a.Result {
		rating := getHistoryDelayRating(a.Delay)
		return getColoredHistoryEntryChar(rating)
	} else {
		return color.RedString("%s", "?")
	}
}

/*
 * Helper function to get colored history
 */

// getHistoryDelayRating returns an arbitrary float between 0 and 1 (lower is better) which
// indicates how good/bad the response was in comparison to the worst response
func getHistoryDelayRating(d time.Duration) float64 {
	return float64(d) / float64(GlobalState.WorstResponseDelay)
}

// show a scale for the usage of the color in the query history
func getHistoryColorScale() string {
	scale := "Scale: "

	worstDelay := GlobalState.WorstResponseDelay

	delays := []float64{
		float64(worstDelay) * 0.6,
		float64(worstDelay) * 0.8,
		float64(worstDelay) * 0.9,
		float64(worstDelay) * 0.95,
		float64(worstDelay) * 1,
	}

	for _, delay := range delays {
		r := getHistoryDelayRating(time.Duration(delay))
		d := time.Duration(delay)

		scale += fmt.Sprintf("%s < %dms ", getColoredHistoryEntryChar(r), d.Milliseconds())
	}

	return scale
}

// return a useful char so the user know how bad the delay is
func getColoredHistoryEntryChar(rating float64) string {
	switch {
	case rating <= 0.6:
		return color.GreenString("%s", ".")
	case rating <= 0.8:
		return color.CyanString("%s", "-")
	case rating <= 0.9:
		return color.BlueString("%s", "+")
	case rating <= 0.95:
		return color.YellowString("%s", "*")
	default:
		return color.MagentaString("%s", "#")
	}
}
