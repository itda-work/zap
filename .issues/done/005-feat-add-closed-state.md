---
number: 5
title: "feat: closed 상태 추가 (취소/보류)"
state: in-progress
labels:
  - feature
assignees:
  - allieus
created_at: 2026-01-15T00:00:00Z
updated_at: 2026-01-15T00:00:00Z
closed_at:
---

## 개요

`.issues/closed/` 디렉토리를 추가하여 취소/보류된 이슈를 관리하는 상태 추가

## 배경

현재 3가지 상태(open, in-progress, done)만 지원하지만, 취소되거나 보류된 이슈를 별도로 관리할 필요가 있음

## 변경 후 디렉토리 구조

```
.issues/
├── open/           # 대기 중
├── in-progress/    # 진행 중
├── done/           # 완료
└── closed/         # 취소/보류 (신규)
```

## 작업 목록

- [x] internal/issue/issue.go 수정 (StateClosed 상수, AllStates, ParseState)
- [x] internal/cli/move.go - closeCmd 추가
- [x] internal/cli/list.go - stateSymbol 추가
- [x] internal/cli/search.go - stateSymbol 추가
- [x] internal/cli/stats.go - stateOrder, stateEmoji 추가
- [x] internal/cli/init.go - 문서 업데이트
- [x] internal/issue/issue_test.go - 테스트 업데이트
- [x] 빌드 및 테스트 검증

## 상태별 기호

| 상태 | 기호 | 설명 |
|------|------|------|
| open | ○ | 대기 중 |
| in-progress | ◐ | 진행 중 |
| done | ● | 완료 |
| closed | ✕ | 취소/보류 (신규) |

## 진행 내역

### 2026-01-15

- 이슈 생성
- 구현 완료:
  - `StateClosed` 상수 및 관련 함수 추가
  - `zap close <number>` 명령어 추가
  - 모든 CLI 출력에 closed 상태 기호(✕) 추가
  - 테스트 케이스 업데이트 (4개 상태)
  - 빌드 및 기능 테스트 완료
