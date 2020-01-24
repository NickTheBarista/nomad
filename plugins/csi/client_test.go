package csi

import (
	"context"
	"fmt"
	"testing"

	csipbv1 "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/protobuf/ptypes/wrappers"
	fake "github.com/hashicorp/nomad/plugins/csi/testing"
	"github.com/stretchr/testify/require"
)

func newTestClient() (*fake.IdentityClient, *fake.ControllerClient, *fake.NodeClient, CSIPlugin) {
	ic := fake.NewIdentityClient()
	cc := fake.NewControllerClient()
	nc := fake.NewNodeClient()
	client := &client{
		identityClient:   ic,
		controllerClient: cc,
		nodeClient:       nc,
	}

	return ic, cc, nc, client
}

func TestClient_RPC_PluginProbe(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		ProbeResponse    *csipbv1.ProbeResponse
		ExpectedResponse bool
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name: "returns false for ready when the provider returns false",
			ProbeResponse: &csipbv1.ProbeResponse{
				Ready: &wrappers.BoolValue{Value: false},
			},
			ExpectedResponse: false,
		},
		{
			Name: "returns true for ready when the provider returns true",
			ProbeResponse: &csipbv1.ProbeResponse{
				Ready: &wrappers.BoolValue{Value: true},
			},
			ExpectedResponse: true,
		},
		{
			/* When a SP does not return a ready value, a CO MAY treat this as ready.
			   We do so because example plugins rely on this behaviour. We may
				 re-evaluate this decision in the future. */
			Name: "returns true for ready when the provider returns a nil wrapper",
			ProbeResponse: &csipbv1.ProbeResponse{
				Ready: nil,
			},
			ExpectedResponse: true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ic, _, _, client := newTestClient()
			defer client.Close()

			ic.NextErr = c.ResponseErr
			ic.NextPluginProbe = c.ProbeResponse

			resp, err := client.PluginProbe(context.TODO())
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			}

			require.Equal(t, c.ExpectedResponse, resp)
		})
	}

}

func TestClient_RPC_PluginInfo(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		InfoResponse     *csipbv1.GetPluginInfoResponse
		ExpectedResponse string
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name: "returns an error if we receive an empty `name`",
			InfoResponse: &csipbv1.GetPluginInfoResponse{
				Name: "",
			},
			ExpectedErr: fmt.Errorf("PluginGetInfo: plugin returned empty name field"),
		},
		{
			Name: "returns the name when successfully retrieved and not empty",
			InfoResponse: &csipbv1.GetPluginInfoResponse{
				Name: "com.hashicorp.storage",
			},
			ExpectedResponse: "com.hashicorp.storage",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ic, _, _, client := newTestClient()
			defer client.Close()

			ic.NextErr = c.ResponseErr
			ic.NextPluginInfo = c.InfoResponse

			resp, err := client.PluginGetInfo(context.TODO())
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			}

			require.Equal(t, c.ExpectedResponse, resp)
		})
	}

}

func TestClient_RPC_PluginGetCapabilities(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		Response         *csipbv1.GetPluginCapabilitiesResponse
		ExpectedResponse *PluginCapabilitySet
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name: "HasControllerService is true when it's part of the response",
			Response: &csipbv1.GetPluginCapabilitiesResponse{
				Capabilities: []*csipbv1.PluginCapability{
					{
						Type: &csipbv1.PluginCapability_Service_{
							Service: &csipbv1.PluginCapability_Service{
								Type: csipbv1.PluginCapability_Service_CONTROLLER_SERVICE,
							},
						},
					},
				},
			},
			ExpectedResponse: &PluginCapabilitySet{hasControllerService: true},
		},
		{
			Name: "HasTopologies is true when it's part of the response",
			Response: &csipbv1.GetPluginCapabilitiesResponse{
				Capabilities: []*csipbv1.PluginCapability{
					{
						Type: &csipbv1.PluginCapability_Service_{
							Service: &csipbv1.PluginCapability_Service{
								Type: csipbv1.PluginCapability_Service_VOLUME_ACCESSIBILITY_CONSTRAINTS,
							},
						},
					},
				},
			},
			ExpectedResponse: &PluginCapabilitySet{hasTopologies: true},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ic, _, _, client := newTestClient()
			defer client.Close()

			ic.NextErr = c.ResponseErr
			ic.NextPluginCapabilities = c.Response

			resp, err := client.PluginGetCapabilities(context.TODO())
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			}

			require.Equal(t, c.ExpectedResponse, resp)
		})
	}
}

