package sitecontent

import "strings"

var builtInSections = []string{SectionStats, SectionAbout, SectionNotices, SectionApp}

func Validate(content Content) (Content, error) {
	var err error
	out := Content{}
	if out.Header, err = header(content.Header); err != nil {
		return Content{}, err
	}
	if out.Home, err = home(content.Home); err != nil {
		return Content{}, err
	}
	if out.Footer, err = footer(content.Footer); err != nil {
		return Content{}, err
	}
	if out.SignIn.Title, err = line("The sign-in title", content.SignIn.Title, MaxShort, false); err != nil {
		return Content{}, err
	}
	if out.SignIn.Text, err = line("The sign-in text", content.SignIn.Text, MaxLine, false); err != nil {
		return Content{}, err
	}
	if out.SEO.Title, err = line("The page title", content.SEO.Title, MaxShort, true); err != nil {
		return Content{}, err
	}
	if out.SEO.Description, err = line("The page description", content.SEO.Description, MaxLine, false); err != nil {
		return Content{}, err
	}
	if out.SEO.ImageURL, err = image("The share image", content.SEO.ImageURL, false); err != nil {
		return Content{}, err
	}
	if out.Pages, err = pages(content.Pages); err != nil {
		return Content{}, err
	}
	return out, nil
}

func header(value Header) (Header, error) {
	var err error
	out := Header{}
	if out.SiteName, err = line("The site name", value.SiteName, MaxShort, true); err != nil {
		return Header{}, err
	}
	if out.Tagline, err = line("The tagline", value.Tagline, MaxLine, false); err != nil {
		return Header{}, err
	}
	if out.LogoURL, err = image("The logo", value.LogoURL, false); err != nil {
		return Header{}, err
	}
	if out.LogoURL == "" {
		out.LogoURL = Default().Header.LogoURL
	}
	if out.Menu, err = links("The header menu", value.Menu); err != nil {
		return Header{}, err
	}
	return out, nil
}

func home(value Home) (Home, error) {
	var err error
	out := Home{}
	if out.HeroEyebrow, err = line("The hero line", value.HeroEyebrow, MaxShort, false); err != nil {
		return Home{}, err
	}
	if out.HeroTitle, err = line("The hero title", value.HeroTitle, MaxLine, true); err != nil {
		return Home{}, err
	}
	if out.HeroText, err = line("The hero text", value.HeroText, MaxText, false); err != nil {
		return Home{}, err
	}
	if len(value.Slides) == 0 || len(value.Slides) > MaxSlides {
		return Home{}, problem("The slideshow needs between 1 and %d photos.", MaxSlides)
	}
	out.Slides = make([]Slide, 0, len(value.Slides))
	for _, slide := range value.Slides {
		picture, err := image("A slideshow photo", slide.ImageURL, true)
		if err != nil {
			return Home{}, err
		}
		caption, err := line("A slideshow caption", slide.Caption, MaxLine, false)
		if err != nil {
			return Home{}, err
		}
		out.Slides = append(out.Slides, Slide{ImageURL: picture, Caption: caption})
	}
	if out.PrimaryCTA, err = link("The first button", value.PrimaryCTA, false); err != nil {
		return Home{}, err
	}
	if out.SecondaryCTA, err = link("The second button", value.SecondaryCTA, false); err != nil {
		return Home{}, err
	}
	if out.AboutTitle, err = line("The about title", value.AboutTitle, MaxLine, false); err != nil {
		return Home{}, err
	}
	if out.AboutText, err = paragraph("The about text", value.AboutText, MaxText); err != nil {
		return Home{}, err
	}
	if len(value.Custom) > MaxCustom {
		return Home{}, problem("The home page can hold at most %d custom sections.", MaxCustom)
	}
	known := make(map[string]bool)
	for _, key := range builtInSections {
		known[key] = true
	}
	out.Custom = make([]CustomSection, 0, len(value.Custom))
	for _, section := range value.Custom {
		if !idPattern.MatchString(section.ID) || known[CustomPrefix+section.ID] {
			return Home{}, problem("A custom section has a missing or repeated id.")
		}
		title, err := line("A custom section title", section.Title, MaxLine, true)
		if err != nil {
			return Home{}, err
		}
		inner, err := blocks("the section "+title, section.Blocks)
		if err != nil {
			return Home{}, err
		}
		known[CustomPrefix+section.ID] = true
		out.Custom = append(out.Custom, CustomSection{ID: section.ID, Title: title, Blocks: inner})
	}
	seen := make(map[string]bool)
	out.Sections = make([]SectionRef, 0, len(known))
	for _, ref := range value.Sections {
		key := strings.TrimSpace(ref.Key)
		if !known[key] {
			return Home{}, problem("The home order names a section that does not exist.")
		}
		if seen[key] {
			return Home{}, problem("The home order names a section twice.")
		}
		seen[key] = true
		out.Sections = append(out.Sections, SectionRef{Key: key, Visible: ref.Visible})
	}
	for _, key := range builtInSections {
		if !seen[key] {
			out.Sections = append(out.Sections, SectionRef{Key: key, Visible: false})
		}
	}
	for _, section := range out.Custom {
		if !seen[CustomPrefix+section.ID] {
			out.Sections = append(out.Sections, SectionRef{Key: CustomPrefix + section.ID, Visible: false})
		}
	}
	return out, nil
}

