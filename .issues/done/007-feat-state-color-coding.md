---
number: 7
title: "feat: 이슈 상태 색상 표시"
state: in-progress
labels:
  - enhancement
  - cli
assignees:
  - allieus
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
closed_at:
---

## 개요

`zap list -a` 출력에서 상태별 기호(●, ◐, ○, ✕)만으로는 구별이 어려움. ANSI 색상을 추가하여 가독성 향상.

## 현재 상태

```
● #1    feat: ...  (done? open?)
● #2    feat: ...  (구별 어려움)
◐ #6    feat: ...  (in-progress)
```

## 색상 설계

| 상태 | 기호 | 색상 | ANSI 코드 |
|------|------|------|-----------|
| open | ○ | 기본(흰색) | - |
| in-progress | ◐ | 노란색 | `\033[33m` |
| done | ● | 녹색 | `\033[32m` |
| closed | ✕ | 회색 | `\033[90m` |

## 작업 목록

- [x] internal/cli/list.go 색상 추가
- [x] internal/cli/search.go 색상 추가
- [x] 테스트 및 검증

## 진행 내역

### 2026-01-16

- 이슈 생성
- 구현 완료:
  - list.go: ANSI 색상 코드 추가 (노란색=in-progress, 녹색=done, 회색=closed)
  - search.go: 동일하게 색상 적용
  - 빌드 및 테스트 완료
