package manifests

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"io/fs"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

var (
	//go:embed testdata
	testdataFS embed.FS

	createRecord = false
)

func TestBuildAll_GoldenRecord(t *testing.T) {
	dir, err := fs.ReadDir(testdataFS, "testdata")
	if err != nil {
		t.Fatalf("can not read testdata dir: %s", err)
	}

	for _, tc := range dir {
		tc := tc

		if !tc.IsDir() {
			continue
		}

		t.Run(tc.Name(), func(t *testing.T) {
			t.Parallel()

			var opts Options
			optsFile, err := testdataFS.Open(filepath.Join("testdata", tc.Name(), "opts.json"))
			if err != nil {
				t.Fatalf("can not open opts file: %s", err)
			}
			defer optsFile.Close()

			if err := json.NewDecoder(optsFile).Decode(&opts); err != nil {
				t.Fatalf("can not decode opts JSON: %s", err)
			}

			objects, err := BuildAll(opts)
			if err != nil {
				t.Errorf("error generating manifests: %s", err)
			}

			for _, obj := range objects {
				obj := obj

				id := fmt.Sprintf("%s-%s.yaml", strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind), obj.GetName())
				t.Run(id, func(t *testing.T) {
					objBytes, err := yaml.Marshal(obj)
					if err != nil {
						t.Errorf("can not encode object %q: %s", obj.GetName(), err)
					}

					fileName := filepath.Join("testdata", tc.Name(), id)
					wantBytes, err := testdataFS.ReadFile(fileName)
					switch {
					case errors.Is(err, fs.ErrNotExist):
						if createRecord {
							if err := os.WriteFile(fileName, objBytes, 0644); err != nil {
								t.Fatalf("can not create object file: %s", err)
							}
						}
						t.Errorf("object does not exist in record: %s", id)
						return
					case err != nil:
						t.Errorf("error reading object file: %s", err)
						return
					}

					if diff := cmp.Diff(objBytes, wantBytes); diff != "" {
						t.Errorf("objects differ: -got+want\n%s", diff)
					}
				})
			}
		})
	}
}
