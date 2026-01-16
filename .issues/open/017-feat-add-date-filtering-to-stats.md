---
number: 17
title: "feat: zap stats에 날짜 필터링 옵션 추가"
state: open
labels:
  - enhancement
  - cli
assignees: []
created_at: 2026-01-16T00:00:00Z
updated_at: 2026-01-16T00:00:00Z
closed_at:
---

## 개요

`zap stats`에 날짜/기간 기반 필터링 옵션을 추가하여 특정 기간의 통계만 조회 가능하게 함.

`zap list`의 날짜 필터링 (#16)과 동일한 옵션 체계 사용.

## 현재 상태

```bash
# 전체 이슈 통계만 표시
zap stats
```

## 제안 옵션

| 옵션 | 설명 | 예시 |
|------|------|------|
| `--today` | 오늘 생성/수정된 이슈 통계 | `zap stats --today` |
| `--since <date>` | 지정일 이후 통계 | `zap stats --since 2026-01-01` |
| `--until <date>` | 지정일 이전 통계 | `zap stats --until 2026-01-15` |
| `--year <YYYY>` | 해당 연도 통계 | `zap stats --year 2026` |
| `--month <YYYY-MM>` | 해당 월 통계 | `zap stats --month 2026-01` |
| `--date <YYYY-MM-DD>` | 특정 날짜 통계 | `zap stats --date 2026-01-16` |

## 사용 예시

```bash
# 오늘 생성/수정된 이슈 통계
zap stats --today

# 이번 달 통계
zap stats --month 2026-01

# 1월 1일 이후 통계
zap stats --since 2026-01-01

# 기간 범위 조합
zap stats --since 2026-01-01 --until 2026-01-15
```

## 작업 목록

- [ ] 날짜 옵션 플래그 추가 (internal/cli/stats.go)
- [ ] Store.Stats()에 날짜 필터링 지원 추가
- [ ] `--today`, `--since`, `--until`, `--year`, `--month`, `--date` 구현
- [ ] 테스트 추가

## 의존성

- #16 (zap list 날짜 필터링) - 공통 날짜 필터링 로직 재사용

## 구현 참고

- #16의 날짜 파싱/필터링 로직을 공통 유틸로 분리하여 재사용
- 통계 출력에 필터링 기간 표시 추가 (예: "Statistics for 2026-01")
