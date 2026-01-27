# PDCA 방법론 - zap 이슈 관리

## PDCA란?

PDCA(Plan-Do-Check-Act)는 지속적 개선을 위한 반복적 관리 방법론입니다.
W. Edwards Deming이 대중화하여 "데밍 사이클"이라고도 합니다.

```
Plan → Do → Check → Act
  ↑                   │
  └───────────────────┘
       (반복 개선)
```

## zap 상태 매핑

| PDCA 단계 | zap 상태 | 의미 | CLI 명령 |
|-----------|----------|------|----------|
| **Plan** | `open` | 목표/기준/접근 방식 정의 | `zap new "제목"` |
| **Do** | `wip` | 계획에 따라 구현 | `zap set wip <N>` |
| **Check** | `check` | 계획 대비 결과 검증 | `zap set check <N>` |
| **Review** | `review` | 외부 리뷰, 피드백 수집 | `zap set review <N>` |
| **Act** | `done` | 개선 조치 기록, 완료 | `zap set done <N>` |
| - | `closed` | 취소/보류 (어느 단계에서든) | `zap set closed <N>` |

## 이슈 본문 PDCA 섹션 템플릿

`zap new "제목"` 명령으로 아래 PDCA 템플릿이 자동 삽입됩니다:

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

## 반복 사이클 워크플로우

### 기본 흐름 (한 사이클)

```
open(Plan) → wip(Do) → check(Check) → review(Review) → done(Act)
```

### 갭 발견 시 반복

check 또는 review 단계에서 갭이 발견되면 wip로 되돌릴 수 있습니다:

```
open → wip → check → (갭 발견!) → wip → check → review → done
```

```bash
# 갭 발견 후 재작업
zap set wip 42    # check/review → wip 으로 복귀
# ... 수정 작업 ...
zap set check 42  # 다시 검증
```

### 여러 사이클

큰 이슈의 경우 이슈 본문에 Cycle 2, Cycle 3 섹션을 추가하여 반복 기록을 유지합니다:

```markdown
## Cycle 2

### Plan
- 이전 사이클 피드백 반영:
- 추가 목표:

### Do
(추가 구현 내용)

### Check
- 이전 갭 해결 여부:
- 새로운 발견:

### Review
- 리뷰어:
- 피드백:

### Act
- 최종 결론:
```

## 실전 예시

### 예시 1: 기능 구현

```bash
# 1. Plan: 이슈 생성
zap new "사용자 인증 시스템 구현"
# → 이슈 본문의 Plan 섹션에 목표, 기준, 접근 방식 기록

# 2. Do: 구현 시작
zap set wip 52
# → Do 섹션에 구현 내용 기록

# 3. Check: 자기 검증
zap set check 52
# → Check 섹션에 테스트 결과, 갭 분석 기록

# 4. Review: 코드 리뷰
zap set review 52
# → Review 섹션에 리뷰어, 피드백 기록

# 5. Act: 완료
zap set done 52
# → Act 섹션에 개선 조치 기록
```

### 예시 2: 갭 발견 시

```bash
zap set check 52
# Check 결과: 에지 케이스 누락 발견!

zap set wip 52     # wip로 복귀하여 재작업
# ... 에지 케이스 처리 추가 ...

zap set check 52   # 다시 검증
# Check 결과: 모두 통과!

zap set review 52  # 리뷰 진행
zap set done 52    # 완료
```
