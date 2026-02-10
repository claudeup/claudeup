// ABOUTME: Scans plugin cache directory to build search index
// ABOUTME: Parses plugin.json and component directories

package pluginsearch

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Scanner scans the plugin cache to build a search index.
type Scanner struct{}

// NewScanner creates a new Scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// pluginJSON represents the structure of .claude-plugin/plugin.json
type pluginJSON struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Keywords    []string `json:"keywords"`
}

// Scan walks the cache directory and builds an index of all plugins.
// Multiple cached versions of the same plugin are deduplicated, keeping only
// the latest version per name@marketplace.
func (s *Scanner) Scan(cacheDir string) ([]PluginSearchIndex, error) {
	// Collect all plugin entries, then deduplicate
	type pluginKey struct{ name, marketplace string }
	best := make(map[pluginKey]PluginSearchIndex)
	bestVersion := make(map[pluginKey]*semver.Version)

	marketplaces, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	for _, marketplace := range marketplaces {
		if !marketplace.IsDir() {
			continue
		}
		marketplacePath := filepath.Join(cacheDir, marketplace.Name())

		pluginDirs, err := os.ReadDir(marketplacePath)
		if err != nil {
			continue
		}

		for _, pluginDir := range pluginDirs {
			if !pluginDir.IsDir() {
				continue
			}
			pluginPath := filepath.Join(marketplacePath, pluginDir.Name())

			versions, err := os.ReadDir(pluginPath)
			if err != nil {
				continue
			}

			for _, version := range versions {
				if !version.IsDir() {
					continue
				}
				versionPath := filepath.Join(pluginPath, version.Name())

				plugin, err := s.parsePlugin(versionPath, marketplace.Name())
				if err != nil {
					continue
				}

				plugin.Skills = s.scanSkills(versionPath)
				plugin.Commands = s.scanComponents(versionPath, "commands")
				plugin.Agents = s.scanComponents(versionPath, "agents")

				key := pluginKey{plugin.Name, plugin.Marketplace}
				sv, svErr := semver.NewVersion(plugin.Version)

				existing, seen := best[key]
				if !seen {
					best[key] = *plugin
					if svErr == nil {
						bestVersion[key] = sv
					}
					continue
				}

				// Both parseable as semver: keep higher version
				if prev, ok := bestVersion[key]; ok && svErr == nil {
					if sv.GreaterThan(prev) {
						best[key] = *plugin
						bestVersion[key] = sv
					}
					continue
				}

				// Fallback: lexicographic comparison of version strings
				if plugin.Version > existing.Version {
					best[key] = *plugin
					if svErr == nil {
						bestVersion[key] = sv
					}
				}
			}
		}
	}

	plugins := make([]PluginSearchIndex, 0, len(best))
	for _, p := range best {
		plugins = append(plugins, p)
	}
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Marketplace != plugins[j].Marketplace {
			return plugins[i].Marketplace < plugins[j].Marketplace
		}
		return plugins[i].Name < plugins[j].Name
	})
	return plugins, nil
}

func (s *Scanner) parsePlugin(pluginPath, marketplace string) (*PluginSearchIndex, error) {
	jsonPath := filepath.Join(pluginPath, ".claude-plugin", "plugin.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var pj pluginJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, err
	}

	return &PluginSearchIndex{
		Name:        pj.Name,
		Description: pj.Description,
		Version:     pj.Version,
		Keywords:    pj.Keywords,
		Marketplace: marketplace,
		Path:        pluginPath,
	}, nil
}

func (s *Scanner) scanSkills(pluginPath string) []ComponentInfo {
	skillsDir := filepath.Join(pluginPath, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}

	var skills []ComponentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(skillsDir, entry.Name())
		skill := s.parseSkillFrontmatter(skillPath, entry.Name())
		skills = append(skills, skill)
	}
	return skills
}

func (s *Scanner) parseSkillFrontmatter(skillPath, dirName string) ComponentInfo {
	skillFile := filepath.Join(skillPath, "SKILL.md")
	file, err := os.Open(skillFile)
	if err != nil {
		return ComponentInfo{Name: dirName, Path: skillPath}
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	inFrontmatter := false
	name := dirName
	description := ""

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if line == "---" {
			if inFrontmatter {
				break // End of frontmatter
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if strings.HasPrefix(line, "name:") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				name = strings.Trim(name, "\"'")
			} else if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				description = strings.Trim(description, "\"'")
			}
		}
	}
	// Check for scanner errors (per Go best practice)
	if fileScanner.Err() != nil {
		return ComponentInfo{Name: dirName, Path: skillPath}
	}

	return ComponentInfo{
		Name:        name,
		Description: description,
		Path:        skillPath,
	}
}

func (s *Scanner) scanComponents(pluginPath, componentType string) []ComponentInfo {
	dir := filepath.Join(pluginPath, componentType)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var components []ComponentInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		components = append(components, ComponentInfo{
			Name: entry.Name(),
			Path: filepath.Join(dir, entry.Name()),
		})
	}
	return components
}
