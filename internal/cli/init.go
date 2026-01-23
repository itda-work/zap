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

이 프로젝트는 로컬 이슈 관리 시스템(.issues/)을 사용합니다.

### 중요: GitHub 이슈가 아닌 로컬 이슈 사용

이슈 조회 시 ` + "`gh issue`" + ` 명령이 아닌 ` + "`zap`" + ` 명령을 사용하세요:

` + "```" + `bash
# ❌ 잘못된 방법
gh issue view 10

# ✅ 올바른 방법
zap show 10
` + "```" + `

### .issues/ 디렉토리 구조

` + "```" + `
.issues/
├── 001-feat-some-feature.md     # state: open
├── 002-fix-some-bug.md          # state: wip
├── 003-feat-completed.md        # state: done
└── 004-cancelled-task.md        # state: closed
` + "```" + `

이슈 상태는 파일의 YAML frontmatter에 있는 ` + "`state`" + ` 필드로 결정됩니다.

### 이슈 생성 (중요!)

#### zap new 명령 사용 (권장)

이슈 생성 시 반드시 ` + "`zap new`" + ` 명령을 사용하세요. 올바른 형식이 자동으로 적용됩니다:

` + "```" + `bash
# 기본 사용법
zap new "이슈 제목"

# 레이블 추가
zap new "버그 수정" -l bug -l urgent

# 담당자 추가
zap new "기능 구현" -a username

# 본문 추가
zap new "이슈 제목" --body "상세 설명 내용"

# 파이프로 본문 전달 (AI 사용 시 유용)
echo "상세 본문 내용" | zap new "이슈 제목"

# 에디터로 본문 작성
zap new "이슈 제목" --editor
` + "```" + `

#### 수동 생성 시 정확한 형식 (zap new 사용 불가 시)

수동으로 이슈를 생성해야 하는 경우, 아래 형식을 **정확히** 따르세요:

` + "```" + `markdown
---
number: 1
title: "이슈 제목"
state: open
labels:
  - bug
  - urgent
assignees:
  - username
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

## 개요

이슈 본문 내용...
` + "```" + `

**필수 검증 체크리스트:**
- [ ] 파일이 ` + "`---`" + `로 시작
- [ ] ` + "`number`" + `: 양의 정수, 파일명과 일치
- [ ] ` + "`title`" + `: 비어있지 않은 문자열 (따옴표 권장)
- [ ] ` + "`state`" + `: open, wip, done, closed 중 하나
- [ ] ` + "`labels`" + `: YAML 배열 형식 (비어있으면 ` + "`[]`" + `)
- [ ] ` + "`assignees`" + `: YAML 배열 형식 (비어있으면 ` + "`[]`" + `)
- [ ] 날짜: RFC3339/ISO8601 형식 (` + "`YYYY-MM-DDTHH:MM:SSZ`" + `)
- [ ] frontmatter가 ` + "`---`" + `로 종료

**파일명 규칙:** ` + "`NNN-slug.md`" + `
- NNN: 3자리 제로패딩 숫자 (예: 001, 024)
- slug: 소문자, 하이픈 구분, 한글 지원
- 예: ` + "`024-feat-user-auth.md`" + `, ` + "`025-버그-수정.md`" + `

### zap CLI 명령어

#### 이슈 생성

` + "```" + `bash
zap new "제목"              # 새 이슈 생성
zap new "제목" -l label     # 레이블과 함께 생성
zap new "제목" -a user      # 담당자와 함께 생성
zap new "제목" -b "본문"    # 본문과 함께 생성
` + "```" + `

#### 목록 조회

` + "```" + `bash
zap list                    # 열린 이슈 (open + wip)
zap list --all              # 전체 이슈
zap list --state open       # 특정 상태만
zap list --label bug        # 레이블 필터
zap list --assignee user    # 담당자 필터
` + "```" + `

#### 상세 보기

` + "```" + `bash
zap show 1                  # 이슈 #1 상세
zap show 1 --raw            # 원본 마크다운 출력
` + "```" + `

#### 상태 변경

상태 변경 시 파일의 frontmatter가 업데이트됩니다 (파일 위치 변경 없음):

` + "```" + `bash
zap set open 1              # state: open (이슈 재오픈)
zap set wip 1               # state: wip (작업 시작)
zap set done 1              # state: done (작업 완료)
zap set closed 1            # state: closed (취소/보류)
` + "```" + `

#### 상태 선택 가이드

| 상태 | 의미 | 사용 시점 |
|------|------|----------|
| ` + "`open`" + ` | 📋 대기 중 | 새로 등록된 이슈, 아직 작업 시작 전 |
| ` + "`wip`" + ` | 🔄 진행 중 | 현재 작업 중인 이슈 |
| ` + "`done`" + ` | ✅ 작업 완료 | 요청한 기능/수정을 **성공적으로 구현**했을 때 |
| ` + "`closed`" + ` | ❌ 진행 안 함 | 취소, 중복, 불필요, 범위 외로 **작업 없이 닫을 때** |

**done vs closed 핵심 구분:**
- 코드를 작성/수정했다 → ` + "`done`" + `
- 작업 없이 닫는다 → ` + "`closed`" + `

#### 검색

` + "```" + `bash
zap list --search "키워드"   # 제목/내용 검색
zap list --title-only       # 제목만 검색
` + "```" + `

#### 통계

` + "```" + `bash
zap stats                   # 상태별 이슈 수, 최근 활동
` + "```" + `

#### 마이그레이션

기존 디렉토리 기반 구조를 사용 중이라면:

` + "```" + `bash
zap migrate                 # 평면 구조로 마이그레이션
zap migrate --dry-run       # 변경 사항 미리보기
` + "```" + `

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
