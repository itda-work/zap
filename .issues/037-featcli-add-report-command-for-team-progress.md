---
number: 37
title: 'feat(cli): add report command for team progress sharing'
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-19T05:15:03Z"
updated_at: "2026-01-19T05:15:09Z"
closed_at: "2026-01-19T05:15:09Z"
---

팀 진행 상황 공유를 위한 보고서 생성 명령어 추가

## 기능
- 커밋 목록 및 관련 이슈 연결
- 이슈 진행 상황 (done, wip, open, closed)
- 파일 변경 통계
- AI 요약 생성

## 사용법
```bash
zap report --days 7
zap report --today
zap report v1.0..HEAD
zap report 10 11 12
zap report --days 7 -o report.md
zap report --days 7 --format json
```

## 옵션
- `--since`, `--until`, `--today`, `--days`, `--weeks`: 날짜 필터
- `-f, --format`: 출력 형식 (markdown, text, json)
- `-o, --output`: 파일 출력
- `--ai`: AI 제공자 선택
- `--no-ai`: AI 요약 생략