func TestClient_RPC_ControllerGetCapabilities(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		Response         *csipbv1.ControllerGetCapabilitiesResponse
		ExpectedResponse *ControllerCapabilitySet
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name: "ignores unknown capabilities",
			Response: &csipbv1.ControllerGetCapabilitiesResponse{
				Capabilities: []*csipbv1.ControllerServiceCapability{
					{
						Type: &csipbv1.ControllerServiceCapability_Rpc{
							Rpc: &csipbv1.ControllerServiceCapability_RPC{
								Type: csipbv1.ControllerServiceCapability_RPC_GET_CAPACITY,
							},
						},
					},
				},
			},
			ExpectedResponse: &ControllerCapabilitySet{},
		},
		{
			Name: "detects list volumes capabilities",
			Response: &csipbv1.ControllerGetCapabilitiesResponse{
				Capabilities: []*csipbv1.ControllerServiceCapability{
					{
						Type: &csipbv1.ControllerServiceCapability_Rpc{
							Rpc: &csipbv1.ControllerServiceCapability_RPC{
								Type: csipbv1.ControllerServiceCapability_RPC_LIST_VOLUMES,
							},
						},
					},
					{
						Type: &csipbv1.ControllerServiceCapability_Rpc{
							Rpc: &csipbv1.ControllerServiceCapability_RPC{
								Type: csipbv1.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
							},
						},
					},
				},
			},
			ExpectedResponse: &ControllerCapabilitySet{
				HasListVolumes:               true,
				HasListVolumesPublishedNodes: true,
			},
		},
		{
			Name: "detects publish capabilities",
			Response: &csipbv1.ControllerGetCapabilitiesResponse{
				Capabilities: []*csipbv1.ControllerServiceCapability{
					{
						Type: &csipbv1.ControllerServiceCapability_Rpc{
							Rpc: &csipbv1.ControllerServiceCapability_RPC{
								Type: csipbv1.ControllerServiceCapability_RPC_PUBLISH_READONLY,
							},
						},
					},
					{
						Type: &csipbv1.ControllerServiceCapability_Rpc{
							Rpc: &csipbv1.ControllerServiceCapability_RPC{
								Type: csipbv1.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
							},
						},
					},
				},
			},
			ExpectedResponse: &ControllerCapabilitySet{
				HasPublishUnpublishVolume: true,
				HasPublishReadonly:        true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, cc, _, client := newTestClient()
			defer client.Close()

			cc.NextErr = tc.ResponseErr
			cc.NextCapabilitiesResponse = tc.Response

			resp, err := client.ControllerGetCapabilities(context.TODO())
			if tc.ExpectedErr != nil {
				require.Error(t, tc.ExpectedErr, err)
			}

			require.Equal(t, tc.ExpectedResponse, resp)
		})
	}
}

