---
number: 36
title: fix-datetime-format에 --analyze 플래그 추가
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-19T02:01:13Z"
updated_at: "2026-01-19T02:11:49Z"
closed_at: "2026-01-19T02:11:49Z"
---

## 개요

`zap fix-datetime-format` 명령에 `--analyze` 플래그를 추가하여 현재 이슈들의 날짜/시각 포맷 분포를 분석하는 기능 구현.

## 배경

- 현재 `parseFlexibleTime`이 5가지 포맷을 valid로 처리함
- 어떤 포맷이 버그인지 정상인지 혼란스러울 때가 있음
- 변환 전 분석, 디버깅/검증, 문서화/정책 용도로 포맷 통계 필요

## 지원 포맷 (parser.go:36-42)

1. `RFC3339` - `2026-01-17T15:47:00Z`
2. `2006-01-02T15:04:05` - `2026-01-17T15:47:00`
3. `2006-01-02 15:04:05` - `2026-01-17 15:47:00`
4. `2006-01-02 15:04` - `2026-01-17 15:47`
5. `2006-01-02` - `2026-01-17`

## 구현 요구사항

### 명령어
```bash
zap fix-datetime-format --analyze
```

### 출력 내용
1. **포맷별 통계**: 5가지 지원 포맷 각각에 해당하는 이슈 개수
2. **필드별 분리**: `created_at`, `updated_at`, `closed_at` 각각 따로 통계
3. **예시 이슈 번호**: 각 포맷에 해당하는 이슈 번호를 함께 표시

### 예상 출력
```
DateTime Format Analysis:

created_at:
  RFC3339 (2026-01-17T15:47:00Z): 10 issues (#1, #3, #5...)
  YYYY-MM-DD HH:MM:               2 issues (#7, #12)
  YYYY-MM-DD:                     1 issue (#15)

updated_at:
  RFC3339 (2026-01-17T15:47:00Z): 12 issues (#1, #2...)
  ...

Summary:
  Total issues: 15
  Already RFC3339: 30 fields
  Need conversion: 5 fields
```

## 기술 참고

- `internal/cli/fix_datetime.go` 수정
- `internal/issue/parser.go`의 `parseFlexibleTime` 포맷 목록 참조
- 원본 문자열 값을 분석해야 하므로 `rawFrontmatter` 활용 필요

---

## 추가 개선 (2026-01-19)

### 문제점 발견

`fix-datetime-format` 실행 시, 원본 파일에 `2026-01-16` (date-only) 포맷이 있어도 "변경 없음"으로 처리됨:

```bash
$ zap fix-datetime-format --dry-run
Dry run complete. Would update 0 issues (36 already correct).
```

**원인**: 현재 로직은 파싱된 `time.Time` 값만 비교하므로, 원본 문자열 포맷이 RFC3339인지 아닌지 감지하지 못함.

```
2026-01-16 → 파싱 → time(2026-01-16 00:00:00 UTC) → RFC3339 비교 → 동일 → 변경 안 함
```

### 개선 방향

1. **원본 포맷 기반 감지**: 원본 문자열 포맷이 RFC3339가 아니면 변환 대상으로 인식
2. **`--git-dates` 확장**: date-only 포맷(`2026-01-16`)도 git history에서 정확한 시간 정보 가져오기

### 예상 동작

```bash
# 기본 동작: date-only → RFC3339 (시간은 00:00:00)
$ zap fix-datetime-format
Issue #13: created_at: 2026-01-16 → 2026-01-16T00:00:00Z

# --git-dates: date-only → git에서 실제 시간 가져오기
$ zap fix-datetime-format --git-dates
Issue #13: created_at: 2026-01-16 → 2026-01-16T15:30:00Z (from git)
```
