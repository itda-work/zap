---
number: 12
title: "feat: 재사용 가능한 AI 모듈 설계"
state: open
labels:
  - feature
  - ai
  - architecture
assignees: []
created_at: 2026-01-16
updated_at: 2026-01-16
---

## 설명

다양한 AI CLI 도구를 추상화하여 zap 전반에서 재사용 가능한 AI 모듈을 설계합니다.

## 활용 사례

| 기능 | 설명 | 관련 이슈 |
|------|------|----------|
| Frontmatter 복구 | 파싱 실패한 이슈 파일 수정 | #11 |
| 이슈 생성 보조 | 제목/본문 자동 생성 | - |
| 이슈 요약 | 긴 이슈 내용 요약 | - |
| 검색 쿼리 확장 | 자연어 → 검색 키워드 | - |
| 번역 | 이슈 다국어 지원 | - |
| 중복 감지 | 유사 이슈 탐지 | - |

## 아키텍처

```
internal/ai/
├── ai.go           # Client 인터페이스 및 팩토리
├── config.go       # 설정 로드/저장
├── prompt.go       # 프롬프트 템플릿 관리
├── providers/
│   ├── claude.go   # Claude CLI
│   ├── codex.go    # OpenAI Codex CLI
│   ├── gemini.go   # Google Gemini CLI
│   └── ollama.go   # Ollama (로컬 LLM)
└── tasks/
    ├── repair.go   # frontmatter 복구
    ├── generate.go # 이슈 생성
    └── summarize.go # 요약
```

## 핵심 인터페이스

### Client 인터페이스

```go
package ai

// Client는 AI CLI 도구의 추상화 인터페이스
type Client interface {
    // 기본 정보
    Name() string
    IsAvailable() bool

    // 핵심 메서드
    Complete(ctx context.Context, req *Request) (*Response, error)
    Stream(ctx context.Context, req *Request) (<-chan StreamChunk, error)
}

type Request struct {
    Prompt      string
    System      string            // 시스템 프롬프트
    MaxTokens   int
    Temperature float64
    Variables   map[string]string // 템플릿 변수
}

type Response struct {
    Content   string
    TokensIn  int
    TokensOut int
    Model     string
    Duration  time.Duration
}
```

### 팩토리 함수

```go
// 설정 기반 클라이언트 생성
func NewClient(cfg *Config) (Client, error)

// 자동 감지 (우선순위: claude > codex > gemini > ollama)
func AutoDetect() (Client, error)

// 특정 프로바이더
func NewClaudeClient(cfg *ClaudeConfig) *ClaudeClient
func NewCodexClient(cfg *CodexConfig) *CodexClient
func NewGeminiClient(cfg *GeminiConfig) *GeminiClient
func NewOllamaClient(cfg *OllamaConfig) *OllamaClient
```

### 프롬프트 템플릿

```go
package ai

type PromptTemplate struct {
    Name        string
    Description string
    System      string
    User        string
    Variables   []string  // 필수 변수 목록
}

// 내장 템플릿
var Templates = map[string]*PromptTemplate{
    "repair-frontmatter": {...},
    "generate-issue":     {...},
    "summarize":          {...},
    "translate":          {...},
}

// 템플릿 렌더링
func (t *PromptTemplate) Render(vars map[string]string) (*Request, error)
```

## 프로바이더 구현

### Claude CLI

```go
type ClaudeClient struct {
    model   string
    binPath string
}

func (c *ClaudeClient) Complete(ctx context.Context, req *Request) (*Response, error) {
    args := []string{"-p", req.Prompt}
    if req.System != "" {
        args = append(args, "--system", req.System)
    }
    if c.model != "" {
        args = append(args, "--model", c.model)
    }

    cmd := exec.CommandContext(ctx, c.binPath, args...)
    output, err := cmd.Output()
    // ...
}
```

### Ollama (로컬)

