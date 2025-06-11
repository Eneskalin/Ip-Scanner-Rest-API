package handlers

import (
	"IpScanner/config"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type PingResult struct {
	Ping   string `json:"ip"`
	IsLive bool   `json:"IsLive"`
	MAC    string `json:"mac"`
	Device string `json:"device"`
}

type MacVendor struct {
	MacPrefix  string `json:"macPrefix"`
	VendorName string `json:"vendorName"`
}

var macDB map[string]string

func getLocalMAC(ip string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var currentIP string
			switch v := addr.(type) {
			case *net.IPNet:
				currentIP = v.IP.String()
			case *net.IPAddr:
				currentIP = v.IP.String()
			}
			if currentIP == ip {
				return iface.HardwareAddr.String()
			}
		}
	}
	return ""
}

func init() {
	macDB = make(map[string]string)
	file, err := os.Open("db/mac-vendors-export.json")
	if err == nil {
		defer file.Close()
		var vendors []MacVendor
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&vendors); err == nil {
			for _, v := range vendors {
				macDB[strings.ToUpper(strings.ReplaceAll(v.MacPrefix, "-", ":"))] = v.VendorName
			}
		}
	}
}

func getDeviceNameFromDB(mac string) string {
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ReplaceAll(mac, " ", "")
	macParts := strings.Split(mac, ":")
	if len(macParts) < 3 {
		return "unknown"
	}
	prefix := strings.Join(macParts[:3], ":")
	if name, ok := macDB[prefix]; ok {
		return name
	}
	return "unknown"
}

func SendPing(ip string) PingResult {
	var device string
	count := config.AppConfig.PingCount
	timeout := config.AppConfig.PingTimeoutMs

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", strconv.Itoa(count), "-w", strconv.Itoa(timeout), ip)
	} else {
		cmd = exec.Command("ping", "-c", strconv.Itoa(count), "-W", strconv.Itoa(timeout/1000), ip)
	}

	err := cmd.Run()
	mac := getMACAddress(ip)
	if config.AppConfig.NmapSupport {
		macNmap, deviceNmap := getDeviceNameWithNmap(ip)
		if macNmap != "" {
			mac = macNmap
		}
		if deviceNmap != "" && deviceNmap != "unknown" {
			device = deviceNmap
		} else {
			device = getDeviceNameFromDB(mac)
		}
	} else {
		device = getDeviceNameFromDB(mac)
	}

	return PingResult{
		Ping:   ip,
		IsLive: err == nil,
		MAC:    mac,
		Device: device,
	}
}

func Ping(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		http.Error(w, "Missing 'ip' query parameter", http.StatusBadRequest)
		return
	}

	result := SendPing(ip)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func isMACAddress(s string) bool {
	_, err := net.ParseMAC(s)
	return err == nil
}

func getMACAddress(ip string) string {

	mac := getLocalMAC(ip)
	if mac != "" {
		return mac
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("arp", "-a", ip)
	} else {
		cmd = exec.Command("arp", "-n", ip)
	}

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) {
			fields := strings.Fields(line)
			if runtime.GOOS == "windows" {
				if len(fields) >= 2 && isMACAddress(fields[1]) {
					return fields[1]
				}
			} else {
				if len(fields) >= 3 && isMACAddress(fields[2]) {
					return fields[2]
				}
			}
		}
	}
	return ""
}

func getDeviceNameWithNmap(ip string) (string, string) {
	_, err := exec.LookPath("nmap")
	if err == nil {
		cmd := exec.Command("nmap", "-sn", "-n", "--host-timeout", "2s", ip)
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			var mac, device string
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "MAC Address:") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						rest := strings.TrimSpace(parts[1])
						fields := strings.SplitN(rest, " ", 2)
						mac = fields[0]
						if len(fields) > 1 {
							start := strings.Index(fields[1], "(")
							end := strings.Index(fields[1], ")")
							if start != -1 && end != -1 && end > start {
								device = fields[1][start+1 : end]
							}
						}
					}
				}
			}
			if mac != "" {
				if device == "" {
					device = "unknown"
				}
				return mac, device
			}
		}
	}

	mac := getMACAddress(ip)
	return mac, "unknown"
}
