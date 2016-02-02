package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cherepski/go-gitlab"
	_ "github.com/go-sql-driver/mysql"
)

var gitlab_url string = "your-gitlab-url"
var database_host string = "your-database-host"
var database_username string = "your-database-username"
var database_password string = "your-database-password"
var database_name string = "your-database-name"
var database_port int = 3306
var ansible_private_key string = "your-private-key-location"
var ansible_extra_vars string = "your-ansible-extra-vars"
var http_port int = 5001
var templates *template.Template = template.Must(template.ParseGlob("templates/*"))
var db *sql.DB

func AuthHandlerFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, err := req.Cookie("private-token")
		if err != nil {
			if err == http.ErrNoCookie {
				http.Redirect(w, req, "/login/", 302)
				return
			} else {
				log.Fatal(err)
			}
		}
		fn(w, req)
	}
}

func GetNewClientViaAuth(req *http.Request) *gitlab.Client {
	private_token, _ := req.Cookie("private-token")
	return gitlab.NewClient(nil, gitlab_url, private_token.Value)
}

func main() {
	// Connect to the DB.
	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", database_username, database_password, database_host, database_port, database_name))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fs := http.FileServer(http.Dir("media"))
	http.Handle("/media/", http.StripPrefix("/media/", fs))
	fs_static := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs_static))
	http.Handle("/", AuthHandlerFunc(Index))
	http.Handle("/deploy/", AuthHandlerFunc(Deploy))
	http.Handle("/logs/", AuthHandlerFunc(Logs))
	http.Handle("/login/", http.HandlerFunc(Login))
	http.Handle("/logout/", AuthHandlerFunc(Logout))
	err = http.ListenAndServe(fmt.Sprintf(":%d", http_port), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}

func Logout(w http.ResponseWriter, req *http.Request) {
	priv_token, err := req.Cookie("private-token")
	if err != nil {
		log.Fatal(err)
	}
	priv_token.MaxAge = -1
	priv_token.Value = ""
	priv_token.Path = "/"
	http.SetCookie(w, priv_token)
	http.Redirect(w, req, "/login/", 302)
}

func Login(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		templates.ExecuteTemplate(w, "login", nil)
	case "POST":
		req.ParseForm()
		username := req.Form["username"][0]
		password := req.Form["password"][0]

		sessionOpts := gitlab.GetSessionOptions{Login: username, Password: password}
		git := gitlab.NewClient(nil, gitlab_url, "")
		session, response, err := git.Session.GetSession(&sessionOpts)
		if err != nil {
			if response.StatusCode == 401 {
				templates.ExecuteTemplate(w, "login", "Invalid Credentials")
				return
			}
			log.Fatal(err)
		}
		http.SetCookie(w, &http.Cookie{Name: "private-token", Value: session.PrivateToken, Path: "/"})
		http.Redirect(w, req, "/", 302)
	default:
		log.Fatal("Invalid HTTP Method")
	}
}

type Log struct {
	Id          string
	User        string
	User_id     string
	Project     string
	Project_id  string
	Commit_hash string
	Version     string
	Comment     string
	Results     string
	Created_on  string
	Modified_on string
}

type IndexProject struct {
	Id          int
	Name        string
	Description string
	Url         string
}

func Index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	rows, err := db.Query("SELECT * FROM `logs` ORDER BY `created_on` DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	logs := []Log{}
	defer rows.Close()
	for rows.Next() {
		var deploylog Log
		err := rows.Scan(&deploylog.Id, &deploylog.User, &deploylog.User_id, &deploylog.Project, &deploylog.Project_id, &deploylog.Commit_hash, &deploylog.Version, &deploylog.Comment, &deploylog.Results, &deploylog.Created_on, &deploylog.Modified_on)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, deploylog)
	}

	git := GetNewClientViaAuth(req)
	user, _, err := git.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	projects, _, err := git.Projects.ListProjects(nil)
	if err != nil {
		log.Fatal(err)
	}

	indexProjectList := []IndexProject{}
	for _, project := range projects {
		description := ""
		url := ""
		description_arr := strings.Split(*project.Description, " : ")
		if len(description_arr) == 1 {
			description = description_arr[0]
		} else if len(description_arr) == 2 {
			description = description_arr[0]
			url = description_arr[1]
		}
		tmpProject := IndexProject{Id: *project.ID, Name: *project.Name, Description: description, Url: url}
		indexProjectList = append(indexProjectList, tmpProject)
	}

	templates.ExecuteTemplate(w, "index", map[string]interface{}{"User": user, "Projects": indexProjectList, "Logs": logs})
}

