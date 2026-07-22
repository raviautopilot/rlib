package sysinfo

import (
	"bufio"
	"encoding/xml"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo represents details about the running OS and hardware.
type SystemInfo struct {
	OS              string          `json:"os"`               // e.g. "linux", "windows", "darwin"
	OSVersion       string          `json:"os_version"`       // e.g. "Ubuntu 22.04"
	AvailableMemory uint64          `json:"available_memory"` // in bytes
	TotalMemory     uint64          `json:"total_memory"`     // in bytes
	Networks        []NetworkInfo   `json:"networks"`
	USBDevices      []USBDeviceInfo `json:"usb_devices"`
}

// NetworkInfo represents a system network interface.
type NetworkInfo struct {
	Name        string   `json:"name"`
	MACAddress  string   `json:"mac_address"`
	IPAddresses []string `json:"ip_addresses"`
	Flags       string   `json:"flags"`
}

// USBDeviceInfo represents a system USB device.
type USBDeviceInfo struct {
	Name         string `json:"name"`
	VendorID     string `json:"vendor_id"`
	ProductID    string `json:"product_id"`
	Manufacturer string `json:"manufacturer"`
	SerialNumber string `json:"serial_number"`
}

// GetSystemInfo retrieves system information using pure Go APIs and standard files.
// It avoids calling OS commands to ensure compatibility and efficiency.
func GetSystemInfo() (*SystemInfo, error) {
	networks, err := GetNetworkInterfaces()
	if err != nil {
		networks = []NetworkInfo{}
	}

	info := &SystemInfo{
		OS:       runtime.GOOS,
		Networks: networks,
	}

	// Fetch OS-specific values at runtime
	switch runtime.GOOS {
	case "linux":
		totalMem, availMem, err := getLinuxMemory()
		if err == nil {
			info.TotalMemory = totalMem
			info.AvailableMemory = availMem
		}

		osVer, err := getLinuxOSVersion()
		if err == nil {
			info.OSVersion = osVer
		} else {
			info.OSVersion = "Linux"
		}

		usbDevs, err := getLinuxUSBDevices()
		if err == nil {
			info.USBDevices = usbDevs
		} else {
			info.USBDevices = []USBDeviceInfo{}
		}

	case "darwin":
		osVer, err := getDarwinOSVersion()
		if err == nil {
			info.OSVersion = osVer
		} else {
			info.OSVersion = "macOS"
		}
		// Darwin memory/USBs are not easily accessible via standard files, we fallback
		info.TotalMemory = 0
		info.AvailableMemory = 0
		info.USBDevices = []USBDeviceInfo{}

	default:
		// Fallbacks for Windows or other operating systems
		info.OSVersion = runtime.GOOS
		info.TotalMemory = 0
		info.AvailableMemory = 0
		info.USBDevices = []USBDeviceInfo{}
	}

	return info, nil
}

// GetNetworkInterfaces returns details of network interfaces.
func GetNetworkInterfaces() ([]NetworkInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var networks []NetworkInfo
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ipAddrs []string
		for _, addr := range addrs {
			ipAddrs = append(ipAddrs, addr.String())
		}

		networks = append(networks, NetworkInfo{
			Name:        iface.Name,
			MACAddress:  iface.HardwareAddr.String(),
			IPAddresses: ipAddrs,
			Flags:       iface.Flags.String(),
		})
	}
	return networks, nil
}

// Linux-specific helper functions using standard Go code

func getLinuxMemory() (uint64, uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	var totalMem, availMem uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		// values are in kB, convert to bytes
		valBytes := val * 1024
		if key == "MemTotal" {
			totalMem = valBytes
		} else if key == "MemAvailable" {
			availMem = valBytes
		}
	}
	return totalMem, availMem, nil
}

func getLinuxOSVersion() (string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		file, err = os.Open("/usr/lib/os-release")
		if err != nil {
			return "Linux", err
		}
	}
	defer file.Close()

	var name, version string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "PRETTY_NAME" {
			return val, nil
		}
		if key == "NAME" {
			name = val
		} else if key == "VERSION" {
			version = val
		}
	}
	if name != "" {
		if version != "" {
			return name + " " + version, nil
		}
		return name, nil
	}
	return "Linux", nil
}

func getLinuxUSBDevices() ([]USBDeviceInfo, error) {
	const usbDir = "/sys/bus/usb/devices"
	entries, err := os.ReadDir(usbDir)
	if err != nil {
		return nil, err
	}

	var devices []USBDeviceInfo
	for _, entry := range entries {
		path := filepath.Join(usbDir, entry.Name())

		// Read idVendor and idProduct
		vendorBytes, err := os.ReadFile(filepath.Join(path, "idVendor"))
		if err != nil {
			continue
		}
		productIDBytes, err := os.ReadFile(filepath.Join(path, "idProduct"))
		if err != nil {
			continue
		}

		vendorID := strings.TrimSpace(string(vendorBytes))
		productID := strings.TrimSpace(string(productIDBytes))

		var name, manufacturer, serial string
		if b, err := os.ReadFile(filepath.Join(path, "product")); err == nil {
			name = strings.TrimSpace(string(b))
		}
		if b, err := os.ReadFile(filepath.Join(path, "manufacturer")); err == nil {
			manufacturer = strings.TrimSpace(string(b))
		}
		if b, err := os.ReadFile(filepath.Join(path, "serial")); err == nil {
			serial = strings.TrimSpace(string(b))
		}

		devices = append(devices, USBDeviceInfo{
			Name:         name,
			VendorID:     vendorID,
			ProductID:    productID,
			Manufacturer: manufacturer,
			SerialNumber: serial,
		})
	}
	return devices, nil
}

// Darwin-specific helper functions using standard Go code

type plist struct {
	Dict dict `xml:"dict"`
}

type dict struct {
	Keys   []string `xml:"key"`
	Values []string `xml:"string"`
}

func getDarwinOSVersion() (string, error) {
	file, err := os.Open("/System/Library/CoreServices/SystemVersion.plist")
	if err != nil {
		return "macOS", err
	}
	defer file.Close()

	var p plist
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&p); err != nil {
		return "macOS", err
	}

	var prodName, prodVersion string
	for i, key := range p.Dict.Keys {
		if i >= len(p.Dict.Values) {
			break
		}
		if key == "ProductName" {
			prodName = p.Dict.Values[i]
		} else if key == "ProductVersion" {
			prodVersion = p.Dict.Values[i]
		}
	}

	if prodName != "" && prodVersion != "" {
		return prodName + " " + prodVersion, nil
	}
	if prodName != "" {
		return prodName, nil
	}
	return "macOS", nil
}
