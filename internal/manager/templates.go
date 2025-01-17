package manager

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func (m *Manager) loadTemplates() error {
	tplPath := *m.conf.TemplatePath
	entries, err := os.ReadDir(tplPath)
	if err != nil {
		return err
	}

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
