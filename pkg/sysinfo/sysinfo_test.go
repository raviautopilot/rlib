package sysinfo

import (
	"runtime"
	"testing"
)

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo failed: %v", err)
	}

	t.Logf("OS: %s", info.OS)
	t.Logf("OS Version: %s", info.OSVersion)
	t.Logf("Total Memory: %d bytes", info.TotalMemory)
	t.Logf("Available Memory: %d bytes", info.AvailableMemory)
	t.Logf("Networks detected: %d", len(info.Networks))
	t.Logf("USB Devices detected: %d", len(info.USBDevices))

	// Verify standard fields are populated
	if info.OS == "" {
		t.Error("Expected OS field to be non-empty")
	}

	// We expect runtime.GOOS to match our field
	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS to be %q, got %q", runtime.GOOS, info.OS)
	}

	if info.OSVersion == "" {
		t.Error("Expected OSVersion to be non-empty")
	}

	// Check memory. On supported platforms, it should be greater than 0
	if info.OS == "linux" || info.OS == "windows" || info.OS == "darwin" {
		if info.TotalMemory == 0 {
			t.Error("Expected TotalMemory to be greater than 0")
		}
		t.Logf("Available Memory percentage: %.2f%%", float64(info.AvailableMemory)/float64(info.TotalMemory)*100)
	}

	// We should always find at least loopback network interface
	if len(info.Networks) == 0 {
		t.Error("Expected at least one network interface")
	}

	for _, net := range info.Networks {
		t.Logf("  - Network: %s (MAC: %s, IPs: %v, Flags: %s)", net.Name, net.MACAddress, net.IPAddresses, net.Flags)
	}

	// Output details of USB devices found
	for _, usb := range info.USBDevices {
		t.Logf("  - USB Device: %s (Vendor: %s, Product: %s, Manufacturer: %s, Serial: %s)",
			usb.Name, usb.VendorID, usb.ProductID, usb.Manufacturer, usb.SerialNumber)
	}
}
