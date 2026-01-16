---
number: 13
title: "feat: update 명령에 --script 옵션 추가"
state: done
labels:
  - feature
  - cli
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 설명

`zap update` 명령에 `--script` 옵션을 추가하여 OS별 install 스크립트를 통한 업데이트 기능을 제공합니다.

## 사용 사례

```bash
# 스크립트를 통한 최신 버전 업데이트
zap update --script

# 특정 버전으로 스크립트 업데이트
zap update --script -v 0.2.0
```

## 구현 계획

### 1. 수정 파일

`internal/cli/update.go`

### 2. Import 추가

```go
import (
    "os/exec"
    "runtime"
    // ... 기존 imports
)
```

### 3. 상수 및 플래그 추가

```go
const (
    installScriptUnix    = "https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.sh"
    installScriptWindows = "https://raw.githubusercontent.com/itda-work/zap/main/scripts/install.ps1"
)

var updateScript bool
```

### 4. init()에 플래그 등록

```go
updateCmd.Flags().BoolVar(&updateScript, "script", false, "Update using OS-specific install script (curl/PowerShell)")
```

### 5. runUpdate() 함수 수정

`--script` 플래그 확인 후 분기 처리:

```go
if updateScript {
    return runScriptUpdate()
}
```

### 6. runScriptUpdate() 함수 추가

```go
func runScriptUpdate() error {
    var cmd *exec.Cmd

    switch runtime.GOOS {
    case "windows":
        script := fmt.Sprintf("iex ((New-Object System.Net.WebClient).DownloadString('%s'))", installScriptWindows)
        if updateVersion != "" {
            script = fmt.Sprintf("$env:ZAP_VERSION='%s'; ", updateVersion) + script
        }
        cmd = exec.Command("powershell", "-Command", script)
    default:
        script := fmt.Sprintf("curl -fsSL %s | bash", installScriptUnix)
        if updateVersion != "" {
            script = fmt.Sprintf("ZAP_VERSION=%s %s", updateVersion, script)
        }
        cmd = exec.Command("bash", "-c", script)
    }

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin

    fmt.Printf("Running install script for %s...\n\n", runtime.GOOS)
    return cmd.Run()
}
```

## 검증 방법

```bash
# 1. 빌드 확인
go build ./...

# 2. 새 플래그 표시 확인
go run ./cmd/zap update --help

# 3. 스크립트 업데이트 테스트
go run ./cmd/zap update --script
```

## 완료 기준

- [x] `--script` 플래그 추가
- [x] Unix 스크립트 업데이트 동작
- [x] Windows PowerShell 업데이트 동작
- [x] `--version` 플래그와 조합 동작
- [x] help 메시지에 옵션 설명 표시
- [x] 업데이트 실패 시 `--script` 옵션 안내 메시지 추가
