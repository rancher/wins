package apis

import (
	"context"

	"github.com/Microsoft/hcsshim"
	"github.com/Microsoft/hcsshim/hcn"
	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type hnsService struct {
}

func (s *hnsService) GetNetwork(_ context.Context, req *types.HnsGetNetworkRequest) (resp *types.HnsGetNetworkResponse, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	var hnsNetwork *types.HnsNetwork

	// get network
	switch opts := req.GetOptions().(type) {
	case *types.HnsGetNetworkRequest_Name:
		if isV2Api() {
			network, err := hcn.GetNetworkByName(opts.Name)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "could not get HNS network %s via v2 api: %v", opts.Name, err)
			}
			hnsNetwork = v2nativeToHnsNetwork(network)
		} else {
			network, err := hcsshim.GetHNSNetworkByName(opts.Name)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "could not get HNS network %s via v1 api: %v", opts.Name, err)
			}
			hnsNetwork = v1nativeToHnsNetwork(network)
		}
	case *types.HnsGetNetworkRequest_Address:
		if isV2Api() {
			network, err := v2getHNSNetworkByAddress(opts.Address)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "could not get HNS network %s via v2 api: %v", opts.Address, err)
			}
			hnsNetwork = v2nativeToHnsNetwork(network)
		} else {
			network, err := v1getHNSNetworkByAddress(opts.Address)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "could not get HNS network %s via v1 api: %v", opts.Address, err)
			}
			hnsNetwork = v1nativeToHnsNetwork(network)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "indicate the HNS network name or address")
	}

	// construct response
	return &types.HnsGetNetworkResponse{
		Data: hnsNetwork,
	}, nil
}

func isV2Api() bool {
	return hcn.V2ApiSupported() == nil
}

func v1getHNSNetworkByAddress(address string) (*hcsshim.HNSNetwork, error) {
	hnsNetworks, err := hcsshim.HNSListNetworkRequest("GET", "", "")
	if err != nil {
		return nil, err
	}
	for _, hnsNetwork := range hnsNetworks {
		for _, nativeSubnet := range hnsNetwork.Subnets {
			if nativeSubnet.AddressPrefix == address {
				return &hnsNetwork, nil
			}
		}
	}
	return nil, hcsshim.NetworkNotFoundError{NetworkName: address}
}

func v2getHNSNetworkByAddress(address string) (*hcn.HostComputeNetwork, error) {
	hnsNetworks, err := hcn.ListNetworks()
	if err != nil {
		return nil, err
	}
	for _, hnsNetwork := range hnsNetworks {
		for _, ipam := range hnsNetwork.Ipams {
			for _, nativeSubnet := range ipam.Subnets {
				if nativeSubnet.IpAddressPrefix == address {
					return &hnsNetwork, nil
				}
			}
		}
	}
	return nil, hcsshim.NetworkNotFoundError{NetworkName: address}
}

func v1nativeToHnsNetwork(nativeData *hcsshim.HNSNetwork) *types.HnsNetwork {
	var subnets []*types.HnsNetworkSubnet
	for _, nativeSubnet := range nativeData.Subnets {
		subnets = append(subnets, &types.HnsNetworkSubnet{
			AddressCIDR:    nativeSubnet.AddressPrefix,
			GatewayAddress: nativeSubnet.GatewayAddress,
		})
	}

	return &types.HnsNetwork{
		ID:           nativeData.Id,
		Type:         nativeData.Type,
		Subnets:      subnets,
		ManagementIP: nativeData.ManagementIP,
	}
}

func v2nativeToHnsNetwork(nativeData *hcn.HostComputeNetwork) *types.HnsNetwork {
	var subnets []*types.HnsNetworkSubnet
	for _, ipam := range nativeData.Ipams {
		for _, nativeSubnet := range ipam.Subnets {
			subnets = append(subnets, &types.HnsNetworkSubnet{
				AddressCIDR:    nativeSubnet.IpAddressPrefix,
				GatewayAddress: nativeSubnet.Routes[0].NextHop,
			})
		}
	}

	var managementIP string
	for _, policy := range nativeData.Policies {
		if policy.Type == hcn.ProviderAddress {
			managementIP = converters.GetStringFormJSON(policy.Settings, "ProviderAddress")
		}
	}

	return &types.HnsNetwork{
		ID:           nativeData.Id,
		Type:         string(nativeData.Type),
		Subnets:      subnets,
		ManagementIP: managementIP,
	}
}
