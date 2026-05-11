# AlphaKey Terraform Provider

알파키(AlphaKey) IDaaS 플랫폼의 리소스를 [Terraform](https://www.terraform.io)으로 선언적으로 관리할 수 있는 Provider입니다.

> 리소스 10개 + 데이터소스 8개 지원

---

## 요구사항

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0

## 설치

### 1. 바이너리 다운로드

[Releases](https://github.com/lgu-idaas/alphakey-terraform-provider/releases) 페이지에서 본인 OS에 맞는 zip을 다운로드합니다.

| OS | 파일 |
|----|------|
| Windows | `terraform-provider-alphakey_0.1.0_windows_amd64.zip` |
| macOS (Apple Silicon) | `terraform-provider-alphakey_0.1.0_darwin_arm64.zip` |
| Linux | `terraform-provider-alphakey_0.1.0_linux_amd64.zip` |

### 2. 바이너리 배치

다운로드한 zip을 원하는 경로에 압축 해제합니다.

```
# 예시 (Windows)
C:\terraform\providers\registry.terraform.io\lgu-idaas\alphakey\0.1.0\windows_amd64\terraform-provider-alphakey.exe

# 예시 (macOS/Linux)
~/terraform/providers/registry.terraform.io/lgu-idaas/alphakey/0.1.0/darwin_arm64/terraform-provider-alphakey
```

### 3. Terraform 설정

Terraform이 로컬 Provider를 인식하도록 설정 파일을 생성합니다.

**Windows:** `%APPDATA%\terraform.rc`
**macOS/Linux:** `~/.terraformrc`

```hcl
provider_installation {
  filesystem_mirror {
    path    = "C:/terraform/providers"  # 바이너리를 배치한 상위 경로
    include = ["registry.terraform.io/lgu-idaas/*"]
  }
  direct {
    exclude = ["registry.terraform.io/lgu-idaas/*"]
  }
}
```

### 4. 초기화

```bash
terraform init
```

### 5. 설치 확인

```bash
terraform plan
```

오류 없이 실행되면 설치 성공입니다.

---

## 사용법

### Provider 설정

```hcl
terraform {
  required_providers {
    alphakey = {
      source  = "registry.terraform.io/lgu-idaas/alphakey"
      version = "0.1.0"
    }
  }
}

provider "alphakey" {
  base_url  = "https://your-company.alphakey.kr"  # 또는 환경변수 ALPHAKEY_BASE_URL
  api_token = var.alphakey_token                   # 또는 환경변수 ALPHAKEY_API_TOKEN
}
```

### Provider 속성

| 속성 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `base_url` | String | Yes | 알파키 테넌트 URL. 환경변수 `ALPHAKEY_BASE_URL`로 대체 가능 |
| `api_token` | String | Yes | API 인증 토큰. 환경변수 `ALPHAKEY_API_TOKEN` 또는 `ALPHAKEY_TOKEN`으로 대체 가능 |
| `request_timeout` | Number | No | HTTP 요청 타임아웃(초). 기본값: 30 |
| `insecure` | Bool | No | TLS 인증서 검증 건너뛰기. 기본값: false |

### 토큰 발급

알파키 관리자 콘솔 > 설정 > OpenAPI 토큰 관리에서 발급합니다.

### 환경변수로 인증 (토큰 하드코딩 방지)

```bash
export ALPHAKEY_BASE_URL="https://your-company.alphakey.kr"
export ALPHAKEY_API_TOKEN="발급받은_토큰"
terraform plan
```

---

## 예제

### 사용자 생성

```hcl
resource "alphakey_user" "hong" {
  dept_name   = "개발팀"
  last_name   = "홍"
  first_name  = "길동"
  email       = "hong@example.com"
  mobile      = "010-1234-5678"
  id_state_yn = "Y"
}
```

### 사용자 그룹 생성 + 멤버 추가

```hcl
resource "alphakey_user_group" "devteam" {
  user_group_name = "개발팀"
  user_group_desc = "개발팀 사용자 그룹"
  user_ids        = [alphakey_user.hong.id]
}
```

### 앱 등록 + 사용자 권한 부여

```hcl
resource "alphakey_app" "figma" {
  lg_saas_id = "figma-saas-id"
}

resource "alphakey_user_app" "hong_figma" {
  saas_id  = alphakey_app.figma.id
  user_ids = [alphakey_user.hong.id]
}
```

### 접속 차단 IP 추가

```hcl
resource "alphakey_blocked_ip" "malicious" {
  ip_address  = "192.168.1.100"
  subnet_mask = "255.255.255.255"
  reason      = "악성 접속 시도 감지"
}
```

### 기존 사용자 조회 (Data Source)

```hcl
data "alphakey_user" "existing" {
  user_id = "pearllee@lguplus.co.kr"
}

output "user_dept" {
  value = data.alphakey_user.existing.dept_name
}
```

---

## 리소스

| 리소스 | 설명 |
|--------|------|
| alphakey_user | 사용자 등록, 수정, 삭제, ID 생성/회수 |
| alphakey_app | SaaS 앱 등록/삭제 |
| alphakey_app_contract | 앱 계약정보 등록/수정 |
| alphakey_user_group | 사용자 그룹 CRUD + 멤버 관리 |
| alphakey_app_group | 앱 그룹 CRUD + 앱/사용자 관리 |
| alphakey_user_app | 사용자-앱 권한 부여/회수 |
| alphakey_blocked_ip | 접속 차단 IP 관리 |
| alphakey_admin | 일반 관리자 추가/삭제 |
| alphakey_mfa_policy | MFA 인증수단 정책 관리 |
| alphakey_password_expiry | 비밀번호 유효기간 설정 |

## 데이터소스

| 데이터소스 | 설명 |
|-----------|------|
| alphakey_user | 단일 사용자 조회 |
| alphakey_users | 사용자 목록 조회 |
| alphakey_app | 단일 앱 조회 |
| alphakey_apps | 앱 목록 조회 |
| alphakey_user_groups | 사용자 그룹 목록 |
| alphakey_app_groups | 앱 그룹 목록 |
| alphakey_departments | 부서 목록 |
| alphakey_app_categories | 앱 카테고리 목록 |

---

## Import

기존 알파키 리소스를 Terraform State로 가져올 수 있습니다.

```bash
terraform import alphakey_user.hong "hong@example.com"
terraform import alphakey_app.figma "saas-id-here"
terraform import alphakey_user_group.devteam "group-id-here"
terraform import alphakey_blocked_ip.malicious "network-policy-id"
```

---

## 문제 해결

| 증상 | 원인 | 해결 |
|------|------|------|
| `terraform init` 시 Provider를 찾을 수 없음 | `.terraformrc` 경로 설정 오류 | `path`가 바이너리의 상위 폴더를 가리키는지 확인 |
| `Access is denied` (Windows) | 보안 소프트웨어가 exe 차단 | PowerShell 관리자 권한으로 `Unblock-File` 실행 |
| `인증 정보가 유효하지 않습니다` | 토큰 만료 또는 오타 | 관리자 콘솔에서 토큰 재발급 |
| `네트워크 연결 실패` | URL 오류 또는 방화벽 | `base_url`에 `https://` 포함 확인, `/openapi` 미포함 확인 |
| `TLS 인증서 검증 실패` | 자체 서명 인증서 환경 | `insecure = true` 설정 (개발 환경만) |

---

## 개발

### 빌드

```bash
go build -o terraform-provider-alphakey
```

### 로컬 설치

```bash
make install
```

### 테스트

```bash
make test
```

---

## 버전 호환성

| 구성 요소 | 지원 버전 |
|----------|----------|
| Terraform | 1.0 이상 |
| Go (빌드 시) | 1.22 이상 |
| 알파키 OpenAPI | v1 |

---

## 라이선스

Copyright © LG유플러스. All rights reserved.

본 소프트웨어는 알파키(AlphaKey) IDaaS 서비스 이용 고객에게 제공됩니다.
