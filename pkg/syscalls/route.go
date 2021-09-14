package syscalls

import (
	"syscall"
	"unsafe"
)

var (
	modiphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procGetIPForwardTable    = modiphlpapi.NewProc("GetIPForwardTable")
	procCreateIPForwardEntry = modiphlpapi.NewProc("CreateIPForwardEntry")
)

// https://docs.microsoft.com/en-us/windows/win32/api/ipmib/ns-ipmib-mib_ipforwardtable
// typedef struct _MIB_IPFORWARDTABLE {
//  DWORD            dwNumEntries;
//  MIB_IPFORWARDROW table[ANY_SIZE];
// }
type IPForwardTable struct {
	NumEntries uint32
	Table      [1]IPForwardRow
}

// https://docs.microsoft.com/en-us/windows/win32/api/ipmib/ns-ipmib-mib_ipforwardrow
// typedef struct _MIB_IPFORWARDROW {
//  DWORD    dwForwardDest;
//  DWORD    dwForwardMask;
//  DWORD    dwForwardPolicy;
//  DWORD    dwForwardNextHop;
//  IF_INDEX dwForwardIfIndex;
//  union {
//    DWORD              dwForwardType;
//    MIB_IPFORWARD_TYPE ForwardType;
//  };
//  union {
//    DWORD               dwForwardProto;
//    MIB_IPFORWARD_PROTO ForwardProto;
//  };
//  DWORD    dwForwardAge;
//  DWORD    dwForwardNextHopAS;
//  DWORD    dwForwardMetric1;
//  DWORD    dwForwardMetric2;
//  DWORD    dwForwardMetric3;
//  DWORD    dwForwardMetric4;
//  DWORD    dwForwardMetric5;
// }
type IPForwardRow struct {
	ForwardDest      uint32
	ForwardMask      uint32
	ForwardPolicy    uint32
	ForwardNextHop   uint32
	ForwardIfIndex   uint32
	ForwardType      uint32
	ForwardProto     uint32
	ForwardAge       uint32
	ForwardNextHopAS uint32
	ForwardMetric1   uint32
	ForwardMetric2   uint32
	ForwardMetric3   uint32
	ForwardMetric4   uint32
	ForwardMetric5   uint32
}

func GetIPForwardTable(ft *IPForwardTable, size *uint32, order bool) (errcode error) {
	var _p0 uint32
	if order {
		_p0 = 1
	} else {
		_p0 = 0
	}
	r0, _, _ := syscall.Syscall(procGetIPForwardTable.Addr(), 3, uintptr(unsafe.Pointer(ft)), uintptr(unsafe.Pointer(size)), uintptr(_p0))
	if r0 != 0 {
		errcode = syscall.Errno(r0)
	}
	return
}

func CreateIPForwardEntry(fr *IPForwardRow) (errcode error) {
	r0, _, _ := syscall.Syscall(procCreateIPForwardEntry.Addr(), 1, uintptr(unsafe.Pointer(fr)), 0, 0)
	if r0 != 0 {
		errcode = syscall.Errno(r0)
	}
	return
}
