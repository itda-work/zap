---
number: 54
title: 'feat: zap move 명령 - 다른 프로젝트로 이슈 이동 지원'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: "2026-01-30T04:37:03Z"
updated_at: "2026-01-30T04:39:38Z"
closed_at: "2026-01-30T04:39:38Z"
---

## 개요

다른 프로젝트로 이슈를 옮겨야 할 때, 단순 파일 복사는 이슈 번호가 틀어진다.
`zap move` 명령으로 대상 프로젝트의 새 이슈 번호를 자동 할당하여 이동한다.

## 사용법

```bash
zap move 5 --to ~/other-project          # 이슈 #5를 다른 프로젝트로 이동 (원본 유지)
zap move 5 --to ~/other-project --delete  # 이동 후 원본 삭제
```

## 설계

### 프로세스
1. 소스 프로젝트에서 이슈 #N 읽기 (파싱)
2. 대상 프로젝트의 .issues/에서 findNextIssueNumber() → 새 번호 M 할당
3. 새 이슈 파일 생성: M-slug.md (number를 M으로 변경, 나머지 보존)
4. 대상 본문 상단에 출처 메모 추가: ← Moved from <source-path> #N
5. 원본 처리: 기본 유지, --delete 시 삭제

### 상세 규칙
- created_at: 원본 시간 유지
- updated_at: 이동 시점으로 갱신
- state: 원본 상태 그대로 유지
- labels, assignees: 원본 그대로 유지
- body: 원본 본문 앞에 출처 메모 한 줄 추가
- 대상 .issues/ 디렉토리 없으면 자동 생성

### 재활용 가능한 기존 코드
- findNextIssueNumber (new.go)
- issue.Parse / issue.Serialize (parser.go)
- generateSlug (new.go)
- getIssuesDirWithDiscovery (root.go)

### 플래그
- --to (필수): 대상 프로젝트 경로
- --delete: 이동 후 원본 파일 삭제
- -d/--dir: 이슈 디렉토리 (기본 .issues)
