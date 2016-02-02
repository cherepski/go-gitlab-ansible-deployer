# go-gitlab-ansible-deployer
Go application to deploy GitLab repositories to production

The aim of this project is to create a user friendly and flexible deployment of GitLab repositories to beta/production servers using Ansible playbooks.  Each GitLab repository should contain at the root of the project ```deployment/deploy.yaml```.  ```deploy.yaml``` is the Ansible playbook that will be run for a selected GitLab project that is chosen to be deployed.  There is an example Ansible playbook found in the root of the project ```deployment/deploy.yaml```.  This Go application expects Ansible to be installed on the same host it is running on.

Requirements:
- Go >= 1.5
- Ansible >= 1.9.4
- MySQL >= 5.6.21

1. go get github.com/cherepski/go-gitlab-ansible-deployer
2. configure the following Go vars at the top of main.go
  - var gitlab_url string = "your-gitlab-url"
  - var database_host string = "your-database-host"
  - var database_username string = "your-database-username"
  - var database_password string = "your-database-password"
  - var database_name string = "your-database-name"
  - var database_port int = 3306
  - var ansible_private_key string = "your-private-key-location"
  - var ansible_extra_vars string = "your-ansible-extra-vars"
  - var http_port int = 5001
3. create MySQL database "your-database-username" at host "your-database-host" and load the MySQL create_initial.sql script
4. change directory into github.com/cherepski/go-gitlab-ansible-deployer
5. go run main.go or go build
5. run the application and you should be able to login to your GitLab account and see your projects listed
6. deploy projects or view logs
