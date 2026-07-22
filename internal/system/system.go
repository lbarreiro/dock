package system

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"
)

type Stats struct {
    CPU  float64
    RAM  float64
    Temp   float64
    Uptime string
}

func GetStats() (Stats, error) {

    cpu, err := cpuUsage()
    if err != nil {
        return Stats{}, err
    }

    ram, err := ramUsage()
    if err != nil {
        return Stats{}, err
    }

    return Stats{
        CPU:    cpu,
        RAM:    ram,
        Temp:   cpuTemperature(),
        Uptime: uptime(),
    }, nil
}

func cpuUsage() (float64, error) {

    idle1, total1, err := readCPU()
    if err != nil {
        return 0, err
    }

    time.Sleep(200 * time.Millisecond)

    idle2, total2, err := readCPU()
    if err != nil {
        return 0, err
    }

    idle := float64(idle2 - idle1)
    total := float64(total2 - total1)

    if total == 0 {
        return 0, nil
    }

    return (1.0 - idle/total) * 100.0, nil
}

func readCPU() (uint64, uint64, error) {

    data, err := os.ReadFile("/proc/stat")
    if err != nil {
        return 0, 0, err
    }

    fields := strings.Fields(strings.Split(string(data), "\n")[0])

    var total uint64

    for _, f := range fields[1:] {

        v, _ := strconv.ParseUint(f, 10, 64)
        total += v

    }

    idle, _ := strconv.ParseUint(fields[4], 10, 64)

    return idle, total, nil

}

func ramUsage() (float64, error) {

    data, err := os.ReadFile("/proc/meminfo")
    if err != nil {
        return 0, err
    }

    var total uint64
    var available uint64

    for _, line := range strings.Split(string(data), "\n") {

        if strings.HasPrefix(line, "MemTotal:") {

            fields := strings.Fields(line)
            total, _ = strconv.ParseUint(fields[1], 10, 64)

        }

        if strings.HasPrefix(line, "MemAvailable:") {

            fields := strings.Fields(line)
            available, _ = strconv.ParseUint(fields[1], 10, 64)

        }

    }

    if total == 0 {
        return 0, nil
    }

    used := total - available

    return float64(used) / float64(total) * 100.0, nil

}
func cpuTemperature() float64 {

    data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
    if err != nil {
        return 0
    }

    value, err := strconv.Atoi(strings.TrimSpace(string(data)))
    if err != nil {
        return 0
    }

    return float64(value) / 1000
}


func uptime() string {

    data, err := os.ReadFile("/proc/uptime")
    if err != nil {
        return "-"
    }

    fields := strings.Fields(string(data))
    secs, err := strconv.Atoi(strings.Split(fields[0], ".")[0])
    if err != nil {
        return "-"
    }

    switch {
    case secs < 3600:
        return fmt.Sprintf("%dm", secs/60)
    case secs < 86400:
        return fmt.Sprintf("%dh", secs/3600)
    default:
        return fmt.Sprintf("%dd", secs/86400)
    }
}