```go
type OllamaClient struct {
    host  string
    model string
}

func (o *OllamaClient) Complete(ctx context.Context, req *Request) (*Response, error) {
    // HTTP API 호출: POST /api/generate
    body := map[string]interface{}{
        "model":  o.model,
        "prompt": req.Prompt,
        "system": req.System,
        "stream": false,
    }
    // ...
}
```

## 설정 파일

`~/.config/zap/ai.yaml`:

```yaml
# 기본 프로바이더 (auto | claude | codex | gemini | ollama)
default: auto

# 프로바이더별 설정
claude:
  model: claude-sonnet-4-20250514
  # bin: /usr/local/bin/claude  # 커스텀 경로

codex:
  model: gpt-4

gemini:
  model: gemini-pro

ollama:
  host: http://localhost:11434
  model: llama3.2

# 공통 설정
options:
  timeout: 30s
  max_tokens: 4096
  temperature: 0.3

# 커스텀 프롬프트 템플릿 경로
templates_dir: ~/.config/zap/prompts/
```

## Task 추상화

특정 작업을 위한 고수준 API:

```go
package ai

// Task는 특정 AI 작업의 추상화
type Task interface {
    Name() string
    Template() *PromptTemplate
    Execute(ctx context.Context, client Client, input interface{}) (interface{}, error)
}

// 내장 Task들
type RepairTask struct{}      // frontmatter 복구
type GenerateTask struct{}    // 이슈 생성
type SummarizeTask struct{}   // 요약
type TranslateTask struct{}   // 번역

// Task 실행 헬퍼
func RunTask(ctx context.Context, taskName string, input interface{}) (interface{}, error) {
    client, err := AutoDetect()
    if err != nil {
        return nil, err
    }
    task := GetTask(taskName)
    return task.Execute(ctx, client, input)
}
```

## CLI 통합

```go
// 모든 AI 관련 명령의 공통 플래그
func addAIFlags(cmd *cobra.Command) {
    cmd.Flags().String("ai", "", "AI provider (claude, codex, gemini, ollama)")
    cmd.Flags().String("model", "", "Model name")
    cmd.Flags().Bool("dry-run", false, "Preview without executing")
}

// AI 클라이언트 가져오기
func getAIClient(cmd *cobra.Command) (ai.Client, error) {
    provider, _ := cmd.Flags().GetString("ai")
    if provider != "" {
        return ai.NewClient(&ai.Config{Provider: provider})
    }
    return ai.AutoDetect()
}
```

## 에러 처리

```go
var (
    ErrNoProvider     = errors.New("no AI provider available")
    ErrProviderFailed = errors.New("AI provider execution failed")
    ErrTimeout        = errors.New("AI request timed out")
    ErrRateLimit      = errors.New("rate limit exceeded")
)

// 재시도 래퍼
func WithRetry(client Client, maxRetries int) Client {
    return &retryClient{client: client, maxRetries: maxRetries}
}
```

## 테스트

```go
// Mock 클라이언트
type MockClient struct {
    responses map[string]*Response
}

func (m *MockClient) Complete(ctx context.Context, req *Request) (*Response, error) {
    if resp, ok := m.responses[req.Prompt]; ok {
        return resp, nil
    }
    return &Response{Content: "mock response"}, nil
}
```

## 완료 기준

- [ ] `ai.Client` 인터페이스 정의
- [ ] Claude CLI 프로바이더 구현
- [ ] Codex CLI 프로바이더 구현
- [ ] Gemini CLI 프로바이더 구현
- [ ] Ollama 프로바이더 구현
- [ ] 자동 감지 로직
- [ ] 프롬프트 템플릿 시스템
- [ ] 설정 파일 지원
- [ ] Task 추상화
- [ ] 에러 처리 및 재시도
- [ ] Mock 클라이언트 (테스트용)
- [ ] 단위 테스트

## 의존 이슈

이 모듈을 사용하는 이슈:
- #11 feat: AI CLI를 활용한 frontmatter 자동 복구
