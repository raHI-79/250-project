package sitecontent

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func encoded(t *testing.T, content Content) []byte {
	t.Helper()
	raw, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("could not encode the content: %v", err)
	}
	return raw
}

func withField(t *testing.T, content Content, path []string, key string, value any) []byte {
	t.Helper()
	var loose map[string]any
	if err := json.Unmarshal(encoded(t, content), &loose); err != nil {
		t.Fatalf("could not decode the content: %v", err)
	}
	target := loose
	for _, step := range path {
		target = target[step].(map[string]any)
	}
	target[key] = value
	raw, err := json.Marshal(loose)
	if err != nil {
		t.Fatalf("could not encode the content: %v", err)
	}
	return raw
}

func TestTheBuiltInContentIsValid(t *testing.T) {
	content, err := Parse(encoded(t, Default()))
	if err != nil {
		t.Fatalf("the built-in content was refused: %v", err)
	}
	if content.Header.SiteName != "SUST CSE" || len(content.Home.Slides) != 4 {
		t.Fatalf("the built-in content changed while being checked: %+v", content.Header)
	}
	if len(content.Home.Sections) != 4 {
		t.Fatalf("the home order has %d sections, want the 4 built-in ones", len(content.Home.Sections))
	}
}

func TestTheDeveloperCreditCannotBeChangedOrHidden(t *testing.T) {
	base := Default()
	attempts := map[string][]byte{
		"a credit field in the footer":    withField(t, base, []string{"footer"}, "credit", "Made by somebody else"),
		"a switch that hides the credit":  withField(t, base, []string{"footer"}, "hide_credit", true),
		"a developer field at the top":    withField(t, base, nil, "developer", map[string]any{"label": "x"}),
		"a developed_by field in the seo": withField(t, base, []string{"seo"}, "developed_by", "x"),
		"a show_credit switch in home":    withField(t, base, []string{"home"}, "show_credit", false),
		"a CoordiNet override":            withField(t, base, []string{"header"}, "coordinet_label", "x"),
	}
	for name, raw := range attempts {
		if _, err := Parse(raw); !errors.Is(err, ErrCredit) {
			t.Errorf("%s returned %v, want the fixed credit error", name, err)
		}
	}
	hidden := base
	hidden.Footer = Footer{Visible: false}
	content, err := Parse(encoded(t, hidden))
	if err != nil {
		t.Fatalf("an empty hidden footer was refused: %v", err)
	}
	raw := string(encoded(t, content))
	if strings.Contains(strings.ToLower(raw), "credit") || strings.Contains(raw, CreditName) {
		t.Fatalf("the stored content carries the credit, so the editor could change it: %s", raw)
	}
	if CreditLabel != "Developed by CoordiNet" || CreditPath != "/about-coordinet" {
		t.Fatalf("the fixed credit changed: %q at %q", CreditLabel, CreditPath)
	}
	custom := base
	custom.Footer.Note = "Developed by somebody else entirely"
	if _, err := Parse(encoded(t, custom)); err != nil {
		t.Fatalf("custom footer text was refused: %v", err)
	}
}

func TestUnknownFieldsAndRawHTMLAreRefused(t *testing.T) {
	base := Default()
	if _, err := Parse(withField(t, base, []string{"home"}, "script", "alert(1)")); err == nil {
		t.Error("an unknown field was accepted")
	}
	if _, err := Parse([]byte(`{"header":`)); err == nil {
		t.Error("broken JSON was accepted")
	}
	html := base
	html.Home.HeroTitle = `<script>alert(1)</script>`
	if _, err := Parse(encoded(t, html)); err == nil {
		t.Error("a title with HTML was accepted")
	}
	long := base
	long.Header.SiteName = strings.Repeat("a", MaxShort+1)
	if _, err := Parse(encoded(t, long)); err == nil {
		t.Error("a site name over the limit was accepted")
	}
	empty := base
	empty.Header.SiteName = "  "
	if _, err := Parse(encoded(t, empty)); err == nil {
		t.Error("an empty site name was accepted")
	}
	if _, err := Parse([]byte(strings.Repeat(" ", MaxContentSize+1))); err == nil {
		t.Error("content over the size limit was accepted")
	}
}

