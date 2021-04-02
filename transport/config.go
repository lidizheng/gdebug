package transport

import (
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

type SecurityType int

const (
	TypeInsecure SecurityType = iota
	TypeTls
)

const GrpcdebugServerConfigEnvName = "GRPCDEBUG_CONFIG"

func (e SecurityType) String() string {
	switch e {
	case TypeInsecure:
		return "Insecure"
	case TypeTls:
		return "TLS"
	default:
		return fmt.Sprintf("%d", int(e))
	}
}

type ServerConfig struct {
	// Only allow two wildcard * and ?
	Pattern string
	// If present, override the given target address
	RealAddress        string
	Security           SecurityType
	IdentityFile       string
	ServerNameOverride string
}

func parseServerPattern(x string) (string, error) {
	var matcher = regexp.MustCompile(`^Server\s+?([A-Za-z0-9-_\.\*\?:]*)$`)
	tokens := matcher.FindStringSubmatch(x)
	if len(tokens) != 2 {
		return "", fmt.Errorf("Invalid server pattern: %v", x)
	}
	return strings.TrimSpace(tokens[1]), nil
}

func parseServerOption(x string) (string, string, error) {
	var matcher = regexp.MustCompile(`^(\w+?)\s+?(\S*)$`)
	tokens := matcher.FindStringSubmatch(x)
	if len(tokens) != 3 {
		return "", "", fmt.Errorf("Invalid server option: %v", x)
	}
	return strings.TrimSpace(tokens[1]), strings.TrimSpace(tokens[2]), nil
}

func LoadServerConfigsFromFile(path string) []ServerConfig {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(bytes), "\n")
	var configs []ServerConfig
	var current *ServerConfig
	for i, line := range lines {
		if strings.HasPrefix(line, "Server") {
			pattern, err := parseServerPattern(line)
			if err != nil {
				log.Fatalf("Failed to parse config [%v:%d]: %v", path, i, err)
			}
			configs = append(configs, ServerConfig{Pattern: pattern})
			current = &configs[len(configs)-1]
		} else {
			stem := strings.TrimSpace(line)
			if stem == "" {
				// Allow black lines, skip them
				continue
			}
			key, value, err := parseServerOption(stem)
			if err != nil {
				log.Fatalf("Failed to parse config [%v:%d]: %v", path, i, err)
			}
			switch key {
			case "RealAddress":
				current.RealAddress = value
			case "Security":
				switch strings.ToLower(value) {
				case "insecure":
					current.Security = TypeInsecure
				case "tls":
					current.Security = TypeTls
				default:
					log.Fatalf("Unsupported security model: %v", value)
				}
			case "IdentityFile":
				current.IdentityFile = value
			case "ServerNameOverride":
				current.ServerNameOverride = value
			}
		}
	}
	log.Printf("Loaded server configs from %v", path)
	return configs
}

func LoadServerConfigs() []ServerConfig {
	if value := os.Getenv(GrpcdebugServerConfigEnvName); value != "" {
		return LoadServerConfigsFromFile(value)
	}
	// Try to load from work directory, if exists
	if _, err := os.Stat("./grpcdebug_config"); err == nil {
		return LoadServerConfigsFromFile("./grpcdebug_config")
	}
	// Try to load from user config directory, if exists
	dir, _ := os.UserConfigDir()
	defaultUserConfig := path.Join(dir, "grpcdebug_config")
	if _, err := os.Stat(defaultUserConfig); err == nil {
		return LoadServerConfigsFromFile(defaultUserConfig)
	}
	return nil
}

func GetServerConfig(target string) ServerConfig {
	for _, config := range LoadServerConfigs() {
		if config.Pattern == target {
			return config
		}
	}
	return ServerConfig{}
}
