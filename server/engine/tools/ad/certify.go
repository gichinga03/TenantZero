// Package ad implements Active Directory attack tool simulations for the TenantZero engine.
package ad

import (
	"fmt"
	"strings"
	"time"

	"github.com/0x4D67/tenantzero/server/engine/providers"
	"github.com/0x4D67/tenantzero/shared/schema"
)

// Certify simulates Certify.exe for ADCS enumeration and abuse.
type Certify struct{}

// Name returns the tool identifier used in the registry.
func (c *Certify) Name() string { return "certify" }
func (c *Certify) Aliases() []string { return []string{"Certify.exe"} }

// Requires returns the minimum access conditions.
func (c *Certify) Requires() []string { return nil }

// Execute dispatches a Certify.exe invocation.
// Format: Certify.exe <subcommand> [flags]
func (c *Certify) Execute(
	cmd schema.ParsedCommand,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	if len(cmd.Args) == 0 {
		return schema.ToolResult{Output: certifyBanner()}
	}

	subcommand := strings.ToLower(cmd.Args[0])
	flags := parseCertifyFlags(cmd.Args[1:])

	switch subcommand {
	case "find":
		return c.find(flags, state, provider)
	case "request":
		return c.request(flags, state, provider)
	default:
		return schema.ToolResult{Output: certifyBanner()}
	}
}

// find enumerates certificate templates.
// With --vulnerable, only templates with ESC flags are shown.
func (c *Certify) find(
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	templates := provider.GetCertTemplates()
	domain := providers.GetDomain(provider)

	_, vuln := flags["vulnerable"]

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("   _____          __  .__  _____       \n")
	sb.WriteString("  /  __ \\ _______|  | |__|/ ____\\_ __  \n")
	sb.WriteString(" |  |_|  |/ __ \\  | | |  \\   __\\  |  \\ \n")
	sb.WriteString("  \\  ___/|  ___/|  |_| |  ||  |  |  |  /\n")
	sb.WriteString("   \\___  >\\___  >____/____/|__|  |____/ \n")
	sb.WriteString("       \\/     \\/\n")
	sb.WriteString("                        by @harmj0y and @tifkin_\n\n")
	sb.WriteString("[*] Action: Find certificate templates\n")

	if vuln {
		sb.WriteString("[*] Using the search filter: (!(msPKI-Enrollment-Flag:1.2.840.113556.1.4.804:=2))\n")
	}

	sb.WriteString("\n")

	// CA info block.
	caName := "corp-CA"
	if domain != "" {
		parts := strings.Split(domain, ".")
		caName = strings.ToUpper(parts[0]) + "-CA"
	}

	fmt.Fprintf(&sb, "[*] Enterprise CAs:\n\n")
	fmt.Fprintf(&sb, "    CA Name                               : %s\n", caName)
	fmt.Fprintf(&sb, "    DNS Hostname                          : dc01.%s\n", domain)
	fmt.Fprintf(&sb, "    FullName                              : dc01.%s\\%s\n", domain, caName)
	fmt.Fprintf(&sb, "    Flags                                 : SUPPORTS_NT_AUTHENTICATION, CA_SERVERTYPE_ADVANCED\n")
	fmt.Fprintf(&sb, "    Cert SubjectName                      : CN=%s, DC=%s\n",
		caName, strings.Join(strings.Split(domain, "."), ", DC="))
	sb.WriteString("\n")

	if vuln {
		sb.WriteString("[!] Vulnerable Certificates Templates :\n\n")
	} else {
		fmt.Fprintf(&sb, "[*] Available Certificates Templates :\n\n")
	}

	count := 0
	for _, tmpl := range templates {
		if vuln && !isVulnerableTemplate(tmpl) {
			continue
		}
		count++
		sb.WriteString(formatCertTemplate(tmpl, caName, domain))
		sb.WriteString("\n")
	}

	if count == 0 {
		if vuln {
			sb.WriteString("    [*] No vulnerable certificate templates found.\n")
		} else {
			sb.WriteString("    [*] No certificate templates found.\n")
		}
	}

	return schema.ToolResult{Output: sb.String()}
}

