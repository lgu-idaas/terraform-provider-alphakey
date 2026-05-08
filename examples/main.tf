terraform {
  required_providers {
    alphakey = {
      source  = "registry.terraform.io/lgu-idaas/alphakey"
      version = "0.1.0"
    }
  }
}

provider "alphakey" {
  base_url  = "https://your-company.alphakey.kr"
  api_token = "여기에_토큰_입력"
}

# 사용자 목록 조회 테스트
data "alphakey_users" "all" {}

output "user_count" {
  value = length(data.alphakey_users.all.users)
}
