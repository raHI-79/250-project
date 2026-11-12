package sitecontent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	CreditName  = "CoordiNet"
	CreditLabel = "Developed by CoordiNet"
	CreditPath  = "/about-coordinet"

	BlockHeading   = "heading"
	BlockText      = "text"
	BlockImage     = "image"
	BlockImageText = "image_text"
	BlockButton    = "button"
	BlockCards     = "cards"
	BlockFAQ       = "faq"

	SectionStats   = "stats"
	SectionNotices = "notices"
	SectionApp     = "app"
	SectionAbout   = "about"
	CustomPrefix   = "custom:"

	MaxShort       = 120
	MaxLine        = 300
	MaxText        = 4000
	MaxLinks       = 12
	MaxSlides      = 10
	MaxBlocks      = 40
	MaxItems       = 24
	MaxCustom      = 20
	MaxPages       = 40
	MaxContentSize = 512 << 10
)

var (
	ErrCredit = errors.New(
		"The developer credit is fixed and cannot be changed, hidden or replaced from the editor.")
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	idPattern   = regexp.MustCompile(`^[a-z0-9]{1,24}$`)
	datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	linkPattern = regexp.MustCompile(`\[([^\]\n]*)\]\(([^)\s]*)\)`)
)

var creditWords = []string{"credit", "developer", "developed_by", "developedby", "coordinet", "powered_by"}

var reservedSlugs = []string{
	"student", "teacher", "cr", "office-portal", "control-room", "profile", "people", "api",
	"uploads", "download", "p", "about-coordinet", "login", "register", "admin", "brand", "campus",
	"assets", "index", "sw", "manifest",
}

type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Slide struct {
	ImageURL string `json:"image_url"`
	Caption  string `json:"caption"`
}

type Card struct {
	Title    string `json:"title"`
	Text     string `json:"text"`
	Date     string `json:"date"`
	ImageURL string `json:"image_url"`
}

type Question struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Block struct {
	Type     string     `json:"type"`
	Text     string     `json:"text,omitempty"`
	ImageURL string     `json:"image_url,omitempty"`
	Alt      string     `json:"alt,omitempty"`
	Side     string     `json:"side,omitempty"`
	Label    string     `json:"label,omitempty"`
	URL      string     `json:"url,omitempty"`
	Cards    []Card     `json:"cards,omitempty"`
	FAQ      []Question `json:"faq,omitempty"`
}

type Header struct {
	SiteName string `json:"site_name"`
	Tagline  string `json:"tagline"`
	LogoURL  string `json:"logo_url"`
	Menu     []Link `json:"menu"`
}

type SectionRef struct {
	Key     string `json:"key"`
	Visible bool   `json:"visible"`
}

type CustomSection struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Blocks []Block `json:"blocks"`
}

type Home struct {
	HeroEyebrow  string          `json:"hero_eyebrow"`
	HeroTitle    string          `json:"hero_title"`
	HeroText     string          `json:"hero_text"`
	Slides       []Slide         `json:"slides"`
	PrimaryCTA   Link            `json:"primary_cta"`
	SecondaryCTA Link            `json:"secondary_cta"`
	AboutTitle   string          `json:"about_title"`
	AboutText    string          `json:"about_text"`
	Sections     []SectionRef    `json:"sections"`
	Custom       []CustomSection `json:"custom"`
}