// request submits a certificate request, optionally abusing ESC1 or ESC4.
func (c *Certify) request(
	flags map[string]string,
	state *schema.SessionState,
	provider providers.Provider,
) schema.ToolResult {
	caName := flags["ca"]
	templateName := flags["template"]
	altname := flags["altname"]

	if caName == "" || templateName == "" {
		return schema.ToolResult{
			Output: "[-] Error: /ca:<ca-name> and /template:<template-name> are required\n",
		}
	}

	// Look up template.
	var target *schema.CertTemplate
	for _, tmpl := range provider.GetCertTemplates() {
		t := tmpl
		if strings.EqualFold(t.Name, templateName) {
			target = &t
			break
		}
	}

	if target == nil {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Error: Template '%s' not found\n", templateName),
		}
	}

	// Check whether the request is a valid abuse path.
	canRequest := false
	abusePath := ""

	if target.ESC1 && altname != "" {
		// ESC1: EnrolleeSuppliesSubject — can request cert with arbitrary SAN.
		canRequest = true
		abusePath = "ESC1 (EnrolleeSuppliesSubject + user-supplied SAN)"
	} else if target.ESC4 {
		// ESC4: template ACL is writeable — abuse without altname.
		canRequest = true
		abusePath = "ESC4 (WritableCertificateTemplate)"
	} else if !isVulnerableTemplate(*target) {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Error: Insufficient access rights on template '%s'\n", templateName),
		}
	} else {
		// Template has ESC flags but not ESC1/4 — can still request under subject's own identity.
		canRequest = true
		abusePath = "standard enrollment"
	}

	if !canRequest {
		return schema.ToolResult{
			Output: fmt.Sprintf("[-] Error: Insufficient access rights on template '%s'\n", templateName),
		}
	}

	subjectID := altname
	if subjectID == "" {
		subjectID = state.User
	}

	pfxBlob := deterministicPFX(templateName, subjectID)

	var sb strings.Builder
	sb.WriteString("[*] Action: Request a Certificates\n\n")
	fmt.Fprintf(&sb, "[*] Current user context    : %s\\%s\n",
		providers.NetBIOSName(providers.GetDomain(provider)), state.User)
	fmt.Fprintf(&sb, "[*] No subject name specified, using current context as subject.\n\n")
	fmt.Fprintf(&sb, "[*] Template                : %s\n", target.Name)
	if altname != "" {
		fmt.Fprintf(&sb, "[*] Subject Alternative Name: %s\n", altname)
	}
	fmt.Fprintf(&sb, "[*] Abuse path              : %s\n\n", abusePath)
	fmt.Fprintf(&sb, "[*] Certificate Authority   : %s\n", caName)
	sb.WriteString("[*] CA Response             : The certificate had been issued.\n")
	sb.WriteString("[*] Request ID              : 42\n\n")
	fmt.Fprintf(&sb, "[*] cert.pem               :\n\n")
	sb.WriteString("-----BEGIN CERTIFICATE-----\n")
	sb.WriteString(pfxBlob[:64] + "\n")
	sb.WriteString(pfxBlob[64:128] + "\n")
	sb.WriteString("... (truncated)\n")
	sb.WriteString("-----END CERTIFICATE-----\n\n")
	fmt.Fprintf(&sb, "[*] Convert to PFX         : Certify.exe createpfx /cert:cert.pem /privatekey:key.pem\n")

	effects := []schema.SideEffect{
		{
			Type:  schema.SideEffectSetLoot,
			Key:   "pfx_certs." + subjectID,
			Value: pfxBlob,
		},
	}

	return schema.ToolResult{Output: sb.String(), SideEffects: effects}
}

// isVulnerableTemplate returns true if any ESC flag is set.
func isVulnerableTemplate(t schema.CertTemplate) bool {
	return t.ESC1 || t.ESC2 || t.ESC3 || t.ESC4 || t.ESC5 || t.ESC6 || t.ESC7 || t.ESC8
}

