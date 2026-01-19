---
number: 38
title: 'docs: zap init 템플릿에 done/closed 상태 구분 가이드 추가'
state: done
labels:
    - docs
assignees: []
created_at: "2026-01-19T05:24:49Z"
updated_at: "2026-01-19T05:25:41Z"
closed_at: "2026-01-19T05:25:41Z"
---

## 문제
사용자가 done과 closed의 차이를 인지하지 못하고, 완료된 작업에 closed를 사용하는 경우 발생.

## 해결
zap init에서 생성하는 지침 파일에 명확한 상태 구분 가이드 추가.

```markdown
### 상태 선택 가이드

| 상태 | 의미 | 사용 시점 |
|------|------|----------|
| done | ✅ 작업 완료 | 요청한 기능/수정을 성공적으로 구현했을 때 |
| closed | ❌ 진행 안 함 | 취소, 중복, 불필요, 범위 외로 더 이상 진행하지 않을 때 |

**핵심 구분**: 
- 코드를 작성/수정했다 → done
- 작업 없이 닫는다 → closed
```