func TestClient_RPC_NodeGetCapabilities(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		Response         *csipbv1.NodeGetCapabilitiesResponse
		ExpectedResponse *NodeCapabilitySet
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name: "ignores unknown capabilities",
			Response: &csipbv1.NodeGetCapabilitiesResponse{
				Capabilities: []*csipbv1.NodeServiceCapability{
					{
						Type: &csipbv1.NodeServiceCapability_Rpc{
							Rpc: &csipbv1.NodeServiceCapability_RPC{
								Type: csipbv1.NodeServiceCapability_RPC_EXPAND_VOLUME,
							},
						},
					},
				},
			},
			ExpectedResponse: &NodeCapabilitySet{},
		},
		{
			Name: "detects stage volumes capability",
			Response: &csipbv1.NodeGetCapabilitiesResponse{
				Capabilities: []*csipbv1.NodeServiceCapability{
					{
						Type: &csipbv1.NodeServiceCapability_Rpc{
							Rpc: &csipbv1.NodeServiceCapability_RPC{
								Type: csipbv1.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
							},
						},
					},
				},
			},
			ExpectedResponse: &NodeCapabilitySet{
				HasStageUnstageVolume: true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			_, _, nc, client := newTestClient()
			defer client.Close()

			nc.NextErr = tc.ResponseErr
			nc.NextCapabilitiesResponse = tc.Response

			resp, err := client.NodeGetCapabilities(context.TODO())
			if tc.ExpectedErr != nil {
				require.Error(t, tc.ExpectedErr, err)
			}

			require.Equal(t, tc.ExpectedResponse, resp)
		})
	}
}

func TestClient_RPC_ControllerPublishVolume(t *testing.T) {
	cases := []struct {
		Name             string
		ResponseErr      error
		Response         *csipbv1.ControllerPublishVolumeResponse
		ExpectedResponse *ControllerPublishVolumeResponse
		ExpectedErr      error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name:             "Handles PublishContext == nil",
			Response:         &csipbv1.ControllerPublishVolumeResponse{},
			ExpectedResponse: &ControllerPublishVolumeResponse{},
		},
		{
			Name: "Handles PublishContext != nil",
			Response: &csipbv1.ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					"com.hashicorp/nomad-node-id": "foobar",
					"com.plugin/device":           "/dev/sdc1",
				},
			},
			ExpectedResponse: &ControllerPublishVolumeResponse{
				PublishContext: map[string]string{
					"com.hashicorp/nomad-node-id": "foobar",
					"com.plugin/device":           "/dev/sdc1",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, cc, _, client := newTestClient()
			defer client.Close()

			cc.NextErr = c.ResponseErr
			cc.NextPublishVolumeResponse = c.Response

			resp, err := client.ControllerPublishVolume(context.TODO(), &ControllerPublishVolumeRequest{})
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			}

			require.Equal(t, c.ExpectedResponse, resp)
		})
	}
}

func TestClient_RPC_NodeStageVolume(t *testing.T) {
	cases := []struct {
		Name        string
		ResponseErr error
		Response    *csipbv1.NodeStageVolumeResponse
		ExpectedErr error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name:        "handles success",
			ResponseErr: nil,
			ExpectedErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, _, nc, client := newTestClient()
			defer client.Close()

			nc.NextErr = c.ResponseErr
			nc.NextStageVolumeResponse = c.Response

			err := client.NodeStageVolume(context.TODO(), "foo", nil, "/foo", &VolumeCapability{})
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestClient_RPC_NodeUnstageVolume(t *testing.T) {
	cases := []struct {
		Name        string
		ResponseErr error
		Response    *csipbv1.NodeUnstageVolumeResponse
		ExpectedErr error
	}{
		{
			Name:        "handles underlying grpc errors",
			ResponseErr: fmt.Errorf("some grpc error"),
			ExpectedErr: fmt.Errorf("some grpc error"),
		},
		{
			Name:        "handles success",
			ResponseErr: nil,
			ExpectedErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, _, nc, client := newTestClient()
			defer client.Close()

			nc.NextErr = c.ResponseErr
			nc.NextUnstageVolumeResponse = c.Response

			err := client.NodeUnstageVolume(context.TODO(), "foo", "/foo")
			if c.ExpectedErr != nil {
				require.Error(t, c.ExpectedErr, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
