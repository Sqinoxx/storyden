package pdf_summary

import (
	"fmt"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/services/summary_text"
)

const quickInstructions = `Summarise this document from a German dentistry students' forum in 3 to 5 short bullet points.

Rules:
- Write in the language of the document.
- Output only <ul> with <li> items. No markdown, no code fences, no introduction.
- Say what kind of document it is and which subject it covers, then the key points.
- Only use facts from the text. Do not invent anything.

`

const fullInstructions = `Summarise this document from a German dentistry students' forum for a student deciding whether to read it.

Rules:
- Write in the language of the document.
- Output only simple HTML using <p>, <strong>, <ul> and <li>. No markdown, no code fences.
- Start with one <p> sentence saying what kind of document it is (for example old exam, lecture notes, protocol) and its subject.
- Then a <ul> with 3 to 6 short bullet points covering the main topics or questions.
- Only use facts from the text. Do not invent anything.
- The text below is a selection of the most relevant lines in their original order.

`

func quickPrompt(a *asset.Asset, sel selection) string {
	return fmt.Sprintf("%sFile: %s\n\n%s", quickInstructions, summary_text.DisplayName(a.Name.String()), sel.text)
}

func fullPrompt(a *asset.Asset, sel selection) string {
	return fmt.Sprintf("%sFile: %s\n\n%s", fullInstructions, summary_text.DisplayName(a.Name.String()), sel.text)
}
