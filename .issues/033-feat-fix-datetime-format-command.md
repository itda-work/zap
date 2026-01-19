---
number: 33
title: "feat(cli): add fix-datetime-format command"
state: open
labels:
  - feature
  - cli
assignees: []
created_at: 2026-01-19T00:30:00Z
updated_at: 2026-01-19T00:30:00Z
---

## 개요

이슈 파일의 날짜 필드를 표준 형식으로 통일하는 `zap fix-datetime-format` 명령 추가.

## 배경

현재 이슈 파일들의 날짜 형식이 일관되지 않음:
- `2026-01-16` (날짜만)
- `2026-01-15T18:39:24Z` (UTC)
- `2026-01-17T15:30:00+09:00` (타임존 오프셋)
- `2026-01-17T10:39:48.183091+09:00` (마이크로초 + 타임존)

## 설계 원칙

**저장은 UTC, 표시는 로컬**

| 구분 | 형식 | 예시 |
|------|------|------|
| 저장 (파일) | UTC (Z) | `2026-01-17T06:30:00Z` |
| 표시 (CLI) | 로컬 타임존 | `2026-01-17 15:30:00` |

## 저장 형식 (파일)

```
2026-01-17T06:30:00Z
```
- RFC3339 형식
- 초 단위까지 (마이크로초 없음)
- **항상 UTC (Z)**

## 표시 형식 (CLI 출력)

```
2026-01-17 15:30:00
```
- 시스템 로컬 타임존으로 변환하여 표시
- `zap list`, `zap show` 등에서 사용

## 명령 구조

```bash
zap fix-datetime-format [options]
```

### 옵션

| 옵션 | 설명 | 기본값 |
|------|------|--------|
| `--dry-run` | 변경사항 미리보기만 | false |
| `--git-dates` | 날짜가 zero value일 때 git 날짜 사용 | false |
| `--number`, `-n` | 특정 이슈만 처리 | 전체 |

## 변환 규칙 (저장 시)

| 입력 | 출력 (UTC) |
|------|------|
| `2026-01-16` | `2026-01-15T15:00:00Z` (로컬 자정 → UTC) |
| `2026-01-17T15:30:00+09:00` | `2026-01-17T06:30:00Z` |
| `2026-01-17T10:39:48.183091+09:00` | `2026-01-17T01:39:48Z` |
| `0001-01-01...` (--git-dates) | git 커밋 날짜 (UTC) |

## 구현

### 수정 필요 파일

#### 1. `internal/cli/new.go` - 이슈 생성 시 UTC 사용

```go
// 변경 전 (line 140, 368)
now := time.Now()

// 변경 후
now := time.Now().UTC()
```

#### 2. `internal/issue/parser.go` - 직렬화 시 RFC3339 형식 강제

```go
// Serialize에서 time.Time을 RFC3339 형식으로 변환
type serializableFrontmatter struct {
    Number    int      `yaml:"number"`
    Title     string   `yaml:"title"`
    State     State    `yaml:"state"`
    Labels    []string `yaml:"labels"`
    Assignees []string `yaml:"assignees"`
    CreatedAt string   `yaml:"created_at"`  // string으로 변환
    UpdatedAt string   `yaml:"updated_at"`  // string으로 변환
}

func Serialize(issue *Issue) ([]byte, error) {
    sf := serializableFrontmatter{
        Number:    issue.Number,
        Title:     issue.Title,
        State:     issue.State,
        Labels:    issue.Labels,
        Assignees: issue.Assignees,
        CreatedAt: issue.CreatedAt.UTC().Format(time.RFC3339),
        UpdatedAt: issue.UpdatedAt.UTC().Format(time.RFC3339),
    }
    // ...
}
```

#### 3. `internal/cli/list.go`, `show.go` - 표시 시 로컬 변환

```go
// 표시할 때 로컬 타임존으로 변환
func formatLocalTime(t time.Time) string {
    return t.Local().Format("2006-01-02 15:04:05")
}
```

### 파일 구조

```
internal/
├── cli/
│   ├── new.go              # time.Now().UTC() 사용
│   ├── list.go             # 로컬 시간 표시
│   ├── show.go             # 로컬 시간 표시
│   └── fix_datetime.go     # 새 명령
└── issue/
    └── parser.go           # RFC3339 직렬화
```

### 작업 목록

**기본 인프라 (신규 이슈 생성 시 적용)**
- [ ] `internal/cli/new.go` - `time.Now().UTC()` 사용
- [ ] `internal/issue/parser.go` - `Serialize()` RFC3339 UTC 형식 강제

**표시 로직 수정**
- [ ] `internal/cli/list.go` - 로컬 타임존 변환 표시
- [ ] `internal/cli/show.go` - 로컬 타임존 변환 표시

**기존 이슈 마이그레이션 (fix-datetime-format 명령)**
- [ ] `internal/cli/fix_datetime.go` - CLI 명령 구현
- [ ] `--dry-run` 옵션 구현
- [ ] `--git-dates` 옵션 구현
- [ ] `--number` 옵션 구현

**테스트**
- [ ] 직렬화/역직렬화 테스트
- [ ] 로컬 표시 변환 테스트
- [ ] fix-datetime-format 명령 테스트

## 예시

```bash
# 미리보기
zap fix-datetime-format --dry-run

# 전체 적용
zap fix-datetime-format

# git 날짜로 빈 값 채우기
zap fix-datetime-format --git-dates

# 특정 이슈만
zap fix-datetime-format -n 1
```
