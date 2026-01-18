---
number: 20
title: 'feat: 이슈 번호 인자에 셸 자동 완성 지원'
state: done
labels:
    - enhancement
    - cli
assignees: []
created_at: 2026-01-17T10:40:03Z
updated_at: 2026-01-17T10:39:48.183091+09:00
closed_at: 2026-01-17T10:39:48.183092+09:00
---

## 개요

`zap show <number>` 등 이슈 번호를 인자로 받는 명령에서 셸 자동 완성(Tab 키) 지원

## 대상 명령

| 명령 | 별칭 | 설명 |
|------|------|------|
| `show` | `s` | 이슈 상세 보기 |
| `open` | `o` | open 상태로 이동 |
| `start` | `wip` | in-progress 상태로 이동 |
| `done` | `d` | done 상태로 이동 |
| `close` | `c` | closed 상태로 이동 |

## 상태별 필터링 시나리오

### show 명령
- **입력**: `zap show <TAB>`
- **출력**: 모든 이슈 표시
- **이유**: 어떤 상태든 조회 가능

### open 명령
- **입력**: `zap open <TAB>`
- **출력**: open 상태가 **아닌** 이슈만 표시
- **이유**: 이미 open인 이슈를 다시 open할 필요 없음
- **표시 대상**: in-progress, done, closed 상태 이슈

### start 명령
- **입력**: `zap start <TAB>`
- **출력**: in-progress 상태가 **아닌** 이슈만 표시
- **이유**: 이미 진행 중인 이슈를 다시 시작할 필요 없음
- **표시 대상**: open, done, closed 상태 이슈

### done 명령
- **입력**: `zap done <TAB>`
- **출력**: done 상태가 **아닌** 이슈만 표시
- **이유**: 이미 완료된 이슈를 다시 완료 처리할 필요 없음
- **표시 대상**: open, in-progress, closed 상태 이슈

### close 명령
- **입력**: `zap close <TAB>`
- **출력**: closed 상태가 **아닌** 이슈만 표시
- **이유**: 이미 닫힌 이슈를 다시 닫을 필요 없음
- **표시 대상**: open, in-progress, done 상태 이슈

## 자동 완성 출력 형식

```
1    #1: 첫 번째 이슈 제목 [open]
2    #2: 두 번째 이슈 제목 [in-progress]
3    #3: 세 번째 이슈 제목 [done]
```

## 테스트 케이스

### TC-1: show 명령 - 모든 이슈 표시
```
Given: open(1), in-progress(2), done(3), closed(4) 이슈 존재
When: `zap show <TAB>`
Then: 1, 2, 3, 4 모두 표시
```

### TC-2: open 명령 - open 제외
```
Given: open(1), in-progress(2), done(3) 이슈 존재
When: `zap open <TAB>`
Then: 2, 3만 표시 (1 제외)
```

### TC-3: start 명령 - in-progress 제외
```
Given: open(1), in-progress(2), done(3) 이슈 존재
When: `zap start <TAB>`
Then: 1, 3만 표시 (2 제외)
```

### TC-4: done 명령 - done 제외
```
Given: open(1), in-progress(2), done(3) 이슈 존재
When: `zap done <TAB>`
Then: 1, 2만 표시 (3 제외)
```

### TC-5: close 명령 - closed 제외
```
Given: open(1), in-progress(2), closed(4) 이슈 존재
When: `zap close <TAB>`
Then: 1, 2만 표시 (4 제외)
```

### TC-6: 숫자 프리픽스 필터링
```
Given: 이슈 1, 2, 10, 11, 20 존재
When: `zap show 1<TAB>`
Then: 1, 10, 11만 표시
```

### TC-7: 두 번째 인자 없음
```
Given: 이슈 1, 2, 3 존재
When: `zap show 1 <TAB>`
Then: 추가 자동 완성 없음 (파일 완성도 없음)
```

## 구현 파일

| 파일 | 작업 |
|------|------|
| `internal/cli/completion.go` | 신규 - 공통 자동 완성 함수 |
| `internal/cli/show.go` | 수정 - ValidArgsFunction 추가 |
| `internal/cli/move.go` | 수정 - ValidArgsFunction 추가 |

## 작업 목록

- [x] completion.go 생성
- [x] show.go에 ValidArgsFunction 추가
- [x] move.go에 상태별 ValidArgsFunction 추가
- [x] 빌드 및 수동 테스트
- [ ] 단위 테스트 작성 (선택)

## 검증 방법

```bash
# 빌드
make build

# zsh 자동 완성 테스트
./zap completion zsh > /tmp/_zap
source /tmp/_zap
zap show <TAB>      # 모든 이슈 표시 확인
zap done <TAB>      # done 상태 제외 확인
```

## 진행 내역

### 2026-01-17

- `internal/cli/completion.go` 생성
  - `completeIssueNumber()`: 모든 이슈 자동완성
  - `completeIssueNumberExcluding()`: 특정 상태 제외 자동완성
- `internal/cli/show.go`: `ValidArgsFunction` 추가
- `internal/cli/move.go`: 4개 명령(open/start/done/close)에 상태별 `ValidArgsFunction` 추가
- `README.md`: 셸 자동완성 가이드 추가 (Bash/Zsh/Fish/PowerShell + Windows 주의사항)