func footer(value Footer) (Footer, error) {
	var err error
	out := Footer{Visible: value.Visible}
	if out.Address, err = paragraph("The address", value.Address, MaxLine); err != nil {
		return Footer{}, err
	}
	if out.Phone, err = line("The phone number", value.Phone, MaxShort, false); err != nil {
		return Footer{}, err
	}
	if out.Email, err = line("The email address", value.Email, MaxShort, false); err != nil {
		return Footer{}, err
	}
	if out.Email != "" && (!strings.Contains(out.Email, "@") || strings.ContainsAny(out.Email, " /:")) {
		return Footer{}, problem("The email address does not look like one.")
	}
	if out.OfficeHours, err = line("The office hours", value.OfficeHours, MaxLine, false); err != nil {
		return Footer{}, err
	}
	if out.Note, err = line("The footer text", value.Note, MaxLine, false); err != nil {
		return Footer{}, err
	}
	if out.QuickLinks, err = links("The quick links", value.QuickLinks); err != nil {
		return Footer{}, err
	}
	if out.SocialLinks, err = links("The social links", value.SocialLinks); err != nil {
		return Footer{}, err
	}
	return out, nil
}

func pages(values []Page) ([]Page, error) {
	if len(values) > MaxPages {
		return nil, problem("The website can hold at most %d pages.", MaxPages)
	}
	seen := make(map[string]bool)
	cleaned := make([]Page, 0, len(values))
	for _, value := range values {
		slug := strings.TrimSpace(value.Slug)
		if !ValidSlug(slug) {
			return nil, problem(
				"The address /p/%s cannot be used. Use small letters, digits and dashes, and not a name the portal already uses.",
				slug)
		}
		if seen[slug] {
			return nil, problem("Two pages share the address /p/%s.", slug)
		}
		seen[slug] = true
		title, err := line("The title of /p/"+slug, value.Title, MaxLine, true)
		if err != nil {
			return nil, err
		}
		seoTitle, err := line("The page title of /p/"+slug, value.SEOTitle, MaxShort, false)
		if err != nil {
			return nil, err
		}
		description, err := line("The description of /p/"+slug, value.SEODescription, MaxLine, false)
		if err != nil {
			return nil, err
		}
		inner, err := blocks("/p/"+slug, value.Blocks)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, Page{
			Slug: slug, Title: title, SEOTitle: seoTitle, SEODescription: description, Blocks: inner,
		})
	}
	return cleaned, nil
}
