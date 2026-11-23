package ad

import (
	"github.com/0x4D67/tenantzero/shared/schema"
)

// SharpHound simulates SharpHound AD enumeration output.
func SharpHound(cmd string, state *schema.SessionState) string {
	return "[*] SharpHound 2.0.0\n[*] Collection complete. Output saved to BloodHound.zip"
}