func Logs(w http.ResponseWriter, req *http.Request) {
	git := GetNewClientViaAuth(req)
	user, _, err := git.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	req.ParseForm()
	if id, ok := req.Form["id"]; ok {
		row := db.QueryRow("SELECT * FROM `logs` WHERE `id` = ?", id[0])
		var deploylog Log
		err := row.Scan(&deploylog.Id, &deploylog.User, &deploylog.User_id, &deploylog.Project, &deploylog.Project_id, &deploylog.Commit_hash, &deploylog.Version, &deploylog.Comment, &deploylog.Results, &deploylog.Created_on, &deploylog.Modified_on)
		if err != nil {
			log.Fatal(err)
		}
		float_ver, err := strconv.ParseFloat(deploylog.Version, 64)
		if err != nil {
			log.Fatal(err)
		}
		deploylog.Version = fmt.Sprintf("%.1f", float_ver)

		templates.ExecuteTemplate(w, "logdetails", map[string]interface{}{"Log": deploylog, "User": user})
		return
	}

	rows, err := db.Query("SELECT * FROM `logs`")
	if err != nil {
		log.Fatal(err)
	}
	logs := []Log{}
	defer rows.Close()
	for rows.Next() {
		var deploylog Log
		err := rows.Scan(&deploylog.Id, &deploylog.User, &deploylog.User_id, &deploylog.Project, &deploylog.Project_id, &deploylog.Commit_hash, &deploylog.Version, &deploylog.Comment, &deploylog.Results, &deploylog.Created_on, &deploylog.Modified_on)
		if err != nil {
			log.Fatal(err)
		}
		logs = append(logs, deploylog)
	}

	templates.ExecuteTemplate(w, "logs", map[string]interface{}{"Logs": logs, "User": user})
}

func Deploy(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	id := req.Form["id"][0]

	git := GetNewClientViaAuth(req)
	user, _, err := git.Users.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	project, _, err := git.Projects.GetProject(id)
	if err != nil {
		log.Fatal(err)
	}
	switch req.Method {
	case "GET":
		commit, status, err := git.Commits.GetCommit(id, "master")
		if status.StatusCode == 404 {
			templates.ExecuteTemplate(w, "master_branch_not_found", map[string]interface{}{"User": user, "Project": project})
			return
		}
		if err != nil {
			log.Fatal(err)
		}

		row := db.QueryRow("SELECT `commit_hash`, `version` FROM `logs` WHERE `project_id` = ? ORDER BY `created_on` DESC LIMIT 1", id)
		var previous_commit string
		var version float64
		err = row.Scan(&previous_commit, &version)
		switch {
		case err == sql.ErrNoRows:
			log.Print("No rows matched")
		case err != nil:
			log.Fatal(err)
		default:
			log.Print(err)
		}

		var prev_commit *gitlab.Commit
		var compare *gitlab.Compare
		var str_suggested_version = "1.0"
		if err != sql.ErrNoRows {
			prev_commit, _, err = git.Commits.GetCommit(id, previous_commit)
			if err != nil {
				log.Fatal(err)
			}

			compareOpts := &gitlab.CompareOptions{From: previous_commit, To: commit.ID}
			compare, _, err = git.Repositories.Compare(id, compareOpts)
			if err != nil {
				log.Fatal(err)
			}

			suggested_version := version + 0.1
			str_suggested_version = fmt.Sprintf("%.1f", suggested_version)
		}

		rawFileContentOpts := &gitlab.RawFileContentOptions{FilePath: "deployment/deploy.yaml"}
		yaml_deploy_file, _, err := git.Repositories.RawFileContent(id, "master", rawFileContentOpts)

		deployable := true
		if len(yaml_deploy_file) == 0 {
			deployable = false
		}

		templates.ExecuteTemplate(w, "deploy", map[string]interface{}{"User": user, "Project": project, "Commit": commit, "Compare": compare, "PrevCommit": prev_commit, "SuggestedVersion": str_suggested_version, "Playbook": string(yaml_deploy_file), "Deployable": deployable})
	case "POST":
		comment := req.Form["comment"][0]
		version := req.Form["version"][0]

		file, err := ioutil.TempFile("/tmp/", "deployer")
		if err != nil {
			log.Fatal(err)
		}

		rawFileContentOpts := &gitlab.RawFileContentOptions{FilePath: "deployment/deploy.yaml"}
		yaml_deploy_file, _, err := git.Repositories.RawFileContent(id, "master", rawFileContentOpts)

		file.Write(yaml_deploy_file)

		out, err := exec.Command("ansible-playbook", file.Name(), "-vvvv", "--extra-vars", ansible_extra_vars, "--private-key", ansible_private_key).CombinedOutput()
		if err != nil {
			log.Fatal(err)
		}
		os.Remove(file.Name())

		createTagOpts := &gitlab.CreateTagOptions{Ref: "master", TagName: version, Message: comment}
		tag, _, err := git.Repositories.CreateTag(id, createTagOpts)
		if err != nil {
			log.Fatal(err)
		}

		_, err = db.Exec("INSERT INTO `logs` (user, user_id, project, project_id, commit_hash, version, comment, results) values (?, ?, ?, ?, ?, ?, ?, ?)", user.Username, user.ID, project.Name, project.ID, tag.Commit.ID, version, comment, string(out))
		if err != nil {
			log.Fatal(err)
		}

		templates.ExecuteTemplate(w, "deploy_confirmation", map[string]interface{}{"User": user, "Project": project, "Tag": tag, "Result": string(out)})
	default:
	}
}
