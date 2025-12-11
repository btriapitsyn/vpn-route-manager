# VPN Route Manager

A macOS tool that automatically bypasses VPN for specific services like Telegram and YouTube by managing network routes.

## Why bypass VPN for certain services?

Corporate VPNs are designed for secure business access, not high-bandwidth streaming or messaging. Routing non-business services through VPN often causes:
- **Poor performance**: Video buffering, low quality, connection drops
- **VPN overload**: Streaming can consume 3-5 Mbps per user, impacting critical business apps
- **Broken features**: Chromecast, AirPlay, and location-based features often fail through VPN
- **No security benefit**: These services don't contain company data

This tool intelligently routes selected services directly to the internet while keeping business traffic secure through VPN.

## What it does

When you connect to VPN, this tool:
- Detects VPN connection automatically
- Adds bypass routes for configured services (Telegram, YouTube by default)
- Routes these services through your regular internet connection
- Removes routes when VPN disconnects

## Installation

1. Download the installer package
2. Run the installer:
```bash
./installer.sh
```

The installer will prompt for your password once to:
- Install the binary to `/usr/local/bin`
- Setup automatic startup
- Configure passwordless sudo access for route commands only
- Enable Telegram and YouTube bypass by default

## Usage

Check status:
```bash
vpn-route-manager status
```

View logs:
```bash
vpn-route-manager logs -f
```

List services:
```bash
vpn-route-manager service list
```

Enable/disable services:
```bash
vpn-route-manager service enable telegram
vpn-route-manager service disable youtube
```

## Uninstall

```bash
vpn-route-manager uninstall
```

## Requirements

- macOS 10.15 or later
- Admin privileges for installation

## How it works

The tool runs as a background service that monitors your VPN connection every 5 seconds. When it detects a VPN connection (works with GlobalProtect, Cisco AnyConnect, FortiClient, OpenVPN, and other corporate VPNs), it adds specific network routes that bypass the VPN tunnel for configured services.

## Available Services

**Enabled by default:**
- **Telegram**: Messaging app  
- **YouTube**: Video streaming and Google services

**Available but disabled:**
- **WhatsApp**: Messaging service
- **YouTube Music**: Music streaming
- **Spotify**: Music streaming
- **Apple Music**: Apple's music service
- **Facebook**: Social network
- **Instagram**: Photo sharing

To enable additional services:
```bash
vpn-route-manager service enable spotify
vpn-route-manager service enable whatsapp
```

Additional services can be configured by adding JSON files to `~/.vpn-route-manager/config/services/`