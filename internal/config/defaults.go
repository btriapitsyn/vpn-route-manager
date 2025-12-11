package config

import (
	"os"
	"path/filepath"
)

// GetDefaultConfig returns the default configuration
func GetDefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		Gateway:       "auto",
		CheckInterval: 5,
		LogDir:        filepath.Join(homeDir, ".vpn-route-manager", "logs"),
		StateDir:      filepath.Join(homeDir, ".vpn-route-manager", "state"),
		Services:      make(map[string]*Service),
		AutoStart:     true,
		Debug:         false,
	}
}

// GetDefaultServiceConfigs returns built-in service configurations
func GetDefaultServiceConfigs() map[string]*Service {
	return map[string]*Service{
		"telegram": {
			Name:        "Telegram",
			Description: "Telegram messaging service",
			Enabled:     true,
			Priority:    100,
			Networks: []string{
				"149.154.160.0/20",
				"149.154.164.0/22",
				"149.154.168.0/22",
				"149.154.172.0/22",
				"91.108.4.0/22",
				"91.108.8.0/22",
				"91.108.12.0/22",
				"91.108.16.0/22",
				"91.108.56.0/22",
				"185.76.151.0/24",
				"95.161.64.0/20",
			},
			Domains: []string{
				"telegram.org",
				"web.telegram.org",
				"api.telegram.org",
			},
		},
		"youtube": {
			Name:        "YouTube",
			Description: "YouTube and Google services",
			Enabled:     true,
			Priority:    90,
			Networks: []string{
				"172.217.0.0/16",
				"142.250.0.0/15",
				"216.58.192.0/19",
				"74.125.0.0/16",
				"64.233.160.0/19",
				"66.249.80.0/20",
				"72.14.192.0/18",
				"209.85.128.0/17",
			},
			Domains: []string{
				"youtube.com",
				"googlevideo.com",
				"google.com",
			},
		},
		"whatsapp": {
			Name:        "WhatsApp",
			Description: "WhatsApp messaging service",
			Enabled:     false,
			Priority:    80,
			Networks: []string{
				"31.13.64.0/18",
				"31.13.24.0/21",
				"31.13.64.0/19",
				"31.13.96.0/19",
				"157.240.0.0/16",
				"173.252.64.0/18",
				"179.60.192.0/22",
				"18.194.0.0/15",
				"34.224.0.0/12",
			},
			Domains: []string{
				"whatsapp.com",
				"whatsapp.net",
				"wa.me",
			},
		},
		"spotify": {
			Name:        "Spotify",
			Description: "Spotify music streaming service",
			Enabled:     false,
			Priority:    70,
			Networks: []string{
				"78.31.8.0/21",
				"193.182.8.0/21",
				"194.68.28.0/22",
				"34.64.0.0/10",
				"35.184.0.0/13",
				"35.192.0.0/14",
				"35.196.0.0/15",
				"104.154.0.0/15",
				"104.196.0.0/14",
				"104.199.64.0/18",
				"35.186.224.0/20",
			},
			Domains: []string{
				"spotify.com",
				"spclient.wg.spotify.com",
				"audio-ak-spotify-com.akamaized.net",
			},
		},
		"apple-music": {
			Name:        "Apple Music",
			Description: "Apple Music streaming service",
			Enabled:     false,
			Priority:    70,
			Networks: []string{
				"17.0.0.0/8",
				"139.178.128.0/17",
				"144.178.0.0/18",
				"63.92.224.0/19",
				"198.183.16.0/20",
				"65.199.22.0/23",
				"192.35.50.0/24",
				"204.79.190.0/24",
			},
			Domains: []string{
				"music.apple.com",
				"itunes.apple.com",
				"audio-ssl.itunes.apple.com",
				"streamingaudio.itunes.apple.com",
			},
		},
		"facebook": {
			Name:        "Facebook",
			Description: "Facebook social network",
			Enabled:     false,
			Priority:    60,
			Networks: []string{
				"31.13.24.0/21",
				"31.13.64.0/18",
				"45.64.40.0/22",
				"66.220.0.0/16",
				"69.63.176.0/20",
				"69.171.0.0/16",
				"74.119.76.0/22",
				"102.132.96.0/20",
				"103.4.96.0/22",
				"129.134.0.0/16",
				"157.240.0.0/16",
				"173.252.64.0/18",
				"179.60.192.0/22",
				"185.60.216.0/22",
				"204.15.20.0/22",
			},
			Domains: []string{
				"facebook.com",
				"fb.com",
				"fbcdn.net",
				"facebook.net",
			},
		},
		"instagram": {
			Name:        "Instagram",
			Description: "Instagram social network",
			Enabled:     false,
			Priority:    60,
			Networks: []string{
				"31.13.24.0/21",
				"31.13.64.0/18",
				"45.64.40.0/22",
				"66.220.0.0/16",
				"69.63.176.0/20",
				"69.171.0.0/16",
				"74.119.76.0/22",
				"102.132.96.0/20",
				"103.4.96.0/22",
				"129.134.0.0/16",
				"157.240.0.0/16",
				"173.252.64.0/18",
				"179.60.192.0/22",
				"185.60.216.0/22",
				"204.15.20.0/22",
			},
			Domains: []string{
				"instagram.com",
				"cdninstagram.com",
				"instagramstatic-a.akamaihd.net",
			},
		},
		"youtube-music": {
			Name:        "YouTube Music",
			Description: "YouTube Music streaming service",
			Enabled:     false,
			Priority:    70,
			Networks: []string{
				"172.217.0.0/16",
				"142.250.0.0/15",
				"216.58.192.0/19",
				"74.125.0.0/16",
				"64.233.160.0/19",
				"66.249.80.0/20",
				"72.14.192.0/18",
				"209.85.128.0/17",
				"34.64.0.0/10",
				"35.184.0.0/13",
			},
			Domains: []string{
				"music.youtube.com",
				"youtubei.googleapis.com",
				"youtube.com",
			},
		},
	}
}