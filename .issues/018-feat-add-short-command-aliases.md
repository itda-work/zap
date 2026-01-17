---
number: 18
title: 'feat: 명령어 짧은 alias 추가'
state: done
labels:
    - enhancement
    - cli
    - ux
assignees: []
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-17T10:15:24.560522+09:00
closed_at: 2026-01-17T10:15:24.560522+09:00
---

## 개요

자주 사용하는 명령어에 짧은 alias를 추가하여 사용 편의성을 높임.

현재는 `list` → `ls` alias만 존재함.

## 현재 상태

```bash
# 긴 명령어만 지원
zap show 15
zap stats
zap start 15
zap done 15
```

## 제안 Alias 목록

| 명령어 | 짧은 Alias | 설명 |
|--------|------------|------|
| `show` | `s` | 이슈 상세 보기 |
| `list` | `ls` | ✅ 이미 존재 |
| `stats` | `st` | 통계 보기 |
| `start` | `wip` | 작업 시작 (work-in-progress) |
| `done` | `d` | 완료 처리 |
| `close` | `c` | 닫기 (취소/보류) |
| `open` | `o` | 열기/재열기 |
| `repair` | `r` | AI 복구 |
| `init` | `i` | 초기화 |
| `update` | `up` | 업데이트 |

## 사용 예시

```bash
# 이슈 상세 보기
zap s 15          # zap show 15

# 통계 보기
zap st            # zap stats
zap st --today    # zap stats --today

# 상태 변경
zap wip 15        # zap start 15
zap d 15          # zap done 15
zap c 15          # zap close 15
zap o 15          # zap open 15

# 기타
zap r --auto      # zap repair --auto
zap i             # zap init
zap up            # zap update
```

## 작업 목록

- [x] `show` → `s` alias 추가
- [x] `stats` → `st` alias 추가
- [x] `start` → `wip` alias 추가
- [x] `done` → `d` alias 추가
- [x] `close` → `c` alias 추가
- [x] `open` → `o` alias 추가
- [x] `repair` → `r` alias 추가
- [x] `init` → `i` alias 추가
- [x] `update` → `up` alias 추가
- [x] 테스트

## 구현 참고

- cobra의 `Aliases` 필드 사용 (list.go 참고)
- `completion`, `help`, `version`은 자주 사용하지 않으므로 제외
- `start` → `wip`은 상태명(`in-progress`)과 연관되어 직관적

## 구현 내역

각 CLI 명령의 `cobra.Command` 구조체에 `Aliases` 필드 추가:

| 파일 | 추가된 Alias |
|------|-------------|
| `internal/cli/show.go` | `s` |
| `internal/cli/stats.go` | `st` |
| `internal/cli/move.go` | `o`, `wip`, `d`, `c` |
| `internal/cli/repair.go` | `r` |
| `internal/cli/init.go` | `i` |
| `internal/cli/update.go` | `up` |
