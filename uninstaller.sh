#!/bin/bash
# VPN Route Manager Uninstaller
# Removes VPN Route Manager from the system

echo "🗑️  VPN Route Manager Uninstaller"
echo "================================"
echo ""

USERNAME=$(whoami)

echo "This will remove:"
echo "  • VPN Route Manager service"
echo "  • Binary from /usr/local/bin"
echo "  • Configuration and logs"
echo "  • Sudo permissions"
echo "  • Any active bypass routes"
echo ""
read -p "Continue with uninstallation? [Y/n] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]] && [[ ! -z $REPLY ]]; then
    echo "Uninstallation cancelled."
    exit 0
fi

echo ""

# Step 1: Stop and unload LaunchAgent
echo "📋 Stopping service..."
PLIST_PATH="$HOME/Library/LaunchAgents/com.${USERNAME}.vpn.route.manager.plist"
if [[ -f "$PLIST_PATH" ]]; then
    if launchctl list | grep -q "com.${USERNAME}.vpn.route.manager"; then
        launchctl unload "$PLIST_PATH" 2>/dev/null || true
        echo "✅ Service stopped"
    fi
    rm -f "$PLIST_PATH"
    echo "✅ LaunchAgent removed"
else
    echo "⚠️  LaunchAgent not found"
fi

# Step 2: Kill any remaining processes
pkill -f vpn-route-manager 2>/dev/null || true

# Step 3: Remove sudo configuration
echo ""
echo "🔐 Removing sudo permissions..."
SUDO_FILES=(
    "/etc/sudoers.d/vpn-route-manager-$USERNAME"
    "/etc/sudoers.d/vpn-route-bypass-$USERNAME"
)
for file in "${SUDO_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        sudo rm -f "$file"
        echo "✅ Removed: $file"
    fi
done

# Step 4: Remove binary
echo ""
echo "🗑️  Removing binary..."
if [[ -f "/usr/local/bin/vpn-route-manager" ]]; then
    sudo rm -f /usr/local/bin/vpn-route-manager
    echo "✅ Binary removed"
else
    echo "⚠️  Binary not found"
fi

# Step 5: Remove configuration and logs
echo ""
echo "📁 Removing configuration and logs..."
if [[ -d "$HOME/.vpn-route-manager" ]]; then
    rm -rf "$HOME/.vpn-route-manager"
    echo "✅ Configuration removed"
fi

# Remove any other log locations
rm -rf "$HOME/Library/Logs/VPNRouteManager" 2>/dev/null || true
rm -f "$HOME/vpn-route-manager.log"* 2>/dev/null || true

# Step 6: Check for active routes
echo ""
echo "🛣️  Checking for active bypass routes..."

# Use netstat to find all bypass routes pointing to local gateway
GATEWAY_ROUTES=$(netstat -rn | grep -E "^(149\.154|91\.108|185\.76\.151|95\.161\.64|172\.217|142\.250|216\.58|74\.125|64\.233|66\.249|72\.14|209\.85|31\.13|157\.240|173\.252|179\.60|18\.194|34\.|78\.31|193\.182|194\.68|35\.|104\.|17\.|139\.178|144\.178|63\.92|198\.183|65\.199|192\.35|204\.79|45\.64|66\.220|69\.|74\.119|102\.132|103\.4|129\.134|185\.60|204\.15)" | grep "192.168" | awk '{print $1}')

if [[ -n "$GATEWAY_ROUTES" ]]; then
    echo "Found active bypass routes. Removing..."
    while IFS= read -r route; do
        sudo route delete "$route" 2>/dev/null || true
        echo "  ✅ Removed route: $route"
    done <<< "$GATEWAY_ROUTES"
else
    echo "✅ No active bypass routes found"
fi

echo ""
echo "=================="
echo "✅ Uninstallation Complete!"
echo "=================="
echo ""
echo "VPN Route Manager has been completely removed from your system."
echo ""
echo "If you want to reinstall later, run:"
echo "  ./installer.sh"