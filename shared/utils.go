package shared

func StringAddr(b string) *string {
	stringVar := b
	return &stringVar
}

func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func pointerOrDefaultString(p *string, def string) string {
	if p != nil {
		return *p
	}

	return def
}

func pointerOrDefaultInt(p *int, def int) int {
	if p != nil {
		return *p
	}

	return def
}

func pointerOrDefaultBool(p *bool, def bool) bool {
	if p != nil {
		return *p
	}

	return def
}
