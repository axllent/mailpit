package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"gopkg.in/yaml.v3"
)

type yamlTags struct {
	Filters []yamlTag `yaml:"filters"`
}

type yamlTag struct {
	Match string `yaml:"match"`
	Tags  string `yaml:"tags"`
}

// Load tags from a configuration from a file, if set
func loadTagsFromConfig(c string) error {
	if c == "" {
		return nil // not set, ignore
	}

	c = filepath.Clean(c)

	if !isFile(c) {
		return fmt.Errorf("[tags] configuration file not found or unreadable: %s", c)
	}

	data, err := os.ReadFile(c)
	if err != nil {
		return fmt.Errorf("[tags] %s", err.Error())
	}

	conf := yamlTags{}

	if err := yaml.Unmarshal(data, &conf); err != nil {
		return err
	}

	if conf.Filters == nil {
		return fmt.Errorf("[tags] missing tag: array in %s", c)
	}

	for _, t := range conf.Filters {
		tags := strings.Split(t.Tags, ",")
		TagFilters = append(TagFilters, autoTag{Match: t.Match, Tags: tags})
	}

	logger.Log().Debugf("[tags] loaded %s from config %s", tools.Plural(len(conf.Filters), "tag filter", "tag filters"), c)

	return nil
}

func loadTagsFromArgs(c string) error {
	if c == "" {
		return nil // not set, ignore
	}

	args := tools.ArgsParser(c)

	for _, a := range args {
		t := strings.Split(a, "=")
		if len(t) > 1 {
			match := strings.TrimSpace(strings.ToLower(strings.Join(t[1:], "=")))
			tags := strings.Split(t[0], ",")
			TagFilters = append(TagFilters, autoTag{Match: match, Tags: tags})
		} else {
			return fmt.Errorf("[tag] error parsing tags (%s)", a)
		}
	}

	logger.Log().Debugf("[tags] loaded %s from CLI args", tools.Plural(len(args), "tag filter", "tag filters"))

	return nil
}
