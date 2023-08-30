package stages

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/images"
)

func TestGetLatestImages(t *testing.T) {
	testCases := []struct {
		name           string
		credentialsDB  credentials.Database
		getLatestTagFn func(
			string,
			images.ImageUpdateStrategy,
			string,
			string,
			[]string,
			string,
			*images.Credentials,
		) (string, error)
		assertions func([]api.Image, error)
	}{
		{
			name: "error getting latest version of an image",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestTagFn: func(
				repoURL string,
				updateStrategy images.ImageUpdateStrategy,
				semverConstraint string,
				allowTags string,
				ignoreTags []string,
				platform string,
				creds *images.Credentials,
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ []api.Image, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error getting latest suitable tag for image",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			getLatestTagFn: func(
				repoURL string,
				updateStrategy images.ImageUpdateStrategy,
				semverConstraint string,
				allowTags string,
				ignoreTags []string,
				platform string,
				creds *images.Credentials,
			) (string, error) {
				return "fake-tag", nil
			},
			assertions: func(images []api.Image, err error) {
				require.NoError(t, err)
				require.Len(t, images, 1)
				require.Equal(
					t,
					api.Image{
						RepoURL: "fake-url",
						Tag:     "fake-tag",
					},
					images[0],
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testSubs := []api.ImageSubscription{
				{
					RepoURL: "fake-url",
				},
			}
			r := reconciler{
				credentialsDB:  testCase.credentialsDB,
				getLatestTagFn: testCase.getLatestTagFn,
			}
			testCase.assertions(
				r.getLatestImages(
					context.Background(),
					"fake-namespace",
					testSubs,
				),
			)
		})
	}
}

func TestFetImageSourceURL(t *testing.T) {
	const testURLPrefix = "fake-url-prefix"
	testCases := []struct {
		name        string
		reconciler  *reconciler
		expectedURL string
	}{
		{
			name:        "no image source URL function found",
			reconciler:  &reconciler{},
			expectedURL: "",
		},
		{
			name: "image source URL function found",
			reconciler: &reconciler{
				imageSourceURLFnsByBaseURL: map[string]func(string, string) string{
					testURLPrefix: func(string, string) string {
						return "fake-url"
					},
				},
			},
			expectedURL: "fake-url",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expectedURL,
				testCase.reconciler.getImageSourceURL(testURLPrefix, "fake-tag"),
			)
		})
	}
}

func TestGetGithubImageSourceURL(t *testing.T) {
	const testTag = "fake-tag"
	testCases := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "with .git suffix",
			baseURL: "https://github.com/akuity/kargo.git",
		},
		{
			name:    "without .git suffix",
			baseURL: "https://github.com/akuity/kargo",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				fmt.Sprintf(
					"%s/tree/%s",
					"https://github.com/akuity/kargo",
					testTag,
				),
				getGithubImageSourceURL(testCase.baseURL, testTag),
			)
		})
	}
}