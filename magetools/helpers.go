package magetools

import (
	"strings"

	"github.com/magefile/mage/sh"
)

func IsGitClean() (bool, error) {
	result, err := sh.Output("git", "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	if result != "" {
		return false, nil
	}
	return true, nil
}

func GetLatestTag() (string, error) {
	result, err := sh.Output("git", "tag", "-l", "--contains", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func GetCommit() (string, error) {
	result, err := sh.Output("git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func ParseModFile(file string) map[string]string {
	split := strings.Split(file, "\n")
	modFileMap := make(map[string]string)
	invalidKeyWords := []string{
		"go", "replace", "module", ")", "(", "//", "require",
	}

	for _, entry := range split {
		entry = strings.TrimSpace(strings.Trim(entry, "\t\n"))
		shouldSkip := false
		for _, e := range invalidKeyWords {
			if strings.HasPrefix(entry, e) {
				shouldSkip = true
				break
			}
		}

		if shouldSkip {
			continue
		}

		if strings.HasSuffix(entry, "indirect") {
			continue
		}

		var name, ver string
		split := strings.Split(entry, " ")
		if len(split) == 4 {
			// replace statement
			name = split[0]
			ver = split[3]
		} else if len(split) == 2 {
			// standard dep
			name = split[0]
			ver = split[1]
		} else {
			continue
		}

		modFileMap[name] = ver
	}

	return modFileMap
}
