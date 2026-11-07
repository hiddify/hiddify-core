# Takes out what hiddify-gateway.rsc added, and gives the router its DNS servers and fasttrack back.
#   /import hiddify-gateway-remove.rsc

:local was [/ip firewall address-list find where list=hiddify-dns-was]
:if ([:len $was] > 0) do={
  :local servers ""
  :foreach entry in=$was do={
    :local address [/ip firewall address-list get $entry address]
    :if ($address != "0.0.0.0") do={
      :if ($servers = "") do={ :set servers $address } else={ :set servers ($servers . "," . $address) }
    }
  }
  /ip dns set servers=$servers
}

:foreach rule in=[/ip firewall filter find where action=fasttrack-connection and comment~"^hiddify: fasttrack off; "] do={
  :local comment [/ip firewall filter get $rule comment]
  /ip firewall filter set $rule disabled=no comment=[:pick $comment 24 [:len $comment]]
}

/ip firewall mangle remove [find comment=hiddify]
/ip firewall address-list remove [find comment=hiddify]
/ip route remove [find comment=hiddify]
/routing table remove [find comment=hiddify]

:put "hiddify: removed, the LAN goes out directly"
