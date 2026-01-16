---
number: 11
title: "feat: AI CLI를 활용한 frontmatter 자동 복구"
state: open
labels:
  - feature
  - ai
  - cli
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 설명

이슈 파일의 frontmatter 파싱이 실패할 때, 로컬 AI CLI 도구를 활용하여 자동으로 복구하는 기능입니다.

## 사용 시나리오

### 자동 복구 제안

```bash
❯ zap list

⚠️ 파싱 실패 (2 files):
  - 158-featcli-부가세-수집.md: missing 'number' field
  - 159-refactor-분개장.md: invalid YAML syntax

AI로 자동 복구하시겠습니까? [Y/n]: y

🤖 claude를 사용하여 복구 중...
  ✓ 158-featcli-부가세-수집.md - 복구 완료
  ✓ 159-refactor-분개장.md - 복구 완료

○ #142 feat(browser): ...
○ #158 feat(cli): 부가세 수집 명령에 interval 옵션 추가
○ #159 refactor: 분개장 captureNetworkResponse를 captureNetworkTraffic으로 변경
◐ #146 feat(wehago): ...

Total: 4 issues
```

### 명시적 복구 명령

```bash
# 특정 파일 복구
zap repair 158

# 모든 파싱 실패 파일 복구
zap repair --all

# AI 도구 지정
zap repair --ai codex --all

# dry-run (변경 내용 미리보기)
zap repair --dry-run --all
```

## 구현 계획

> **Note**: AI 모듈 기반은 #12에서 구현. 이 이슈는 repair Task 구현에 집중.

### 1. RepairTask 구현 (`internal/ai/tasks/repair.go`)

```go
type RepairInput struct {
    FilePath string
    Content  string
    Filename string
}

type RepairOutput struct {
    Content  string
    Changes  []string  // 변경 사항 목록
    Fixed    bool
}

type RepairTask struct{}

func (t *RepairTask) Name() string { return "repair-frontmatter" }

func (t *RepairTask) Execute(ctx context.Context, client ai.Client, input interface{}) (interface{}, error) {
    in := input.(*RepairInput)

    req, _ := ai.Templates["repair-frontmatter"].Render(map[string]string{
        "filename": in.Filename,
        "content":  in.Content,
    })

    resp, err := client.Complete(ctx, req)
    if err != nil {
        return nil, err
    }

    return &RepairOutput{
        Content: resp.Content,
        Fixed:   true,
    }, nil
}
```

### 2. repair 명령 (`internal/cli/repair.go`)

```go
var repairCmd = &cobra.Command{
    Use:   "repair [number]",
    Short: "AI를 사용하여 이슈 파일 복구",
    RunE:  runRepair,
}

func runRepair(cmd *cobra.Command, args []string) error {
    client, err := getAIClient(cmd)  // #12에서 제공
    if err != nil {
        return fmt.Errorf("AI not available: %w", err)
    }

    // 파싱 실패한 파일 목록 가져오기
    failures := store.GetParseFailures()  // #9에서 구현

    for _, f := range failures {
        result, err := ai.RunTask(ctx, "repair-frontmatter", &RepairInput{
            FilePath: f.Path,
            Content:  f.Content,
            Filename: f.Name,
        })
        // ...
    }
}
```

### 3. 프롬프트 템플릿

```yaml
# ~/.config/zap/prompts/repair-frontmatter.yaml
name: repair-frontmatter
system: |
  You are a YAML frontmatter repair assistant.
  Fix issues in markdown frontmatter for issue tracking files.
user: |
  Fix the YAML frontmatter in this issue file.
  Filename: {{.filename}}

  Rules:
  - Must start and end with ---
  - Required: number, title, state, labels, assignees, created_at, updated_at
  - Extract number from filename if missing (e.g., "158-feat..." → 158)
  - state: open | in-progress | done | closed

  Content:
  {{.content}}

  Return ONLY the corrected file, no explanation.
```

## 안전장치

1. **백업**: 복구 전 원본 파일 `.backup` 확장자로 백업
2. **diff 표시**: 변경 내용 diff로 표시 후 확인
3. **dry-run**: 실제 변경 없이 미리보기
4. **rollback**: `zap repair --undo` 마지막 복구 취소

## 완료 기준

- [ ] RepairTask 구현
- [ ] `zap repair` 명령 구현
- [ ] 프롬프트 템플릿 작성
- [ ] dry-run 및 백업 기능
- [ ] `zap list`에서 자동 복구 제안 (opt-in)
- [ ] 단위 테스트

## 의존성

- **#12** feat: 재사용 가능한 AI 모듈 설계 (AI Client, Template 시스템)
- **#9** fix: zap list가 파싱 실패한 이슈를 조용히 무시함 (실패 목록 API)
