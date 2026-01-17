---
number: 21
title: 'feat: 이슈 간 참조 관계 추적 및 그래프 시각화'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-17T12:00:00Z
updated_at: 2026-01-17T11:28:11.316579+09:00
closed_at: 2026-01-17T11:28:11.316579+09:00
---

## 개요

이슈 본문에서 `#123` 형태로 다른 이슈를 언급했을 때, 이 참조 관계를 추적하고 시각화하는 기능.

## 핵심 요구사항

| 항목 | 설명 |
|------|------|
| 언급 인식 | 본문에서 `#숫자` 패턴 파싱 (예: #1, #23, #456) |
| 관계 방향 | 양방향 (언급한 이슈 + 언급된 이슈) |
| 존재 검증 | 실제 존재하지 않는 이슈 번호는 무시 |
| 탐색 범위 | 전체 그래프 (연결된 모든 이슈 끝까지 탐색) |
| 시각화 | 텍스트 트리 구조 |

## 명령어 인터페이스

### `zap show <number> --refs`

해당 이슈의 연결 그래프를 텍스트 트리로 표시 (연결 거리순 정렬)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Issue #5: 현재 이슈 제목
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
State:    open
...

Referenced Issues:
├── → #3 연결된 이슈 A [done]
│   └── → #7 2단계 연결 [open]
├── → #12 연결된 이슈 B [in-progress]
└── ← #8 이 이슈를 참조함 [open]
    └── ← #15 2단계 역참조 [closed]

(→: mentions, ←: mentioned by)
```

### `zap list --refs`

목록에서 각 이슈 옆에 참조 수 표시

```
[open]   #1   이슈 제목 A (refs: 3)
[wip]    #2   이슈 제목 B (refs: 1)
[done]   #3   이슈 제목 C
[open]   #4   이슈 제목 D (refs: 5)

Total: 4 issues
```

- `refs: N` = 해당 이슈가 언급한 수 + 해당 이슈를 언급한 수
- 참조가 없는 이슈는 `(refs: N)` 생략

## 기술 설계

### 참조 파싱

```go
// internal/issue/refs.go

// ExtractRefs extracts issue references (#N) from text
func ExtractRefs(text string) []int {
    // 정규식: #(\d+)
    // 중복 제거 후 반환
}
```

### 그래프 구조

```go
type RefGraph struct {
    // 이슈 번호 -> 해당 이슈가 언급한 이슈 번호들
    Mentions map[int][]int
    // 이슈 번호 -> 해당 이슈를 언급한 이슈 번호들
    MentionedBy map[int][]int
}

// BuildRefGraph builds reference graph for all issues
func (s *Store) BuildRefGraph() (*RefGraph, error)

// GetConnectedIssues returns all issues connected to the given issue
// using BFS traversal (handles cycles)
func (g *RefGraph) GetConnectedIssues(issueNum int) []ConnectedIssue

type ConnectedIssue struct {
    Number   int
    Distance int        // 현재 이슈로부터의 거리
    Direction string    // "mentions" or "mentioned_by"
    Path     []int      // 연결 경로
}
```

### 순환 참조 처리

- BFS 탐색 시 visited set 유지
- 이미 방문한 이슈는 재탐색하지 않음

## 구현 파일

| 파일 | 작업 |
|------|------|
| `internal/issue/refs.go` | 신규 - 참조 파싱 및 그래프 로직 |
| `internal/issue/refs_test.go` | 신규 - 참조 관련 테스트 |
| `internal/cli/show.go` | 수정 - `--refs` 플래그 추가 |
| `internal/cli/list.go` | 수정 - `--refs` 플래그 추가 |

## 테스트 케이스

### TC-1: 기본 참조 파싱
```
Given: 본문에 "See #1 and #2 for details"
When: ExtractRefs() 호출
Then: [1, 2] 반환
```

### TC-2: 존재하지 않는 이슈 무시
```
Given: #1, #2 이슈 존재, 본문에 "#1 #2 #999" 언급
When: 그래프 빌드
Then: #1, #2만 그래프에 포함, #999 무시
```

### TC-3: 양방향 관계
```
Given: #1이 #2를 언급, #3이 #1을 언급
When: show 1 --refs
Then: → #2 (mentions), ← #3 (mentioned by) 모두 표시
```

### TC-4: 순환 참조 처리
```
Given: #1 → #2 → #3 → #1 (순환)
When: show 1 --refs
Then: 무한 루프 없이 모든 연결 표시
```

### TC-5: 다단계 연결
```
Given: #1 → #2 → #3 → #4
When: show 1 --refs
Then: 모든 연결이 거리순으로 표시
  → #2 (distance 1)
    → #3 (distance 2)
      → #4 (distance 3)
```

### TC-6: list --refs 카운트
```
Given: #1이 #2, #3 언급 / #4가 #1 언급
When: list --refs
Then: #1 (refs: 3), #2 (refs: 1), #3 (refs: 1), #4 (refs: 1)
```

## 작업 목록

- [x] `internal/issue/refs.go` 생성
  - [x] `ExtractRefs()` 함수
  - [x] `RefGraph` 구조체
  - [x] `BuildRefGraph()` 메서드
  - [x] `GetConnectedIssues()` 메서드 (BFS)
  - [x] `BuildTree()` 메서드 (트리 구조 생성)
- [x] `internal/issue/refs_test.go` 테스트 작성
- [x] `internal/cli/show.go` 수정
  - [x] `--refs` 플래그 추가
  - [x] 트리 렌더링 함수
- [x] `internal/cli/list.go` 수정
  - [x] `--refs` 플래그 추가
  - [x] 참조 카운트 표시
- [x] 빌드 및 테스트

## 진행 내역

### 2026-01-17

**구현 완료:**

- `internal/issue/refs.go` 생성
  - `ExtractRefs()`: 정규식으로 `#숫자` 패턴 추출, 중복 제거, 정렬
  - `RefGraph`: Mentions/MentionedBy 양방향 맵 + Issues 인덱스
  - `BuildRefGraph()`: 모든 이슈에서 참조 추출, 존재하지 않는 이슈 무시
  - `GetConnectedIssues()`: BFS 탐색, 순환 참조 처리, 거리순 정렬
  - `BuildTree()`: 부모-자식 관계로 트리 구조 생성

- `internal/issue/refs_test.go` 테스트 작성
  - ExtractRefs 테스트 (기본, 중복, 멀티라인 등)
  - GetConnectedIssues 테스트 (기본, 순환, 고립 노드)
  - GetRefCount 테스트
  - BuildTree 테스트

- `internal/cli/show.go` 수정
  - `--refs` 플래그 추가
  - `printRefsGraph()`: 참조 그래프 출력 (이슈 상세와 동일한 구분선 스타일)
  - `printRefTree()`: 재귀적 트리 렌더링 (├── └── │ 문자 사용)
  - 상태별 색상 적용 (done=초록, wip=노란, closed=회색)

- `internal/cli/list.go` 수정
  - `--refs` 플래그 추가
  - `(refs: N)` 형식으로 참조 수 표시 (회색)

## 고려사항

### 성능
- 전체 그래프 빌드는 모든 이슈 파일을 읽어야 함
- 이슈 수가 많을 경우 캐싱 고려 (향후)

### 확장 가능성
- 향후 DOT 포맷 출력 지원 가능 (`--format dot`)
- 향후 깊이 제한 옵션 가능 (`--depth N`)
