---
number: 47
title: zap init 명령 H2 레벨로 변경 및 프로젝트명 H1 자동 추가
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-23T16:32:50Z"
updated_at: "2026-01-23T16:36:11Z"
closed_at: "2026-01-23T16:36:11Z"
---

## 개요

zap init 명령에서 생성되는 markdown 내용의 제목 레벨 변경

## 변경 내용

1. `# zap - Local Issue Management` → `## zap - Local Issue Management` (H2로 변경)
2. 새 파일 생성 시 프로젝트 폴더명을 H1 제목으로 최상단에 추가
   - 예: `my-project` → `# My Project`
   - 하이픈을 공백으로, 각 단어 첫 글자 대문자화

## 예시 출력

### 새 파일 생성 시
```markdown
# My Project

## zap - Local Issue Management

이 프로젝트는 로컬 이슈 관리 시스템...
```

### 기존 파일에 append 시
```markdown
(기존 내용)

---

## zap - Local Issue Management

이 프로젝트는 로컬 이슈 관리 시스템...
```

## 이유

매번 수동으로 # 1개를 ##로 변경하고, 프로젝트 제목을 추가하는 것이 번거로움
