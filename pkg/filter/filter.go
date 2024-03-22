package filter

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

type Filter struct {
	regexpResource *regexp.Regexp
	regexpObject   *regexp.Regexp
}

func Execute(planLines string, filterCfg map[string]map[string]any, eot, diff bool) error {
	filters, err := populateFilters(filterCfg)
	if err != nil {
		return fmt.Errorf("failed to populate filters. %v", err)
	}
	getPlan(planLines, filters, eot, diff)
	return nil
}

func populateFilters(filterConfig map[string]map[string]any) ([]Filter, error) {
	var filters []Filter
	for key, val := range filterConfig {
		for k, v := range val {
			// Regex to match resource e.g. `  ~ resource "helm_release" "argocd" {`
			regexpResource, err := regexp.Compile(`^  [~+-] resource "` + key + `" "` + k + `" {$`)
			if err != nil {
				return nil, fmt.Errorf("failed to create resource regex. %v", err)
			}
			// Regex to match object e.g. `  ~ "customresourcedefinition.apiextensions.k8s.io/apiextensions.k8s.io/v1/applicationsets.argoproj.io" = {`
			regexpObject, err := regexp.Compile(`^ +[~+-] "(` + v.(string) + `)" += {$`)
			if err != nil {
				return nil, fmt.Errorf("failed to create object regex. %v", err)
			}
			filters = append(filters, Filter{
				regexpResource: regexpResource,
				regexpObject:   regexpObject,
			})
		}
	}
	return filters, nil
}

// Iterate over every input line and evaluate it against the given filters
func getPlan(planLines string, filters []Filter, isHideEot bool, isDiffMode bool) string {
	var regexpObject *regexp.Regexp
	var objectActLine, objectEndLine string
	var isResourceMatch, isObjectMatch, isEot bool
	var toHideEot, toHide, toAdd, toChange, toDestroy int
	var outBuff bytes.Buffer
	// The last line of a terraform resource is `}` prefixed by 4 whitespaces
	resourceEndLine := "    }"

	scanner := bufio.NewScanner(strings.NewReader(planLines))

	for scanner.Scan() {
		line := scanner.Text()
		if !isResourceMatch {
			// Look for the first line of the resource to filter
			for _, filter := range filters {
				if filter.regexpResource.MatchString(line) {
					regexpObject = filter.regexpObject
					isResourceMatch = true
				}
			}
			outBuff.WriteString(getLine(line, isHideEot, isDiffMode, &isEot, &toHideEot))
		} else {
			if !isObjectMatch {
				// Look for the last line of the filtered resource
				if line == resourceEndLine {
					isResourceMatch = false
					// Look for the first line of the object to filter
				} else if indexes := regexpObject.FindStringSubmatchIndex(line); indexes != nil {
					// Note the object indentations to be able to spot its children and the last line
					objectActLine = strings.Repeat(" ", indexes[2]+1)
					objectEndLine = strings.Repeat(" ", indexes[2]-1) + "}"
					toHide, toAdd, toChange, toDestroy = 0, 0, 0, 0
					isObjectMatch = true
				}
				outBuff.WriteString(getLine(line, isHideEot, isDiffMode, &isEot, &toHideEot))
			} else {
				// Look for the last line of the filtered object
				if line == objectEndLine {
					isObjectMatch = false
					comment := fmt.Sprintf("%s# (%d lines hidden: %d to add, %d to change, %d to destroy)", objectActLine, toHide, toAdd, toChange, toDestroy)
					outBuff.WriteString(comment)
					outBuff.WriteString(line)
					// Look for the action signs [+~-] and increment the corresponding stats of the filtered object
				} else {
					if strings.HasPrefix(line, objectActLine+"+") {
						toAdd++
					} else if strings.HasPrefix(line, objectActLine+"~") {
						toChange++
					} else if strings.HasPrefix(line, objectActLine+"-") {
						toDestroy++
					}
					toHide++
				}
			}
		}
	}

	return outBuff.String()
}

// Decide whether to print the line raw or modify / hide it
func getLine(line string, isHideEot bool, isDiffMode bool, isEot *bool, toHideEot *int) string {
	var outBuff bytes.Buffer
	// Hide the lines within EOT blocks if necessary
	if *isEot && strings.HasPrefix(strings.TrimSpace(line), "EOT") {
		if isHideEot {
			comment := strings.Replace(line, "EOT", fmt.Sprintf("  # (%d lines hidden)", *toHideEot), 1)
			outBuff.WriteString(comment)
		}
		outBuff.WriteString(line)
		*isEot = false
		return ""
	} else if *isEot {
		if isHideEot {
			*toHideEot++
		} else {
			outBuff.WriteString(line)
		}
		return ""
	} else if strings.HasSuffix(strings.TrimSpace(line), "EOT") {
		*toHideEot = 0
		*isEot = true
	}

	if isDiffMode {
		// Move the action signs [+~-] to the start of the line to imitate diff syntax
		re := regexp.MustCompile(`^(?P<space> +)(?P<action>[+~-])(?P<text> .*)`)
		swapped := re.ReplaceAllString(line, "${action}${space}${text}")
		swapped = strings.Replace(swapped, "~ ", "!~", 1)
		outBuff.WriteString(swapped)
	} else {
		outBuff.WriteString(line)
	}
	return outBuff.String()
}
