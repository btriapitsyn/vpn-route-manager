#!/bin/bash
# VPN Route Manager Installer
# This script installs a pre-built vpn-route-manager binary

set -e

echo "🚀 VPN Route Manager Installer"
echo "=============================="
echo ""

# Check if running on macOS
if [[ "$(uname)" != "Darwin" ]]; then
    echo "❌ This installer is for macOS only"
    exit 1
fi

# Check if binary exists in same directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY_PATH="$SCRIPT_DIR/vpn-route-manager"

if [[ ! -f "$BINARY_PATH" ]]; then
    echo "❌ Binary not found: $BINARY_PATH"
    echo ""
    echo "Please ensure vpn-route-manager binary is in the same"
    echo "directory as this installer script."
    exit 1
fi

USERNAME=$(whoami)

echo "📋 Installation Summary:"
echo "  • User: $USERNAME"
echo "  • Binary: $BINARY_PATH"
echo "  • Install to: /usr/local/bin/vpn-route-manager"
echo ""
read -p "Continue with installation? [Y/n] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ ! -z $REPLY ]]; then
    echo "Installation cancelled."
    exit 0
fi

# Step 1: Install binary
echo ""
echo "📦 Installing binary..."
sudo mkdir -p /usr/local/bin
sudo cp "$BINARY_PATH" /usr/local/bin/vpn-route-manager
sudo chmod 755 /usr/local/bin/vpn-route-manager
sudo xattr -cr /usr/local/bin/vpn-route-manager

# Verify installation
if ! command -v vpn-route-manager &> /dev/null; then
    echo "❌ Binary installation failed"
    exit 1
fi

echo "✅ Binary installed successfully"

# Step 2: Setup sudo permissions
echo ""
echo "🔐 Setting up sudo permissions..."
echo "This allows the service to manage network routes without password prompts."

SUDO_FILE="/etc/sudoers.d/vpn-route-manager-$USERNAME"
SUDO_CONTENT="$USERNAME ALL=(root) NOPASSWD: /sbin/route"

# Create temporary file
TEMP_FILE="/tmp/vpn-route-sudoers-$$"
echo "$SUDO_CONTENT" > "$TEMP_FILE"

# Validate and install
if visudo -c -f "$TEMP_FILE"; then
    sudo cp "$TEMP_FILE" "$SUDO_FILE"
    sudo chmod 440 "$SUDO_FILE"
    rm -f "$TEMP_FILE"
    echo "✅ Sudo permissions configured"
else
    echo "❌ Invalid sudoers syntax"
    rm -f "$TEMP_FILE"
    exit 1
fi

# Test sudo access
if ! sudo -n route get default >/dev/null 2>&1; then
    echo "⚠️  Sudo configuration may need manual adjustment"
fi

# Step 3: Create directories
echo ""
echo "📁 Creating configuration directories..."
mkdir -p ~/.vpn-route-manager/{config,logs,state}
mkdir -p ~/.vpn-route-manager/config/services

# Step 4: Create default configuration
echo ""
echo "⚙️  Creating default configuration..."
cat > ~/.vpn-route-manager/config/config.json << EOF
{
  "gateway": "auto",
  "check_interval": 5,
  "log_dir": "$HOME/.vpn-route-manager/logs",
  "state_dir": "$HOME/.vpn-route-manager/state",
  "auto_start": true,
  "debug": false,
  "services": {}
}
EOF

# Step 5: Create default service configurations
echo ""
echo "📋 Setting up default services..."

# Telegram service
cat > ~/.vpn-route-manager/config/services/telegram.json << 'EOF'
{
  "telegram": {
    "name": "Telegram",
    "description": "Telegram messaging service",
    "enabled": true,
    "priority": 100,
    "networks": [
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
      "95.161.64.0/20"
    ]
  }
}
EOF

# YouTube service
cat > ~/.vpn-route-manager/config/services/youtube.json << 'EOF'
{
  "youtube": {
    "name": "YouTube",
    "description": "YouTube and Google services",
    "enabled": true,
    "priority": 90,
    "networks": [
      "172.217.0.0/16",
      "142.250.0.0/15",
      "216.58.192.0/19",
      "74.125.0.0/16",
      "64.233.160.0/19",
      "66.249.80.0/20",
      "72.14.192.0/18",
      "209.85.128.0/17"
    ]
  }
}
EOF

# Install additional service configurations
echo ""
echo "📋 Installing additional service configurations..."

# WhatsApp service
cat > ~/.vpn-route-manager/config/services/whatsapp.json << 'EOF'
{
  "whatsapp": {
    "name": "WhatsApp",
    "description": "WhatsApp messaging service",
    "enabled": false,
    "priority": 80,
    "networks": [
      "31.13.64.0/18",
      "31.13.24.0/21",
      "31.13.64.0/19",
      "31.13.96.0/19",
      "157.240.0.0/16",
      "173.252.64.0/18",
      "179.60.192.0/22",
      "18.194.0.0/15",
      "34.224.0.0/12"
    ],
    "domains": [
      "whatsapp.com",
      "whatsapp.net",
      "wa.me"
    ]
  }
}
EOF
echo "  ✅ Installed: whatsapp"

