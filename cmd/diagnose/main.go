// +build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const (
	TWAIN_DLL = "TWAINDSM.dll"

	MSG_GETFIRST = 0x0004
	MSG_GETNEXT  = 0x0005
	MSG_OPENDSM  = 0x0301
	MSG_CLOSEDSM = 0x0302

	DG_CONTROL   = 0x0001
	DAT_IDENTITY = 0x0003

	TWRC_SUCCESS    = 0
	TWRC_ENDOFLIST  = 7
)

type TW_IDENTITY struct {
	Id              uint32
	Version         [8]uint16
	ProtocolMajor   uint16
	ProtocolMinor   uint16
	SupportedGroups uint32
	Manufacturer    [34]uint16
	ProductFamily   [34]uint16
	ProductName     [34]uint16
}

func utf16ToString(u16 []uint16) string {
	length := 0
	for i, c := range u16 {
		if c == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		length = len(u16)
	}

	runes := make([]rune, length)
	for i := 0; i < length; i++ {
		runes[i] = rune(u16[i])
	}
	return string(runes)
}

func utf16FromString(s string) []uint16 {
	result := make([]uint16, len(s))
	for i, c := range s {
		result[i] = uint16(c)
	}
	return result
}

func main() {
	fmt.Println("===================================")
	fmt.Println("Scanner Detection Diagnostic Tool")
	fmt.Println("===================================\n")

	// Test 1: WIA Detection
	fmt.Println("TEST 1: WIA Scanner Detection")
	fmt.Println("------------------------------")
	testWIA()

	fmt.Println()

	// Test 2: TWAIN Detection
	fmt.Println("TEST 2: TWAIN Scanner Detection")
	fmt.Println("--------------------------------")
	testTWAIN()

	fmt.Println("\n===================================")
	fmt.Println("Diagnostic Complete")
	fmt.Println("===================================")
}

func testWIA() {
	ole.CoInitialize(0)
	defer ole.CoUninitialize()

	// Try WIA 2.0 first
	fmt.Println("Trying WIA 2.0 (Vista+)...")
	deviceMgrRaw, err := oleutil.CreateObject("WIA.DeviceManager")
	if err != nil {
		fmt.Printf("❌ WIA 2.0 not available: %v\n", err)

		// Try WIA 1.0
		fmt.Println("Trying WIA 1.0 (XP)...")
		deviceMgrRaw, err = oleutil.CreateObject("WIA.CommonDialog")
		if err != nil {
			fmt.Printf("❌ WIA 1.0 also not available: %v\n", err)
			return
		}
		fmt.Println("✓ WIA 1.0 available")
	} else {
		fmt.Println("✓ WIA 2.0 available")
	}
	defer deviceMgrRaw.Release()

	deviceMgr, err := deviceMgrRaw.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		fmt.Printf("❌ Failed to get IDispatch: %v\n", err)
		return
	}
	defer deviceMgr.Release()

	// Get device list
	deviceInfosRaw, err := oleutil.GetProperty(deviceMgr, "DeviceInfos")
	if err != nil {
		fmt.Printf("❌ Failed to get DeviceInfos: %v\n", err)
		return
	}
	defer deviceInfosRaw.Clear()

	deviceInfos := deviceInfosRaw.ToIDispatch()
	defer deviceInfos.Release()

	countRaw, err := oleutil.GetProperty(deviceInfos, "Count")
	if err != nil {
		fmt.Printf("❌ Failed to get device count: %v\n", err)
		return
	}
	count := int(countRaw.Val)

	fmt.Printf("Found %d WIA device(s)\n\n", count)

	// List all devices
	for i := 1; i <= count; i++ {
		deviceInfoRaw, err := oleutil.GetProperty(deviceInfos, "Item", i)
		if err != nil {
			fmt.Printf("  Device %d: Error getting info: %v\n", i, err)
			continue
		}

		deviceInfo := deviceInfoRaw.ToIDispatch()

		// Get device properties
		deviceIDRaw, _ := oleutil.GetProperty(deviceInfo, "DeviceID")
		deviceID := deviceIDRaw.ToString()

		propsRaw, err := oleutil.GetProperty(deviceInfo, "Properties")
		if err != nil {
			fmt.Printf("  Device %d: %s (no properties)\n", i, deviceID)
			deviceInfo.Release()
			continue
		}

		props := propsRaw.ToIDispatch()

		// Get device name
		namePropRaw, err := oleutil.GetProperty(props, "Item", "Name")
		deviceName := "Unknown"
		if err == nil {
			nameProp := namePropRaw.ToIDispatch()
			nameValueRaw, err := oleutil.GetProperty(nameProp, "Value")
			if err == nil {
				deviceName = nameValueRaw.ToString()
			}
			nameProp.Release()
		}

		// Get device type
		typePropRaw, err := oleutil.GetProperty(props, "Item", 4) // WIA_DIP_DEV_TYPE = 4
		deviceType := 0
		if err == nil {
			typeProp := typePropRaw.ToIDispatch()
			typeValueRaw, err := oleutil.GetProperty(typeProp, "Value")
			if err == nil {
				deviceType = int(typeValueRaw.Val)
			}
			typeProp.Release()
		}

		props.Release()
		deviceInfo.Release()

		fmt.Printf("  Device %d:\n", i)
		fmt.Printf("    Name: %s\n", deviceName)
		fmt.Printf("    ID: %s\n", deviceID)
		fmt.Printf("    Type: %d (1=Scanner, 2=Camera, 3=Video)\n", deviceType)
		fmt.Println()
	}
}

