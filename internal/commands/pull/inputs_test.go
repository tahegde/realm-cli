package pull

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/10gen/realm-cli/internal/cloud/realm"
	"github.com/10gen/realm-cli/internal/local"
	u "github.com/10gen/realm-cli/internal/utils/test"
	"github.com/10gen/realm-cli/internal/utils/test/assert"
	"github.com/10gen/realm-cli/internal/utils/test/mock"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestPullInputsResolve(t *testing.T) {
	t.Run("should return an error if run from outside a project directory and no app-dir is set", func(t *testing.T) {
		profile, teardown := mock.NewProfileFromTmpDir(t, "pull_input_test")
		defer teardown()

		var i inputs
		assert.Equal(t, errProjectNotFound{}, i.Resolve(profile, nil))
	})

	t.Run("when run inside a project directory", func(t *testing.T) {
		profile, teardown := mock.NewProfileFromTmpDir(t, "pull_input_test")
		defer teardown()

		assert.Nil(t, ioutil.WriteFile(
			filepath.Join(profile.WorkingDirectory, local.FileConfig.String()),
			[]byte(`{"config_version":20210101,"app_id":"eggcorn-abcde","name":"eggcorn"}`),
			0666,
		))

		t.Run("should set inputs from app if no flags are set", func(t *testing.T) {
			var i inputs
			assert.Nil(t, i.Resolve(profile, nil))

			assert.Equal(t, profile.WorkingDirectory, i.Target)
			assert.Equal(t, "eggcorn-abcde", i.From)
			assert.Equal(t, realm.AppConfigVersion20210101, i.AppVersion)
		})

		t.Run("should return an error if app version flag is different from the project value", func(t *testing.T) {
			i := inputs{AppVersion: realm.AppConfigVersion20200603}
			assert.Equal(t, errConfigVersionMismatch, i.Resolve(profile, nil))
		})
	})

	t.Run("resolving the target flag should work", func(t *testing.T) {
		homeDir, teardown := u.SetupHomeDir("")
		defer teardown()

		for _, tc := range []struct {
			description    string
			targetFlag     string
			expectedTarget string
		}{
			{
				description:    "should expand the target flag to include the user home directory",
				targetFlag:     "~/my/project/root",
				expectedTarget: filepath.Join(homeDir, "my/project/root"),
			},
			{
				description:    "should resolve the target flag to account for relative paths",
				targetFlag:     "../../cmd",
				expectedTarget: filepath.Join(homeDir, "../../cmd"),
			},
		} {
			t.Run(tc.description, func(t *testing.T) {
				profile := mock.NewProfile(t)

				i := inputs{Target: tc.targetFlag}
				assert.Nil(t, i.Resolve(profile, nil))

				assert.Equal(t, tc.expectedTarget, i.Target)
			})
		}
	})
}

func TestPullInputsResolveFrom(t *testing.T) {
	t.Run("should do nothing if to is not set", func(t *testing.T) {
		var i inputs
		tt, err := i.resolveFrom(nil, nil)
		assert.Nil(t, err)
		assert.Equal(t, from{}, tt)
	})

	t.Run("should return the app id and group id of specified app if to is set to app", func(t *testing.T) {
		var appFilter realm.AppFilter
		app := realm.App{
			ID:          primitive.NewObjectID().Hex(),
			GroupID:     primitive.NewObjectID().Hex(),
			ClientAppID: "test-app-abcde",
			Name:        "test-app",
		}

		client := mock.RealmClient{}
		client.FindAppsFn = func(filter realm.AppFilter) ([]realm.App, error) {
			appFilter = filter
			return []realm.App{app}, nil
		}

		i := inputs{Project: app.GroupID, From: app.ClientAppID}

		f, err := i.resolveFrom(nil, client)
		assert.Nil(t, err)

		assert.Equal(t, from{GroupID: app.GroupID, AppID: app.ID}, f)
		assert.Equal(t, realm.AppFilter{GroupID: app.GroupID, App: app.ClientAppID}, appFilter)
	})
}
