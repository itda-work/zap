---
number: 16
title: 'feat: zap list에 날짜 필터링 옵션 추가'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-16T23:03:27Z
updated_at: 2026-01-16T23:29:56Z
---

## 개요

`zap list`에 날짜/기간 기반 필터링 옵션을 추가하여 특정 기간의 이슈만 조회 가능하게 함.

Issue 구조체에 이미 `created_at`, `updated_at`, `closed_at` 필드가 존재하므로 구현 가능.

## 현재 상태

```bash
# 날짜 필터링 불가
zap list --state open  # 모든 open 이슈 표시
```

## 제안 옵션

| 옵션 | 설명 | 예시 |
|------|------|------|
| `--today` | 오늘 생성/수정된 이슈 | `zap list --today` |
| `--since <date>` | 지정일 이후 | `zap list --since 2026-01-01` |
| `--until <date>` | 지정일 이전 | `zap list --until 2026-01-15` |
| `--year <YYYY>` | 해당 연도 이슈 | `zap list --year 2026` |
| `--month <YYYY-MM>` | 해당 월 이슈 | `zap list --month 2026-01` |
| `--date <YYYY-MM-DD>` | 특정 날짜 이슈 | `zap list --date 2026-01-16` |
| `--days <N>` | 최근 N일 이슈 | `zap list --days 7` |
| `--weeks <N>` | 최근 N주 이슈 | `zap list --weeks 2` |

## 사용 예시

```bash
# 오늘 생성/수정된 이슈
zap list --today

# 이번 달 open 이슈
zap list --month 2026-01 --state open

# 1월 1일 이후 bug 레이블 이슈
zap list --since 2026-01-01 --label bug

# 기간 범위 조합
zap list --since 2026-01-01 --until 2026-01-15

# 최근 7일 이슈
zap list --days 7

# 최근 2주 이슈
zap list --weeks 2
```

## 작업 목록

- [x] 날짜 옵션 플래그 추가 (internal/cli/list.go)
- [x] DateFilter 공통 유틸 생성 (internal/cli/datefilter.go)
- [x] `--today`, `--since`, `--until`, `--year`, `--month`, `--date` 구현
- [x] `--days`, `--weeks` 옵션 추가
- [x] 기존 필터들과 AND 조건으로 동작
- [x] 날짜 파싱 및 유효성 검사
- [ ] 테스트 추가

## 구현 참고

- 날짜 필터 기준: `created_at` 또는 `updated_at` (둘 다 검사)
- 날짜 형식: ISO 8601 (YYYY-MM-DD)

## 구현 기록

### 2026-01-16

- `DateFilter` 구조체와 공통 유틸 생성 (internal/cli/datefilter.go)
- `--today`, `--since`, `--until`, `--year`, `--month`, `--date` 플래그 추가
- `--days`, `--weeks` 플래그 추가 (최근 N일/N주 필터)
- `FilterIssuesByDate()` 함수 구현
- `GetDateRange()` 메서드로 날짜 범위 계산
- created_at 또는 updated_at 기준으로 필터링 (둘 중 하나만 범위 내에 있으면 포함)
