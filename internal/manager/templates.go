package manager

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

func (m *Manager) loadTemplates() error {
	tplPath := *m.conf.TemplatePath
	entries, err := os.ReadDir(tplPath)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			// TODO: Support sub directories
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		fullPath := filepath.Join(tplPath, name)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		files = append(files, string(content))
	}

	for _, file := range files {
		t, err := template.New("judge-tmpl").Parse(file)
		if err != nil {
			return err
		}
		m.tmpls = append(m.tmpls, t)
	}

	return nil
}
