package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init [agent]",
	Aliases: []string{"i"},
	Short:   "Initialize agent instruction file",
	Long:    `Initialize an instruction file for AI coding assistants.

Supported agents:
  claude    Create CLAUDE.md for Claude Code
  codex     Create AGENTS.md for OpenAI Codex CLI
  gemini    Create GEMINI.md for Google Gemini

Either an agent name or --path flag is required.

Examples:
  zap init claude                  # Create CLAUDE.md in project root
  zap init codex                   # Create AGENTS.md in project root
  zap init --path AI_GUIDE.md      # Create AI_GUIDE.md directly`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"claude", "codex", "gemini"},
	RunE:      runInit,
}

var initPath string

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initPath, "path", "", "File path for instruction file (default: CLAUDE.md/AGENTS.md/GEMINI.md)")
}

// agentFilenames maps agent names to their default filenames
var agentFilenames = map[string]string{
	"claude": "CLAUDE.md",
	"codex":  "AGENTS.md",
	"gemini": "GEMINI.md",
}

func runInit(cmd *cobra.Command, args []string) error {
	// Require either agent argument or --path flag
	if len(args) == 0 && initPath == "" {
		return fmt.Errorf("either an agent name (claude, codex, gemini) or --path flag is required")
	}

	// Get project directory from -C flag
	projectDir, err := getProjectDir(cmd)
	if err != nil {
		return err
	}

	// Determine target file path
	var targetFile string
	if initPath != "" {
		// Use provided file path
		if filepath.IsAbs(initPath) {
			targetFile = initPath
		} else {
			targetFile = filepath.Join(projectDir, initPath)
		}
	} else {
		// Use agent's default filename
		agent := strings.ToLower(args[0])
		filename, ok := agentFilenames[agent]
		if !ok {
			return fmt.Errorf("unsupported agent: %s (supported: claude, codex, gemini)", agent)
		}
		targetFile = filepath.Join(projectDir, filename)
	}

	// Generate instruction content
	content := generateInstructions()

	// Create parent directory if needed
	if dir := filepath.Dir(targetFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Check if file exists
	if _, err := os.Stat(targetFile); err == nil {
		// File exists, append to it
		f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		// Add separator before appending
		separator := "\n\n---\n\n"
		if _, err := f.WriteString(separator + content); err != nil {
			return fmt.Errorf("failed to append to file: %w", err)
		}

		fmt.Printf("✅ Appended zap instructions to %s\n", targetFile)
	} else {
		// Create new file with project title as H1
		absProjectDir, err := filepath.Abs(projectDir)
		if err != nil {
			absProjectDir = projectDir
		}
		projectName := filepath.Base(absProjectDir)
		projectTitle := toTitleCase(projectName)
		fullContent := fmt.Sprintf("# %s\n\n%s", projectTitle, content)

		if err := os.WriteFile(targetFile, []byte(fullContent), 0644); err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		fmt.Printf("✅ Created %s\n", targetFile)
	}

	return nil
}

// toTitleCase converts "my-project" to "My Project"
func toTitleCase(s string) string {
	words := strings.Split(strings.ReplaceAll(s, "-", " "), " ")
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func generateInstructions() string {
	return `## zap - Local Issue Management

### 중요: GitHub 이슈가 아닌 로컬 이슈 사용

이 프로젝트는 로컬 이슈 관리 시스템(.issues/)을 사용합니다.
이슈 관리 시 ` + "`gh issue`" + ` 명령이 아닌 ` + "`zap`" + ` 명령을 사용하세요. 올바른 형식이 자동으로 적용됩니다.

### zap CLI 명령어

#### 명령 예시

` + "```bash" + `
zap new "제목"              # 새 이슈 생성
zap new "제목" -l label     # 레이블과 함께 생성
zap new "제목" -a user      # 담당자와 함께 생성
zap new "제목" -b "본문"    # 본문과 함께 생성

zap list                    # 목록 조회
zap show 1                  # 상세보기

# 상태 변경
# 상태 변경 시 파일의 frontmatter가 업데이트됩니다 (파일 위치 변경 없음):
zap set open 1              # state: open (이슈 재오픈)
zap set wip 1               # state: wip (작업 시작)
zap set done 1              # state: done (작업 완료)
zap set closed 1            # state: closed (취소/보류)
` + "```" + `

**done vs closed 핵심 구분:**
- 코드를 작성/수정했다 → ` + "`done`" + `
- 작업 없이 닫는다 → ` + "`closed`" + `

### 커밋 메시지 규칙

이슈 관련 커밋은 메시지 끝에 이슈 번호를 포함하고, 구현이 완료되면 구현 내역과 이슈 파일을 함께 커밋하세요:

` + "```bash" + `
git commit -m "feat: 로그인 기능 구현 (#1)"
git commit -m "fix: 버그 수정 (#23)"
` + "```" + `

**Skills/Commands/Agents 사용 기록 (선택사항):**

이슈 작업 중 skills, commands, agents를 사용한 경우, 커밋 메시지 하단(footer)에 기록하면 작업 컨텍스트를 유지하는 데 도움이 됩니다:

` + "```bash" + `
git commit -m "feat: 게임 초기화 구조 구현 (#5)

GDevelop 기반의 프로젝트 구조 설정 및 기본 씬 구성 완료

Skills: /game:init (프로젝트 템플릿 생성)
Commands: /clarify (요구사항 명확화)"
` + "```" + `

포함할 정보:
- ` + "`Skills:`" + ` - 사용한 skill과 목적 (예: ` + "`/game:init (프로젝트 템플릿)`" + `)
- ` + "`Commands:`" + ` - 사용한 command와 목적 (예: ` + "`/clarify (요구사항 정리)`" + `)
- ` + "`Agents:`" + ` - 사용한 agent와 목적 (예: ` + "`codex-exec (알고리즘 최적화)`" + `)

### 워크플로우

1. **새 이슈 생성**: ` + "`zap new \"이슈 제목\"`" + ` 실행
2. **작업 시작**: ` + "`zap set wip <number>`" + ` 실행
3. **작업 완료**: ` + "`zap set done <number>`" + ` 실행
4. **취소/보류**: ` + "`zap set closed <number>`" + ` 실행

### 주의사항

- **이슈 생성 시 반드시 ` + "`zap new`" + ` 명령을 사용하세요** (파싱 오류 방지)
- 이슈 번호는 고유해야 합니다
- 파일명의 번호와 frontmatter의 number가 일치해야 합니다
- 상태 변경 시 frontmatter의 state 필드가 업데이트됩니다
`
}
