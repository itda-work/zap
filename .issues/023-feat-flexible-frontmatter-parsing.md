---
number: 23
title: "Flexible frontmatter date field parsing"
state: done
created_at: 2026-01-17T16:30:00Z
updated_at: 2026-01-17T16:45:00Z
---

# #023 Flexible frontmatter date field parsing

## 문제

외부 프로젝트(예: dice)의 이슈 파일이 zap 파서와 호환되지 않음:

| 구분 | zap 형식 | 외부 형식 |
|------|----------|-----------|
| 필드명 | `created_at` / `updated_at` | `created` / `updated` |
| 날짜 | `2026-01-17T15:47:00Z` | `2026-01-17 15:47` |

결과: `zap show`에서 날짜가 `0001-01-01 00:00`으로 표시됨

## 해결 방안

### 1. 중간 파싱 구조체 도입
```go
type rawFrontmatter struct {
    // 두 형식 모두 지원
    CreatedAt string `yaml:"created_at"`
    Created   string `yaml:"created"`
    UpdatedAt string `yaml:"updated_at"`
    Updated   string `yaml:"updated"`
}
```

### 2. 유연한 날짜 파싱 함수
```go
func parseFlexibleTime(s string) (time.Time, error)
// 지원 형식:
// - 2026-01-17T15:47:00Z (ISO8601)
// - 2026-01-17 15:47
// - 2026-01-17
```

### 3. 변환 로직
- `created` 또는 `created_at` 중 존재하는 값 사용
- `updated` 또는 `updated_at` 중 존재하는 값 사용

## 수정 파일
- [x] `internal/issue/parser.go` - 파싱 로직 수정
- [x] `internal/issue/parser_test.go` - 테스트 케이스 추가

## 호환성
- 기존 `created_at`/`updated_at` 형식 계속 지원
- 새로운 `created`/`updated` 형식도 지원
- 저장 시에는 기존 형식 유지
