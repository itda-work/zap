---
number: 51
title: 'PDCA 방법론 접목: 상태 모델 확장 및 섹션 기반 이슈 관리'
state: done
labels:
    - feature
    - pdca
assignees: []
created_at: "2026-01-27T08:07:29Z"
updated_at: "2026-01-27T09:09:41Z"
closed_at: "2026-01-27T09:09:41Z"
---

## 배경

현재 zap의 이슈 라이프사이클(open → wip → done/closed)에는 검증(Check)과 리뷰(Review) 단계가 부재하여, "done"의 의미가 "코딩 끝"에 불과하고 "검증된 완료"가 아님.

PDCA(Plan-Do-Check-Act) 방법론을 접목하면 품질 하한선을 높일 수 있음.

## 문제

1. 계획 없이 바로 구현 시작 → 성공 기준 불명확
2. 자기 검증 단계 없음 → "된 것 같으면" done 처리
3. 외부 검토 프로세스 미내장 → 리뷰가 선택적
4. 시행착오 기록 없음 → 반복 사이클 추적 불가

## 제안 상태 모델

```
open → wip → check → review → done
                        ↓
                     closed (어느 단계에서든 취소 가능)
```

| 상태 | PDCA 매핑 | 의미 |
|------|----------|------|
| open | Plan | 계획 수립. 목표와 성공 기준 기술 |
| wip | Do | 구현 진행 중 |
| check | Check | 자기 검증. Plan 기준 대비 결과 확인 |
| review | Check+ | 외부 검토 또는 self-review (코드 리뷰, 전체 점검) |
| done | Act 완료 | 검증 통과, 개선 완료 |
| closed | - | 취소/보류 |

## 반복(Iterate) 방식

- check/review에서 갭 발견 시: check → wip 또는 review → wip로 되돌림
- 이슈 본문에 Cycle N으로 반복 기록

## 이슈 본문 PDCA 섹션 템플릿

```markdown
## Cycle 1
### Plan
- 목표:
- 성공 기준:
- 접근 방식:

### Do
(구현 내용 기록)

### Check
- 설계 대비 일치도:
- 발견된 갭:
- 테스트 결과:

### Review
- 리뷰어:
- 피드백:

### Act
- 개선 조치:
- 다음 사이클 필요 여부:
```

## 구현 범위

### 핵심 기능

1. **State 추가**: check, review 상태를 issue.State에 추가
2. **CLI 반영**: zap set check/review 명령 지원
3. **리스트 표시**: active 상태 = open+wip+check+review (list 기본 표시), done+closed는 --all
4. **색상**: check=cyan, review=magenta 등 구분 색상
5. **템플릿**: zap new 시 PDCA 섹션 자동 삽입 (기본 동작)
6. **상태 전환 안내**: 전환 시 1줄 힌트 출력 (예: `Tip: Check 섹션에 검증 결과를 기록하세요`)

### 문서화

7. **docs/PDCA.md**: PDCA 방법론 설명 문서 생성
   - PDCA 개념 및 4단계 순환 설명
   - zap에서의 PDCA 적용 방법 (상태 매핑, 섹션 템플릿, 반복 사이클)
   - 워크플로우 예시
8. **docs/PDCA.html**: PDCA.md 기반 HTML 문서 생성
   - 시각적으로 보기 좋은 웹 페이지 형태
   - PDCA 사이클 다이어그램 포함

### 스코프 외 (별도 이슈)

- 웹UI(`internal/web/`)의 PDCA 상태 반영은 별도 이슈로 분리 → 웹UI 제거 후 불필요

### zap init 프롬프트 반영

9. **zap init 프롬프트에 PDCA 지침 추가** (`internal/cli/init.go`의 `generateInstructions()`)
   - 기존 워크플로우 섹션을 PDCA 6상태 워크플로우로 교체
   - 간결한 PDCA 가이드 추가 (번잡하지 않게):
     - 상태별 의미 테이블 (open=Plan, wip=Do, check=Check, review=Review, done=Act완료)
     - PDCA 섹션 템플릿 요약
     - 갭 발견 시 반복(check/review → wip) 안내
   - 기존 `zap set` 예시에 check/review 추가
10. **CLAUDE.md 업데이트**: 프로젝트 CLAUDE.md에도 PDCA 워크플로우 반영

## 핵심 장점

- done의 품질 보증: "코딩 끝"이 아닌 "검증된 완료"
- 계획-결과 추적: 이슈 하나에 전체 사이클 기록
- 반복 시각화: Cycle 1, 2, 3으로 시행착오 문서화
- 리뷰 명시화: 자기 검증(check)과 외부 검증(review) 분리

---

## 구현 결과

### 수정 파일 (15개)

| # | 파일 | 변경 내용 |
|---|------|----------|
| 1 | `internal/issue/issue.go` | StateCheck, StateReview 상수, AllStates(6개), ActiveStates(4개), ParseState, IsActive 확장 |
| 2 | `internal/issue/issue_test.go` | 6개 상태 기대값으로 테스트 업데이트 |
| 3 | `internal/cli/color.go` | magenta 계열 색상 상수 및 테마 변수 추가 |
| 4 | `internal/cli/list.go` | stateStyle 맵에 check/review 추가, Long 설명 및 --state 도움말 갱신 |
| 5 | `internal/cli/watch.go` | printWatchStats에 Check/Review 카운트, stateStyle 맵 추가 |
| 6 | `internal/cli/show.go` | stateColor()에 check→cyan, review→magenta 추가 |
| 7 | `internal/cli/stats.go` | 6개 상태 순서, ◑(check)/◕(review) 이모지 추가 |
| 8 | `internal/cli/utils.go` | statePriority 6단계 확장 (done→closed→review→check→wip→open) |
| 9 | `internal/cli/report.go` | stateOrder, stateNames에 check/review 추가 |
| 10 | `internal/cli/move.go` | autocomplete, 에러 메시지, printTransitionTip() 추가 |
| 11 | `internal/cli/new.go` | PDCA 템플릿 기본 삽입 (appendPDCATemplate) |
| 12 | `internal/cli/fix_state.go` | knownStateMappings에 checking/verify/reviewing 등 추가 |
| 13 | `internal/cli/init.go` | generateInstructions() PDCA 워크플로우로 교체 |
| 14 | `internal/ai/prompt.go` | repair-frontmatter 프롬프트에 check/review 상태 추가 |
| 15 | `CLAUDE.md` | 6단계 워크플로우, PDCA 상태 테이블, 갭 반복 안내 |

### 신규 파일 (1개)

| # | 파일 | 내용 |
|---|------|------|
| 1 | `docs/PDCA.md` | PDCA 개념, 상태 매핑, 템플릿, 반복 워크플로우, 실전 예시 |

### 코드 리뷰 반영

- **버그 수정**: `internal/ai/prompt.go`의 repair-frontmatter 프롬프트가 기존 4상태만 명시하여, `zap repair --auto` 실행 시 AI가 check/review 상태를 잘못 "수정"할 수 있는 데이터 오염 경로 → 6상태로 수정
- **설계 변경**: `--pdca` 플래그를 제거하고 PDCA 템플릿을 기본 동작으로 변경 (모든 새 이슈에 자동 삽입)
