package licenselib

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	yaml "gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

func init() {
	loadAllLicenses()
}

var (
	licenses []*LicenseData
)

type (
	bigram struct{ r0, r1 rune }
	Match  struct {
		Coefficient float64
		License     *LicenseData
	}
	matchSorter []Match
)

func (s Match) String() string {
	return fmt.Sprintf("%s\t%0.2f", s.License.SpdxID, s.Coefficient*100.0)
}

func (m matchSorter) Len() int           { return len(m) }
func (m matchSorter) Less(i, j int) bool { return m[i].Coefficient > m[j].Coefficient }
func (m matchSorter) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }

type LicenseData struct {
	Title        string              `yaml:"title"`
	SpdxID       string              `yaml:"spdx-id"`
	RedirectFrom string              `yaml:"redirect_from"`
	Source       string              `yaml:"source"`
	Description  string              `yaml:"description"`
	How          string              `yaml:"how"`
	Conditions   []string            `yaml:"conditions"`
	Permissions  []string            `yaml:"permissions"`
	Limitations  []string            `yaml:"limitations"`
	Using        []map[string]string `yaml:"using"`

	Text string `yaml:"-"`
}

func bigrams(s string) map[bigram]bool {
	result := make(map[bigram]bool)
	last := rune(0)
	for i, c := range s {
		if i > 0 {
			result[bigram{last, c}] = true
		}
		last = c
	}
	return result
}

func diceCoefficient(a, b string) float64 {
	ba, bb := bigrams(a), bigrams(b)
	var intersection float64
	for a, _ := range ba {
		if bb[a] {
			intersection += 1
		}
	}
	return intersection * 2.0 / float64(len(ba)+len(bb))
}

// File reads from a file
func File(s string) ([]Match, error) {
	f, err := os.Open(s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}
	defer f.Close()

	return Reader(f)
}

// Reader uses an io.Reader
func Reader(r io.Reader) ([]Match, error) {
	txt, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var matches []Match
	for _, lic := range licenses {
		matches = append(matches, Match{
			Coefficient: diceCoefficient(lic.Text, string(txt)),
			License:     lic,
		})
	}

	sort.Sort(matchSorter(matches))
	return matches, nil
}

func readLicenseData(path string) (*LicenseData, error) {
	txt, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fragments := bytes.SplitN(txt, []byte("---\n"), 3)
	if len(fragments) != 3 {
		return nil, errors.New("want 3 fragments")
	}

	var lic LicenseData
	if err = yaml.Unmarshal(fragments[1], &lic); err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", filepath.Base(path))
	}

	lic.Text = string(fragments[2])

	return &lic, nil
}

func loadAllLicenses() {
	base, err := getBasePath("")
	if err != nil {
		panic("could not find base path: " + err.Error())
	}
	licenseDir := filepath.Join(base, "_licenses")
	files, err := filepath.Glob(filepath.Join(licenseDir, "*.txt"))
	if err != nil {
		panic("could not load license files: " + err.Error())
	}

	for _, f := range files {
		lic, err := readLicenseData(f)
		if err != nil {
			panic("could not load license: " + err.Error())
		}

		licenses = append(licenses, lic)
	}
}

var basePackage = "github.com/aarondl/licenser/licenselib"

func getBasePath(baseDirConfig string) (string, error) {
	if len(baseDirConfig) > 0 {
		return baseDirConfig, nil
	}

	p, _ := build.Default.Import(basePackage, "", build.FindOnly)
	if p != nil && len(p.Dir) > 0 {
		return p.Dir, nil
	}

	return os.Getwd()
}
