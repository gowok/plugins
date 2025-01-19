package circuit

import (
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/gowok/gowok/maps"
)

type Config struct {
	Duration   string
	MaxRetry   int
	MaxFailure int
}

type configOpt struct {
	duration   time.Duration
	maxRetry   int
	maxFailure int
}

var (
	re = regexp.MustCompile(`(\d+)([dhms])`)
)

func parseConfig(configMap map[string]any) (configOpt, error) {
	var OptionConfig Config

	err := maps.ToStruct(maps.Get[map[string]any](configMap, "circuitBreaker"), &OptionConfig)
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return configOpt{}, err
	}

	config := configOpt{
		duration: parseDurationToTimeDuration(OptionConfig.Duration),
		maxRetry: OptionConfig.MaxRetry,
		maxFailure: func() int {
			if OptionConfig.MaxFailure < 0 {
				return 5
			}
			return OptionConfig.MaxFailure
		}(),
	}

	return config, nil
}

func parseDurationToTimeDuration(durationStr string) time.Duration {

	defaultDuration := 10 * time.Second

	matches := re.FindAllStringSubmatch(durationStr, -1)
	if len(matches) == 0 {
		return defaultDuration
	}

	var totalDuration time.Duration

	for _, match := range matches {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return defaultDuration
		}

		unit := match[2]
		switch unit {
		case "d":
			totalDuration += time.Duration(value) * 24 * time.Hour
		case "h":
			totalDuration += time.Duration(value) * time.Hour
		case "m":
			totalDuration += time.Duration(value) * time.Minute
		case "s":
			totalDuration += time.Duration(value) * time.Second
		default:
			return defaultDuration
		}
	}

	return totalDuration

}