func testTWAIN() {
	// Try to load TWAIN DSM
	dsmLib, err := syscall.LoadDLL(TWAIN_DLL)
	if err != nil {
		fmt.Printf("❌ TWAIN DSM not found (%s)\n", TWAIN_DLL)
		fmt.Println("   Please install TWAIN drivers from:")
		fmt.Println("   https://www.twain.org/")
		return
	}
	defer dsmLib.Release()

	fmt.Printf("✓ TWAIN DSM found (%s)\n", TWAIN_DLL)

	dsmEntry, err := dsmLib.FindProc("DSM_Entry")
	if err != nil {
		fmt.Printf("❌ DSM_Entry function not found: %v\n", err)
		return
	}

	fmt.Println("✓ DSM_Entry function found")

	// Initialize application identity
	appIdentity := &TW_IDENTITY{
		ProtocolMajor:   2,
		ProtocolMinor:   3,
		SupportedGroups: DG_CONTROL | 0x0002, // DG_IMAGE
	}
	copy(appIdentity.Manufacturer[:], utf16FromString("Scanner Service"))
	copy(appIdentity.ProductFamily[:], utf16FromString("Document Scanner"))
	copy(appIdentity.ProductName[:], utf16FromString("ScanServer"))

	// Open DSM
	fmt.Println("\nOpening TWAIN Data Source Manager...")
	ret, _, _ := dsmEntry.Call(
		uintptr(unsafe.Pointer(appIdentity)),
		0,
		DG_CONTROL,
		DAT_IDENTITY,
		MSG_OPENDSM,
		0,
	)

	if ret != TWRC_SUCCESS {
		fmt.Printf("❌ Failed to open DSM, error code: %d\n", ret)
		return
	}
	fmt.Println("✓ DSM opened successfully")

	// Ensure we close DSM when done
	defer func() {
		dsmEntry.Call(
			uintptr(unsafe.Pointer(appIdentity)),
			0,
			DG_CONTROL,
			DAT_IDENTITY,
			MSG_CLOSEDSM,
			0,
		)
		fmt.Println("✓ DSM closed")
	}()

	// Enumerate data sources
	fmt.Println("\nEnumerating TWAIN data sources...")
	var dsIdentity TW_IDENTITY
	ret, _, _ = dsmEntry.Call(
		uintptr(unsafe.Pointer(appIdentity)),
		0,
		DG_CONTROL,
		DAT_IDENTITY,
		MSG_GETFIRST,
		uintptr(unsafe.Pointer(&dsIdentity)),
	)

	if ret == TWRC_ENDOFLIST {
		fmt.Println("❌ No TWAIN data sources found")
		return
	}

	if ret != TWRC_SUCCESS {
		fmt.Printf("❌ Failed to get first data source, error code: %d\n", ret)
		return
	}

	// List all data sources
	scannerCount := 0
	for {
		scannerCount++

		productName := utf16ToString(dsIdentity.ProductName[:])
		manufacturer := utf16ToString(dsIdentity.Manufacturer[:])
		productFamily := utf16ToString(dsIdentity.ProductFamily[:])

		fmt.Printf("\n  Data Source %d:\n", scannerCount)
		fmt.Printf("    Product: %s\n", productName)
		fmt.Printf("    Manufacturer: %s\n", manufacturer)
		fmt.Printf("    Family: %s\n", productFamily)
		fmt.Printf("    ID: %d\n", dsIdentity.Id)

		// Get next
		ret, _, _ = dsmEntry.Call(
			uintptr(unsafe.Pointer(appIdentity)),
			0,
			DG_CONTROL,
			DAT_IDENTITY,
			MSG_GETNEXT,
			uintptr(unsafe.Pointer(&dsIdentity)),
		)

		if ret == TWRC_ENDOFLIST {
			break
		}

		if ret != TWRC_SUCCESS {
			fmt.Printf("\n⚠ Enumeration ended with error code: %d\n", ret)
			break
		}
	}

	fmt.Printf("\n✓ Found %d TWAIN data source(s)\n", scannerCount)
}
