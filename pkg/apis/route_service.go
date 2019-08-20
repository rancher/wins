package apis

import (
	"context"
	"fmt"
	"net"
	"unsafe"

	"github.com/rancher/wins/pkg/converters"
	"github.com/rancher/wins/pkg/panics"
	"github.com/rancher/wins/pkg/syscalls"
	"github.com/rancher/wins/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type routeService struct {
}

func (s *routeService) Add(ctx context.Context, req *types.RouteAddRequest) (resp *types.Void, respErr error) {
	defer panics.DealWith(func(recoverObj interface{}) {
		respErr = status.Errorf(codes.Unknown, "panic %v", recoverObj)
	})

	var addrIPNs []*net.IPNet
	for _, addr := range req.GetAddresses() {
		_, ipn, err := net.ParseCIDR(addr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not recognize address %s: %v", addr, err)
		}

		addrIPNs = append(addrIPNs, ipn)
	}

	// find out how big our buffer needs to be
	b := make([]byte, 1)
	ft := (*syscalls.IpForwardTable)(unsafe.Pointer(&b[0]))
	ol := uint32(0)
	syscalls.GetIpForwardTable(ft, &ol, false)

	// start to get table
	b = make([]byte, ol)
	ft = (*syscalls.IpForwardTable)(unsafe.Pointer(&b[0]))
	if err := syscalls.GetIpForwardTable(ft, &ol, false); err != nil {
		return nil, status.Errorf(codes.Internal, "could not get IP table: %v", err)
	}

	// iterate to find
	for i := 0; i < int(ft.NumEntries); i++ {
		row := *(*syscalls.IpForwardRow)(unsafe.Pointer(
			uintptr(unsafe.Pointer(&ft.Table[0])) + uintptr(i)*uintptr(unsafe.Sizeof(ft.Table[0])), // head idx + offset
		))

		if converters.Inet_ntoa(row.ForwardDest, false) != "0.0.0.0" {
			continue
		}

		for _, addrIPN := range addrIPNs {
			ip, mask := ipnetToString(addrIPN)
			// clone route configuration
			row.ForwardDest = converters.Inet_aton(ip, false)
			row.ForwardMask = converters.Inet_aton(mask, false)
			err := syscalls.CreateIpForwardEntry(&row)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "could not create IP forward entry: %v", err)
			}
		}

		// construct response
		return &types.Void{}, nil
	}

	return nil, status.Errorf(codes.Internal, "there isn't a default gateway with a destination of 0.0.0.0")
}

func ipnetToString(ipNet *net.IPNet) (addr string, mask string) {
	a, m := ipNet.IP, ipNet.Mask
	return a.String(), fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}
