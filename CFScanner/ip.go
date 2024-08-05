package CFScanner

import (
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

func InitRandSeed() {
	rand.Seed(time.Now().UnixNano())
}

func isIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

func randIPEndWith(num byte) byte {
	if num == 0 { // For /32, which is a single IP
		return byte(0)
	}
	return byte(rand.Intn(int(num)))
}

type IPRanges struct {
	ips     []*net.IPAddr
	mask    string
	firstIP net.IP
	ipNet   *net.IPNet
}

func newIPRanges() *IPRanges {
	return &IPRanges{
		ips: make([]*net.IPAddr, 0),
	}
}

// If it's a single IP, add the subnet mask; otherwise, get the subnet mask (r.mask)
func (r *IPRanges) fixIP(ip string) string {
	// If it doesn't contain '/', it's not an IP range but a single IP, so we need to add /32 or /128 subnet mask
	if i := strings.IndexByte(ip, '/'); i < 0 {
		if isIPv4(ip) {
			r.mask = "/32"
		} else {
			r.mask = "/128"
		}
		ip += r.mask
	} else {
		r.mask = ip[i:]
	}
	return ip
}

// Parse the IP range and get the IP, IP range, and subnet mask
func (r *IPRanges) parseCIDR(ip string) {
	var err error
	if r.firstIP, r.ipNet, err = net.ParseCIDR(r.fixIP(ip)); err != nil {
		log.Fatalln("ParseCIDR err", err)
	}
}

func (r *IPRanges) appendIPv4(d byte) {
	r.appendIP(net.IPv4(r.firstIP[12], r.firstIP[13], r.firstIP[14], d))
}

func (r *IPRanges) appendIP(ip net.IP) {
	r.ips = append(r.ips, &net.IPAddr{IP: ip})
}

// Return the minimum value and the number of available IPs in the fourth segment of the IP
func (r *IPRanges) getIPRange() (minIP, hosts byte) {
	minIP = r.firstIP[15] & r.ipNet.Mask[3] // Minimum value of the fourth segment of the IP

	// Get the number of hosts based on the subnet mask
	m := net.IPv4Mask(255, 255, 255, 255)
	for i, v := range r.ipNet.Mask {
		m[i] ^= v
	}
	total, _ := strconv.ParseInt(m.String(), 16, 32) // Total number of available IPs
	if total > 255 {                                 // Adjust the number of available IPs in the fourth segment
		hosts = 255
		return
	}
	hosts = byte(total)
	return
}

func (r *IPRanges) chooseIPv4() {
	if r.mask == "/32" { // For a single IP, no need to randomize, just add itself
		r.appendIP(r.firstIP)
	} else {
		minIP, hosts := r.getIPRange()    // Get the minimum value and the number of available IPs in the fourth segment of the IP
		for r.ipNet.Contains(r.firstIP) { // As long as the IP is within the IP range, continue looping and randomizing
			if TestAll { // If testing all IPs
				for i := 0; i <= int(hosts); i++ { // Iterate from the minimum value to the maximum value of the last segment of the IP
					r.appendIPv4(byte(i) + minIP)
				}
			} else { // Randomize the last segment of the IP as 0.0.0.X
				r.appendIPv4(minIP + randIPEndWith(hosts))
			}
			r.firstIP[14]++ // 0.0.(X+1).X
			if r.firstIP[14] == 0 {
				r.firstIP[13]++ // 0.(X+1).X.X
				if r.firstIP[13] == 0 {
					r.firstIP[12]++ // (X+1).X.X.X
				}
			}
		}
	}
}

func (r *IPRanges) chooseIPv6() {
	if r.mask == "/128" { // For a single IP, no need to randomize, just add itself
		r.appendIP(r.firstIP)
	} else {
		var tempIP uint8                  // Temporary variable to store the value of the previous segment
		for r.ipNet.Contains(r.firstIP) { // As long as the IP is within the IP range, continue looping and randomizing
			r.firstIP[15] = randIPEndWith(255) // Randomize the last segment of the IP
			r.firstIP[14] = randIPEndWith(255) // Randomize the last segment of the IP

			targetIP := make([]byte, len(r.firstIP))
			copy(targetIP, r.firstIP)
			r.appendIP(targetIP) // Add the IP to the IP pool

			for i := 13; i >= 0; i-- { // Randomize from the third-to-last segment
				tempIP = r.firstIP[i]              // Store the value of the previous segment
				r.firstIP[i] += randIPEndWith(255) // Randomize 0~255 and add it to the current segment
				if r.firstIP[i] >= tempIP {        // If the value of the current segment is greater than or equal to the value of the previous segment, it means the randomization was successful, so we can exit the loop
					break
				}
			}
		}
	}
}

func loadIPRanges(sample bool, num int) []*net.IPAddr {
	ranges := newIPRanges()
	if IPText != "" { // Get IP range data from the parameters
		IPs := strings.Split(IPText, ",") // Split by comma and iterate through the array
		for _, IP := range IPs {
			IP = strings.TrimSpace(IP) // Remove leading and trailing whitespace characters (spaces, tabs, newlines, etc.)
			if IP == "" {              // Skip empty values (e.g., consecutive commas)
				continue
			}
			ranges.parseCIDR(IP) // Parse the IP range and get the IP, IP range, and subnet mask
			if isIPv4(IP) {      // Generate all IPv4/IPv6 addresses to be tested (single/random/all)
				ranges.chooseIPv4()
			} else {
				ranges.chooseIPv6()
			}
		}
	} else { // Get IP range data from a file
		return nil
	}
	if sample {
		return RandomSelect(ranges.ips, num)
	} else {
		return ranges.ips
	}
}
