terraform {
  required_providers {
    alphakey = {
      source  = "lgu-idaas/alphakey"
      version = "~> 0.1.0"
    }
  }
}

# 인증 정보는 환경변수로 설정:
#   export ALPHAKEY_BASE_URL="https://your-tenant.alphakey.kr"
#   export ALPHAKEY_API_TOKEN="your-api-token"
provider "alphakey" {}
