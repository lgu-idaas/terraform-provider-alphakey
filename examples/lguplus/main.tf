terraform {
  required_providers {
    alphakey = {
      source  = "registry.terraform.io/lgu-idaas/alphakey"
      version = "0.1.0"
    }
  }
}

provider "alphakey" {
  base_url  = "https://lguplus.alphakey.kr"
  api_token = "8PyZjoznBPwJTY0u_sB00JQLLvt26TX0n7fcoUwOtVg"
}

resource "alphakey_user_group" "test" {
  user_group_name = "terraform-lguplus"
  user_group_desc = "Terraform test - lguplus tenant"
}
