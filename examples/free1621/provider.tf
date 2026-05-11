terraform {
  required_providers {
    alphakey = {
      source  = "registry.terraform.io/lgu-idaas/alphakey"
      version = "0.1.0"
    }
  }
}

provider "alphakey" {
  base_url  = "https://free1621.alphakey.kr"
  api_token = "AOt991g7gMTtmQ1NqiO-CcdbU7_wrQClX1EUsFhAf1o"
}
