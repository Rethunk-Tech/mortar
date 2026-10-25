package devsvc

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// thunderstorePackage is a Thunderstore package's Team-Name, the form manifest.json's dependencies use.
var thunderstorePackage = regexp.MustCompile(`^[A-Za-z0-9_]+-[A-Za-z0-9_]+$`)

// AddDependency records pkg at version as a ThunderstoreDependency of the project in dir, which the template's pack
// step writes into manifest.json; an existing entry for pkg takes the new version.
func (s *Service) AddDependency(gameID, dir, pkg, version string) error {
	p, err := s.project(gameID, dir)
	if err != nil {
		return err
	}
	if !thunderstorePackage.MatchString(pkg) || !packageVersion.MatchString(version) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s %s is not a Thunderstore package and version", pkg, version))
	}
	csproj, ok := csprojOf(p.Dir)
	if !ok {
		return usererr.Wrap(usererr.NotFound, errors.New("the project file is missing"))
	}
	raw, err := fsx.ReadFile(csproj)
	if err != nil {
		return err
	}
	return fsx.WriteFile(csproj, []byte(withDependency(string(raw), pkg, version)), 0o644) // #nosec G306 -- a tracked project file
}

// withDependency sets pkg's ThunderstoreDependency in a project file: in place when it is listed, after the last
// listed one otherwise, or in a new ItemGroup at the end.
func withDependency(project, pkg, version string) string {
	item := fmt.Sprintf(`<ThunderstoreDependency Include="%s" Version="%s" />`, pkg, version)
	listed := regexp.MustCompile(`<ThunderstoreDependency\s+Include="` + regexp.QuoteMeta(pkg) + `"[^>]*/>`)
	if listed.MatchString(project) {
		return listed.ReplaceAllLiteralString(project, item)
	}
	if last := strings.LastIndex(project, "<ThunderstoreDependency"); last >= 0 {
		lineStart := strings.LastIndex(project[:last], "\n") + 1
		end := last + strings.Index(project[last:], "/>") + len("/>")
		return project[:end] + "\n" + project[lineStart:last] + item + project[end:]
	}
	closing := strings.LastIndex(project, "</Project>")
	if closing < 0 {
		return project
	}
	return project[:closing] + "  <ItemGroup>\n    " + item + "\n  </ItemGroup>\n" + project[closing:]
}
