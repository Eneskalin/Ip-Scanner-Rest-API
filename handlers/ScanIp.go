package handlers

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "sync"
)

func expandCustomRange(input string) ([]string, error) {
    parts := strings.Split(input, "/")
    if len(parts) != 2 {
        return nil, fmt.Errorf("Invalid range format. Use format like 192.168.1.1/24")
    }

    baseIP := parts[0]
    endOctet, err := strconv.Atoi(parts[1])
    if err != nil || endOctet < 0 || endOctet > 255 {
        return nil, fmt.Errorf("Invalid end range")
    }

    octets := strings.Split(baseIP, ".")
    if len(octets) != 4 {
        return nil, fmt.Errorf("Invalid base IP")
    }

    startOctet, err := strconv.Atoi(octets[3])
    if err != nil {
        return nil, fmt.Errorf("Invalid start IP")
    }

    if endOctet < startOctet {
        return nil, fmt.Errorf("End of range must be >= start")
    }

    basePrefix := strings.Join(octets[:3], ".")
    var ips []string
    for i := startOctet; i <= endOctet; i++ {
        ips = append(ips, fmt.Sprintf("%s.%d", basePrefix, i))
    }

    return ips, nil
}





func ScanIp(w http.ResponseWriter, r *http.Request) {
    ipRange := r.URL.Query().Get("ip")
    if ipRange == "" {
        http.Error(w, "Missing 'ip' query parameter", http.StatusBadRequest)
        return
    }

    ips, err := expandCustomRange(ipRange)
    if err != nil {
        http.Error(w, "Invalid IP range format. Use format like 192.168.1.1/24", http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    w.Write([]byte("[")) 
    flusher.Flush()

    ipChan := make(chan string)
    resultChan := make(chan []byte)
    const workerCount = 20
    var wg sync.WaitGroup

  
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for ip := range ipChan {
                result := SendPing(ip) 
                raw, _ := json.Marshal(result)
                var m map[string]interface{}
                json.Unmarshal(raw, &m)

                mac, ok := m["mac"].(string)
                if !ok || mac == "" || mac =="ff-ff-ff-ff-ff-ff" {
                    continue
                }

                m["ip"] = ip
                resJSON, _ := json.Marshal(m)
                resultChan <- resJSON
            }
        }()
    }

    
    go func() {
        for _, ip := range ips {
            ipChan <- ip
        }
        close(ipChan)
        wg.Wait()
        close(resultChan)
    }()

    
    first := true
    for resJSON := range resultChan {
        if !first {
            w.Write([]byte(","))
        }
        first = false
        w.Write(resJSON)
        flusher.Flush()
    }

    w.Write([]byte("]")) 
}
