# Sends the network of a MikroTik (RouterOS 7) through hiddify-core in router mode, running on
# another Linux box or OpenWrt router, or as a container on this router (docs/router-mode.md).
#
#   :global hiddifyGateway 172.17.0.2
#   /import hiddify-gateway.rsc
#
# hiddifyGateway is the IPv4 address of the box running `hiddify-core run --router`. It must be on
# a subnet of its own (a container's veth, a VLAN, another port), not on the LAN: on the LAN its
# replies reach the devices directly, and RouterOS drops the connections it sees half of.
#
# What it does, all marked with the comment "hiddify" (hiddify-gateway-remove.rsc takes it out):
# - a routing table "hiddify" whose default route is the box; while the box does not answer pings,
#   the LAN goes out directly as before
# - connections from the WAN (to this router, and port forwards) are marked once at their start;
#   their replies keep the main table, so they still work
# - the LAN's other traffic to the internet - TCP, UDP and ICMP - takes the "hiddify" table
# - the router's DNS asks the box, so names are resolved through the proxy, not by the ISP
# - fasttrack is turned off: a fasttracked packet skips the routing mark

:global hiddifyGateway
:if ([:typeof $hiddifyGateway] = "nothing" || [:typeof $hiddifyGateway] = "nil") do={
  :error "set the box's address first: :global hiddifyGateway 172.17.0.2"
}
:local gw [:toip $hiddifyGateway]
:if ([:typeof $gw] != "ip") do={ :error "hiddifyGateway is not an IPv4 address: $hiddifyGateway" }

/routing table
:if ([:len [find name=hiddify]] = 0) do={ add name=hiddify fib comment=hiddify }

/ip route
remove [find comment=hiddify]
add dst-address=0.0.0.0/0 gateway=$gw routing-table=hiddify check-gateway=ping comment=hiddify

/ip firewall address-list
remove [find list=hiddify-direct]
:foreach net in={10.0.0.0/8; 172.16.0.0/12; 192.168.0.0/16; 100.64.0.0/10; 169.254.0.0/16; 224.0.0.0/4; 255.255.255.255/32} do={
  add list=hiddify-direct address=$net comment=hiddify
}

/ip firewall mangle
remove [find comment=hiddify]
add chain=prerouting in-interface-list=WAN connection-state=new action=mark-connection \
    new-connection-mark=hiddify-inbound passthrough=yes comment=hiddify
add chain=prerouting in-interface-list=LAN src-address=("!" . $gw) dst-address-list=!hiddify-direct \
    connection-mark=no-mark action=mark-routing new-routing-mark=hiddify passthrough=yes comment=hiddify

:foreach rule in=[/ip firewall filter find where action=fasttrack-connection and !disabled] do={
  /ip firewall filter set $rule disabled=yes comment=("hiddify: fasttrack off; " . [/ip firewall filter get $rule comment])
}

:local servers [/ip dns get servers]
:if ([:len [/ip firewall address-list find where list=hiddify-dns-was]] = 0) do={
  # 0.0.0.0 stands for no servers of its own (only the dynamic ones), for the removal to give back
  /ip firewall address-list add list=hiddify-dns-was address=0.0.0.0 comment=hiddify
  :foreach s in=$servers do={ /ip firewall address-list add list=hiddify-dns-was address=$s comment=hiddify }
}
/ip dns set servers=$gw

:put "hiddify: the LAN goes through $gw (remove with /import hiddify-gateway-remove.rsc)"
