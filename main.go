package main

import (
	"flag"
	"fmt"
	"github.com/Anthrazz/parallel-check/plugins"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/gosuri/uilive"
	"github.com/miekg/dns"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/ts"
)

// Variables to hold command line arguments
var (
	Domain            = flag.String("d", "example.com", "dns check: `domain` that should be queried")
	DomainType        = flag.String("dns-type", "A", "dns check: what for DNS `record type` should be queried?")
	MaxCount          = flag.Int("c", 0, "exit after `count` tests")
	WaitTime          = flag.Duration("w", 1*time.Second, "delay between two checks (prefix duration with ms or s)")
	TimeoutForQueries = flag.Duration("t", 1*time.Second, "timeout for checks (prefix duration with ms or s)")
	IPv4              = flag.Bool("4", false, "use IPv4")
	IPv6              = flag.Bool("6", false, "use IPv6")

	PluginToUse string

	GlobalState GlobalStateType // contains different variables and functions for the global state of the program
)

/*********/
/* Types */
/*********/

type GlobalStateType struct {
	Domain      string        // Domain address for which the IP should be asked
	RecordType  uint16        // RecordType contains the type of the wanted DNS Record Type
	TestCounter int           // TestCounter contains the global counter how many DNS requests were already sent
	Server      []Server      // Server contains a slice with all Server
	Timeout     time.Duration // timeout to wait for DNS query answer

	// Automatically set:
	Mutex                sync.Mutex
	WorstResponseDelay   time.Duration // worst response delay over all resolver, dynamically readjusted
	MaximumHistoryLength int           // Maximum length of the query history, will be readjusted automatically
	LongestIPLength      int           // how many chars are in the longest DNS resolver IP?
	Pause                bool          // Set to true to pause the output and tests
	ResetState           bool          // Set to true to clear history and restart tests
}

// InitGlobalStateType creates a new Global State Type struct with some safe defaults
func InitGlobalStateType() GlobalStateType {
	return GlobalStateType{
		Domain:               "example.com",
		RecordType:           dns.TypeA,
		MaximumHistoryLength: 13,
		LongestIPLength:      len("Server"), // Length of Header "SERVER"
		Timeout:              *TimeoutForQueries,
	}
}

func (gs *GlobalStateType) SetWorstResponseDelay(d time.Duration) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()

	if gs.WorstResponseDelay < d {
		gs.WorstResponseDelay = d
	}
}
func (gs *GlobalStateType) AddServer(ip string, testPlugin string) error {
	// Create a new server
	s, err := newServer(testPlugin)
	if err != nil {
		return err
	}

	pluginConfig := plugins.PluginConfig{
		"IPAddress":  ip,
		"Domain":     *Domain,
		"RecordType": *DomainType,
		"Timeout":    gs.Timeout.String(),
	}
	if *IPv4 {
		pluginConfig["IPv4"] = "true"
	}
	if *IPv6 {
		pluginConfig["IPv6"] = "true"
	}

	// Set the Test config
	// TODO: load correct config for the wanted plugin
	if err = s.TestPlugin.SetConfig(pluginConfig); err != nil {
		return err
	}

	// and append it
	gs.Server = append(gs.Server, s)

	// set the length of the longest IP, needed for AutoScaleQueryHistory()
	if len(ip) > gs.LongestIPLength {
		gs.LongestIPLength = len(ip)
	}

	return nil
}

// AutoScaleQueryHistory Sets a new Query History Length if the user rescales the terminal
func (gs *GlobalStateType) AutoScaleQueryHistory() {
	// get terminal size and calculate a new size
	// 87 chars is the table long without the "DNS SERVER" column
	size, _ := ts.GetSize()
	newSize := size.Col() - 87 - gs.LongestIPLength

	// Do not scale under 13 history entries because table header "QUERY HISTORY"
	// is 13 chars long, so we can use the already allocated space
	if newSize > 13 {
		gs.MaximumHistoryLength = newSize
	}
}

// QueryResolver do execute a Server.ExecuteQuery on all set Server in go routines
func (gs *GlobalStateType) QueryResolver() {
	gs.TestCounter++

	var wg sync.WaitGroup
	for i := range gs.Server {
		wg.Add(1)

		// ask all DNS resolver asynchronous
		go func(i int) {
			defer wg.Done()
			gs.Server[i].ExecuteQuery()
		}(i)
	}

	// While waiting check if the Query History must be rescaled because the terminal size could have changed
	gs.AutoScaleQueryHistory()

	wg.Wait()
}

