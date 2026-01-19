---
number: 10
title: 'feat: -C 옵션으로 프로젝트 디렉토리 지정'
state: done
labels:
    - feature
    - cli
assignees: []
created_at: 2026-01-16T19:44:31Z
updated_at: 2026-01-16T23:29:56Z
---

## 설명

`git -C <path>` 처럼 zap 명령에 `-C` 옵션을 추가하여 프로젝트 홈 경로를 지정할 수 있도록 합니다.

## 사용 사례

현재 디렉토리가 아닌 다른 프로젝트의 이슈를 관리할 때:

```bash
# 현재 방식: 디렉토리 이동 필요
cd ~/Apps/taxhero-kr/alfred
zap list
cd -

# 제안 방식: -C 옵션 사용
zap -C ~/Apps/taxhero-kr/alfred list
zap -C ~/Apps/taxhero-kr/alfred show 142
zap -C ~/Apps/taxhero-kr/alfred start 142
```

## 구현 계획

### 1. root.go에 -C 플래그 추가

```go
func init() {
    rootCmd.PersistentFlags().StringP("dir", "d", ".issues", "Issues directory path")
    rootCmd.PersistentFlags().StringP("project", "C", "", "Run as if zap was started in <path>")
}
```

### 2. 경로 해석 로직

`-C`가 지정되면 `.issues` 경로를 해당 프로젝트 기준으로 해석:

```go
func getIssuesDir(cmd *cobra.Command) string {
    projectDir, _ := cmd.Flags().GetString("project")
    issuesDir, _ := cmd.Flags().GetString("dir")

    if projectDir != "" {
        return filepath.Join(projectDir, issuesDir)
    }
    return issuesDir
}
```

### 3. git 호환성

git의 `-C` 옵션과 동일하게 동작:
- 상대 경로 및 절대 경로 모두 지원
- 여러 번 사용 시 누적: `zap -C foo -C bar` → `foo/bar`
- `~` 확장 지원

## 영향 범위

모든 서브커맨드에 적용:
- `list`, `show`, `search`
- `open`, `start`, `done`, `close`
- `stats`, `init`

## 완료 기준

- [x] `-C` / `--project` 플래그 추가
- [x] 모든 서브커맨드에서 동작 확인
- [x] 상대/절대 경로 지원
- [x] `~` (홈 디렉토리) 확장
- [x] 존재하지 않는 경로 에러 처리
- [x] help 메시지 업데이트
- [x] README 문서 업데이트

## 진행 내역

### 2026-01-16

구현 완료:

- `root.go`: `-C/--project` PersistentFlag 추가 (StringArrayP로 여러 번 사용 가능)
- `root.go`: `expandTilde()`, `getProjectDir()`, `getIssuesDir()` 헬퍼 함수 추가
- 모든 서브커맨드 수정: list, show, move(open/start/done/close), search, stats, repair, init
- `init.go`: AI 에이전트 지침에 "GitHub 이슈가 아닌 로컬 이슈 사용" 안내 추가
- `README.md`: `-C` 옵션 사용법 문서화

## 참고

- `git -C <path>`: https://git-scm.com/docs/git#Documentation/git.txt--Cltpathgt