func TestLinksAreHTTPOrSitePaths(t *testing.T) {
	good := []string{"https://www.sust.edu", "http://example.org/a?b=1", "/p/admission", "/student/login"}
	for _, value := range good {
		if !ValidLink(value) {
			t.Errorf("ValidLink(%q) = false, want true", value)
		}
	}
	bad := []string{"", "javascript:alert(1)", "//evil.example", "data:text/html,x", "mailto:a@b.c",
		"ftp://example.org", "https://", "p/admission", "/a b", `/a"onclick`, "https://a.b/<x>"}
	for _, value := range bad {
		if ValidLink(value) {
			t.Errorf("ValidLink(%q) = true, want false", value)
		}
	}
	content := Default()
	content.Header.Menu = []Link{{Label: "Admission", URL: "javascript:alert(1)"}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("a menu link with a script address was accepted")
	}
	content.Header.Menu = []Link{{Label: "Admission", URL: "/p/admission"}}
	content.Footer.QuickLinks = []Link{{Label: "University", URL: "https://www.sust.edu"}}
	content.Footer.SocialLinks = []Link{{Label: "Facebook", URL: "https://facebook.com/sust"}}
	clean, err := Parse(encoded(t, content))
	if err != nil {
		t.Fatalf("good links were refused: %v", err)
	}
	if len(clean.Header.Menu) != 1 || len(clean.Footer.QuickLinks) != 1 || len(clean.Footer.SocialLinks) != 1 {
		t.Fatalf("the extra links were lost: %+v", clean.Footer)
	}
	content.Home.AboutText = "Read [the rules](javascript:alert(1))"
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("formatted text with a script link was accepted")
	}
	content.Home.AboutText = "Read **the** *rules* at [the office](https://www.sust.edu)\n- one\n- two"
	if _, err := Parse(encoded(t, content)); err != nil {
		t.Errorf("formatted text with bold, italic, a list and a link was refused: %v", err)
	}
}

func TestImagesMustBeUploadedOrBuiltIn(t *testing.T) {
	good := []string{"/uploads/photos/0123456789abcdef0123456789abcdef.jpg", "/brand/sust-logo.png",
		"/campus/hero-building.jpg"}
	for _, value := range good {
		if !ValidImage(value) {
			t.Errorf("ValidImage(%q) = false, want true", value)
		}
	}
	bad := []string{"", "https://evil.example/a.png", "/uploads/photos/", "/uploads/../secret", "//x/y.png",
		"/api/site/content", "data:image/png;base64,AAAA"}
	for _, value := range bad {
		if ValidImage(value) {
			t.Errorf("ValidImage(%q) = true, want false", value)
		}
	}
	content := Default()
	content.Home.Slides = nil
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("a slideshow without photos was accepted")
	}
	content.Home.Slides = []Slide{{ImageURL: "https://evil.example/a.png", Caption: "x"}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("a slideshow photo from another site was accepted")
	}
	content.Home.Slides = []Slide{{ImageURL: "/campus/hero-building.jpg", Caption: "IICT"}}
	content.Header.LogoURL = ""
	clean, err := Parse(encoded(t, content))
	if err != nil {
		t.Fatalf("one slide and no logo were refused: %v", err)
	}
	if clean.Header.LogoURL != "/brand/sust-logo.png" {
		t.Fatalf("an empty logo became %q, want the SUST logo", clean.Header.LogoURL)
	}
}

