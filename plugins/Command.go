package plugins

import (
	"errors"
	"fmt"
	"github.com/mattn/go-shellwords"
	"os/exec"
	"time"
)

/**
CommandCollector should be used to execute a custom command and use
its exitcode as the test result.

Its still WIP and not ready for use.
*/

import (
	"context"
)

type CommandCollector struct {
	command []string
	name    string
	timeout time.Duration
}

func (c *CommandCollector) SetConfig(m map[string]string) error {
	if _, ok := m["Command"]; !ok {
		return errors.New("missing Command")
	}

	var err error
	c.command, err = shellwords.Parse(m["Command"])
	if err != nil {
		return fmt.Errorf("could not parse command: %w", err)
	}

	return nil
}

func (c *CommandCollector) GetName() string {
	return c.name
}

func (c *CommandCollector) ExecuteTest() (DataPointInterface, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.command[0], c.command[1:]...)

	now := time.Now()
	err := cmd.Run()
	delay := time.Since(now)

	// result not ok
	if err != nil {
		return DataPoint{
				delay:  delay,
				result: false,
			},
			nil
	}

	// result ok
	return DataPoint{
			delay:  delay,
			result: true,
		},
		nil
}

func (c *CommandCollector) New() PluginInterface {
	return &CommandCollector{}
}

func (c *CommandCollector) SetTimeout(duration time.Duration) {
	c.timeout = duration
}
