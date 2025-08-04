#!/bin/bash
echo "🤙 🐍 Welcome to $(hostname)"
echo

# System status
uptime_info=$(uptime | awk -F'load average:' '{print $2}' | xargs)
memory_info=$(free | awk 'NR==2{printf "%.0f%%", $3*100/$2}')
echo "📊 Load: $uptime_info | 🧠 Memory: $memory_info"

# Disk usage with warning
disk_usage=$(df / | awk 'NR==2{print $5}' | sed 's/%//')
if [ "$disk_usage" -gt 80 ]; then
    disk_emoji="⚠️💾"
else
    disk_emoji="💾"
fi
echo "$disk_emoji Disk: $(df -h / | awk 'NR==2{print $3"/"$2" ("$5")"}')"

# System Updates (Fedora CoreOS)
echo
echo "📦 System Updates:"
if command -v rpm-ostree &> /dev/null; then
    # Get rpm-ostree status
    ostree_status=$(rpm-ostree status 2>/dev/null)

    # Check for staged updates
    if echo "$ostree_status" | grep -q "update staged"; then
        staged_version=$(echo "$ostree_status" | grep -A1 "update staged" | grep -oP '(?<=: )[0-9.]+' | head -1)
        echo "  🔄 Update staged: $staged_version"

        # Check if reboot is delayed
        if echo "$ostree_status" | grep -q "reboot delayed"; then
            delay_reason=$(echo "$ostree_status" | grep -oP '(?<=reboot delayed ).*' | head -1)
            echo "  ⏸️  Reboot delayed: $delay_reason"
        else
            echo "  🔄 Reboot required to apply updates"
        fi
    # Check for available updates
    elif rpm-ostree upgrade --check 2>&1 | grep -q "AvailableUpdate"; then
        echo "  🆕 Updates available - run 'sudo rpm-ostree upgrade' to apply"
    else
        echo "  ✅ System up to date"
    fi

    # Show Zincati status if present
    if echo "$ostree_status" | grep -q "AutomaticUpdatesDriver: Zincati"; then
        driver_state=$(echo "$ostree_status" | grep -oP '(?<=DriverState: )[^;]+' | head -1)
        echo "  🤖 Auto-updates: $driver_state"
    fi

    # Show current deployment version
    current_version=$(echo "$ostree_status" | grep -A1 "●" | grep "Version:" | awk '{print $2}')
    if [ -n "$current_version" ]; then
        echo "  📍 Current: $current_version"
    fi
else
    echo "  ℹ️  rpm-ostree not found"
fi

# Network info
echo
# Get MAC address of primary network interface
primary_mac=$(ip link show | awk '/state UP/ {getline; if(/link\/ether/) {print $2; exit}}')
if [ -z "$primary_mac" ]; then
    primary_mac=$(ip link show | awk '/link\/ether/ && !/00:00:00:00:00:00/ {print $2; exit}')
fi
mac_display="${primary_mac:-unavailable}"

echo "🌐 Local IPs: $(ip -4 addr show | awk '/inet / && !/127.0.0.1/ {printf "%s ", $2}' | sed 's/ $//')"
external_ip=$(curl -s --max-time 3 ifconfig.me 2>/dev/null || echo "unavailable")
echo "🌍 External IP: $external_ip | 📱 MAC: $mac_display"

# Time and last login
echo "⏰ $(date '+%Y-%m-%d %H:%M:%S %Z')"
last_login=$(last -n 1 $USER | head -1 | awk '{for(i=4;i<=6;i++) printf $i" "; print ""}' | sed 's/ $//')
echo "🔒 Last login: $last_login"

# SSH keys
echo "🔑 $USER keys:"
if [ -f /home/$USER/.ssh/authorized_keys.d/ignition ]; then
    ssh-keygen -lf /home/$USER/.ssh/authorized_keys.d/ignition
fi

# Bootc Container Info
echo
echo "🐳 Bootc Container:"
if [ -f /etc/iago/machine-info ]; then
    machine_name=$(grep "MACHINE_NAME=" /etc/iago/machine-info | cut -d'=' -f2)
    service_name="bootc-${machine_name}.service"
    
    if systemctl list-units --type=service | grep -q "$service_name"; then
        # Service status and uptime
        if systemctl is-active --quiet "$service_name"; then
            status="active (running)"
            uptime_info=$(systemctl show "$service_name" --property=ActiveEnterTimestamp | cut -d'=' -f2)
            if [ -n "$uptime_info" ] && [ "$uptime_info" != "0" ]; then
                uptime_display=$(systemd-analyze timespan --since="$uptime_info" 2>/dev/null | awk '{print $NF}' || echo "unknown")
            else
                uptime_display="unknown"
            fi
        else
            status="inactive"
            uptime_display="stopped"
        fi
        
        # Container information
        container_name="bootc-${machine_name}"
        if podman container exists "$container_name" 2>/dev/null; then
            # Get container image
            container_image=$(podman inspect "$container_name" --format '{{.ImageName}}' 2>/dev/null || echo "unknown")
            
            # Get container creation time and calculate days ago
            container_created=$(podman inspect "$container_name" --format '{{.Created}}' 2>/dev/null)
            if [ -n "$container_created" ]; then
                container_epoch=$(date -d "$container_created" +%s 2>/dev/null || echo "0")
                current_epoch=$(date +%s)
                if [ "$container_epoch" != "0" ] && [ "$container_epoch" -le "$current_epoch" ]; then
                    days_ago=$(( (current_epoch - container_epoch) / 86400 ))
                    formatted_date=$(date -d "$container_created" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
                    
                    if [ $days_ago -eq 0 ]; then
                        last_update="$formatted_date (today)"
                    elif [ $days_ago -eq 1 ]; then
                        last_update="$formatted_date (1 day ago)"
                    else
                        last_update="$formatted_date ($days_ago days ago)"
                    fi
                else
                    last_update="unknown"
                fi
            else
                last_update="unknown"
            fi
            
            # Get restart count
            restart_count=$(systemctl show "$service_name" --property=NRestarts | cut -d'=' -f2 2>/dev/null || echo "0")
            
            # Get health status
            health_status=$(podman healthcheck run "$container_name" >/dev/null 2>&1 && echo "healthy" || echo "unknown")
            
            # Display container info
            echo "  📦 Image: $container_image"
            echo "  🔄 Status: $status | ⏱️ Uptime: $uptime_display"
            echo "  🆕 Last Update: $last_update"
            echo "  🔁 Restarts: $restart_count | ✅ Health: $health_status"
        else
            echo "  📦 Service: $service_name"
            echo "  🔄 Status: $status (no container found)"
        fi
    else
        echo "  ⚠️  No bootc service found for machine: $machine_name"
    fi
else
    echo "  ⚠️  Machine info not found (/etc/iago/machine-info)"
fi
echo
