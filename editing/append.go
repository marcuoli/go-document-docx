package editing

// Append adds a styled literal run. It does not translate or parse its text.
// Whitespace remains meaningful; newlines become soft breaks on serialization.
func (p *Paragraph) Append(text string, style TextStyle) error {
	if err := validText(text); err != nil {
		return err
	}
	if err := validateStyle(style); err != nil {
		return err
	}
	text = normalizedBreaks(text)
	index := len(p.native.Runs())
	run, err := p.native.AddRun()
	if err != nil {
		return err
	}
	if err := run.SetText(text); err != nil {
		return err
	}
	p.styles[index] = style
	return styleRun(run, style)
}