# YouTube Music service
cat > ~/.vpn-route-manager/config/services/youtube-music.json << 'EOF'
{
  "youtube-music": {
    "name": "YouTube Music",
    "description": "YouTube Music streaming service",
    "enabled": false,
    "priority": 85,
    "networks": [
      "172.217.0.0/16",
      "142.250.0.0/15",
      "216.58.192.0/19",
      "74.125.0.0/16",
      "64.233.160.0/19",
      "66.249.80.0/20",
      "72.14.192.0/18",
      "209.85.128.0/17",
      "34.64.0.0/10",
      "35.184.0.0/13"
    ],
    "domains": [
      "music.youtube.com",
      "youtubei.googleapis.com",
      "youtube.com"
    ]
  }
}
EOF
echo "  ✅ Installed: youtube-music"

# Spotify service
cat > ~/.vpn-route-manager/config/services/spotify.json << 'EOF'
{
  "spotify": {
    "name": "Spotify",
    "description": "Spotify music streaming service",
    "enabled": false,
    "priority": 75,
    "networks": [
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
      "35.186.224.0/20"
    ],
    "domains": [
      "spotify.com",
      "spclient.wg.spotify.com",
      "audio-ak-spotify-com.akamaized.net"
    ]
  }
}
EOF
echo "  ✅ Installed: spotify"

# Apple Music service
cat > ~/.vpn-route-manager/config/services/apple-music.json << 'EOF'
{
  "apple-music": {
    "name": "Apple Music",
    "description": "Apple Music streaming service",
    "enabled": false,
    "priority": 70,
    "networks": [
      "17.0.0.0/8",
      "139.178.128.0/17",
      "144.178.0.0/18",
      "63.92.224.0/19",
      "198.183.16.0/20",
      "65.199.22.0/23",
      "192.35.50.0/24",
      "204.79.190.0/24"
    ],
    "domains": [
      "music.apple.com",
      "itunes.apple.com",
      "audio-ssl.itunes.apple.com",
      "streamingaudio.itunes.apple.com"
    ]
  }
}
EOF
echo "  ✅ Installed: apple-music"

# Facebook service
cat > ~/.vpn-route-manager/config/services/facebook.json << 'EOF'
{
  "facebook": {
    "name": "Facebook",
    "description": "Facebook social network",
    "enabled": false,
    "priority": 65,
    "networks": [
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
      "204.15.20.0/22"
    ],
    "domains": [
      "facebook.com",
      "fb.com",
      "fbcdn.net",
      "facebook.net"
    ]
  }
}
EOF
echo "  ✅ Installed: facebook"

# Instagram service
cat > ~/.vpn-route-manager/config/services/instagram.json << 'EOF'
{
  "instagram": {
    "name": "Instagram",
    "description": "Instagram social network",
    "enabled": false,
    "priority": 60,
    "networks": [
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
      "204.15.20.0/22"
    ],
    "domains": [
      "instagram.com",
      "cdninstagram.com",
      "instagramstatic-a.akamaihd.net"
    ]
  }
}
EOF
echo "  ✅ Installed: instagram"

echo "  ✅ Installed 6 additional services"

# Step 6: Create LaunchAgent
echo ""
echo "🎯 Creating LaunchAgent..."
PLIST_PATH="$HOME/Library/LaunchAgents/com.${USERNAME}.vpn.route.manager.plist"
mkdir -p "$HOME/Library/LaunchAgents"

# Remove old plist if exists to ensure fresh config
if [[ -f "$PLIST_PATH" ]]; then
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    rm -f "$PLIST_PATH"
fi

cat > "$PLIST_PATH" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.${USERNAME}.vpn.route.manager</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/vpn-route-manager</string>
        <string>start</string>
        <string>--daemon</string>
    </array>
    
    <key>WorkingDirectory</key>
    <string>/usr/local/bin</string>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <true/>
    
    <key>ProcessType</key>
    <string>Background</string>
    
    <key>StandardOutPath</key>
    <string>$HOME/.vpn-route-manager/logs/stdout.log</string>
    
    <key>StandardErrorPath</key>
    <string>$HOME/.vpn-route-manager/logs/stderr.log</string>
    
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>$HOME</string>
        <key>USER</key>
        <string>$USERNAME</string>
    </dict>
    
    <key>ThrottleInterval</key>
    <integer>10</integer>
    
    <key>ExitTimeOut</key>
    <integer>30</integer>
    
    <key>Nice</key>
    <integer>1</integer>
    
    <key>LowPriorityIO</key>
    <true/>
</dict>
</plist>
EOF

# Step 7: Load LaunchAgent
echo ""
echo "🚀 Starting service..."
launchctl load "$PLIST_PATH" 2>/dev/null || {
    echo "⚠️  Service already loaded, reloading..."
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    launchctl load "$PLIST_PATH"
}

# Wait for service to start
sleep 3

# Step 8: Verify installation
echo ""
echo "=================="
echo "✅ Installation Complete!"
echo "=================="
echo ""

# Check service status
echo "📊 Service Status:"
vpn-route-manager status || {
    echo "⚠️  Service may need a moment to start"
    echo "Try again with: vpn-route-manager status"
}

echo ""
echo "📋 Quick Start Commands:"
echo "  • Check status:    vpn-route-manager status"
echo "  • View logs:       vpn-route-manager logs -f"
echo "  • List services:   vpn-route-manager service list"
echo "  • Enable service:  vpn-route-manager service enable <name>"
echo "  • Disable service: vpn-route-manager service disable <name>"
echo ""
echo "💡 Default enabled services:"
echo "  • Telegram - Messaging app"
echo "  • YouTube - Video streaming"
echo ""
echo "📖 For more help: vpn-route-manager --help"