func (gs *GlobalStateType) TogglePause() {
	if gs.Pause {
		gs.Pause = false
	} else {
		gs.Pause = true
	}
}

func (gs *GlobalStateType) Reset() {
	for i, _ := range gs.Server {
		gs.Server[i].Reset()
	}

	GlobalState.TestCounter = 0
	GlobalState.WorstResponseDelay = 0
}

// UpdateTimeouts updates the timeout on each tested instance
func (gs *GlobalStateType) UpdateTimeouts() {
	for i, _ := range gs.Server {
		gs.Server[i].TestPlugin.SetTimeout(gs.Timeout)
	}
}

/*
 * Command
 */

// Command is used to communicate with some Go Routines for Output and User Handling
type Command struct {
	Command string
}

/*************/
/* Functions */
/*************/

// Wait some time until the next queries and terminal write
// Return true when the max queries are reached
func sleep(duration time.Duration) bool {
	// exit if the maximum query count is reached
	if *MaxCount != 0 && GlobalState.TestCounter >= *MaxCount {
		return true
	}

	time.Sleep(duration)
	return false
}

func printHelp() {
	fmt.Println("Usage: " + os.Args[0] + " [<arguments>] <IP> [<IP> ...]")
	fmt.Println()
	fmt.Println("This tool do execute a check with the given address in a regular interval and prints")
	fmt.Println("the results to the terminal.")
	fmt.Println()
	fmt.Println("Interactive Keyboard Shortcuts:")
	fmt.Println("  Q: Quit")
	fmt.Println("  P: Pause")
	fmt.Println("  R: Reset")
	fmt.Println("  Arrow Key Up: Increase Wait Time between Checks")
	fmt.Println("  Arrow Key Down: Decrease Wait Time")
	fmt.Println("  Arrow Key Left: Decrease Timeout")
	fmt.Println("  Arrow Key Right: Increase Timeout")
	fmt.Println()
	fmt.Println("Arguments:")
	flag.PrintDefaults()
}

// Configure and parse all command line flags
func parseFlags() *GlobalStateType {
	flag.Usage = printHelp
	flag.Parse()

	gs := InitGlobalStateType()

	// Add DNS server
	for _, server := range flag.Args() {
		err := gs.AddServer(server, PluginToUse)
		if err != nil {
			fmt.Printf("Could not add server %s: %s\n", server, err)
			os.Exit(1)
		}
	}
	if len(gs.Server) == 0 {
		fmt.Println("No servers given!")
		printHelp()
		os.Exit(1)
	}

	return &gs
}

// registerPlugins registers all available plugins
func registerPlugins() {
	Plugins.Register(
		"DNS",
		"dns",
		&plugins.DNSCollector{},
	)
	Plugins.Register(
		"Ping",
		"ping",
		&plugins.PingCollector{},
	)

	flag.StringVar(&PluginToUse,
		"plugin", "dns",
		"which check plugin should be used. Available: "+fmt.Sprintf("%s", Plugins.GetAvailablePlugins()),
	)
	flag.StringVar(&PluginToUse,
		"p", "dns",
		"shorthand for --plugin",
	)
}

func main() {
	os.Exit(run())
}

func run() int {
	registerPlugins()
	GlobalState = *parseFlags()

	// Start rendering routine
	chRender := make(chan Command)
	var wgRender sync.WaitGroup
	wgRender.Add(1)
	go renderRoutine(&wgRender, chRender)

	// Start Keyboard listen routine
	go keyboardRoutine(chRender)

	// Clear Console Screen
	chRender <- Command{Command: "clearConsole"}

	// Start Main Loop which coordinate queries and rendering
MainLoop:
	for {
		startLoop := time.Now()

		// reset all stats if needed
		if GlobalState.ResetState {
			GlobalState.Reset()
			GlobalState.ResetState = false
		}

		// execute all tests
		if !GlobalState.Pause {
			GlobalState.QueryResolver()
		}

		// render the user interface
		chRender <- Command{Command: "renderTable"}

		elapsedTimeSinceStart := time.Since(startLoop)
		// calculate how long the tests in the last frame have taken and only wait up to the max wait time
		timeToSleep := *WaitTime - elapsedTimeSinceStart
		if elapsedTimeSinceStart >= *WaitTime {
			timeToSleep = time.Duration(0)
		}

		if end := sleep(timeToSleep); end {
			break MainLoop
		}
	}

	// End Rendering Thread
	close(chRender)
	wgRender.Wait()
	return 0
}

