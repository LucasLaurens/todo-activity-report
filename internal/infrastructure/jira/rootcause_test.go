package jira

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSplitRootCauseExtractsCauseAndCorrection(t *testing.T) {
	t.Parallel()
	// Arrange — the template flattened by extractADFText: one section per line.
	text := strings.Join([]string{
		"🎯 Problème",
		"Le compteur affichait 11 pièces alors que le profil en montrait 0.",
		"💥 Cause",
		"Deux endpoints filtrent le statut du programme différemment.",
		"🧑‍💻 Correction",
		"- Back : ajout du filtre status ∈ [ACTIVE, COMPLETED].",
		"- Front : v-if total > 0 masque la carte vide.",
		"🔒 Prévention",
		"Tests de non-régression back + front.",
	}, "\n")

	// Act
	rootCause, correction := splitRootCause(text)

	// Assert
	if rootCause != "Deux endpoints filtrent le statut du programme différemment." {
		t.Fatalf("unexpected root cause: %q", rootCause)
	}
	wantFix := "- Back : ajout du filtre status ∈ [ACTIVE, COMPLETED]. - Front : v-if total > 0 masque la carte vide."
	if correction != wantFix {
		t.Fatalf("unexpected correction: %q", correction)
	}
}

func TestSplitRootCauseIgnoresVulgariseVariants(t *testing.T) {
	t.Parallel()
	// Arrange — technical Cause/Correction kept, the lay duplicate dropped.
	text := strings.Join([]string{
		"💥 Cause (technique)",
		"strip_tags() retirait toutes les balises HTML.",
		"🧠 Cause vulgarisée",
		"Le texte était nettoyé trop agressivement.",
		"🧑‍💻 Correction",
		"strip_tags($content, '<a>').",
		"🛠️ Correction vulgarisée",
		"On garde les liens.",
	}, "\n")

	// Act
	rootCause, correction := splitRootCause(text)

	// Assert
	if rootCause != "strip_tags() retirait toutes les balises HTML." {
		t.Fatalf("expected technical cause only, got %q", rootCause)
	}
	if correction != "strip_tags($content, '<a>')." {
		t.Fatalf("expected technical correction only, got %q", correction)
	}
}

func TestSplitRootCauseDropsUnfilledTemplate(t *testing.T) {
	t.Parallel()
	// Arrange — placeholder instructions never replaced by the author.
	text := strings.Join([]string{
		"🎯 Problème : description simple DU POINT DE VUE DU CLIENT",
		"💥 Cause : raison technique : Compréhensible par un non technique",
		"🧑‍💻 Correction : ce qui a été fait",
	}, "\n")

	// Act
	rootCause, correction := splitRootCause(text)

	// Assert
	if rootCause != "" || correction != "" {
		t.Fatalf("expected empty for unfilled template, got rc=%q fix=%q", rootCause, correction)
	}
}

func TestSplitRootCauseKeepsPlainTextVerbatim(t *testing.T) {
	t.Parallel()
	// Arrange — a non-template free-text field (no section headers).
	text := "Race condition on cache invalidation."

	// Act
	rootCause, correction := splitRootCause(text)

	// Assert
	if rootCause != text {
		t.Fatalf("expected verbatim root cause, got %q", rootCause)
	}
	if correction != "" {
		t.Fatalf("expected no correction, got %q", correction)
	}
}

func TestExtractADFTextPreservesSectionLineBreaks(t *testing.T) {
	t.Parallel()
	// Arrange — emoji+bold header, hard break, body, then a bulleted correction.
	adf := `{
		"type":"doc","content":[
			{"type":"paragraph","content":[
				{"type":"text","text":"💥 "},
				{"type":"text","text":"Cause","marks":[{"type":"strong"}]},
				{"type":"hardBreak"},
				{"type":"text","text":"Endpoints divergents."}
			]},
			{"type":"bulletList","content":[
				{"type":"listItem","content":[{"type":"paragraph","content":[
					{"type":"text","text":"Back : filtre ajouté."}
				]}]}
			]}
		]
	}`
	var node map[string]any
	if err := json.Unmarshal([]byte(adf), &node); err != nil {
		t.Fatalf("invalid ADF fixture: %v", err)
	}

	// Act
	got := extractADFText(node)

	// Assert
	want := "💥 Cause\nEndpoints divergents.\n- Back : filtre ajouté."
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
