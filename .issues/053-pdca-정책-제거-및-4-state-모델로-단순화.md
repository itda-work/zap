---
number: 53
title: PDCA 정책 제거 및 4-state 모델로 단순화
state: done
labels:
    - refactor
assignees: []
created_at: "2026-01-29T10:05:59Z"
updated_at: "2026-01-29T10:05:59Z"
---

## 변경 내역

- check/review 상태 제거, open/wip/done/closed 4개 상태만 유지
- PDCA 템플릿 자동 삽입 제거 (zap new)
- docs/PDCA.md 삭제
- fix-state에서 check/review → wip 하위 호환 매핑 유지
- build 출력을 ./bin/ 경로로 변경
- .gitignore에 bin/, .sisyphus/ 추가

## 관련 커밋

- 5e5eccf refactor: PDCA 정책 제거 및 4-state 모델로 단순화
