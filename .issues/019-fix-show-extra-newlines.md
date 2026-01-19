---
number: 19
title: 'fix: show 명령 본문 렌더링 시 불필요한 개행 출력'
state: done
labels:
    - bug
    - cli
assignees: []
created_at: 2026-01-17T10:23:25Z
updated_at: 2026-01-17T10:23:16.814577+09:00
closed_at: 2026-01-17T10:23:16.814577+09:00
---

## 개요

`zap show` 명령에서 본문(body) 렌더링 시 각 줄 사이에 불필요한 빈 줄이 출력됨.

## 현상

체크리스트, 제목 등의 마크다운 요소 사이에 추가 개행이 발생:

```
## 작업 목록

[✓] show → s alias 추가

[✓] stats → st alias 추가

[✓] start → wip alias 추가
```

## 원인

glamour 라이브러리의 기본 스타일에 margin/block_suffix가 설정되어 있음.
현재 `compactStyle`은 일부 요소만 처리하고 있어 누락된 요소에서 여백 발생.

### 현재 처리 요소 (4개)
- list, item, paragraph, code_block

### 누락된 요소 (12개)
- document, block_quote, heading, h1~h6
- hr, code, table, definition_list, definition_description
- html_block, html_span

## 수정 방안

`compactStyle`에 모든 여백 관련 요소 추가:

| 요소 | 설정 |
|------|------|
| document | `margin: 0` |
| block_quote | `margin: 0` |
| heading | `margin: 0`, `block_suffix: ""` |
| h1~h6 | `margin: 0`, `block_suffix: ""` |
| hr | `format: "--------"` |
| code | `margin: 0` |
| code_block | `margin: 0` |
| table | `margin: 0` |
| definition_list | `margin: 0` |
| definition_description | `block_prefix: ""` |
| html_block | `margin: 0` |
| html_span | `margin: 0` |

## 작업 목록

- [x] compactStyle JSON 완성 (16개 요소)
- [x] 테스트 마크다운 파일 생성 (모든 요소 포함)
- [x] 테스트 코드 작성 (연속 개행 검증)
- [x] 빌드 및 검증

## 영향 범위

- `internal/cli/show.go`
- `internal/cli/show_test.go` (신규)
- `internal/cli/testdata/all_elements.md` (신규)
