---
number: 14
title: "feat: list 출력 시 status별 전체 행 색상 적용 및 터미널 감지"
state: done
labels:
  - feature
  - cli
  - ux
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 설명

`zap list` 출력에서 status별로 전체 행에 색상을 적용하고, 터미널 색상 지원 여부를 감지하여 안전하게 처리합니다.

## 현재 상태

### 1. 색상 적용 범위 제한

현재는 기호(symbol)에만 색상이 적용됨:

```
◐ #1    작업 중인 이슈    ← 기호만 노란색, 나머지는 기본색
```

### 2. 터미널 감지 없음

```go
// list.go:92-96 - 하드코딩된 ANSI 코드
const (
    colorReset  = "\033[0m"
    colorYellow = "\033[33m"
    // ...
)
```

**문제점**:
- 색상 미지원 터미널: `[33m◐[0m #1 제목` 출력 가능
- 파이프/리다이렉트 (`zap list > file.txt`): escape 코드가 파일에 포함
- `NO_COLOR` 환경변수: 무시됨 (표준 미지원)

## 제안

### 1. 전체 행 색상 적용

```
◐ #1    작업 중인 이슈 [bug]     ← 전체 노란색
● #2    완료된 이슈              ← 전체 초록색
✕ #3    닫힌 이슈                ← 전체 회색
○ #4    열린 이슈                ← 기본색 (또는 파란색)
```

### 2. 터미널 색상 지원 감지

이미 간접 의존성으로 포함된 라이브러리 활용:
- `mattn/go-isatty v0.0.20` - TTY 감지
- `muesli/termenv v0.16.0` - 터미널 환경 감지

## 구현 계획

### 1. 색상 유틸리티 추가 (`internal/cli/color.go`)

```go
package cli

import (
    "os"
    "github.com/mattn/go-isatty"
)

var colorEnabled bool

func init() {
    // NO_COLOR 환경변수 지원 (https://no-color.org/)
    if os.Getenv("NO_COLOR") != "" {
        colorEnabled = false
        return
    }

    // TTY 여부 확인
    colorEnabled = isatty.IsTerminal(os.Stdout.Fd()) ||
                   isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func colorize(text, color string) string {
    if !colorEnabled || color == "" {
        return text
    }
    return color + text + colorReset
}
```

### 2. list.go 수정

```go
func printIssueList(issues []*issue.Issue, skippedCount int) {
    for _, iss := range issues {
        style := stateStyle[iss.State]
        labels := ""
        if len(iss.Labels) > 0 {
            labels = fmt.Sprintf(" [%s]", strings.Join(iss.Labels, ", "))
        }

        line := fmt.Sprintf("%s #%-4d %s%s", style.symbol, iss.Number, iss.Title, labels)
        fmt.Println(colorize(line, style.color))
    }
    // ...
}
```

### 3. search.go 동일 패턴 적용

search 명령도 동일한 출력 로직 사용 중 (search.go:52-67)

## 영향 범위

- `internal/cli/list.go` - 목록 출력
- `internal/cli/search.go` - 검색 결과 출력
- `internal/cli/repair.go` - diff 출력 (이미 색상 사용 중)
- 새 파일: `internal/cli/color.go` - 색상 유틸리티

## 완료 기준

- [x] 색상 유틸리티 함수 추가 (`colorize`, `colorEnabled`)
- [x] `NO_COLOR` 환경변수 지원
- [x] TTY 감지 (파이프/리다이렉트 시 색상 비활성화)
- [x] `zap list` 전체 행 색상 적용
- [x] `zap search` 전체 행 색상 적용
- [ ] `--no-color` 플래그 추가 (선택) - 미구현, NO_COLOR 환경변수로 대체 가능
- [x] 기존 테스트 통과 확인

## 구현 결과

### 변경된 파일

| 파일 | 변경 내용 |
|------|----------|
| `internal/cli/color.go` | 새 파일 - 색상 유틸리티 (`colorize`, `colorEnabled`) |
| `internal/cli/list.go` | 전체 행 색상 적용, 중복 색상 상수 제거 |
| `internal/cli/search.go` | 전체 행 색상 적용, 하이라이트 TTY 감지 추가 |
| `internal/cli/repair.go` | `colorize` 함수 사용으로 변경 |

### 테스트 결과

```bash
# 터미널에서 실행 - 색상 적용됨
./zap list -a

# NO_COLOR 환경변수 - 색상 없음
NO_COLOR=1 ./zap list -a

# 파이프 출력 - 색상 없음 (TTY 감지)
./zap list -a | cat
```

## 참고

- NO_COLOR 표준: https://no-color.org/
- mattn/go-isatty: https://github.com/mattn/go-isatty
