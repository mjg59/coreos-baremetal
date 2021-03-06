package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	logtest "github.com/Sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"

	"github.com/coreos/coreos-baremetal/bootcfg/server"
	"github.com/coreos/coreos-baremetal/bootcfg/storage/storagepb"
	fake "github.com/coreos/coreos-baremetal/bootcfg/storage/testfakes"
)

func TestCloudHandler(t *testing.T) {
	content := `#cloud-config
coreos:
  etcd2:
    name: {{.uuid}}
  units:
    - name: {{.service_name}}
`
	expected := `#cloud-config
coreos:
  etcd2:
    name: a1b2c3d4
  units:
    - name: etcd2
`
	store := &fake.FixedStore{
		Profiles:     map[string]*storagepb.Profile{fake.Group.Profile: fake.Profile},
		CloudConfigs: map[string]string{fake.Profile.CloudId: content},
	}
	logger, _ := logtest.NewNullLogger()
	srv := NewServer(&Config{Logger: logger})
	c := server.NewServer(&server.Config{Store: store})
	h := srv.cloudHandler(c)
	ctx := withGroup(context.Background(), fake.Group)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	h.ServeHTTP(ctx, w, req)
	// assert that:
	// - Cloud config is rendered with Group metadata and selectors
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, expected, w.Body.String())
}

func TestCloudHandler_MissingCtxProfile(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	srv := NewServer(&Config{Logger: logger})
	c := server.NewServer(&server.Config{Store: &fake.EmptyStore{}})
	h := srv.cloudHandler(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	h.ServeHTTP(context.Background(), w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCloudHandler_MissingCloudConfig(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	srv := NewServer(&Config{Logger: logger})
	c := server.NewServer(&server.Config{Store: &fake.EmptyStore{}})
	h := srv.cloudHandler(c)
	ctx := withProfile(context.Background(), fake.Profile)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	h.ServeHTTP(ctx, w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCloudHandler_MissingTemplateMetadata(t *testing.T) {
	content := `#cloud-config
coreos:
  etcd2:
    name: {{.missing_key}}
`
	store := &fake.FixedStore{
		Profiles:     map[string]*storagepb.Profile{fake.Group.Profile: fake.Profile},
		CloudConfigs: map[string]string{fake.Profile.CloudId: content},
	}
	logger, _ := logtest.NewNullLogger()
	srv := NewServer(&Config{Logger: logger})
	c := server.NewServer(&server.Config{Store: store})
	h := srv.cloudHandler(c)
	ctx := withGroup(context.Background(), fake.Group)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	h.ServeHTTP(ctx, w, req)
	// assert that:
	// - Cloud-config template rendering errors because "missing_key" is not
	// present in the Group metadata
	assert.Equal(t, http.StatusNotFound, w.Code)
}
