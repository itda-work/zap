---
number: 9
title: "fix: zap list가 파싱 실패한 이슈를 조용히 무시함"
state: open
labels:
  - bug
  - cli
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 문제

`zap list` 명령이 파싱에 실패한 이슈 파일을 사용자에게 알리지 않고 조용히 건너뜁니다.

### 재현 방법

```bash
❯ zap list
○ #142 feat(browser): BrowserManager에서 기존 브라우저 연결 지원 [enhancement, browser]
◐ #146 feat(wehago): 신용카드 수집 서비스 구현 (collect-credit-card)

Total: 2 issues

❯ tree ./.issues/open
./.issues/open
├── 142-feat-browsermanager-기존-브라우저-연결-지원.md
├── 158-featcli-부가세-수집-명령에-interval-옵션-추가.md
└── 159-refactor-분개장-captureNetworkResponse를-captureNetworkTraffic으로-변경.md
```

`.issues/open`에 3개의 파일이 있지만 `zap list`는 1개만 표시합니다.

### 원인

`store.go:64-67`에서 파싱 실패 시 에러를 무시합니다:

```go
issue, err := Parse(filePath)
if err != nil {
    // 파싱 실패한 파일은 건너뜀
    continue
}
```

파싱 실패 원인:
- frontmatter에 `number:` 필드 누락
- frontmatter가 `---`로 시작/종료되지 않음
- YAML 문법 오류

## 해결 방안

### 옵션 A: 경고 표시 (권장)
파싱 실패한 파일을 목록 끝에 경고와 함께 표시:

```
○ #142 feat(browser): ...
◐ #146 feat(wehago): ...

⚠️ 파싱 실패 (2 files):
  - 158-featcli-부가세-수집-명령에-interval-옵션-추가.md: missing 'number' field
  - 159-refactor-분개장-captureNetworkResponse...md: frontmatter not closed

Total: 2 issues (2 skipped)
```

### 옵션 B: 파일명에서 번호 추출
`number:` 필드가 없으면 파일명에서 추출 시도:
- `158-featcli-...md` → `number: 158`

### 옵션 C: --strict 플래그
- 기본: 경고만 표시
- `--strict`: 파싱 실패 시 에러 반환

## 구현 계획

1. `Store.loadFromDir`에서 파싱 에러 수집
2. `Store.List` 반환 타입에 경고 추가 또는 별도 메서드
3. `runList`에서 경고 출력
4. 선택적: 파일명에서 번호 추출 fallback

## 완료 기준

- [ ] 파싱 실패한 파일에 대한 경고 표시
- [ ] 실패 원인(구체적 에러 메시지) 포함
- [ ] `--quiet` 플래그로 경고 숨기기 옵션
- [ ] 단위 테스트 작성
