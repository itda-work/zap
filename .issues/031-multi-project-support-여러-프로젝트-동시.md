---
number: 31
title: 'Multi-project support: 여러 프로젝트 동시 관리 기능'
state: done
labels:
    - enhancement
    - cli
    - web
assignees: []
created_at: 2026-01-18T19:27:25.450044+09:00
updated_at: 2026-01-19T08:10:09.393993+09:00
closed_at: 2026-01-19T08:10:09.393993+09:00
---

## Summary
여러 `-C` 플래그로 독립적인 프로젝트들을 동시에 관리할 수 있도록 zap CLI를 확장합니다.

## Requirements
- **프로젝트 식별자**: 자동(디렉토리명) + 수동(`alias:path`) 둘 다 지원
- **CLI 출력 형식**: 접두어 (`zap/#1`, `alfred/#5`)
- **Web UI**: 통합 목록 + 프로젝트 뱃지
- **번호 충돌**: 허용 (각 프로젝트 독립 번호 체계)
- **이슈 지정 문법**: `alfred/#5` 형식과 `--project alfred 5` 둘 다 지원

## Usage Examples
```bash
# 두 프로젝트 목록 보기
zap -C . -C ~/alfred list

# 별칭 지정
zap -C main:. -C sub:~/alfred list

# 특정 이슈 보기
zap -C . -C ~/alfred show alfred/#5
zap -C . -C ~/alfred show 5 --project alfred

# 상태 변경
zap -C . -C ~/alfred done zap/#1

# Web UI
zap -C . -C ~/alfred serve
```

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] `internal/project/project.go` - Project, ProjectRef, ProjectSpec 타입
- [ ] `internal/project/multistore.go` - MultiStore 구현
- [ ] `internal/project/projectissue.go` - ProjectIssue 타입

### Phase 2: CLI Commands
- [ ] `internal/cli/root.go` - getProjectSpecs(), getMultiStore()
- [ ] `internal/cli/list.go` - 다중 프로젝트 출력
- [ ] `internal/cli/show.go` - 이슈 참조 파싱
- [ ] `internal/cli/move.go` - 이슈 참조 파싱
- [ ] `internal/cli/new.go` - --project 플래그

### Phase 3: Web UI
- [ ] `internal/web/server.go` - MultiStore 지원
- [ ] `internal/web/handlers.go` - 프로젝트 컨텍스트
- [ ] `internal/web/templates/dashboard.html` - 프로젝트 뱃지
