package apis

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/syscalls"
	"github.com/rancher/wins/pkg/types"
	"golang.org/x/sys/windows"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type networkService struct {
}

func (s *networkService) Get(ctx context.Context, req *types.NetworkGetRequest) (resp *types.NetworkGetResponse, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	name := req.GetName()
	addr := req.GetAddress()
	index := -1
	if name == "" && addr == "" {
		ifIdx, err := getDefaultAdapterIndex()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not get default adapter index: %v", err)
		}

		index = ifIdx
	}

	// find out how big our buffer needs to be
	b := make([]byte, 1)
	ai := (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	ol := uint32(0)
	syscall.GetAdaptersInfo(ai, &ol)

	// start to get info
	b = make([]byte, 1)
	ai = (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	if err := syscall.GetAdaptersInfo(ai, &ol); err != nil {
		return nil, status.Errorf(codes.Internal, "could not call system GetAdaptersInfo: %v", err)
	}

	// iterate to find
	for ; ai != nil; ai = ai.Next {
		if ai.Type != windows.IF_TYPE_ETHERNET_CSMACD {
			continue
		}

		aiDescription := converters.UnsafeUTF16BytesToString(ai.Description[:])
		aiIndex := int(ai.Index)

		var aiAddress, aiMask string
		for ipl := &ai.IpAddressList; ipl != nil; ipl = ipl.Next {
			aiAddress = converters.UnsafeUTF16BytesToString(ipl.IpAddress.String[:])
			aiMask = converters.UnsafeUTF16BytesToString(ipl.IpMask.String[:])
			if aiAddress != "" && aiMask != "" {
				break
			}
		}

		var aiGatewayAddress string
		for gwl := &ai.GatewayList; gwl != nil; gwl = gwl.Next {
			aiGatewayAddress = converters.UnsafeUTF16BytesToString(gwl.IpAddress.String[:])
			if aiGatewayAddress != "" {
				break
			}
		}

		if addr == aiAddress || name == aiDescription || index == aiIndex {
			hostname, err := os.Hostname()
			if err != nil {
				return nil, status.Errorf(codes.Internal, "could not get system hostname: %v", err)
			}

			return &types.NetworkGetResponse{
				Data: nativeToNetworkAdatper(aiIndex, aiGatewayAddress, aiAddress, aiMask, hostname),
			}, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "could not get adapter")
}

func nativeToNetworkAdatper(idx int, gw string, address string, mask string, hn string) *types.NetworkAdapter {
	addressIPNet := &net.IPNet{
		IP:   net.ParseIP(address),
		Mask: net.IPv4Mask(0xff, 0xff, 0xff, 0xff),
	}

	subnetAddressIPNet := &net.IPNet{
		IP:   net.ParseIP(address),
		Mask: make(net.IPMask, net.IPv4len),
	}
	for i, mask := range strings.SplitN(mask, ".", 4) {
		aInt, _ := strconv.Atoi(mask)
		aIntByte := byte(aInt)
		subnetAddressIPNet.IP[12+i] &= aIntByte
		subnetAddressIPNet.Mask[i] = aIntByte
	}

	return &types.NetworkAdapter{
		InterfaceIndex: strconv.Itoa(idx),
		GatewayAddress: gw,
		HostName:       hn,
		AddressCIDR:    addressIPNet.String(),
		SubnetCIDR:     subnetAddressIPNet.String(),
	}
}

func getDefaultAdapterIndex() (int, error) {
	// find out how big our buffer needs to be
	b := make([]byte, 1)
	ft := (*syscalls.IpForwardTable)(unsafe.Pointer(&b[0]))
	ol := uint32(0)
	syscalls.GetIpForwardTable(ft, &ol, false)

	// start to get table
	b = make([]byte, ol)
	ft = (*syscalls.IpForwardTable)(unsafe.Pointer(&b[0]))
	if err := syscalls.GetIpForwardTable(ft, &ol, false); err != nil {
		return -1, err
	}

	// iterate to find
	for i := 0; i < int(ft.NumEntries); i++ {
		row := *(*syscalls.IpForwardRow)(unsafe.Pointer(
			uintptr(unsafe.Pointer(&ft.Table[0])) + uintptr(i)*uintptr(unsafe.Sizeof(ft.Table[0])), // head idx + offset
		))

		if converters.Inet_ntoa(row.ForwardDest, false) != "0.0.0.0" {
			continue
		}

		return int(row.ForwardIfIndex), nil
	}

	return -1, errors.New("there isn't a default gateway with a destination of 0.0.0.0")
}
