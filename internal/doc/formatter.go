package doc

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

func FormatOutput(content, format string) (string, error) {
	switch strings.ToLower(format) {
	case "markdown":
		return content, nil
	case "html":
		return convertMarkdownToHTML(content)
	case "json":
		return convertToJSON(content)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func convertMarkdownToHTML(markdown string) (string, error) {
	htmlTemplate := `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Documentation</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1, h2, h3, h4, h5, h6 {
            margin-top: 24px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.25;
        }
        h1 { border-bottom: 1px solid #eaecef; padding-bottom: 10px; }
        h2 { border-bottom: 1px solid #eaecef; padding-bottom: 8px; }
        code {
            background-color: #f6f8fa;
            border-radius: 3px;
            font-size: 85%;
            margin: 0;
            padding: 0.2em 0.4em;
        }
        pre {
            background-color: #f6f8fa;
            border-radius: 6px;
            font-size: 85%;
            line-height: 1.45;
            overflow: auto;
            padding: 16px;
        }
        pre code {
            background-color: transparent;
            border: 0;
            display: inline;
            line-height: inherit;
            margin: 0;
            max-width: auto;
            padding: 0;
            word-wrap: normal;
        }
        blockquote {
            border-left: 4px solid #dfe2e5;
            margin: 0;
            padding: 0 16px;
            color: #6a737d;
        }
        table {
            border-collapse: collapse;
            width: 100%;
        }
        table th, table td {
            border: 1px solid #dfe2e5;
            padding: 6px 13px;
        }
        table th {
            background-color: #f6f8fa;
            font-weight: 600;
        }
        ul, ol {
            padding-left: 2em;
        }
        li {
            margin-bottom: 0.25em;
        }
    </style>
</head>
<body>
    <div id="content">{{.Content}}</div>
</body>
</html>`

	tmpl, err := template.New("html").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// 簡単なMarkdown→HTML変換（基本的なもののみ）
	htmlContent := simpleMarkdownToHTML(markdown)

	var result strings.Builder
	err = tmpl.Execute(&result, map[string]interface{}{
		"Content": template.HTML(htmlContent),
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return result.String(), nil
}

func convertToJSON(content string) (string, error) {
	doc := map[string]interface{}{
		"type":    "documentation",
		"format":  "markdown",
		"content": content,
	}

	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}

func simpleMarkdownToHTML(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	inCodeBlock := false
	var codeBlockLang string

	for _, line := range lines {
		// コードブロックの処理
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				result.WriteString("</code></pre>\n")
				inCodeBlock = false
			} else {
				codeBlockLang = strings.TrimPrefix(line, "```")
				if codeBlockLang != "" {
					result.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">", codeBlockLang))
				} else {
					result.WriteString("<pre><code>")
				}
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			result.WriteString(escapeHTML(line) + "\n")
			continue
		}

		// ヘッダーの処理
		if strings.HasPrefix(line, "# ") {
			result.WriteString(fmt.Sprintf("<h1>%s</h1>\n", escapeHTML(strings.TrimPrefix(line, "# "))))
		} else if strings.HasPrefix(line, "## ") {
			result.WriteString(fmt.Sprintf("<h2>%s</h2>\n", escapeHTML(strings.TrimPrefix(line, "## "))))
		} else if strings.HasPrefix(line, "### ") {
			result.WriteString(fmt.Sprintf("<h3>%s</h3>\n", escapeHTML(strings.TrimPrefix(line, "### "))))
		} else if strings.HasPrefix(line, "#### ") {
			result.WriteString(fmt.Sprintf("<h4>%s</h4>\n", escapeHTML(strings.TrimPrefix(line, "#### "))))
		} else if strings.HasPrefix(line, "##### ") {
			result.WriteString(fmt.Sprintf("<h5>%s</h5>\n", escapeHTML(strings.TrimPrefix(line, "##### "))))
		} else if strings.HasPrefix(line, "###### ") {
			result.WriteString(fmt.Sprintf("<h6>%s</h6>\n", escapeHTML(strings.TrimPrefix(line, "###### "))))
		} else if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			// リストアイテムの処理（簡略化）
			result.WriteString(fmt.Sprintf("<li>%s</li>\n", escapeHTML(strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* "))))
		} else if strings.TrimSpace(line) == "" {
			result.WriteString("<br>\n")
		} else {
			// 通常のテキスト行
			result.WriteString(fmt.Sprintf("<p>%s</p>\n", escapeHTML(line)))
		}
	}

	return result.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}