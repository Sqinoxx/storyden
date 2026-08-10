#!/usr/bin/env bash
# Oeffnet HTTP/HTTPS auf einer frischen Oracle Cloud VM.
#
# Oracle-Images (Ubuntu) blockieren eingehenden Traffic auf ZWEI Ebenen:
#   1) Die VCN Security List / Network Security Group im Oracle-Dashboard
#      (muss manuell im Browser freigegeben werden, siehe README)
#   2) iptables auf der VM selbst (macht dieses Skript)
#
# Ausfuehren auf der VM: sudo bash oracle-firewall.sh
set -euo pipefail

for port in 80 443; do
  iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null \
    || iptables -I INPUT 5 -m state --state NEW -p tcp --dport "$port" -j ACCEPT
done

# HTTP/3 laeuft ueber UDP 443.
iptables -C INPUT -p udp --dport 443 -j ACCEPT 2>/dev/null \
  || iptables -I INPUT 5 -m state --state NEW -p udp --dport 443 -j ACCEPT

netfilter-persistent save

echo "Ports 80/443 (TCP + UDP) freigegeben und Regeln persistiert."
iptables -L INPUT -n --line-numbers | grep -E "80|443|Chain INPUT"