func TestEveryBlockTypeIsCheckedAndKept(t *testing.T) {
	picture := "/uploads/photos/0123456789abcdef0123456789abcdef.jpg"
	good := []Block{
		{Type: BlockHeading, Text: "Admission 2027"},
		{Type: BlockText, Text: "Apply **before** the deadline.\n- Form\n- Fee"},
		{Type: BlockImage, ImageURL: picture, Alt: "The IICT building"},
		{Type: BlockImageText, ImageURL: picture, Alt: "A lab", Text: "Our labs", Side: "right"},
		{Type: BlockButton, Label: "Apply now", URL: "https://admission.sust.edu"},
		{Type: BlockCards, Cards: []Card{{Title: "Orientation", Text: "For new students", Date: "2026-12-01", ImageURL: picture}}},
		{Type: BlockFAQ, FAQ: []Question{{Question: "Who can apply?", Answer: "Anyone with HSC."}}},
	}
	content := Default()
	content.Home.Custom = []CustomSection{{ID: "news1", Title: "News", Blocks: good}}
	content.Home.Sections = append([]SectionRef{{Key: "custom:news1", Visible: true}}, content.Home.Sections...)
	clean, err := Parse(encoded(t, content))
	if err != nil {
		t.Fatalf("a section with every block type was refused: %v", err)
	}
	kept := clean.Home.Custom[0].Blocks
	if len(kept) != len(good) {
		t.Fatalf("kept %d blocks, want %d", len(kept), len(good))
	}
	for index, block := range kept {
		if block.Type != good[index].Type {
			t.Errorf("block %d became %q, want %q", index, block.Type, good[index].Type)
		}
	}
	if kept[3].Side != "right" || kept[5].Cards[0].Date != "2026-12-01" || kept[6].FAQ[0].Answer == "" {
		t.Fatalf("the blocks lost their details: %+v", kept)
	}
	if clean.Home.Sections[0].Key != "custom:news1" || !clean.Home.Sections[0].Visible {
		t.Fatalf("the custom section lost its place in the home order: %+v", clean.Home.Sections)
	}
	bad := map[string]Block{
		"an unknown type":             {Type: "html", Text: "<b>x</b>"},
		"a heading without text":      {Type: BlockHeading},
		"text with a tag":             {Type: BlockText, Text: "<img src=x onerror=alert(1)>"},
		"an image without alt text":   {Type: BlockImage, ImageURL: picture},
		"an image from another site":  {Type: BlockImage, ImageURL: "https://x.y/z.png", Alt: "x"},
		"a button with a script link": {Type: BlockButton, Label: "Go", URL: "javascript:alert(1)"},
		"a button without a label":    {Type: BlockButton, URL: "/p/x"},
		"cards without a card":        {Type: BlockCards},
		"a card with a bad date":      {Type: BlockCards, Cards: []Card{{Title: "x", Date: "tomorrow"}}},
		"a question without answer":   {Type: BlockFAQ, FAQ: []Question{{Question: "Why?"}}},
	}
	for name, block := range bad {
		content.Home.Custom = []CustomSection{{ID: "news1", Title: "News", Blocks: []Block{block}}}
		if _, err := Parse(encoded(t, content)); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestSectionsCanBeOrderedHiddenAndNeverLost(t *testing.T) {
	content := Default()
	content.Home.Sections = []SectionRef{{Key: SectionApp, Visible: true}, {Key: SectionStats, Visible: false}}
	clean, err := Parse(encoded(t, content))
	if err != nil {
		t.Fatalf("a shorter home order was refused: %v", err)
	}
	if clean.Home.Sections[0].Key != SectionApp || clean.Home.Sections[1].Visible {
		t.Fatalf("the order or the hidden section changed: %+v", clean.Home.Sections)
	}
	if len(clean.Home.Sections) != 4 {
		t.Fatalf("sections left out of the order were dropped: %+v", clean.Home.Sections)
	}
	content.Home.Sections = []SectionRef{{Key: "custom:ghost", Visible: true}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("an order naming a missing section was accepted")
	}
	content.Home.Sections = []SectionRef{{Key: SectionApp, Visible: true}, {Key: SectionApp, Visible: true}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("an order naming a section twice was accepted")
	}
	content.Home.Sections = nil
	content.Home.Custom = []CustomSection{{ID: "a1", Title: "A"}, {ID: "a1", Title: "B"}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("two custom sections with one id were accepted")
	}
}

func TestPageAddressesAreUniqueAndNeverReplaceAnAppRoute(t *testing.T) {
	good := []string{"admission", "admission-2027", "a", "lab-rules-1"}
	for _, slug := range good {
		if !ValidSlug(slug) {
			t.Errorf("ValidSlug(%q) = false, want true", slug)
		}
	}
	bad := []string{"", "Admission", "admission 2027", "-admission", "admission-", "a--b", "a/b", "../x",
		"student", "teacher", "cr", "office-portal", "control-room", "api", "uploads", "profile", "p",
		"about-coordinet", strings.Repeat("a", 61)}
	for _, slug := range bad {
		if ValidSlug(slug) {
			t.Errorf("ValidSlug(%q) = true, want false", slug)
		}
	}
	content := Default()
	content.Pages = []Page{
		{Slug: "admission", Title: "Admission", SEOTitle: "Admission · SUST CSE", SEODescription: "How to apply",
			Blocks: []Block{{Type: BlockHeading, Text: "How to apply"}}},
		{Slug: "labs", Title: "Labs"},
	}
	clean, err := Parse(encoded(t, content))
	if err != nil {
		t.Fatalf("two good pages were refused: %v", err)
	}
	if len(clean.Pages) != 2 || clean.Pages[0].SEOTitle != "Admission · SUST CSE" {
		t.Fatalf("the pages lost their fields: %+v", clean.Pages)
	}
	content.Pages = append(content.Pages, Page{Slug: "admission", Title: "Again"})
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("two pages with one address were accepted")
	}
	content.Pages = []Page{{Slug: "student", Title: "Students"}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("a page named like an app route was accepted")
	}
	content.Pages = []Page{{Slug: "labs"}}
	if _, err := Parse(encoded(t, content)); err == nil {
		t.Error("a page without a title was accepted")
	}
}
