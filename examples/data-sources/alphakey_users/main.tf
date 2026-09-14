data "alphakey_users" "all" {}

output "user_count" {
  value = length(data.alphakey_users.all.users)
}
