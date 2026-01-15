package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <agent>",
	Short: "Initialize agent instruction file",
	Long: `Initialize an instruction file for AI coding assistants.

Supported agents:
  claude    Create CLAUDE.md for Claude Code
  codex     Create AGENTS.md for OpenAI Codex CLI
  gemini    Create GEMINI.md for Google Gemini

Examples:
  lim init claude                       # Create CLAUDE.md in project root
  lim init claude --path AI_GUIDE.md    # Create AI_GUIDE.md instead
  lim init codex --path docs/AGENTS.md  # Create docs/AGENTS.md`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"claude", "codex", "gemini"},
	RunE:      runInit,
}

var initPath string

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVar(&initPath, "path", "", "File path for instruction file (default: CLAUDE.md/AGENTS.md/GEMINI.md)")
}

// agentConfig holds configuration for each agent type
type agentConfig struct {
	filename string
	header   string
}

var agentConfigs = map[string]agentConfig{
	"claude": {
		filename: "CLAUDE.md",
		header:   "# Local Issue Management (lim) - Claude Instructions",
	},
	"codex": {
		filename: "AGENTS.md",
		header:   "# Local Issue Management (lim) - Codex Instructions",
	},
	"gemini": {
		filename: "GEMINI.md",
		header:   "# Local Issue Management (lim) - Gemini Instructions",
	},
}

func runInit(cmd *cobra.Command, args []string) error {
	agent := strings.ToLower(args[0])

	config, ok := agentConfigs[agent]
	if !ok {
		return fmt.Errorf("unsupported agent: %s (supported: claude, codex, gemini)", agent)
	}

	// Determine target file path
	var targetFile string
	if initPath != "" {
		// Use provided file path directly
		targetFile = initPath
	} else {
		// Default to agent's default filename in project root
		targetFile = config.filename
	}

	// Generate instruction content
	content := generateInstructions(config.header)

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

		fmt.Printf("✅ Appended lim instructions to %s\n", targetFile)
	} else {
		// Create new file
		if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		fmt.Printf("✅ Created %s\n", targetFile)
	}

	return nil
}

func generateInstructions(header string) string {
	return header + `

이 프로젝트는 로컬 이슈 관리 시스템(.issues/)을 사용합니다.

## .issues/ 디렉토리 구조

` + "```" + `
.issues/
├── open/           # 새로 생성된 이슈
├── in-progress/    # 진행 중인 이슈
└── done/           # 완료된 이슈
` + "```" + `

이슈 파일은 해당 상태의 디렉토리에 위치하며, 상태 변경 시 파일이 다른 디렉토리로 이동됩니다.

## 이슈 파일 형식

이슈 파일은 YAML frontmatter와 Markdown 본문으로 구성됩니다:

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

파일명 규칙: ` + "`NNN-slug.md`" + ` (예: ` + "`001-feat-user-auth.md`" + `)

## lim CLI 명령어

### 목록 조회

` + "```" + `bash
lim list                    # 열린 이슈 (open + in-progress)
lim list --all              # 전체 이슈
lim list --state open       # 특정 상태만
lim list --label bug        # 레이블 필터
lim list --assignee user    # 담당자 필터
` + "```" + `

### 상세 보기

` + "```" + `bash
lim show 1                  # 이슈 #1 상세
lim show 1 --raw            # 원본 마크다운 출력
` + "```" + `

### 상태 변경

` + "```" + `bash
lim open 1                  # → open/ (이슈 재오픈)
lim start 1                 # → in-progress/ (작업 시작)
lim done 1                  # → done/ (작업 완료)
` + "```" + `

### 검색

` + "```" + `bash
lim search "키워드"          # 제목/내용 검색
lim search --title "키워드"  # 제목만 검색
` + "```" + `

### 통계

` + "```" + `bash
lim stats                   # 상태별 이슈 수, 최근 활동
` + "```" + `

### TUI 모드

` + "```" + `bash
lim                         # TUI 모드 진입
lim tui                     # TUI 모드 진입 (명시적)
` + "```" + `

TUI 단축키:
- ` + "`j/k`" + ` 또는 ` + "`↑/↓`" + `: 이동
- ` + "`Enter`" + `: 상세 보기
- ` + "`1/2/3`" + `: 상태별 필터 (open/in-progress/done)
- ` + "`0`" + `: 전체 보기
- ` + "`r`" + `: 새로고침
- ` + "`/`" + `: 검색
- ` + "`q`" + `: 종료

## 워크플로우

1. **새 이슈 생성**: ` + "`.issues/open/NNN-slug.md`" + ` 파일을 직접 생성
2. **작업 시작**: ` + "`lim start <number>`" + ` 실행
3. **작업 완료**: ` + "`lim done <number>`" + ` 실행

## 주의사항

- 이슈 번호는 고유해야 합니다
- 파일명의 번호와 frontmatter의 number가 일치해야 합니다
- 상태 변경 시 파일이 자동으로 해당 디렉토리로 이동됩니다
`
}
