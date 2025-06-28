package doc

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Analyzer struct {
	srcPath  string
	template string
}

type SourceInfo struct {
	Files     []FileInfo `json:"files"`
	Languages []string   `json:"languages"`
	Structure string     `json:"structure"`
	Summary   string     `json:"summary"`
}

type FileInfo struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Size     int64  `json:"size"`
}

func NewAnalyzer(srcPath, template string) (*Analyzer, error) {
	return &Analyzer{
		srcPath:  srcPath,
		template: template,
	}, nil
}

func (a *Analyzer) Analyze() (*SourceInfo, error) {
	sourceInfo := &SourceInfo{
		Files:     []FileInfo{},
		Languages: []string{},
	}

	// ファイル情報の収集
	err := filepath.WalkDir(a.srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリをスキップ
		if d.IsDir() {
			return nil
		}

		// 隠しファイルとバイナリファイルをスキップ
		if shouldSkipFile(path) {
			return nil
		}

		// ファイル情報の読み取り
		fileInfo, err := a.analyzeFile(path)
		if err != nil {
			// エラーログを出力してスキップ
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze file %s: %v\n", path, err)
			return nil
		}

		sourceInfo.Files = append(sourceInfo.Files, *fileInfo)

		// 言語情報の収集
		if fileInfo.Language != "" && !contains(sourceInfo.Languages, fileInfo.Language) {
			sourceInfo.Languages = append(sourceInfo.Languages, fileInfo.Language)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// プロジェクト構造の生成
	sourceInfo.Structure = a.generateStructure()

	// サマリーの生成
	sourceInfo.Summary = a.generateSummary(sourceInfo)

	return sourceInfo, nil
}

func (a *Analyzer) analyzeFile(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// ファイルサイズ制限（1MB）
	if stat.Size() > 1024*1024 {
		return &FileInfo{
			Path:     path,
			Content:  fmt.Sprintf("[File too large: %d bytes]", stat.Size()),
			Language: detectLanguage(path),
			Size:     stat.Size(),
		}, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// バイナリファイルの検出
	if isBinaryContent(content) {
		return &FileInfo{
			Path:     path,
			Content:  "[Binary file]",
			Language: detectLanguage(path),
			Size:     stat.Size(),
		}, nil
	}

	return &FileInfo{
		Path:     path,
		Content:  string(content),
		Language: detectLanguage(path),
		Size:     stat.Size(),
	}, nil
}

func (a *Analyzer) generateStructure() string {
	var structure strings.Builder
	
	err := filepath.WalkDir(a.srcPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if shouldSkipFile(path) && !d.IsDir() {
			return nil
		}

		// 相対パスを計算
		relPath, err := filepath.Rel(a.srcPath, path)
		if err != nil {
			relPath = path
		}

		// インデントレベルを計算
		depth := strings.Count(relPath, string(filepath.Separator))
		indent := strings.Repeat("  ", depth)

		if d.IsDir() {
			structure.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			structure.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
		}

		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error generating structure: %v", err)
	}

	return structure.String()
}

func (a *Analyzer) generateSummary(sourceInfo *SourceInfo) string {
	var summary strings.Builder
	
	summary.WriteString(fmt.Sprintf("Source analysis for: %s\n", a.srcPath))
	summary.WriteString(fmt.Sprintf("Total files: %d\n", len(sourceInfo.Files)))
	summary.WriteString(fmt.Sprintf("Languages detected: %s\n", strings.Join(sourceInfo.Languages, ", ")))
	
	// ファイル種別の統計
	langCounts := make(map[string]int)
	for _, file := range sourceInfo.Files {
		if file.Language != "" {
			langCounts[file.Language]++
		}
	}
	
	if len(langCounts) > 0 {
		summary.WriteString("File counts by language:\n")
		for lang, count := range langCounts {
			summary.WriteString(fmt.Sprintf("  %s: %d\n", lang, count))
		}
	}

	return summary.String()
}

func shouldSkipFile(path string) bool {
	fileName := filepath.Base(path)
	
	// 隠しファイル
	if strings.HasPrefix(fileName, ".") {
		return true
	}

	// 一般的なスキップ対象
	skipDirs := []string{"node_modules", ".git", ".vscode", "vendor", "target", "dist", "build"}
	skipExts := []string{".exe", ".dll", ".so", ".dylib", ".jpg", ".png", ".gif", ".pdf", ".zip", ".tar", ".gz"}

	for _, dir := range skipDirs {
		if strings.Contains(path, dir) {
			return true
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	for _, skipExt := range skipExts {
		if ext == skipExt {
			return true
		}
	}

	return false
}

func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	langMap := map[string]string{
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".py":   "Python",
		".java": "Java",
		".c":    "C",
		".cpp":  "C++",
		".h":    "C/C++ Header",
		".rs":   "Rust",
		".rb":   "Ruby",
		".php":  "PHP",
		".sh":   "Shell",
		".bash": "Bash",
		".zsh":  "Zsh",
		".md":   "Markdown",
		".yaml": "YAML",
		".yml":  "YAML",
		".json": "JSON",
		".xml":  "XML",
		".html": "HTML",
		".css":  "CSS",
		".scss": "SCSS",
		".sass": "Sass",
		".sql":  "SQL",
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return ""
}

func isBinaryContent(content []byte) bool {
	// 最初の512バイトをチェック
	checkSize := 512
	if len(content) < checkSize {
		checkSize = len(content)
	}

	for i := 0; i < checkSize; i++ {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}