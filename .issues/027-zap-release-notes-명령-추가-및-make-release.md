---
number: 27
title: zap release-notes 명령 추가 및 make release 연동
state: done
labels:
    - feat
    - ai
assignees: []
created_at: 2026-01-18T11:27:41.291804+09:00
updated_at: 2026-01-18T11:30:49.255119+09:00
closed_at: 2026-01-18T11:30:49.255119+09:00
---

## 개요

`make release` 실행 시 AI를 활용하여 커밋 로그 기반의 정돈된 릴리즈 노트를 생성하는 기능 추가.

## 구현 내용

### 1. `zap release-notes` 명령 추가

**기본 동작:**
- 이전 태그부터 현재까지의 커밋 로그 수집
- 변경된 파일 통계 포함
- 관련 .issues/ 이슈 참조 포함
- 기존 `internal/ai` 패키지로 AI CLI 자동 감지 (claude > codex > gemini)
- AI가 커밋들을 분석하여 정돈된 릴리즈 노트 생성

**사용법:**
\`\`\`bash
zap release-notes              # 최신 태그 ~ HEAD
zap release-notes v0.6.6       # v0.6.6 ~ HEAD  
zap release-notes v0.6.5 v0.6.6  # v0.6.5 ~ v0.6.6
zap release-notes --output RELEASE.md  # 파일로 저장
\`\`\`

**출력 옵션:**
- 기본: stdout으로 Markdown 출력
- `--output FILE`: 지정 파일에 저장

**참고:**
- 개발용 명령으로 `Hidden: true` 설정되어 `zap --help`에서 숨김 처리
- 직접 호출(`zap release-notes`)은 가능

### 2. Makefile 연동

`make release` 실행 시:
1. `zap release-notes`로 릴리즈 노트 생성
2. `gh release create`에 생성된 노트 전달

## 수집할 정보

- 커밋 로그 메시지
- 변경 파일 통계 (추가/수정/삭제 파일 수)
- 관련 이슈 참조 (.issues/ 디렉토리)

## 기술 스택

- 기존 `internal/ai` 패키지 활용 (AutoDetect 함수)
- git 명령어로 커밋 로그 및 통계 수집
