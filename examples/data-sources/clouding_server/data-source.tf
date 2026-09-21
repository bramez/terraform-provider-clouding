##############################
# Data source: clouding_server #
##############################

data "clouding_server" "example" {
  id = "06Wq42P0BJr2eDVE"
}

output "server_public_ip" {
  value = data.clouding_server.example.public_ip
}
