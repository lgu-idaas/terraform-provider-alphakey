# 사용자 관리

resource "alphakey_user" "testuser1" {
  dept_name   = "개발팀"
  last_name   = "테"
  first_name  = "라폼"
  email       = "tfuser02@alphakey.kr"
  mobile      = "010-0000-0002"
  id_state_yn = "N"
}
