# AlphaKey Terraform Provider

[AlphaKey](https://alphakey.co.kr) IDaaS 플랫폼의 리소스를 Terraform으로 선언적으로 관리할 수 있는 Provider입니다.

사용자, 앱, 그룹, 보안 정책 등을 코드로 관리하고, 반복 작업을 자동화할 수 있습니다.

## 요구사항

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.0
- AlphaKey OpenAPI 토큰 (관리자 콘솔 > 설정 > OpenAPI 토큰 관리에서 발급)

## 사용법

```hcl
terraform {
  required_providers {
    alphakey = {
      source  = "lgu-idaas/alphakey"
      version = "~> 0.1.0"
    }
  }
}

provider "alphakey" {
  base_url  = "https://your-tenant.alphakey.kr"
  api_token = var.alphakey_token
}
```

### 환경변수로 인증 (권장)

```bash
export ALPHAKEY_BASE_URL="https://your-tenant.alphakey.kr"
export ALPHAKEY_API_TOKEN="your-api-token"
```

```hcl
provider "alphakey" {}
```

## Provider 속성

| 속성 | 타입 | 필수 | 설명 |
|------|------|------|------|
| `base_url` | String | Yes | 알파키 테넌트 URL. 환경변수 `ALPHAKEY_BASE_URL`로 대체 가능 |
| `api_token` | String | Yes | API 인증 토큰. 환경변수 `ALPHAKEY_API_TOKEN`으로 대체 가능 |
| `request_timeout` | Number | No | HTTP 요청 타임아웃(초). 기본값: 30 |
| `insecure` | Bool | No | TLS 인증서 검증 건너뛰기. 기본값: false |

## 리소스

| 리소스 | 설명 |
|--------|------|
| `alphakey_user` | 사용자 등록, 수정, 삭제, ID 생성/회수 |
| `alphakey_app` | SaaS 앱 등록/삭제 |
| `alphakey_app_contract` | 앱 계약정보 등록/수정 |
| `alphakey_user_group` | 사용자 그룹 CRUD + 멤버 관리 |
| `alphakey_app_group` | 앱 그룹 CRUD + 앱/사용자 관리 |
| `alphakey_user_app` | 사용자-앱 권한 부여/회수 |
| `alphakey_blocked_ip` | 접속 차단 IP 관리 |
| `alphakey_admin` | 일반 관리자 추가/삭제 |
| `alphakey_mfa_policy` | MFA 인증수단 정책 관리 |
| `alphakey_password_expiry` | 비밀번호 유효기간 설정 |

## 데이터소스

| 데이터소스 | 설명 |
|-----------|------|
| `alphakey_user` | 단일 사용자 조회 |
| `alphakey_users` | 사용자 목록 조회 |
| `alphakey_app` | 단일 앱 조회 |
| `alphakey_apps` | 앱 목록 조회 |
| `alphakey_user_groups` | 사용자 그룹 목록 |
| `alphakey_app_groups` | 앱 그룹 목록 |
| `alphakey_departments` | 부서 목록 |
| `alphakey_app_categories` | 앱 카테고리 목록 |

## 예제

### 사용자 생성

```hcl
resource "alphakey_user" "hong" {
  dept_name   = "개발팀"
  last_name   = "홍"
  first_name  = "길동"
  email       = "hong@example.com"
  mobile      = "010-0000-0000"
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
resource "alphakey_app" "myapp" {
  lg_saas_id       = "0"
  contract_user_id = "admin@example.com"
}

resource "alphakey_user_app" "hong_myapp" {
  saas_id  = alphakey_app.myapp.id
  user_ids = [alphakey_user.hong.id]
}
```

### 접속 차단 IP 추가

```hcl
resource "alphakey_blocked_ip" "suspicious" {
  ip_address   = "192.168.1.100"
  block_reason = "악성 접속 시도 감지"
}
```

### 기존 사용자 조회 (Data Source)

```hcl
data "alphakey_user" "existing" {
  user_id = "user@example.com"
}

output "user_dept" {
  value = data.alphakey_user.existing.dept_name
}
```

## Import

기존 알파키 리소스를 Terraform State로 가져올 수 있습니다.

```bash
terraform import alphakey_user.hong "hong@example.com"
terraform import alphakey_app.myapp "saas-id"
terraform import alphakey_user_group.devteam "group-id"
terraform import alphakey_blocked_ip.suspicious "network-policy-id"
```

## 개발

```bash
# 빌드
go build -o terraform-provider-alphakey

# 로컬 설치
make install

# 테스트
make test
```

## 버전 호환성

| 구성 요소 | 지원 버전 |
|----------|----------|
| Terraform | 1.0 이상 |
| Go (빌드 시) | 1.22 이상 |
| AlphaKey OpenAPI | v1 |

## 라이선스

[MPL-2.0](./LICENSE)
