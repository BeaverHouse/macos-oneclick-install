package install

import (
	"austinhome/internal/constants"
	"net/http"

	"github.com/BeaverHouse/go-common/errorhandle"
	"go.yaml.in/yaml/v3"
)

const cicdApplicationSetsURL = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/argocd/applicationsets/"

type applicationSource struct {
	RepoURL        string `yaml:"repoURL"`
	TargetRevision string `yaml:"targetRevision"`
}

type applicationSet struct {
	Spec struct {
		Template struct {
			Spec struct {
				Source  applicationSource   `yaml:"source"`
				Sources []applicationSource `yaml:"sources"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

// cicdVersion returns the targetRevision that Argo CD deploys from repoURL,
// so the CLI installs the same version that Argo CD later takes over.
func cicdVersion(applicationSetFile, repoURL string) (string, error) {
	url := cicdApplicationSetsURL + applicationSetFile
	resp, err := http.Get(url)
	if err != nil {
		return "", constants.ErrOperationFailed("fetch "+url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errorhandle.ErrInternalMsg("fetch " + url + " returned " + resp.Status)
	}

	var appSet applicationSet
	if err := yaml.NewDecoder(resp.Body).Decode(&appSet); err != nil {
		return "", constants.ErrOperationFailed("parse "+url, err)
	}

	spec := appSet.Spec.Template.Spec
	for _, source := range append(spec.Sources, spec.Source) {
		if source.RepoURL == repoURL {
			return source.TargetRevision, nil
		}
	}
	return "", errorhandle.ErrNotFound(repoURL + " source in " + url)
}
