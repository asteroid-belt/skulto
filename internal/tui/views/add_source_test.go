package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
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
