# 사용자 그룹 관리

resource "alphakey_user_group" "devteam" {
  user_group_name = "TF-개발팀2"
  user_group_desc = "Terraform으로 생성한 개발팀 그룹"
}
