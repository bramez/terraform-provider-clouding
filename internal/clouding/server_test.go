package clouding

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServerID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
			{
			  "id": "ke8vlrXPjxO1oq3m",
			  "name": "database-server",
			  "hostname": "db.example.com",
			  "vCores": 1,
			  "ramGb": 4,
			  "flavor": "1x4",
			  "volumeSizeGb": 15,
			  "image": {
			    "id": "lo1qJ9oZb1xGMEgD",
			    "name": "CelestiaOS 2.04 (64 Bit)"
			  },
			  "status": "Active",
			  "powerState": "Running",
			  "features": [
			    "Backups",
			    "PrivateNetwork"
			  ],
			  "createdAt": "2022-12-19T12:00:00.0000000Z",
			  "dnsAddress": "0447ff27-2d5f-4888-9822-46ea09048cb4.clouding.host",
			  "publicIp": "185.256.254.180",
			  "privateIp": "10.20.10.1",
			  "sshKeyId": "Dd8v0nXJ1924rayY",
			  "firewalls": [
			    {
			      "id": "LywOkvx5LWAp28NP",
			      "name": "Allow all private traffic"
			    },
			    {
			      "id": "JLB82xyP8aWOrqeN",
			      "name": "Allow MySQL"
			    }
			  ],
			  "snapshots": [],
			  "backups": [
			    {
			      "id": "3lo1qJ9oO19GMEgD",
			      "createdAt": "2023-01-03T12:00:00.0000000Z",
			      "status": "Creating"
			    },
			    {
			      "id": "mR2Dn6xgD49OMPyE",
			      "createdAt": "2023-01-02T12:00:00.0000000Z",
			      "status": "Created"
			    },
			    {
			      "id": "wa7BmZXbRoXe2Mjn",
			      "createdAt": "2023-01-01T12:00:00.0000000Z",
			      "status": "Created"
			    },
			    {
			      "id": "NAQopLWpMbxMmr32",
			      "createdAt": "2022-12-31T12:00:00.0000000Z",
			      "status": "Created"
			    }
			  ],
			  "backupPreferences": {
			    "slots": 4,
			    "frequency": "OneDay"
			  },
			  "cost": {
			    "pricePerHour": 0.014004,
			    "pricePerMonthApprox": 10.22292
			  }
			}	
		`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	server := Server{
		ID: "ke8vlrXPjxO1oq3m",
		AccessConfiguration: &AccessConfiguration{
			SshKeyID:     "Dd8v0nXJ1924rayY",
			Password:     "",
			SavePassword: false,
		},
		Volume: &Volume{
			Source: "image",
			ID:     "lo1qJ9oZb1xGMEgD",
			SsdGb:  15,
		},
	}

	err = client.GetServerID(&server)
	if err != nil {
		t.Errorf("getting error calling GetServerID: %s", err)
	}

	assert.Equal(t, "ke8vlrXPjxO1oq3m", server.ID)
	assert.Equal(t, "database-server", server.Name)
	assert.Equal(t, "db.example.com", server.Hostname)
	assert.Equal(t, float64(1), server.VCores)
	assert.Equal(t, 4, server.RamGb)
	assert.Equal(t, "1x4", server.Flavor)
	assert.Equal(t, int64(15), server.VolumeSizeGb)
	assert.Equal(t, "lo1qJ9oZb1xGMEgD", server.Image.ID)
	assert.Equal(t, "CelestiaOS 2.04 (64 Bit)", server.Image.Name)
	assert.Equal(t, "Active", server.Status)
	assert.Equal(t, "Running", server.PowerState)
	assert.Equal(t, "Backups", server.Features[0])
	assert.Equal(t, "PrivateNetwork", server.Features[1])
	assert.Equal(t, "0447ff27-2d5f-4888-9822-46ea09048cb4.clouding.host", server.DnsAddress)
	assert.Equal(t, "185.256.254.180", server.PublicIP)
	assert.Equal(t, "10.20.10.1", server.PrivateIP)
	assert.Equal(t, "Dd8v0nXJ1924rayY", server.SshKeyID)
	assert.Equal(t, "LywOkvx5LWAp28NP", server.Firewalls[0].ID)
	assert.Equal(t, "Allow all private traffic", server.Firewalls[0].Name)
	assert.Equal(t, "JLB82xyP8aWOrqeN", server.Firewalls[1].ID)
	assert.Equal(t, "Allow MySQL", server.Firewalls[1].Name)
	assert.Equal(t, "3lo1qJ9oO19GMEgD", server.Backups[0].ID)
	assert.Equal(t, "2023-01-03T12:00:00.0000000Z", server.Backups[0].CreatedAt)
	assert.Equal(t, "Creating", server.Backups[0].Status)
	assert.Equal(t, "mR2Dn6xgD49OMPyE", server.Backups[1].ID)
	assert.Equal(t, "2023-01-02T12:00:00.0000000Z", server.Backups[1].CreatedAt)
	assert.Equal(t, "Created", server.Backups[1].Status)
	assert.Equal(t, "wa7BmZXbRoXe2Mjn", server.Backups[2].ID)
	assert.Equal(t, "2023-01-01T12:00:00.0000000Z", server.Backups[2].CreatedAt)
	assert.Equal(t, "Created", server.Backups[2].Status)
	assert.Equal(t, "NAQopLWpMbxMmr32", server.Backups[3].ID)
	assert.Equal(t, "2022-12-31T12:00:00.0000000Z", server.Backups[3].CreatedAt)
	assert.Equal(t, "Created", server.Backups[3].Status)
	assert.Equal(t, int64(4), server.BackupPreference.Slots)
	assert.Equal(t, "OneDay", server.BackupPreference.Frequency)
	assert.Equal(t, 0.014004, server.Cost.PricePerHour)
	assert.Equal(t, 10.22292, server.Cost.PricePerMonthApprox)
}

func TestCreateServer(t *testing.T) {
	t.Parallel()
	var server Server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte(`
			{
  				"id": "Q7y1OZWlknXmk6l3",
  				"name": "my server",
  				"hostname": "my-server.example.com",
  				"vCores": 1,
  				"ramGb": 2,
  				"flavor": "1x2",
  				"volumeSizeGb": 10,
  				"image": {
  				  "id": "lo1qJ9oZb1xGMEgD",
  				  "name": "CelestiaOS 2.04 (64 Bit)"
  				},
  				"status": "Spawning",
  				"pendingFeatures": [
  				  "PrivateNetwork"
  				],
  				"pendingFirewalls": [
  				  "LywOkvx5LWAp28NP"
  				],
  				"requestedAccessConfiguration": {
  				  "sshKeyId": "Dd8v0nXJ1924rayY",
  				  "hasPassword": false,
  				  "savePassword": false
  				},
  				"backupPreferences": null,
  				"action": {
  				  "id": "ZPlL0kxDyR9Q3Yb5",
  				  "status": "inProgress",
  				  "type": "create",
  				  "startedAt": "2023-01-03T12:00:00.0000000Z",
  				  "completedAt": null,
  				  "resourceId": "Q7y1OZWlknXmk6l3",
  				  "resourceType": "server"
  				}
			}
		`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	server = Server{
		Name:       "my server",
		Hostname:   "my-server.example.com",
		FlavorID:   "1x2",
		FirewallID: "LywOkvx5LWAp28NP",
		AccessConfiguration: &AccessConfiguration{
			SshKey:       "Dd8v0nXJ1924rayY",
			Password:     "",
			SavePassword: false,
		},
		Volume: &Volume{
			Source: "image",
			ID:     "lo1qJ9oZb1xGMEgD",
			SsdGb:  10,
		},
		EnablePrivateNetwork:          true,
		EnableStrictAntiDDoSFiltering: false,
		UserData:                      "",
		BackupPreference:              &BackupPreference{},
	}

	err = client.CreateServer(&server)
	if err != nil {
		t.Errorf("getting error calling CreateServer: %s", err)
	}
	assert.Equal(t, "Q7y1OZWlknXmk6l3", server.ID)
	assert.Equal(t, "my server", server.Name)
	assert.Equal(t, "my-server.example.com", server.Hostname)
	assert.Equal(t, "1x2", server.FlavorID)
	assert.Equal(t, "LywOkvx5LWAp28NP", server.FirewallID)
	assert.Equal(t, "Dd8v0nXJ1924rayY", server.AccessConfiguration.SshKeyID)
	assert.Equal(t, "", server.AccessConfiguration.Password)
	assert.Equal(t, false, server.AccessConfiguration.SavePassword)
	assert.Equal(t, "image", server.Volume.Source)
	assert.Equal(t, "lo1qJ9oZb1xGMEgD", server.Volume.ID)
	assert.Equal(t, int64(10), server.Volume.SsdGb)
	assert.Equal(t, true, server.EnablePrivateNetwork)
	assert.Equal(t, false, server.EnableStrictAntiDDoSFiltering)
	assert.Equal(t, "", server.UserData)
	assert.Equal(t, "", server.BackupPreference.Frequency)
	assert.Equal(t, int64(0), server.BackupPreference.Slots)
}

func TestDeleteServer(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte(`
		{
		  "id": "mR2Dn6xgLD9OMPyE",
		  "status": "inProgress",
		  "type": "delete",
		  "startedAt": "2023-01-03T12:00:00.0000000Z",
		  "completedAt": null,
		  "resourceId": "7y1OZWl2ZE9mk6l3",
		  "resourceType": "server"
		}
		`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	action, err := client.DeleteServer("mR2Dn6xgLD9OMPyE")
	if err != nil {
		t.Errorf("getting error calling DeleteServer: %s", err)
	}

	assert.Equal(t, "mR2Dn6xgLD9OMPyE", action.ID)
	assert.Equal(t, "inProgress", action.Status)
	assert.Equal(t, "delete", action.Type)
	assert.Equal(t, "2023-01-03T12:00:00.0000000Z", action.StartedAt)
	assert.Equal(t, "", action.CompletedAt)
	assert.Equal(t, "7y1OZWl2ZE9mk6l3", action.ResourceID)
	assert.Equal(t, "server", action.ResourceType)
}

func TestResizeServer(t *testing.T) {
	t.Parallel()
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("error reading request body: %s", err)
		}
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`
		{
		  "id": "awqYZWO4njxQyOV0",
		  "status": "inProgress",
		  "type": "resize",
		  "startedAt": "2023-01-03T12:00:00.0000000Z",
		  "completedAt": null,
		  "resourceId": "7y1OZWl2ZE9mk6l3",
		  "resourceType": "server"
		}
		`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	action, err := client.ResizeServer("7y1OZWl2ZE9mk6l3", ResizeServerRequest{FlavorID: "2x8", VolumeSizeGb: 50})
	if err != nil {
		t.Errorf("getting error calling ResizeServer: %s", err)
	}

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v1/servers/7y1OZWl2ZE9mk6l3/resize", gotPath)
	assert.JSONEq(t, `{"flavorId":"2x8","volumeSizeGb":50}`, gotBody)

	assert.Equal(t, "awqYZWO4njxQyOV0", action.ID)
	assert.Equal(t, "inProgress", action.Status)
	assert.Equal(t, "resize", action.Type)
	assert.Equal(t, "7y1OZWl2ZE9mk6l3", action.ResourceID)
	assert.Equal(t, "server", action.ResourceType)
}

// Volume only: the omitted flavor must disappear from the body, because the API
// reads an absent field as "leave the flavor alone".
func TestResizeServerOnlyVolume(t *testing.T) {
	t.Parallel()
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("error reading request body: %s", err)
		}
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, err = w.Write([]byte(`{"id":"awqYZWO4njxQyOV0","status":"pending","type":"resize"}`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	_, err = client.ResizeServer("7y1OZWl2ZE9mk6l3", ResizeServerRequest{VolumeSizeGb: 50})
	if err != nil {
		t.Errorf("getting error calling ResizeServer: %s", err)
	}

	assert.JSONEq(t, `{"volumeSizeGb":50}`, gotBody)
}

// A validation 400 (e.g. a size below the current one) must reach the user with
// the API's own detail, not as a generic error.
func TestResizeServerValidationError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`
		{
		  "title": "One or more validation errors occurred.",
		  "status": 400,
		  "detail": "The volume size must be equal or greater than the current.",
		  "errors": {"volumeSizeGb": ["Invalid size"]}
		}
		`))
		if err != nil {
			t.Errorf("error writing response: %s", err)
		}
	}))

	client, err := NewAPI("token123", WithEndpoint(srv.URL))
	if err != nil {
		t.Errorf("getting error creating NewAPI: %s", err)
	}

	_, err = client.ResizeServer("7y1OZWl2ZE9mk6l3", ResizeServerRequest{VolumeSizeGb: 5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "The volume size must be equal or greater than the current.")
	assert.Contains(t, err.Error(), "volumeSizeGb")
}
