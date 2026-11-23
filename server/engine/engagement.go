package engine

import (
	"fmt"
	"io"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/0x4D67/tenantzero/shared/schema"
)

const (
	pageMargin    = 15.0
	headerFontSz  = 18.0
	subHeaderSz   = 10.0
	sectionFontSz = 12.0
	bodyFontSz    = 10.0
	footerFontSz  = 8.0
	rowHeight     = 7.0
	sectionPad    = 4.0
)

// Generate builds a PDF engagement letter from the given Engagement and writes
// it to w. The document follows a fixed A4 layout:
//
//   - TenantZero letterhead (bold 18pt) + CONFIDENTIAL subtitle (red 10pt)
//   - Client block: name, industry, engagement type, mode
//   - Scope table: two-column In Scope / Out of Scope, zebra-striped
//   - Rules of engagement: numbered list
//   - Contacts table: Name | Role | Email
//   - Engagement window
//   - Footer: fictional disclaimer in italic grey 8pt
//
// External dependency: github.com/jung-kurt/gofpdf
func Generate(eng *schema.Engagement, w io.Writer) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pageMargin, pageMargin, pageMargin)
	pdf.AddPage()

	addHeader(pdf)
	addClientBlock(pdf, eng)
	addScopeTable(pdf, eng)
	addRulesOfEngagement(pdf, eng)
	addContactsTable(pdf, eng)
	addEngagementWindow(pdf, eng)
	addFooter(pdf)

	return pdf.Output(w)
}

// addHeader renders the TenantZero letterhead and CONFIDENTIAL subtitle.
func addHeader(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "B", headerFontSz)
	pdf.CellFormat(0, 10, "TenantZero Consulting", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", subHeaderSz)
	pdf.SetTextColor(180, 0, 0)
	pdf.CellFormat(0, 6, "CONFIDENTIAL — ENGAGEMENT LETTER", "", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	pdf.SetDrawColor(100, 100, 100)
	pdf.Line(pageMargin, pdf.GetY(), 210-pageMargin, pdf.GetY())
	pdf.Ln(4)
}

// addClientBlock renders client name, industry, engagement type and mode.
func addClientBlock(pdf *gofpdf.Fpdf, eng *schema.Engagement) {
	addSectionHeader(pdf, "Client Information")

	rows := [][]string{
		{"Client", eng.Client.Name},
		{"Industry", eng.Client.Industry},
		{"Engagement Type", "pentest"},
		{"Mode", eng.Mode},
	}
	for _, r := range rows {
		pdf.SetFont("Helvetica", "B", bodyFontSz)
		pdf.CellFormat(50, rowHeight, r[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", bodyFontSz)
		pdf.CellFormat(0, rowHeight, r[1], "", 1, "L", false, 0, "")
	}
	pdf.Ln(sectionPad)
}

// addScopeTable renders a two-column scope table with zebra striping.
func addScopeTable(pdf *gofpdf.Fpdf, eng *schema.Engagement) {
	addSectionHeader(pdf, "Scope")

	colW := (210 - 2*pageMargin) / 2

	// Column headers
	pdf.SetFont("Helvetica", "B", bodyFontSz)
	pdf.SetFillColor(220, 220, 220)
	pdf.CellFormat(colW, rowHeight, "In Scope", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colW, rowHeight, "Out of Scope", "1", 1, "C", true, 0, "")

	maxRows := len(eng.Scope.InScope)
	if len(eng.Scope.OutOfScope) > maxRows {
		maxRows = len(eng.Scope.OutOfScope)
	}

	for i := 0; i < maxRows; i++ {
		fill := i%2 == 0
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.SetFont("Helvetica", "", bodyFontSz)

		inCell := ""
		if i < len(eng.Scope.InScope) {
			inCell = eng.Scope.InScope[i]
		}
		outCell := ""
		if i < len(eng.Scope.OutOfScope) {
			outCell = eng.Scope.OutOfScope[i]
		}

		pdf.CellFormat(colW, rowHeight, "  "+inCell, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(colW, rowHeight, "  "+outCell, "1", 1, "L", fill, 0, "")
	}
	pdf.Ln(sectionPad)
}

// addRulesOfEngagement renders a numbered list of RoE items.
func addRulesOfEngagement(pdf *gofpdf.Fpdf, eng *schema.Engagement) {
	addSectionHeader(pdf, "Rules of Engagement")
	pdf.SetFont("Helvetica", "", bodyFontSz)
	for i, rule := range eng.RulesOfEngagement.Rules {
		pdf.MultiCell(0, rowHeight,
			fmt.Sprintf("%d. %s", i+1, rule),
			"", "L", false,
		)
	}
	pdf.Ln(sectionPad)
}

// addContactsTable renders a three-column Name | Role | Email contacts table.
func addContactsTable(pdf *gofpdf.Fpdf, eng *schema.Engagement) {
	addSectionHeader(pdf, "Contacts")

	pageW := 210 - 2*pageMargin
	colW := [3]float64{pageW * 0.3, pageW * 0.3, pageW * 0.4}

	pdf.SetFont("Helvetica", "B", bodyFontSz)
	pdf.SetFillColor(220, 220, 220)
	headers := [3]string{"Name", "Role", "Email"}
	for i, h := range headers {
		pdf.CellFormat(colW[i], rowHeight, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", bodyFontSz)
	for j, c := range []string{} {
		fill := j%2 == 0
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(colW[0], rowHeight, "  "+c, "1", 0, "L", fill, 0, "")
		pdf.CellFormat(colW[1], rowHeight, "  N/A", "1", 0, "L", fill, 0, "")
		pdf.CellFormat(colW[2], rowHeight, "  N/A", "1", 1, "L", fill, 0, "")
	}
	pdf.Ln(sectionPad)
}

// addEngagementWindow renders the engagement start and end dates.
func addEngagementWindow(pdf *gofpdf.Fpdf, eng *schema.Engagement) {
	addSectionHeader(pdf, "Engagement Window")
	pdf.SetFont("Helvetica", "", bodyFontSz)
	pdf.CellFormat(0, rowHeight,
		fmt.Sprintf("%s — %s", "N/A", "N/A"),
		"", 1, "L", false, 0, "")
	pdf.Ln(sectionPad)
}

// addFooter writes the fictional disclaimer in italic grey 8pt at the current
// cursor position (not an absolute page footer — the document is not paginated).
func addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetFont("Helvetica", "I", footerFontSz)
	pdf.SetTextColor(130, 130, 130)
	pdf.Line(pageMargin, pdf.GetY(), 210-pageMargin, pdf.GetY())
	pdf.Ln(2)
	pdf.CellFormat(0, 5,
		"This document is fictional and generated for training purposes only.",
		"", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

// addSectionHeader renders a bold 12pt section heading with a light grey
// background cell spanning the full content width.
func addSectionHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", sectionFontSz)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(0, 8, "  "+strings.ToUpper(title), "", 1, "L", true, 0, "")
	pdf.Ln(2)
}