---
number: 57
title: watch AI 요약에 gemini flash 모델 하드코딩 적용
state: done
labels:
    - enhancement
assignees: []
created_at: "2026-01-31T07:01:46Z"
updated_at: "2026-01-31T08:45:09Z"
closed_at: "2026-01-31T08:45:09Z"
---

## 배경

zap watch --ai 의 이슈 변경 요약 시 gemini 기본 모델(Pro)을 사용 중.
한 줄 80자 요약에 Pro는 과잉 스펙이며, Flash 모델이면 충분.

## 테스트

이 파일을 수정해서 watch가 AI 요약을 생성하는지 확인 중입니다. (두 번째 변경)

## 변경 사항

- watch.go의 fetchAISummary()에서 gemini provider일 때 flash 모델 기본 사용
- Request.Model에 gemini-2.0-flash 지정

## 기대 효과

- 응답 속도 향상 (Pro 대비 수배 빠름)
- 실시간 watch UI의 UX 개선
- 비용 절감
