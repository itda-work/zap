---
number: 2
title: 'feat: Markdown syntax highlighting 지원'
state: done
labels:
    - enhancement
    - tui
assignees: []
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
---

## 개요

이슈 상세 보기 시 Markdown 콘텐츠에 syntax highlighting을 적용합니다.

## 현재 상황

- `lim show` CLI 명령어: plain text로 출력
- `lim tui` 상세 뷰: plain text로 출력
- 코드 블록, 헤더, 링크 등이 구분되지 않아 가독성 저하

## 제안

### 옵션 1: glamour 라이브러리 사용 (추천)

[charmbracelet/glamour](https://github.com/charmbracelet/glamour)는 터미널용 Markdown 렌더러입니다.

```go
import "github.com/charmbracelet/glamour"

renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),
    glamour.WithWordWrap(80),
)
out, _ := renderer.Render(markdownContent)
fmt.Print(out)
```

**장점:**
- Bubble Tea/Lipgloss와 같은 Charm 생태계
- 코드 블록 syntax highlighting (chroma 기반)
- 다크/라이트 테마 자동 감지
- 터미널 너비에 맞춰 word wrap

**단점:**
- 추가 의존성

### 옵션 2: 직접 ANSI 스타일링

간단한 패턴 매칭으로 기본적인 스타일링만 적용

**장점:**
- 의존성 없음
- 가벼움

**단점:**
- 코드 블록 syntax highlighting 미지원
- 복잡한 마크다운 처리 어려움

## 적용 범위

1. **CLI `lim show`**: glamour로 렌더링된 출력
2. **TUI 상세 뷰**: glamour로 렌더링된 내용 표시
3. **TUI 목록**: 제목만 표시하므로 변경 불필요

## 작업 목록

- [x] glamour 의존성 추가
- [x] CLI show 명령어에 glamour 렌더링 적용
- [x] TUI 상세 뷰에 glamour 렌더링 적용
- [x] 터미널 너비에 맞는 word wrap 설정
- [ ] 테마 설정 옵션 추가 (선택, 추후)

## 진행 내역

### 2026-01-15

- glamour v0.10.0 의존성 추가
- CLI `lim show` 명령어에 glamour 렌더링 적용 (`internal/cli/show.go`)
- TUI 상세 뷰에 glamour 렌더링 적용 (`internal/tui/app.go`)
- 자동 스타일 감지 (`glamour.WithAutoStyle()`)
- 터미널 너비에 맞는 word wrap 적용

## 참고

- [glamour GitHub](https://github.com/charmbracelet/glamour)
- [glow](https://github.com/charmbracelet/glow) - glamour 기반 CLI 마크다운 뷰어
