package doc

import (
	"fmt"
	"strings"
)

type TemplateInfo struct {
	Name        string
	Description string
	Extensions  []string
	Example     string
}

var supportedTemplates = map[string]TemplateInfo{
	"readme": {
		Name:        "README",
		Description: "Comprehensive project README with installation, usage, and contribution guidelines",
		Extensions:  []string{".md"},
		Example:     "gelf doc -s . -d README.md -t readme",
	},
	"api": {
		Name:        "API Documentation",
		Description: "Detailed API documentation with function signatures, parameters, and examples",
		Extensions:  []string{".md", ".html"},
		Example:     "gelf doc -s ./pkg -d docs/api.md -t api",
	},
	"changelog": {
		Name:        "Changelog",
		Description: "CHANGELOG.md template following Keep a Changelog format",
		Extensions:  []string{".md"},
		Example:     "gelf doc -s . -d CHANGELOG.md -t changelog",
	},
	"architecture": {
		Name:        "Architecture Documentation",
		Description: "System architecture overview with components, data flow, and design patterns",
		Extensions:  []string{".md", ".html"},
		Example:     "gelf doc -s . -d ARCHITECTURE.md -t architecture",
	},
	"godoc": {
		Name:        "Go Package Documentation",
		Description: "Go-style package documentation following godoc conventions",
		Extensions:  []string{".md"},
		Example:     "gelf doc -s ./pkg -d docs/package.md -t godoc",
	},
}

func GetTemplateInfo(templateName string) (TemplateInfo, bool) {
	info, exists := supportedTemplates[templateName]
	return info, exists
}

func ListTemplates() []TemplateInfo {
	var templates []TemplateInfo
	for _, info := range supportedTemplates {
		templates = append(templates, info)
	}
	return templates
}

func GetTemplateNames() []string {
	var names []string
	for name := range supportedTemplates {
		names = append(names, name)
	}
	return names
}

func ValidateTemplate(templateName string) error {
	if _, exists := supportedTemplates[templateName]; !exists {
		availableTemplates := strings.Join(GetTemplateNames(), ", ")
		return fmt.Errorf("unsupported template '%s'. Available templates: %s", templateName, availableTemplates)
	}
	return nil
}

func GetTemplateHelp() string {
	var help strings.Builder
	help.WriteString("Available documentation templates:\n\n")
	
	for _, template := range ListTemplates() {
		help.WriteString(fmt.Sprintf("• %s\n", template.Name))
		help.WriteString(fmt.Sprintf("  %s\n", template.Description))
		help.WriteString(fmt.Sprintf("  Example: %s\n\n", template.Example))
	}
	
	return help.String()
}

func RecommendOutputExtension(templateName, format string) string {
	switch format {
	case "html":
		return ".html"
	case "json":
		return ".json"
	default: // markdown
		return ".md"
	}
}

func ValidateTemplateForLanguage(templateName, language string) error {
	// すべてのテンプレートが日本語と英語をサポート
	supportedLanguages := []string{"english", "japanese", "ja", "en"}
	
	if language == "" {
		return nil // 言語指定なしは有効
	}
	
	language = strings.ToLower(language)
	for _, supported := range supportedLanguages {
		if language == supported {
			return nil
		}
	}
	
	return fmt.Errorf("unsupported language '%s'. Supported languages: %s", language, strings.Join(supportedLanguages, ", "))
}