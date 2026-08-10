package github

type (
	user struct {
		Login string `json:"login"`
	}

	repository struct {
		Name          string      `json:"name"`
		FullName      string      `json:"full_name"`
		DefaultBranch string      `json:"default_branch"`
		Owner         user        `json:"owner"`
		Parent        *repository `json:"parent,omitempty"`
		Permissions   permissions `json:"permissions"`
	}

	permissions struct {
		Push bool `json:"push"`
	}

	gitReference struct {
		Ref    string    `json:"ref"`
		Object gitObject `json:"object"`
	}

	gitObject struct {
		SHA string `json:"sha"`
	}

	gitCommit struct {
		SHA     string      `json:"sha"`
		Tree    gitObject   `json:"tree"`
		Parents []gitObject `json:"parents"`
	}

	commitDetails struct {
		SHA     string      `json:"sha"`
		Parents []gitObject `json:"parents"`
		Files   []pullFile  `json:"files"`
	}

	pullRequest struct {
		Number  int      `json:"number"`
		HTMLURL string   `json:"html_url"`
		Head    pullHead `json:"head"`
	}

	pullHead struct {
		Ref  string      `json:"ref"`
		SHA  string      `json:"sha"`
		Repo *repository `json:"repo"`
	}

	pullFile struct {
		Filename string `json:"filename"`
		Status   string `json:"status"`
	}

	contentResponse struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	createdTree struct {
		SHA string `json:"sha"`
	}

	createdCommit struct {
		SHA string `json:"sha"`
	}
)
