package config

type Config struct {
    PingCount     int
    PingTimeoutMs int
    NmapSupport bool
    WithGui bool
}

var AppConfig = Config{
    PingCount:     2,
    PingTimeoutMs: 2000, 
    NmapSupport: false,
    WithGui: true,
}
