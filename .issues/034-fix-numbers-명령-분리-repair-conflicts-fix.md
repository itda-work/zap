---
number: 34
title: fix-numbers 명령 분리 (repair --conflicts → fix-numbers)
state: done
labels:
    - refactor
    - cli
assignees: []
created_at: 2026-01-19T09:49:52.436249+09:00
updated_at: 2026-01-19T09:49:57.263343+09:00
closed_at: 2026-01-19T09:49:57.263343+09:00
---

## 개요

`repair --conflicts` 기능을 별도의 `fix-numbers` 명령으로 분리하여 명령 책임을 명확히 함.

## 변경 사항

| 명령 | 역할 |
|------|------|
| `repair` | YAML frontmatter 파싱 오류 복구 |
| `fix-numbers` | 이슈 번호 충돌 감지 및 해결 |

## 변경 파일

| 파일 | 액션 | 설명 |
|------|------|------|
| `internal/cli/utils.go` | 신규 | 공유 함수 (`confirm`, `getAIClient`) |
| `internal/cli/fix_numbers.go` | 신규 | fix-numbers 명령 구현 |
| `internal/cli/repair.go` | 수정 | `--conflicts` 제거, 충돌 코드 제거 |
| `internal/cli/migrate.go` | 수정 | `confirmMigrate()` → `confirm()` 호출 |

## fix-numbers 플래그

```
--dry-run    변경 없이 미리보기
--yes, -y    확인 없이 실행
--ai         AI CLI 지정 (claude, codex, gemini)
--no-ai      AI 검증 건너뛰기
```

## 네이밍 컨벤션

`fix-` prefix로 수정 관련 명령 일관성 유지:
- `fix-datetime-format`: 날짜 형식 수정
- `fix-numbers`: 이슈 번호 충돌 수정