// formatCertTemplate renders a single certificate template block in Certify output format.
func formatCertTemplate(t schema.CertTemplate, caName, domain string) string {
	var sb strings.Builder

	enrollFlag := "0x00000020 (PEND_ALL_REQUESTS)"
	if t.ESC1 {
		enrollFlag = "0x00000040 (ENROLLEE_SUPPLIES_SUBJECT)"
	}

	nameFlag := "SUBJECT_ALT_REQUIRE_UPN, SUBJECT_ALT_REQUIRE_EMAIL"
	if t.ESC1 {
		nameFlag = "ENROLLEE_SUPPLIES_SUBJECT"
	}

	fmt.Fprintf(&sb, "    Template Name                         : %s\n", t.Name)
	fmt.Fprintf(&sb, "    Domain                                : %s\n", domain)
	fmt.Fprintf(&sb, "    CA                                    : %s\n", caName)
	fmt.Fprintf(&sb, "    Schema Version                        : 2\n")
	fmt.Fprintf(&sb, "    Validity Period                       : 1 year\n")
	fmt.Fprintf(&sb, "    Renewal Period                        : 6 weeks\n")
	fmt.Fprintf(&sb, "    msPKI-Certificate-Name-Flag           : %s\n", nameFlag)
	fmt.Fprintf(&sb, "    msPKI-Enrollment-Flag                 : %s\n", enrollFlag)
	fmt.Fprintf(&sb, "    Authorized Signatures Required        : 0\n")
	fmt.Fprintf(&sb, "    pkiextendedkeyusage                   : Client Authentication\n")

	if t.ESC1 {
		fmt.Fprintf(&sb, "    [!] Enrollee Supplies Subject           : TRUE\n")
	}

	sb.WriteString("    Permissions\n")
	sb.WriteString("      Enrollment Permissions\n")
	sb.WriteString("        Enrollment Rights                   : Domain Users\n")

	if t.ESC4 {
		sb.WriteString("      Object Control Permissions\n")
		sb.WriteString("        WriteDacl Principals              : Domain Users\n")
		sb.WriteString("        WriteOwner Principals             : Domain Users\n")
		sb.WriteString("        WriteProperty Principals          : Domain Users\n")
		fmt.Fprintf(&sb, "    [!] Vulnerable Flags                    : WriteProperty (ESC4)\n")
	}

	// Annotate other ESC flags.
	for _, flag := range escFlagAnnotations(t) {
		fmt.Fprintf(&sb, "    [!] %s\n", flag)
	}

	return sb.String()
}

// escFlagAnnotations returns human-readable ESC flag annotations for a template.
func escFlagAnnotations(t schema.CertTemplate) []string {
	var out []string
	if t.ESC2 {
		out = append(out, "ESC2: Any Purpose EKU or no EKU set")
	}
	if t.ESC3 {
		out = append(out, "ESC3: Certificate Request Agent EKU")
	}
	if t.ESC5 {
		out = append(out, "ESC5: PKI object access control misconfiguration")
	}
	if t.ESC6 {
		out = append(out, "ESC6: EDITF_ATTRIBUTESUBJECTALTNAME2 flag on CA")
	}
	if t.ESC7 {
		out = append(out, "ESC7: Vulnerable CA permissions")
	}
	if t.ESC8 {
		out = append(out, "ESC8: NTLM relay to AD CS HTTP endpoints")
	}
	return out
}

// deterministicPFX returns a fake base64 PFX blob seeded from template+subject.
func deterministicPFX(templateName, subject string) string {
	h := fnvNTLM(templateName + ":" + subject)
	// Extend to look plausibly like a PFX blob.
	h2 := fnvNTLM(subject + ":" + templateName)
	h3 := fnvNTLM(templateName + subject + time.Now().Format("20060102"))
	return h + h2 + h3
}

// parseCertifyFlags extracts /key:value and /key (boolean) pairs from Certify args.
func parseCertifyFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, a := range args {
		stripped := strings.TrimPrefix(a, "/")
		stripped = strings.TrimPrefix(stripped, "--")
		if idx := strings.Index(stripped, ":"); idx >= 0 {
			flags[strings.ToLower(stripped[:idx])] = stripped[idx+1:]
		} else {
			flags[strings.ToLower(stripped)] = "true"
		}
	}
	return flags
}

func certifyBanner() string {
	return `
   _____          __  .__  _____
  /  __ \ _______|  | |__|/ ____\_ __
 |  |_|  |/ __ \  | | |  \   __\  |  \
  \  ___/|  ___/|  |_| |  ||  |  |  |  /
   \___  >\___  >____/____/|__|  |____/
       \/     \/
                       by @harmj0y and @tifkin_

Certify.exe  -  enumerate and abuse Active Directory certificate services.

Usage:
  Certify.exe find [/vulnerable]
  Certify.exe request /ca:<ca-name> /template:<template-name> [/altname:<upn>]
`
}