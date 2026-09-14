resource "alphakey_blocked_ip" "example" {
  ip_address   = "192.168.1.100"
  block_reason = "악성 접속 시도 감지"
}