type Footer struct {
	Visible     bool   `json:"visible"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	OfficeHours string `json:"office_hours"`
	Note        string `json:"note"`
	QuickLinks  []Link `json:"quick_links"`
	SocialLinks []Link `json:"social_links"`
}

type SignIn struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type SEO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

type Page struct {
	Slug           string  `json:"slug"`
	Title          string  `json:"title"`
	SEOTitle       string  `json:"seo_title"`
	SEODescription string  `json:"seo_description"`
	Blocks         []Block `json:"blocks"`
}

type Content struct {
	Header Header `json:"header"`
	Home   Home   `json:"home"`
	Footer Footer `json:"footer"`
	SignIn SignIn `json:"sign_in"`
	SEO    SEO    `json:"seo"`
	Pages  []Page `json:"pages"`
}

func Default() Content {
	return Content{
		Header: Header{
			SiteName: "SUST CSE",
			Tagline:  "Department of Computer Science and Engineering",
			LogoURL:  "/brand/sust-logo.png",
			Menu:     []Link{},
		},
		Home: Home{
			HeroEyebrow: "Established 1991 · Sylhet, Bangladesh",
			HeroTitle:   "Class and exam coordination for the CSE department",
			HeroText: "One record for the weekly routine, rescheduled classes, exams and notices. " +
				"Class representatives draft the plan, teachers confirm it, and every student reads " +
				"the published result.",
			Slides: []Slide{
				{ImageURL: "/campus/hero-oldGate.jpg", Caption: "SUST Iconic Old Gate, Kumargaon"},
				{ImageURL: "/campus/hero-building.jpg", Caption: "IICT Building, SUST campus"},
				{ImageURL: "/campus/hero-monument.jpg", Caption: "Shaheed Minar, SUST campus"},
				{ImageURL: "/campus/hero-entrance.jpg", Caption: "University main entrance, Kumargaon"},
			},
			PrimaryCTA:   Link{Label: "Student sign in", URL: "/student/login"},
			SecondaryCTA: Link{Label: "Teacher sign in", URL: "/teacher/login"},
			AboutTitle:   "Everything a class day needs, in one record",
			AboutText: "The department runs its weekly routine, the classes and exams ahead, teacher " +
				"confirmations, notices, materials, polls and messages here.",
			Sections: []SectionRef{
				{Key: SectionStats, Visible: true},
				{Key: SectionAbout, Visible: true},
				{Key: SectionNotices, Visible: false},
				{Key: SectionApp, Visible: true},
			},
			Custom: []CustomSection{},
		},
		Footer: Footer{
			Visible:     true,
			Address:     "Academic Building A, University Avenue\nKumargaon, Sylhet 3114\nBangladesh",
			Note:        "For class and exam coordination inside the department only.",
			QuickLinks:  []Link{},
			SocialLinks: []Link{},
		},
		SignIn: SignIn{},
		SEO: SEO{
			Title:       "SUST CSE · Class and exam coordination",
			Description: "The weekly routine, classes, exams and notices of the CSE department of SUST.",
		},
		Pages: []Page{},
	}
}

func problem(format string, args ...any) error {
	return errors.New(fmt.Sprintf(format, args...))
}

func creditKey(key string) bool {
	lowered := strings.ToLower(key)
	for _, word := range creditWords {
		if strings.Contains(lowered, word) {
			return true
		}
	}
	return false
}

func findCreditKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, inner := range typed {
			if creditKey(key) || findCreditKey(inner) {
				return true
			}
		}
	case []any:
		for _, inner := range typed {
			if findCreditKey(inner) {
				return true
			}
		}
	}
	return false
}

func Parse(raw []byte) (Content, error) {
	if len(raw) > MaxContentSize {
		return Content{}, problem("The website content is too large to save.")
	}
	var loose any
	if err := json.Unmarshal(raw, &loose); err != nil {
		return Content{}, problem("The website content could not be read.")
	}
	if findCreditKey(loose) {
		return Content{}, ErrCredit
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var content Content
	if err := decoder.Decode(&content); err != nil {
		return Content{}, problem("The website content has a field the editor does not know.")
	}
	clean, err := Validate(content)
	if err != nil {
		return Content{}, err
	}
	return clean, nil
}

func line(name, value string, limit int, required bool) (string, error) {
	cleaned := strings.Join(strings.Fields(value), " ")
	if required && cleaned == "" {
		return "", problem("%s is required.", name)
	}
	if utf8.RuneCountInString(cleaned) > limit {
		return "", problem("%s is longer than %d characters.", name, limit)
	}
	if strings.ContainsAny(cleaned, "<>") {
		return "", problem("%s cannot contain < or >.", name)
	}
	return cleaned, nil
}

func paragraph(name, value string, limit int) (string, error) {
	cleaned := strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if utf8.RuneCountInString(cleaned) > limit {
		return "", problem("%s is longer than %d characters.", name, limit)
	}
	if strings.ContainsAny(cleaned, "<>") {
		return "", problem("%s cannot contain < or >. Use **bold**, *italic*, lists and [links](https://…).", name)
	}
	for _, match := range linkPattern.FindAllStringSubmatch(cleaned, -1) {
		if !ValidLink(match[2]) {
			return "", problem("%s has a link that is not http(s) or a page of this site.", name)
		}
	}
	return cleaned, nil
}

func ValidLink(value string) bool {
	link := strings.TrimSpace(value)
	if link == "" || len(link) > MaxLine || strings.ContainsAny(link, " \t\n\"'<>\\") {
		return false
	}
	lowered := strings.ToLower(link)
	if strings.HasPrefix(lowered, "https://") || strings.HasPrefix(lowered, "http://") {
		return len(link) > len("https://")
	}
	return strings.HasPrefix(link, "/") && !strings.HasPrefix(link, "//")
}

func ValidImage(value string) bool {
	image := strings.TrimSpace(value)
	if image == "" || len(image) > MaxLine || strings.ContainsAny(image, " \t\n\"'<>\\") {
		return false
	}
	if strings.Contains(image, "..") || strings.HasPrefix(image, "//") {
		return false
	}
	for _, prefix := range []string{"/uploads/photos/", "/brand/", "/campus/"} {
		if strings.HasPrefix(image, prefix) && len(image) > len(prefix) {
			return true
		}
	}
	return false
}

func image(name, value string, required bool) (string, error) {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" && !required {
		return "", nil
	}
	if !ValidImage(cleaned) {
		return "", problem("%s must be an image uploaded here.", name)
	}
	return cleaned, nil
}

func link(name string, value Link, required bool) (Link, error) {
	label, err := line(name+" text", value.Label, MaxShort, false)
	if err != nil {
		return Link{}, err
	}
	target := strings.TrimSpace(value.URL)
	if label == "" && target == "" && !required {
		return Link{}, nil
	}
	if label == "" {
		return Link{}, problem("%s needs its text.", name)
	}
	if !ValidLink(target) {
		return Link{}, problem("%s must link to http(s) or to a page of this site, like /p/admission.", name)
	}
	return Link{Label: label, URL: target}, nil
}

func links(name string, values []Link) ([]Link, error) {
	if len(values) > MaxLinks {
		return nil, problem("%s can hold at most %d links.", name, MaxLinks)
	}
	cleaned := make([]Link, 0, len(values))
	for _, value := range values {
		entry, err := link(name, value, true)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned, nil
}

func ValidSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 60 || !slugPattern.MatchString(slug) {
		return false
	}
	for _, reserved := range reservedSlugs {
		if slug == reserved {
			return false
		}
	}
	return true
}

func blocks(name string, values []Block) ([]Block, error) {
	if len(values) > MaxBlocks {
		return nil, problem("%s can hold at most %d blocks.", name, MaxBlocks)
	}
	cleaned := make([]Block, 0, len(values))
	for index, value := range values {
		entry, err := block(fmt.Sprintf("Block %d of %s", index+1, name), value)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, entry)
	}
	return cleaned, nil
}

func block(name string, value Block) (Block, error) {
	var err error
	out := Block{Type: value.Type}
	switch value.Type {
	case BlockHeading:
		out.Text, err = line(name, value.Text, MaxLine, true)
	case BlockText:
		out.Text, err = paragraph(name, value.Text, MaxText)
		if err == nil && out.Text == "" {
			err = problem("%s needs its text.", name)
		}
	case BlockImage:
		if out.ImageURL, err = image(name, value.ImageURL, true); err != nil {
			return Block{}, err
		}
		out.Alt, err = line(name+" description", value.Alt, MaxLine, true)
	case BlockImageText:
		if out.ImageURL, err = image(name, value.ImageURL, true); err != nil {
			return Block{}, err
		}
		if out.Alt, err = line(name+" description", value.Alt, MaxLine, true); err != nil {
			return Block{}, err
		}
		out.Side = "left"
		if value.Side == "right" {
			out.Side = "right"
		}
		out.Text, err = paragraph(name, value.Text, MaxText)
	case BlockButton:
		var target Link
		target, err = link(name, Link{Label: value.Label, URL: value.URL}, true)
		out.Label, out.URL = target.Label, target.URL
	case BlockCards:
		out.Cards, err = cards(name, value.Cards)
	case BlockFAQ:
		out.FAQ, err = questions(name, value.FAQ)
	default:
		return Block{}, problem("%s has an unknown type.", name)
	}
	if err != nil {
		return Block{}, err
	}
	return out, nil
}

func cards(name string, values []Card) ([]Card, error) {
	if len(values) == 0 || len(values) > MaxItems {
		return nil, problem("%s needs between 1 and %d cards.", name, MaxItems)
	}
	cleaned := make([]Card, 0, len(values))
	for _, value := range values {
		title, err := line(name+" card title", value.Title, MaxLine, true)
		if err != nil {
			return nil, err
		}
		text, err := paragraph(name+" card text", value.Text, MaxText)
		if err != nil {
			return nil, err
		}
		date := strings.TrimSpace(value.Date)
		if date != "" && !datePattern.MatchString(date) {
			return nil, problem("%s has a card date that is not like 2026-11-14.", name)
		}
		picture, err := image(name+" card image", value.ImageURL, false)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, Card{Title: title, Text: text, Date: date, ImageURL: picture})
	}
	return cleaned, nil
}

func questions(name string, values []Question) ([]Question, error) {
	if len(values) == 0 || len(values) > MaxItems {
		return nil, problem("%s needs between 1 and %d questions.", name, MaxItems)
	}
	cleaned := make([]Question, 0, len(values))
	for _, value := range values {
		question, err := line(name+" question", value.Question, MaxLine, true)
		if err != nil {
			return nil, err
		}
		answer, err := paragraph(name+" answer", value.Answer, MaxText)
		if err != nil {
			return nil, err
		}
		if answer == "" {
			return nil, problem("%s has a question without an answer.", name)
		}
		cleaned = append(cleaned, Question{Question: question, Answer: answer})
	}
	return cleaned, nil
}
