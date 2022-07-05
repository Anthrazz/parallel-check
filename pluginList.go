package main

import (
	"errors"

	"github.com/Anthrazz/parallel-check/plugins"
)

var (
	Plugins pluginList
)

type pluginList []pluginConfiguration

type pluginConfiguration struct {
	name                string
	nameCommandlineFlag string
	Collector           plugins.PluginInterface
}

func (p pluginList) Register(name string, commandLineName string, collector plugins.PluginInterface) {
	Plugins = append(Plugins, pluginConfiguration{
		name:                name,
		nameCommandlineFlag: commandLineName,
		Collector:           collector,
	})
}

func (p pluginList) GetNewPlugin(cmdName string) (plugins.PluginInterface, error) {
	for _, plugin := range Plugins {
		if plugin.nameCommandlineFlag == cmdName {
			return plugin.Collector.New(), nil
		}
	}
	return nil, errors.New("plugin not registered")
}

func (p pluginList) GetAvailablePlugins() (s []string) {
	for _, p := range Plugins {
		s = append(s, p.nameCommandlineFlag)
	}
	return s
}
