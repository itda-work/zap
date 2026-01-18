---
number: 33
title: "feat(cli): add fix-datetime-format command"
state: open
labels:
  - feature
  - cli
assignees: []
created_at: 2026-01-19T09:30:00.000000+09:00
updated_at: 2026-01-19T09:30:00.000000+09:00
---

## 개요

이슈 파일의 날짜 필드를 표준 형식으로 통일하는 `zap fix-datetime-format` 명령 추가.

## 배경

현재 이슈 파일들의 날짜 형식이 일관되지 않음:
- `2026-01-16` (날짜만)
- `2026-01-15T18:39:24Z` (UTC)
- `2026-01-17T15:30:00+09:00` (타임존 오프셋)
- `2026-01-17T10:39:48.183091+09:00` (마이크로초 + 타임존)

## 표준 형식

```
2026-01-17T10:39:48.183091+09:00
```
- RFC3339Nano 형식
- 마이크로초 포함 (6자리)
- 로컬 타임존 오프셋

## 명령 구조

```bash
zap fix-datetime-format [options]
```

### 옵션

| 옵션 | 설명 | 기본값 |
|------|------|--------|
| `--dry-run` | 변경사항 미리보기만 | false |
| `--timezone` | 타임존 지정 (예: `Asia/Seoul`) | 시스템 로컬 |
| `--git-dates` | 날짜가 zero value일 때 git 날짜 사용 | false |
| `--number`, `-n` | 특정 이슈만 처리 | 전체 |

## 변환 규칙

| 입력 | 출력 |
|------|------|
| `2026-01-16` | `2026-01-16T00:00:00.000000+09:00` |
| `2026-01-15T18:39:24Z` | `2026-01-16T03:39:24.000000+09:00` |
| `2026-01-17T15:30:00+09:00` | `2026-01-17T15:30:00.000000+09:00` |
| `0001-01-01...` (--git-dates) | git 커밋 날짜 |

## 구현

### 파일 구조

```
internal/
├── cli/
│   └── fix_datetime.go    # CLI 명령
└── issue/
    └── datetime.go        # 날짜 파싱/변환 로직
```

### 작업 목록

- [ ] `internal/issue/datetime.go` - 유연한 날짜 파싱 함수
- [ ] `internal/issue/datetime.go` - 표준 형식 변환 함수
- [ ] `internal/cli/fix_datetime.go` - CLI 명령 구현
- [ ] `--dry-run` 옵션 구현
- [ ] `--git-dates` 옵션 구현
- [ ] `--timezone` 옵션 구현
- [ ] `--number` 옵션 구현
- [ ] 테스트 작성

## 예시

```bash
# 미리보기
zap fix-datetime-format --dry-run

# 전체 적용
zap fix-datetime-format

# git 날짜로 빈 값 채우기
zap fix-datetime-format --git-dates

# 특정 이슈만
zap fix-datetime-format -n 1
```
