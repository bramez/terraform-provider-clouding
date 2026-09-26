package clouding

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// notFoundServer stands in for the Clouding API answering a GET for a resource
// that no longer exists, with the problem+json body the API really returns.
func notFoundServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(`
			{
			  "type": "https://tools.ietf.org/html/rfc7231#section-6.5.4",
			  "title": "Not Found",
			  "status": 404,
			  "detail": "Resource not found"
			}
		`))
		if err != nil {
			t.Errorf("error writing the not found response: %s", err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Each getter behind a resource Read must report a deleted resource as
// ErrNotFound, so the provider can drop it from state instead of failing the
// plan. Without this, deleting a resource outside Terraform makes every later
// `terraform plan` abort during refresh.
func TestGetServerIDNotFound(t *testing.T) {
	t.Parallel()
	srv := notFoundServer(t)

	client, err := NewAPI("token", WithEndpoint(srv.URL))
	assert.NoError(t, err)

	server := Server{ID: "ke8vlrXPjxO1oq3m"}
	err = client.GetServerID(&server)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got: %v", err)
}

func TestGetFirewallIDNotFound(t *testing.T) {
	t.Parallel()
	srv := notFoundServer(t)

	client, err := NewAPI("token", WithEndpoint(srv.URL))
	assert.NoError(t, err)

	_, err = client.GetFirewallID("LywOkvx5LWAp28NP")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got: %v", err)
}

func TestGetFirewallRuleNotFound(t *testing.T) {
	t.Parallel()
	srv := notFoundServer(t)

	client, err := NewAPI("token", WithEndpoint(srv.URL))
	assert.NoError(t, err)

	_, err = client.GetFirewallRule("JLB82xyP8aWOrqeN")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got: %v", err)
}

func TestGetSshKeyIDNotFound(t *testing.T) {
	t.Parallel()
	srv := notFoundServer(t)

	client, err := NewAPI("token", WithEndpoint(srv.URL))
	assert.NoError(t, err)

	_, err = client.GetSshKeyID("Dd8v0nXJ1924rayY")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got: %v", err)
}

// A non-404 failure must stay a hard error: a transient 500 or an expired token
// must never be mistaken for "the resource is gone" and silently wipe state.
func TestGetServerIDServerErrorIsNotNotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`{"title":"Internal Server Error","status":500,"detail":"boom"}`))
		if err != nil {
			t.Errorf("error writing the error response: %s", err)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := NewAPI("token", WithEndpoint(srv.URL))
	assert.NoError(t, err)

	server := Server{ID: "ke8vlrXPjxO1oq3m"}
	err = client.GetServerID(&server)

	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrNotFound), "a 500 must not be reported as ErrNotFound")
}