// keyboardRoutine does process all keyboard inputs.
//
// It gets a reference to the render channel to start a redrawn of the user interface.
func keyboardRoutine(chRender chan Command) {
	keysEvents, err := keyboard.GetKeys(3)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	for event := range keysEvents {
		if event.Err != nil {
			panic(event.Err)
		}
		//fmt.Printf("You pressed: rune %q, key %X\r\n", event.Rune, event.Key)

		// Pause
		if event.Rune == 'p' || event.Rune == 'P' {
			GlobalState.TogglePause()
		}
		// Quit
		if event.Rune == 'q' || event.Rune == 'Q' {
			os.Exit(0)
		}
		// Reset - Set Variable to do reset between tests
		if event.Rune == 'r' || event.Rune == 'R' {
			GlobalState.ResetState = true
		}

		// Arrow Key down -> decrease delay
		if event.Key == keyboard.KeyArrowDown {
			if WaitTime.Milliseconds() >= int64(110) {
				*WaitTime = *WaitTime - (100 * time.Millisecond)
			}
		}
		// Arrow Key up -> increase delay
		if event.Key == keyboard.KeyArrowUp {
			*WaitTime = *WaitTime + (100 * time.Millisecond)
		}

		// Arrow Key left -> decrease timeout
		if event.Key == keyboard.KeyArrowLeft {
			if GlobalState.Timeout.Milliseconds() >= int64(110) {
				GlobalState.Timeout = GlobalState.Timeout - (100 * time.Millisecond)
				GlobalState.UpdateTimeouts()
			}
		}
		// Arrow Key right -> increase timeout
		if event.Key == keyboard.KeyArrowRight {
			GlobalState.Timeout = GlobalState.Timeout + (100 * time.Millisecond)
			GlobalState.UpdateTimeouts()
		}

		// Re-render Table
		chRender <- Command{Command: "renderTable"}
	}

}

func renderRoutine(wg *sync.WaitGroup, commands <-chan Command) {
	defer wg.Done()
	writer := uilive.New()

	for cmd := range commands {
		switch cmd.Command {
		case "quit":
			return
		case "clearConsole":
			fmt.Print("\033[H\033[2J")

		// Rewrite the whole console output
		case "renderTable":
			// Rewrite the whole table to allow a down scale of the query history column
			table := tablewriter.NewWriter(writer)
			table.SetHeader(
				[]string{"Server", "Success", "Errors", "Error %", "Last", "Average", "Best", "Worst", "Query History"},
			)
			table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
			table.SetCenterSeparator("|")

			for _, resolver := range GlobalState.Server {

				table.Append(
					[]string{
						resolver.TestPlugin.GetName(),
						fmt.Sprintf("%d", resolver.SuccessQueries),
						fmt.Sprintf("%d", resolver.ErrorQueries),
						strconv.FormatFloat(resolver.GetErrorPercentage(), 'f', 2, 64) + "%",
						fmt.Sprintf("%.2f ms", float64(resolver.LastDelay/time.Microsecond)/1000),
						fmt.Sprintf("%.2f ms", float64(resolver.AverageDelay/time.Microsecond)/1000),
						fmt.Sprintf("%.2f ms", float64(resolver.BestDelay/time.Microsecond)/1000),
						fmt.Sprintf("%.2f ms", float64(resolver.WorstDelay/time.Microsecond)/1000),
						resolver.GetQueryHistory(),
					},
				)
			}

			table.Render()

			// Print some additional infos
			_, _ = fmt.Fprintf(writer, "\n%s\n", "  "+getHistoryColorScale())
			_, _ = fmt.Fprintf(writer, "  Query History: %d Requests / ~%s\n", GlobalState.MaximumHistoryLength,
				time.Duration(
					int(*WaitTime+GlobalState.WorstResponseDelay)*GlobalState.MaximumHistoryLength,
				).Round(time.Second),
			)
			_, _ = fmt.Fprintf(writer, "  Timeout: %s | Delay: %s", GlobalState.Timeout, *WaitTime)
			if GlobalState.Pause {
				_, _ = fmt.Fprintf(writer, " | Pause Active\n")
			} else {
				_, _ = fmt.Fprintf(writer, "\n")
			}
			_, _ = fmt.Fprintf(writer, "  Tests: %s\n", PluginToUse)

			err := writer.Flush()
			if err != nil {
				fmt.Printf("Error has happened at write to terminal: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
