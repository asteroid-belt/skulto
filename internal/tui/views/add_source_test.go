package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestAddSourceView_IsAcceptingTextInput_followsInputFocus(t *testing.T) {
	asv := NewAddSourceView(nil, &config.Config{})

	// Before Init, the input is not focused
	assert.False(t, asv.IsAcceptingTextInput())

	// Init focuses the input
	asv.Init()
	assert.True(t, asv.IsAcceptingTextInput())

	// Manually blur
	asv.input.Blur()
	assert.False(t, asv.IsAcceptingTextInput())
}

func TestAddSourceView_UpdateKey_BracketedPasteNormalizesQueryAndFragment(t *testing.T) {
	asv := NewAddSourceView(nil, &config.Config{})
	asv.Init()

	back, ok := asv.UpdateKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("https://github.com/owner/repo?ref=main#section"),
		Paste: true,
	})

	assert.False(t, back)
	assert.False(t, ok)
	assert.Equal(t, "https://github.com/owner/repo", asv.input.Value())
	assert.Equal(t, "", asv.error)
}

func TestAddSourceView_UpdateKey_BracketedPasteTruncatesExtraPath(t *testing.T) {
	asv := NewAddSourceView(nil, &config.Config{})
	asv.Init()

	_, _ = asv.UpdateKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("https://github.com/owner/repo/tree/main"),
		Paste: true,
	})

	assert.Equal(t, "https://github.com/owner/repo", asv.input.Value())
	assert.Equal(t, "", asv.error)
}

func TestAddSourceView_UpdateKey_BracketedPasteInvalidDoesNotOverrideInput(t *testing.T) {
	asv := NewAddSourceView(nil, &config.Config{})
	asv.Init()
	asv.input.SetValue("https://github.com/existing/repo")

	_, _ = asv.UpdateKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("not a url"),
		Paste: true,
	})

	assert.Equal(t, "https://github.com/existing/repo", asv.input.Value())
	assert.Contains(t, asv.error, "Invalid pasted URL")
}

func TestAddSourceView_UpdateKey_BracketedPasteClearsPreviousError(t *testing.T) {
	asv := NewAddSourceView(nil, &config.Config{})
	asv.Init()
	asv.error = "some prior error"

	_, _ = asv.UpdateKey(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune("https://gitlab.com/group/project"),
		Paste: true,
	})

	assert.Equal(t, "https://gitlab.com/group/project", asv.input.Value())
	assert.Equal(t, "", asv.error)
}
