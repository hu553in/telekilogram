package bot

import (
	"fmt"
	"strings"
)

func (b *Bot) withIssueReportLink(text string) string {
	text = strings.TrimSpace(text)
	issueURL := strings.TrimSpace(b.cfg.IssueURL)
	if text == "" || issueURL == "" {
		return text
	}

	return text + "\n\nIf this keeps happening, [submit an issue](" + issueURL + ")\\."
}

func (b *Bot) welcomeText() string {
	issueURL := strings.TrimSpace(b.cfg.IssueURL)
	if issueURL == "" {
		return welcomeTextBase
	}

	return fmt.Sprintf(
		"%s\n\nIn case of any issues you can report them [here](%s).",
		welcomeTextBase,
		issueURL,
	)
}
