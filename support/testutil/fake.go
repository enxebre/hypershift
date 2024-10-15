package testutil

import "github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/imageprovider"

var _ imageprovider.ReleaseImageProvider = &fakeImageProvider{}

func FakeImageProvider() imageprovider.ReleaseImageProvider {
	return &fakeImageProvider{}
}

type fakeImageProvider struct {
}

// Version implements imageprovider.ReleaseImageProvider.
func (f *fakeImageProvider) Version() string {
	return "4.18"
}

func (f *fakeImageProvider) GetImage(key string) string {
	return key
}

func (f *fakeImageProvider) ImageExist(key string) (string, bool) {
	return key, true
}
