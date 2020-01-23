package reversesearch

import (
	"regexp"
	"time"
)

// utility function for checking errors
func check(e error) {
	if e != nil {
		panic(e)
	}
}

// utility method for parsing time object
func parseTime(format string, timeStr string) time.Time {
	t, err := time.Parse(format, timeStr)
	check(err)
	return t
}

func compileRegexp(regStr string) *regexp.Regexp {
	regC, err := regexp.Compile(regStr)
	check(err)
	return regC
}

func compileRegexps(regArr []string) []*regexp.Regexp {
	regExpsC := make([]*regexp.Regexp, len(regArr))
	for i, regStr := range regArr {
		regC := compileRegexp(regStr)
		regExpsC[i] = regC
	}
	return regExpsC
}
